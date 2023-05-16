package powerbi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"

)

const (
	reportsURI = "/reports"
)

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

// GetReport returns report by reportID
func (c *ServiceClient) GetReport(consumerID interface{}, accessData domain.AccessData, reportID string) (*domain.Report, error) {
	r, err := c.get(consumerID, accessData, reportsURI+"/"+reportID, func(b io.ReadCloser) (interface{}, error) {
		return domain.DeserializeReport(b)
	})
	if err != nil {
		c.logger.Error("couldn't get report", zap.Error(err))

		return nil, err
	}

	return r.(*domain.Report), nil
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

func pagesURI(reportID string) string {
	return fmt.Sprintf("%v/%v/pages", reportsURI, reportID)
}

func (c *ServiceClient) get(consumerID interface{}, accessData domain.AccessData, resource string, deserialize func(reader io.ReadCloser) (interface{}, error)) (interface{}, error) {
	ctx := context.Background()
	if consumerID != nil {
		var err error
		accessData, err = c.tokenCache.Get(ctx, consumerID)
		if err != nil {
			return nil, err
		}
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
