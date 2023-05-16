package powerbi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"go.uber.org/zap"

	
)

const (
	reportsURI = "/reports"
	groupsURI  = "/groups"
)

// MyWorkspaceGroup group with Reports from My Workspace
var MyWorkspaceGroup = domain.Group{
	ID:   "",
	Name: "My Workspace",
}

// TokenCacheManager interface describe a contract for working with token cache
type TokenCacheManager interface {
	Get(ctx context.Context, consumerID interface{}) (domain.AccessData, error)
	Update(ctx context.Context, consumerID interface{}, accessData domain.AccessData) error
}

// ServiceClient is a service client for MS PowerBi
type ServiceClient struct {
	oauthConfig oauth.Config
	config      *config.PowerBiClientConfig
	tokenCache  TokenCacheManager
	logger      *zap.Logger
}

// NewServiceClient creates new instance of ServiceClient
func NewServiceClient(
	oauthConfig oauth.Config,
	config *config.PowerBiClientConfig,
	tokenCache TokenCacheManager,
	l *zap.Logger,
) *ServiceClient {
	return &ServiceClient{
		oauthConfig: oauthConfig,
		config:      config,
		tokenCache:  tokenCache,
		logger:      l,
	}
}

// GetGroupedReports returns reports grouped by groups (workspaces)
func (c *ServiceClient) GetGroupedReports(consumerID interface{}) (allReports domain.GroupedReports, err error) {
	allReports = domain.NewGroupedReports()
	var allReportsMutex = sync.Mutex{}

	errorsChan := make(chan error)
	w1 := sync.WaitGroup{}
	w1.Add(2)

	go func() {
		defer w1.Done()

		rs, err := c.GetReports(consumerID)
		if err != nil {
			c.logger.Error("couldn't get reports", zap.Error(err), zap.Any("consumerID", consumerID))
			errorsChan <- err

			return
		}

		if len(rs.Value) > 0 {
			allReportsMutex.Lock()
			allReports[&MyWorkspaceGroup] = rs
			allReportsMutex.Unlock()
		}
	}()

	go func() {
		defer w1.Done()

		gs, err := c.GetGroups(consumerID)
		if err != nil {
			c.logger.Error("couldn't get groups", zap.Error(err), zap.Any("consumerID", consumerID))
			errorsChan <- err

			return
		}

		w2 := sync.WaitGroup{}

		for _, g := range gs.Value {
			w2.Add(1)

			go func(g *domain.Group) {
				defer w2.Done()

				rs, err := c.GetReportsInGroup(consumerID, g.ID)
				if err != nil {
					c.logger.Error("couldn't get reports in group", zap.Error(err), zap.String("groupID", g.ID))
					errorsChan <- err

					return
				}

				if len(rs.Value) > 0 {
					allReportsMutex.Lock()
					allReports[g] = rs
					allReportsMutex.Unlock()
				}
			}(g)
		}

		w2.Wait()
	}()

	go func() {
		w1.Wait()
		close(errorsChan)
	}()

	err = <-errorsChan
	return allReports, err
}

// GetReport returns report by reportID
func (c *ServiceClient) GetReport(consumerID interface{}, reportID string) (*domain.Report, error) {
	r, err := c.get(consumerID, reportsURI+"/"+reportID, func(b io.ReadCloser) (interface{}, error) {
		return domain.DeserializeReport(b)
	})
	if err != nil {
		c.logger.Error("couldn't get report", zap.Error(err), zap.String("reportID", reportID), zap.Any("consumerID", consumerID))

		return nil, err
	}

	return r.(*domain.Report), nil
}

// GetReports returns reports from My Workspace
func (c *ServiceClient) GetReports(consumerID interface{}) (*domain.ReportsContainer, error) {
	r, err := c.get(consumerID, reportsURI, func(b io.ReadCloser) (interface{}, error) {
		return domain.DeserializeReports(b)
	})
	if err != nil {
		c.logger.Error("couldn't get reports", zap.Error(err), zap.Any("consumerID", consumerID))

		return nil, err
	}

	return r.(*domain.ReportsContainer), nil
}

// GetGroups returns reports groups
func (c *ServiceClient) GetGroups(consumerID interface{}) (*domain.Groups, error) {
	r, err := c.get(consumerID, groupsURI, func(b io.ReadCloser) (interface{}, error) {
		return domain.DeserializeGroups(b)
	})
	if err != nil {
		c.logger.Error("couldn't get groups", zap.Error(err))

		return nil, err
	}

	return r.(*domain.Groups), nil
}

