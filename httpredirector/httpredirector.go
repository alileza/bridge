package httpredirector

import (
	"fmt"
	"net/http"
	"net/url"

	"bridge/opengraph"
)

const DefaultRedirectURL = "https://alileza.me/"

// HTTPRedirector is a struct that holds the routes and their destinations
type HTTPRedirector struct {
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
		routes = append(routes, Route{
			Preview: v.(string),
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
		http.Redirect(w, r, destinationURL, http.StatusFound)
		return
	}

	headElements, err := opengraph.FetchMetaTags(r.Context(), destinationURL)
	if err != nil {
		http.Redirect(w, r, destinationURL, http.StatusFound)
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
