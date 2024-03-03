package redirector

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

type Redirector struct {
	Routes         sync.Map
	RoutesFilepath string
	StaticFilepath string
	ProxyEnabled   bool
	ProxyURL       string
	Store          Store
}

type Store interface {
	WriteFile(data []byte) error
	ReadFile() ([]byte, error)
}

func (f *Redirector) AddRoute(path, targetURL string) {
	if path == "" {
		return
	}
	f.Routes.Store(path, targetURL)
}

func (f *Redirector) GetRoute(path string) (string, bool) {
	path = trimearlyslash(path)

	target, ok := f.Routes.Load(path)
	if !ok {
		return "", false
	}
	return target.(string), true
}

func (f *Redirector) ReloadRoutes() error {
	blob, err := f.Store.ReadFile()
	if err != nil {
		return err
	}

	var routes map[string]string
	if err := yaml.Unmarshal(blob, &routes); err != nil {
		return err
	}

	f.Routes = sync.Map{}

	for path, target := range routes {
		f.AddRoute(path, target)
	}

	return nil
}

const responseBody = `
<!DOCTYPE html>
<html>
  <head>
		<script async src="https://www.googletagmanager.com/gtag/js?id=G-G205NCWYZC"></script>
		<script>
		window.dataLayer = window.dataLayer || [];
		function gtag(){dataLayer.push(arguments);}
		gtag('js', new Date());

		gtag('config', 'G-G205NCWYZC');
		</script>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	%s
	</head>
	<body>
	<noscript>
		<a href="%s">Click here to continue to: %s</a>
	</noscript>
	<script>
		window.location.replace("%s");
	</script>
	</body>
</html>
`

func generateHTML(targetURL string) []byte {
	og, err := getOpenGraphTagsHTML(targetURL)
	if err != nil {
		log.Println("error getting Open Graph tags:", err)
	}

	return []byte(fmt.Sprintf(responseBody, targetURL, og, targetURL, targetURL))
}

func (f *Redirector) HandleForward(w http.ResponseWriter, r *http.Request) {
	target, ok := f.GetRoute(r.URL.Path)
	if !ok {
		if f.ProxyEnabled {
			f.ProxyHandler(w, r)
			return
		}
		if f.StaticFilepath != "" {
			f.StaticFileHandler(w, r)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// might be interesting for analytics
	json.NewEncoder(os.Stdout).Encode(map[string]string{
		"path":        r.URL.Path,
		"target":      target,
		"user_agent":  r.UserAgent(),
		"referer":     r.Referer(),
		"remote_addr": r.RemoteAddr,
		"host":        r.Host,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	})

	w.Write(generateHTML(target))
}

func (f *Redirector) HandleGetRoutes(w http.ResponseWriter, r *http.Request) {
	routes := make(map[string]string)
	f.Routes.Range(func(key, value interface{}) bool {
		routes[key.(string)] = value.(string)
		return true
	})

	json.NewEncoder(w).Encode(routes)
}

func (f *Redirector) HandleReloadRoutes(w http.ResponseWriter, r *http.Request) {
	f.ReloadRoutes()

	w.WriteHeader(http.StatusAccepted)
}

func (f *Redirector) HandlePutRoute(w http.ResponseWriter, r *http.Request) {
	var Route struct {
		Path   string   `json:"path"`
		Target string   `json:"target"`
		Tags   []string `json:"tags"` // not used
	}
	if err := json.NewDecoder(r.Body).Decode(&Route); err != nil {
		log.Println("error decoding route:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	f.AddRoute(trimearlyslash(Route.Path), Route.Target)

	routes := make(map[string]string)
	f.Routes.Range(func(key, value interface{}) bool {
		routes[key.(string)] = value.(string)
		return true
	})

	b, err := yaml.Marshal(routes)
	if err != nil {
		log.Println("error marshalling routes:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := f.Store.WriteFile(b); err != nil {
		log.Println("error writing routes file:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (rdr *Redirector) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	destURL, err := url.Parse(rdr.ProxyURL)
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

func (rdr *Redirector) StaticFileHandler(w http.ResponseWriter, r *http.Request) {
	filePath := rdr.StaticFilepath + r.URL.Path
	if _, err := os.Stat(filePath); err != nil {
		http.ServeFile(w, r, rdr.StaticFilepath+"/index.html")
		return
	}

	http.FileServer(http.Dir(rdr.StaticFilepath)).ServeHTTP(w, r)
}

func trimearlyslash(s string) string {
	if len(s) > 0 && s[0] == '/' {
		return s[1:]
	}
	return s
}
