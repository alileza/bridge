package portal

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type UIHandler struct {
	StaticFilepath string
	ProxyURL       string
}

func NewUIHandler(staticFilepath string, proxyURL string) *UIHandler {
	return &UIHandler{
		StaticFilepath: staticFilepath,
		ProxyURL:       proxyURL,
	}
}

func (p *UIHandler) Handler(proxyEnabled bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proxyEnabled {
			p.ProxyHandler(w, r)
			return
		}
		p.StaticFileHandler(w, r)
	})
}

func (p *UIHandler) StaticFileHandler(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path == "/" {
		r.URL.Path = "/index.html"
	}
	log.Println("ui/dist/" + r.URL.Path)
	b, err := assets.ReadFile("ui/dist" + r.URL.Path)
	if err != nil {
		log.Println("Error reading file", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if strings.HasSuffix(r.URL.Path, ".js") {
		w.Header().Set("Content-Type", "application/javascript")
	}
	if strings.HasSuffix(r.URL.Path, ".css") {
		w.Header().Set("Content-Type", "text/css")
	}
	w.Write(b)

	// http.FileServer(http.Dir(p.StaticFilepath)).ServeHTTP(w, r)
}

func (p *UIHandler) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	destURL, err := url.Parse(p.ProxyURL)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(destURL)

	r.Host = destURL.Host
	r.URL.Host = destURL.Host
	r.URL.Scheme = destURL.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Header.Set("X-Forwarded-Proto", r.URL.Scheme)

	proxy.ServeHTTP(w, r)
}
