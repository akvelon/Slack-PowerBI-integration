package oauth

import (
	"bytes"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

// Config is extended structure of configuration
type Config struct {
	oauth2.Config

	ResponseType string

	ResponseMode string

	Resource string

	LogoutEndpoint LogoutEndpoint
}

// LogoutEndpoint is logout endpoint of configuration
type LogoutEndpoint struct {
	LogoutURL string
}

// AuthCodeURL method prepare authorization url
func (c *Config) AuthCodeURL(hashUserID string) string {
	v := url.Values{
		"client_id": {c.ClientID},
	}

	if c.ResponseType != "" {
		v.Set("response_type", c.ResponseType)
	}
	if c.RedirectURL != "" {
		v.Set("redirect_uri", c.RedirectURL)
	}
	if c.ResponseMode != "" {
		v.Set("response_mode", c.ResponseMode)
	}
	if c.Resource != "" {
		v.Set("resource", c.Resource)
	}
	if hashUserID != "" {
		v.Set("state", hashUserID)
	}
	if len(c.Scopes) > 0 {
		v.Set("scope", strings.Join(c.Scopes, ","))
	}

	return mergeURLAndParams(c.Endpoint.AuthURL, v)
}

// LogoutCodeURL method prepare logout url
func (c *Config) LogoutCodeURL() string {
	v := url.Values{
		"prompt": {"select_account"},
	}

	return mergeURLAndParams(c.LogoutEndpoint.LogoutURL, v)
}

func mergeURLAndParams(endpoint string, values url.Values) string {
	var buf bytes.Buffer
	buf.WriteString(endpoint)

	if strings.Contains(endpoint, "?") {
		buf.WriteByte('&')
	} else {
		buf.WriteByte('?')
	}

	buf.WriteString(values.Encode())
	return buf.String()
}
