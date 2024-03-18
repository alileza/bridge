package portal

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
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
	filePath := p.StaticFilepath + r.URL.Path
	if _, err := os.Stat(filePath); err != nil {
		http.ServeFile(w, r, p.StaticFilepath+"/index.html")
		return
	}

	http.FileServer(http.Dir(p.StaticFilepath)).ServeHTTP(w, r)
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
