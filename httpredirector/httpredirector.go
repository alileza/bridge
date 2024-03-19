package httpredirector

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"bridge/opengraph"

	"github.com/prometheus/client_golang/prometheus"
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
	prometheus.MustRegister(forwardCounter)
}

const DefaultRedirectURL = "https://alileza.me/"

// HTTPRedirector is a struct that holds the routes and their destinations
type HTTPRedirector struct {
	BaseURL         string
	EnableOpengraph bool
	Storage         Storage
}

type Route struct {
	Preview string `json:"preview"`
	Key     string `json:"key"`
	URL     string `json:"url"`
}

type Storage interface {
	Store(key any, value any)
	Load(key any) (value any, ok bool)
	Delete(key any)
	Range(f func(key any, value any) bool)
}

// LoadRoutes loads a map of routes into the redirector
func (rdr *HTTPRedirector) LoadRoutes(routes map[string]string) {
	for k, v := range routes {
		rdr.Storage.Store(k, v)
	}
}

func (rdr *HTTPRedirector) ListRoutes() []Route {
	var routes []Route

	rdr.Storage.Range(func(k, v interface{}) bool {
		i, _ := opengraph.GenerateBarcode(rdr.BaseURL + k.(string))
		routes = append(routes, Route{
			Preview: "data:image/png;base64," + opengraph.EncodeImageToBase64(i),
			Key:     k.(string),
			URL:     v.(string),
		})
		return true
	})
	return routes
}

// AddRoute adds a route to the redirector it can be path or full with hostname without scheme
func (rdr *HTTPRedirector) SetRoute(key string, destURL string) error {
	_, err := url.ParseRequestURI(destURL)
	if err != nil {
		return fmt.Errorf("invalid destination URL: %w", err)
	}

	if !strings.Contains(key, "/") && !strings.Contains(key, ".") {
		key = "/" + key
	}

	rdr.Storage.Store(key, destURL)
	return nil
}

// RemoveRoute removes a route from the redirector
func (rdr *HTTPRedirector) RemoveRoute(key string) error {
	if _, ok := rdr.Storage.Load(key); !ok {
		return fmt.Errorf("route not found: %s", key)
	}

	rdr.Storage.Delete(key)
	return nil
}

// GetRedirectURL returns the destination for a given route
func (rdr *HTTPRedirector) GetRedirectURL(key *url.URL) string {
	keyWithHost := key.Host + key.Path
	dest, ok := rdr.Storage.Load(keyWithHost)
	if ok {
		return dest.(string)
	}

	dest, ok = rdr.Storage.Load(key.Path)
	if ok {
		return dest.(string)
	}

	return DefaultRedirectURL
}

func (rdr *HTTPRedirector) Handler(w http.ResponseWriter, r *http.Request) {
	destinationURL := rdr.GetRedirectURL(r.URL)

	if !rdr.EnableOpengraph {
		forwardCounter.With(prometheus.Labels{"path": r.URL.Path}).Inc()
		http.Redirect(w, r, destinationURL, http.StatusFound)
		return
	}

	headElements, err := opengraph.FetchMetaTags(r.Context(), destinationURL)
	if err != nil {
		forwardCounter.With(prometheus.Labels{"path": r.URL.Path}).Inc()
		http.Redirect(w, r, destinationURL, http.StatusFound)
		return
	}

	forwardCounter.With(prometheus.Labels{"path": r.URL.Path}).Inc()
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf(responseBody, headElements, destinationURL, destinationURL, destinationURL)))
}

const responseBody = `
<!DOCTYPE html>
<html>
    <head>
		%s
		<title>Bridge - github.com/alileza/bridge</title>
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
