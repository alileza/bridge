package httpredirector

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const DefaultRedirectURL = "https://alileza.me/"

type HTTPRedirector struct {
	Storage Storage
}

type Route struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

type Storage interface {
	Store(key any, value any)
	Load(key any) (value any, ok bool)
	Delete(key any)
	Range(f func(key any, value any) bool)
}

// ListRoutes returns a list of all routes in the redirector
func (rdr *HTTPRedirector) ListRoutes(host string) []Route {
	var routes []Route

	rdr.Storage.Range(func(k, v interface{}) bool {
		if !strings.HasPrefix(k.(string), host) {
			return true
		}

		routes = append(routes, Route{
			Key: k.(string),
			URL: v.(string),
		})
		return true
	})

	return routes
}

// AddRoute adds a route to the redirector it can be path or full with hostname without scheme
func (rdr *HTTPRedirector) SetRoute(r *http.Request, key string, destURL string) error {
	_, err := url.ParseRequestURI(destURL)
	if err != nil {
		return fmt.Errorf("invalid destination URL: %w", err)
	}

	if key[0] != '/' {
		key = "/" + key
	}

	key = r.Host + key

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
