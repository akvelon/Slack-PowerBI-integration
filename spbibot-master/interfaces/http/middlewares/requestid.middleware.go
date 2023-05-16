package middlewares

import (
	"net/http"


)

// NewRequestIDMiddleware creates a new request id middleware.
func NewRequestIDMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(utils.WithRequestID(r.Context()))
		r = r.WithContext(utils.WithRequestInfo(r.Context(), r))
		w.Header().Set("X-Request-Id", utils.RequestID(r.Context()))
		h.ServeHTTP(w, r)
	})
}