// GetReportsInGroup returns reports in group
func (c *ServiceClient) GetReportsInGroup(consumerID interface{}, groupID string) (*domain.ReportsContainer, error) {
	r, err := c.get(consumerID, reportByGroupURI(groupID), func(b io.ReadCloser) (interface{}, error) {
		return domain.DeserializeReports(b)
	})
	if err != nil {
		c.logger.Error("couldn't get reports in group", zap.Error(err))

		return nil, err
	}

	return r.(*domain.ReportsContainer), nil
}

// GetPages retrieves report pages.
func (c *ServiceClient) GetPages(customerID interface{}, reportID string) (*domain.PagesContainer, error) {
	r, err := c.get(customerID, pagesURI(reportID), func(b io.ReadCloser) (interface{}, error) {
		return domain.DeserializePagesContainer(b)
	})
	if err != nil {
		c.logger.Error("couldn't get pages", zap.Error(err), zap.String("reportID", reportID), zap.Any("customerID", customerID))

		return nil, err
	}

	return r.(*domain.PagesContainer), nil
}

// RefreshTokens implements "refresh_token" grant type.
func (c *ServiceClient) RefreshTokens(refreshToken string) (domain.AccessData, error) {
	headers := map[string]string{
		constants.HTTPHeaderContentType: constants.MIMETypeURLEncodedForm,
	}
	form := url.Values{
		"client_id":     {c.config.ClientID},
		"client_secret": {c.config.ClientSecret},
		"grant_type":    {clients.RefreshTokenType},
		"refresh_token": {refreshToken},
	}
	body := strings.NewReader(form.Encode())
	res, err := clients.HandleHTTPRequest(http.MethodPost, c.config.Endpoint.TokenURL, headers, body, false)
	if err != nil {
		return nil, err
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			c.logger.Error("couldn't close body", zap.Error(err))
		}
	}()

	if contentType := res.Header.Values(constants.HTTPHeaderContentType); utils.Contains(contentType, constants.MIMETypeJSON) {
		return nil, domain.ErrUnexpectedContentType(contentType)
	}

	if s := res.StatusCode; s != http.StatusOK {
		return nil, domain.ErrUnexpectedStatusCode(s)
	}

	return clients.DeserializeToken(res.Body)
}

func reportByGroupURI(groupID string) string {
	return fmt.Sprintf("%s/%v%s", groupsURI, groupID, reportsURI)
}

func pagesURI(reportID string) string {
	return fmt.Sprintf("%v/%v/pages", reportsURI, reportID)
}

func (c *ServiceClient) get(consumerID interface{}, resource string, deserialize func(reader io.ReadCloser) (interface{}, error)) (interface{}, error) {
	ctx := context.Background()
	accessData, err := c.tokenCache.Get(ctx, consumerID)
	if err != nil {
		return nil, err
	}

	accessToken := accessData.GetAccessToken()
	if accessToken == "" {
		return nil, domain.ErrNotFound
	}

	return c.executeHTTPRequest(ctx, consumerID, resource, accessData, deserialize, true)
}

func (c *ServiceClient) executeHTTPRequest(ctx context.Context, consumerID interface{}, resource string, accessData domain.AccessData, deserialize func(reader io.ReadCloser) (interface{}, error), refreshTokenIfNeeded bool) (interface{}, error) {
	l := utils.WithContext(ctx, c.logger)

	headers := map[string]string{
		constants.HTTPHeaderAuthorization: constants.BearerTokenType + accessData.GetAccessToken(),
	}

	res, err := clients.HandleHTTPRequest(http.MethodGet, c.config.APIURL+resource, headers, nil, false)
	if err != nil {
		return nil, err
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			l.Error("couldn't close body", zap.Error(err))
		}
	}()

	if c := res.Header.Values(constants.HTTPHeaderContentType); utils.Contains(c, constants.MIMETypeJSON) {
		return nil, domain.ErrUnexpectedContentType(c)
	}

	if s := res.StatusCode; s != http.StatusOK {
		if s != http.StatusForbidden || !refreshTokenIfNeeded {
			return nil, domain.ErrUnexpectedStatusCode(s)
		}

		newAccessData, err := c.RefreshTokens(accessData.GetRefreshToken())
		if err != nil {
			return nil, err
		}

		err = c.tokenCache.Update(ctx, consumerID, newAccessData)
		if err != nil {
			return nil, err
		}

		return c.executeHTTPRequest(ctx, consumerID, resource, newAccessData, deserialize, false)
	}

	r, err := deserialize(res.Body)
	if err != nil {
		return nil, err
	}

	return r, nil
}
