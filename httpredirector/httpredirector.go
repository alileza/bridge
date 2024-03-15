package httpforwarder

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"bridge/opengraph"
)

// HTTPRedirector is a struct that holds the routes and their destinations
type HTTPRedirector struct {
	EnableOpengraph bool

	routes sync.Map
}

// LoadRoutes loads a map of routes into the redirector
func (rdr *HTTPRedirector) LoadRoutes(routes map[string]string) {
	for k, v := range routes {
		rdr.routes.Store(k, v)
	}
}

// AddRoute adds a route to the redirector it can be path or full with hostname without scheme
func (rdr *HTTPRedirector) AddRoute(key string, destURL string) {
	rdr.routes.Store(key, destURL)
}

// RemoveRoute removes a route from the redirector
func (rdr *HTTPRedirector) RemoveRoute(key string) {
	rdr.routes.Delete(key)
}

// GetRedirectURL returns the destination for a given route
func (rdr *HTTPRedirector) GetRedirectURL(key *url.URL) string {
	keyWithHost := key.Host + key.Path
	dest, ok := rdr.routes.Load(keyWithHost)
	if ok {
		return dest.(string)
	}

	dest, ok = rdr.routes.Load(key.Path)
	if ok {
		return dest.(string)
	}

	return "https://alileza.me/"
}

func (rdr *HTTPRedirector) Handler(w http.ResponseWriter, r *http.Request) {
	destinationURL := rdr.GetRedirectURL(r.URL)

	if !rdr.EnableOpengraph {
		http.Redirect(w, r, destinationURL, http.StatusMovedPermanently)
		return
	}

	headElements, err := opengraph.FetchMetaTags(r.Context(), destinationURL)
	if err != nil {
		http.Redirect(w, r, destinationURL, http.StatusMovedPermanently)
		return
	}

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
