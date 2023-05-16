package middlewares

import (
	"net/http"

	"github.com/julienschmidt/httprouter"


)

// NewRouterMiddleware creates a new router middleware.
func NewRouterMiddleware(router *httprouter.Router) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(constants.HTTPHeaderContentType, constants.MIMETypeJSON)
		router.ServeHTTP(w, r)
	})
}
