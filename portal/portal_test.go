package portal

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alileza/bridge/audit"
	"github.com/alileza/bridge/auth"
	"github.com/alileza/bridge/httpredirector"
	"github.com/alileza/bridge/storage"
)

func newTestServer(t *testing.T, gh *auth.GitHub) (http.Handler, *audit.Log, httpredirector.Storage) {
	dir := t.TempDir()
	store, err := storage.NewJSONFileStorage(filepath.Join(dir, "routes"))
	if err != nil {
		t.Fatal(err)
	}
	auditLog, err := audit.Open(filepath.Join(dir, "routes.audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { auditLog.Close() })

	s := NewServer(&Options{
		Logger:     log.New(io.Discard, "", 0),
		Redirector: &httpredirector.HTTPRedirector{Storage: store},
		Auth:       gh,
		Audit:      auditLog,
	})
	return s.srv.Handler, auditLog, store
}

func do(h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Host = "go.example.com"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRouteLifecycleIsAudited(t *testing.T) {
	h, auditLog, _ := newTestServer(t, nil)

	if rec := do(h, "PUT", "/api/routes", `{"key":"/gh","url":"https://github.com"}`); rec.Code != http.StatusAccepted {
		t.Fatalf("create status = %d", rec.Code)
	}
	do(h, "PUT", "/api/routes", `{"key":"/gh","url":"https://github.com"}`) // no-op, not audited
	do(h, "PUT", "/api/routes", `{"key":"/gh","url":"https://github.com/alileza"}`)

	rec := do(h, "GET", "/gh", "")
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "https://github.com/alileza" {
		t.Fatalf("redirect = %d %q", rec.Code, rec.Header().Get("Location"))
	}

	var routes []routeView
	json.NewDecoder(do(h, "GET", "/api/routes", "").Body).Decode(&routes)
	if len(routes) != 1 || routes[0].UpdatedBy != "anonymous" || routes[0].UpdatedAt == nil {
		t.Fatalf("routes = %+v, want one route updated by anonymous", routes)
	}

	do(h, "DELETE", "/api/routes", `{"key":"go.example.com/gh"}`)

	entries := auditLog.Recent(10, "")
	var actions []string
	for _, e := range entries {
		actions = append(actions, e.Action)
	}
	if got := strings.Join(actions, ","); got != "delete,update,create" {
		t.Fatalf("audit actions = %s, want delete,update,create", got)
	}
	if entries[0].PreviousURL != "https://github.com/alileza" || entries[1].PreviousURL != "https://github.com" {
		t.Errorf("previous URLs not recorded: %+v", entries)
	}
}

func TestAuthProtectsPortalButNotRedirects(t *testing.T) {
	gh, err := auth.NewGitHub(auth.Config{ClientID: "id", ClientSecret: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	h, _, store := newTestServer(t, gh)
	store.Set("go.example.com/gh", "https://github.com")

	if rec := do(h, "GET", "/gh", ""); rec.Code != http.StatusFound || rec.Header().Get("Location") != "https://github.com" {
		t.Errorf("public redirect = %d %q, want 302 to https://github.com", rec.Code, rec.Header().Get("Location"))
	}

	for _, tc := range []struct{ method, target, body string }{
		{"GET", "/api/routes", ""},
		{"GET", "/api/audit", ""},
		{"PUT", "/api/routes", `{"key":"/x","url":"https://x"}`},
		{"DELETE", "/api/routes", `{"key":"go.example.com/x"}`},
	} {
		if rec := do(h, tc.method, tc.target, tc.body); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want 401", tc.method, tc.target, rec.Code)
		}
	}

	if rec := do(h, "GET", "/", ""); rec.Code != http.StatusFound || rec.Header().Get("Location") != "/auth/login" {
		t.Errorf("GET / = %d %q, want redirect to /auth/login", rec.Code, rec.Header().Get("Location"))
	}

	var me map[string]any
	json.NewDecoder(do(h, "GET", "/api/me", "").Body).Decode(&me)
	if me["auth_enabled"] != true || me["authenticated"] != false {
		t.Errorf("/api/me = %v", me)
	}

	if rec := do(h, "GET", "/healthz", ""); rec.Code != http.StatusOK {
		t.Errorf("/healthz status = %d, want 200", rec.Code)
	}
}
