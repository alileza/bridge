package portal

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alileza/bridge/httpredirector"
)

type fakeStorage struct{ routes []httpredirector.Route }

func (f *fakeStorage) Set(string, string) error              { return nil }
func (f *fakeStorage) Get(string) (string, error)            { return "", nil }
func (f *fakeStorage) Delete(string) error                   { return nil }
func (f *fakeStorage) List() ([]httpredirector.Route, error) { return f.routes, nil }
func (f *fakeStorage) Reload() error                         { return nil }

func TestMetrics(t *testing.T) {
	m := NewMetrics("1.2.3", &fakeStorage{routes: []httpredirector.Route{
		{Key: "a.io/x"}, {Key: "a.io/y"}, {Key: "b.io/z"},
	}})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /x", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusFound) })
	h := m.Instrument(mux)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/x", nil))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/nope", nil))
	m.redirects.Inc("a.io")
	m.forwarded.Inc(`a.io/we"ird`)

	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	out := rec.Body.String()

	for _, want := range []string{
		`bridge_build_info{version="1.2.3",goversion="`,
		`bridge_storage_up 1`,
		`bridge_routes{host="a.io"} 2`,
		`bridge_routes{host="b.io"} 1`,
		`bridge_http_requests_total{handler="GET /x",code="302"} 1`,
		`bridge_http_requests_total{handler="unmatched",code="404"} 1`,
		`bridge_http_request_duration_seconds_bucket{handler="GET /x",le="+Inf"} 1`,
		`bridge_http_request_duration_seconds_count{handler="GET /x"} 1`,
		`bridge_redirects_total{host="a.io"} 1`,
		`bridge_routes_forwarded_total{key="a.io/we\"ird"} 1`,
		"# TYPE go_goroutines gauge",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("metrics output missing %q\n%s", want, out)
		}
	}
}
