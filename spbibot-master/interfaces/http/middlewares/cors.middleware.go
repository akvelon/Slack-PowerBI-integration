package middlewares

import (
	"net/http"
)

// NewCORSMiddleware creates a new CORS middleware.
func NewCORSMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", r.Header.Get("Allow"))
		h.ServeHTTP(w, r)
	})
}
