package middlewares

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/slack-go/slack"
	"go.uber.org/zap"

	
)

// NewVerifySlackRequestMiddleware adds Slack request signature verification.
func NewVerifySlackRequestMiddleware(h httprouter.Handle, c *config.SlackConfig, l *zap.Logger) httprouter.Handle {
	if !c.VerifyRequests {
		return h
	}

	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		l := utils.WithContext(r.Context(), l)

		if err := verifySlackRequest(r, c.SigningSecret); err != nil {
			l.Error("couldn't verify request", zap.Error(err))
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		h(w, r, p)
	}
}

func verifySlackRequest(r *http.Request, slackSecret string) error {
	v, err := slack.NewSecretsVerifier(r.Header, slackSecret)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	if _, err := v.Write(b); err != nil {
		return err
	}

	r.Body = ioutil.NopCloser(bytes.NewReader(b))

	return v.Ensure()
}
