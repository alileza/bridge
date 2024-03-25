package httpredirector

import (
	"fmt"
	"net/http"
	"net/url"
)

type HTTPRedirector struct {
	Storage Storage
}

type Route struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

type Storage interface {
	Set(key string, url string) (err error)
	Get(key string) (url string, err error)
	Delete(key string) (err error)
	List() (listURL []Route, err error)
	Reload() (err error)
}

func (rdr *HTTPRedirector) ListAllRoutes() ([]Route, error) {
	return rdr.Storage.List()
}

// ListRoutes returns a list of all routes in the redirector
func (rdr *HTTPRedirector) ListRoutes(host string) ([]Route, error) {
	routes, err := rdr.Storage.List()
	if err != nil {
		return nil, err
	}
	return filterRoutes(routes, host), nil
}

func filterRoutes(routes []Route, host string) []Route {
	var filteredRoutes []Route
	for _, route := range routes {
		if route.Key == host {
			filteredRoutes = append(filteredRoutes, route)
		}
	}
	return filteredRoutes
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

	return rdr.Storage.Set(key, destURL)
}

// RemoveRoute removes a route from the redirector
func (rdr *HTTPRedirector) RemoveRoute(key string) error {
	if _, err := rdr.Storage.Get(key); err != nil {
		return fmt.Errorf("route not found: %s => %w", key, err)
	}

	return rdr.Storage.Delete(key)
}
