package http

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// HealthCheckHandler handles health check requests
type HealthCheckHandler struct {
	router *httprouter.Router
}

// ConfigureHealthCheck configures routes for health check
func ConfigureHealthCheck(router *httprouter.Router) {
	router.GET("/healthcheck", HealthCheck)
}

// HealthCheck handler just returns status ok
func HealthCheck(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.WriteHeader(http.StatusOK)
}
