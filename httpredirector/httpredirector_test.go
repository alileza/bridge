package httpredirector

import (
	"net/url"
	"sync"
	"testing"
)

// Test loading multiple routes at once
func TestLoadRoutes(t *testing.T) {
	rdr := &HTTPRedirector{routes: sync.Map{}}
	routes := map[string]string{
		"/key1": "http://destination1.com",
		"/key2": "http://destination2.com",
	}
	rdr.LoadRoutes(routes)

	for k, v := range routes {
		dest, ok := rdr.routes.Load(k)
		if !ok {
			t.Fatalf("expected to find route for key %s, but did not", k)
		}
		if dest != v {
			t.Fatalf("expected destination %s, got %s", v, dest)
		}
	}
}

// Test the GetRedirectURL function
func TestGetRedirectURL(t *testing.T) {
	tests := []struct {
		name          string
		setupActions  func(rdr *HTTPRedirector)
		key           string
		expectedDest  string
		expectedFound bool
	}{
		{
			name: "find route with path only",
			setupActions: func(rdr *HTTPRedirector) {
				rdr.AddRoute("/path-only", "http://destination-path-only.com")
			},
			key:           "/path-only",
			expectedDest:  "http://destination-path-only.com",
			expectedFound: true,
		},
		{
			name: "find route with domain and path",
			setupActions: func(rdr *HTTPRedirector) {
				rdr.AddRoute("example.com/path", "http://destination-domain-path.com")
			},
			key:           "http://example.com/path",
			expectedDest:  "http://destination-domain-path.com",
			expectedFound: true,
		},
		{
			name:          "fallback to default for non-existing route",
			setupActions:  func(rdr *HTTPRedirector) {},
			key:           "http://nonexisting.com/path",
			expectedDest:  "https://alileza.me/",
			expectedFound: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rdr := &HTTPRedirector{routes: sync.Map{}}
			tc.setupActions(rdr)

			parsedKey, _ := url.Parse(tc.key)
			dest := rdr.GetRedirectURL(parsedKey)
			if dest != tc.expectedDest {
				t.Fatalf("expected destination %s, got %s for key %s", tc.expectedDest, dest, tc.key)
			}
		})
	}
}
