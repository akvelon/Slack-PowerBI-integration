package clients

import (
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"
)

const (
	// RefreshTokenType represents grant type for Refresh token
	RefreshTokenType = "refresh_token"
)

// Token represents access + refresh token pair.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// GetAccessToken returns AccessToken
func (t Token) GetAccessToken() string {
	return t.AccessToken
}

// GetRefreshToken returns RefreshToken
func (t Token) GetRefreshToken() string {
	return t.RefreshToken
}

// DeserializeToken returns Token object from json
func DeserializeToken(b io.ReadCloser) (*Token, error) {
	d := json.NewDecoder(b)
	t := Token{}
	if err := d.Decode(&t); err != nil {
		return nil, err
	}

	return &t, nil
}

// HandleHTTPRequest executes requests
func HandleHTTPRequest(method string, url string, headers map[string]string, body io.Reader, isCloseBody bool) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)

	if err != nil {
		return nil, err
	}

	for key, element := range headers {
		req.Header.Add(key, element)
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	if isCloseBody {
		defer func() {
			err = resp.Body.Close()
			if err != nil {
				zap.L().Error("couldn't close body", zap.Error(err))
			}
		}()
	}

	return resp, nil
}
