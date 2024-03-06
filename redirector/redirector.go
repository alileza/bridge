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

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v2"
)

var (
	// Define a Prometheus counter to track requests to the forward handler.
	forwardCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redirector_forward_requests_total",
			Help: "Total number of requests to the forward handler.",
		},
		[]string{"path"},
	)
)

func init() {
	// Register Prometheus metrics.
	prometheus.MustRegister(forwardCounter)
}

type Redirector struct {
	Routes         sync.Map
	RoutesFilepath string
	StaticFilepath string
	ProxyEnabled   bool
	ProxyURL       string
	Store          Store

	ReloadLimiter *rate.Limiter
	LastReload    time.Time

	UpdateLimiter *rate.Limiter
	LastUpdate    time.Time
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

const responseBody = `
<!DOCTYPE html>
<html>
    <head>
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
	head, err := FetchHeadElements(targetURL)
	if err != nil {
		log.Println("error getting Open Graph tags:", err)
	}

	return []byte(MinifyHTML(fmt.Sprintf(responseBody, head, targetURL, targetURL, targetURL)))
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
	forwardCounter.With(prometheus.Labels{"path": r.URL.Path}).Inc()

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
	if !f.ReloadLimiter.Allow() {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	f.LastReload = time.Now()

	f.ReloadRoutes()

	w.WriteHeader(http.StatusAccepted)
}

func (f *Redirector) HandlePutRoute(w http.ResponseWriter, r *http.Request) {
	// limit the size of the request body
	r.Body = http.MaxBytesReader(w, r.Body, 50*1024)
	if r.ContentLength > 50*1024 {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	// limit the number of requests per second
	if !f.UpdateLimiter.Allow() {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}
	f.LastUpdate = time.Now()

	// decode the request body
	var Route struct {
		Path   string `json:"path"`
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&Route); err != nil {
		json.NewEncoder(w).Encode(map[string]any{
			"error":   err,
			"message": `invalid request body, expecting something like: {"path": "me", "target":"https://alileza.me"},`,
		})
		log.Println("[400]", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// add the route to the in-memory map
	f.AddRoute(trimearlyslash(Route.Path), Route.Target)

	// transform sync.Map to map[string]string so it can be marshalled to yaml
	routes := make(map[string]string)
	f.Routes.Range(func(key, value interface{}) bool {
		routes[key.(string)] = value.(string)
		return true
	})

	// marshal the routes to yaml bytes
	b, err := yaml.Marshal(routes)
	if err != nil {
		log.Println("[500]", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// write the routes to the file
	if err := f.Store.WriteFile(b); err != nil {
		log.Println("[500] i/o", err)
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
