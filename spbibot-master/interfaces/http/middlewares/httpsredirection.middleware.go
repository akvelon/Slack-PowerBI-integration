package middlewares

import (
	"net/http"
	"net/url"

)

// NewHTTPSRedirectionMiddleware adds HTTP to HTTPS redirection.
func NewHTTPSRedirectionMiddleware(h http.Handler, httpsPort int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			httpsURL := url.URL{
				Scheme:     constants.HTTPScon,
				Opaque:     r.URL.Opaque,
				User:       r.URL.User,
				Host:       r.Host,
				Path:       r.URL.Path,
				RawPath:    r.URL.RawPath,
				ForceQuery: r.URL.ForceQuery,
				RawQuery:   r.URL.RawQuery,
				Fragment:   r.URL.Fragment,
			}
			httpsURL.Host = utils.JoinHostPort(httpsURL.Hostname(), httpsPort)
			if w.Header().Get(constants.HTTPHeaderContentType) != "" {
				w.Header().Set(constants.HTTPHeaderContentType, "")
			}
			http.Redirect(w, r, httpsURL.String(), http.StatusMovedPermanently)

			return
		}

		h.ServeHTTP(w, r)
	}
}
