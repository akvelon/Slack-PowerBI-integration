package middlewares

import (
	"net/http"
	"net/http/httputil"

	"go.uber.org/zap"
)

// NewRequestLoggingMiddleware creates a new request logging middleware.
func NewRequestLoggingMiddleware(h http.Handler, c *config.RequestLoggingConfig, l *zap.Logger) http.Handler {
	if !c.Enable {
		return h
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := utils.WithContext(r.Context(), l)

		request, err := httputil.DumpRequest(r, c.DumpBody)
		if err != nil {
			l.Warn("couldn't dump request", zap.Error(err))
		} else {
			l.Debug("handling request", zap.String("request", string(request)))
		}

		h.ServeHTTP(w, r)
	})
}
