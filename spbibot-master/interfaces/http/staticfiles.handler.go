package http

import (
	"net/http"

	"github.com/julienschmidt/httprouter"

)

const (
	staticFilesRootHTTPRoute = "static"
)

// StaticFilesHandler handles routes for serving static files
type StaticFilesHandler struct {
	router *httprouter.Router
}

// ConfigureStaticFilesHandler configure routes for serving static files
func ConfigureStaticFilesHandler(router *httprouter.Router) {
	s := &StaticFilesHandler{
		router: router,
	}

	router.ServeFiles("/"+staticFilesRootHTTPRoute+"/*filepath", http.Dir("static"))

	router.GET("/", s.Index)
	router.GET("/payments-changes", s.PaymentsChanges)
	router.GET("/privacy-policy", s.PrivacyPolicy)
	router.GET("/terms-and-conditions", s.TermsAndConditions)
	router.GET("/style.css", s.serveStylesFile)
	router.GET("/assets/*.svg", s.serveAssetFile)
	router.GET("/robots.txt", s.serveRobotsFile)
	router.GET("/sitemap.xml", s.serveSitemapFile)
}

// Index returns main landing page
func (s *StaticFilesHandler) Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.serveFile(w, r, "index.html")
}

// PaymentsChanges returns PaymentsChanges page
func (s *StaticFilesHandler) PaymentsChanges(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.serveFile(w, r, "paymentsChanges.html")
}

// PrivacyPolicy returns PrivacyPolicy page
func (s *StaticFilesHandler) PrivacyPolicy(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.serveFile(w, r, "privacyPolicy.html")
}

// TermsAndConditions returns TermsOfService page
func (s *StaticFilesHandler) TermsAndConditions(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.serveFile(w, r, "termsAndConditions.html")
}

func (s *StaticFilesHandler) serveFile(w http.ResponseWriter, r *http.Request, fileName string) {
	w.Header().Set(constants.HTTPHeaderContentType, constants.MIMETypeHTML)
	http.ServeFile(w, r, "static/"+fileName)
}

func (s *StaticFilesHandler) serveStylesFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set(constants.HTTPHeaderContentType, constants.MIMETypeCSS)
	http.ServeFile(w, r, "static/"+r.URL.Path)
}

func (s *StaticFilesHandler) serveAssetFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set(constants.HTTPHeaderContentType, constants.MIMETypeSVG)
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	http.ServeFile(w, r, "static/"+r.URL.Path)
}

func (s *StaticFilesHandler) serveRobotsFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set(constants.HTTPHeaderContentType, constants.MIMETypeText)
	http.ServeFile(w, r, "static/"+r.URL.Path)
}

func (s *StaticFilesHandler) serveSitemapFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set(constants.HTTPHeaderContentType, constants.MIMETypeXML)
	http.ServeFile(w, r, "static/"+r.URL.Path)
}
