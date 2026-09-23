package auth

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSessionSignVerify(t *testing.T) {
	g, _ := NewGitHub(Config{ClientID: "id", ClientSecret: "secret", SessionSecret: []byte("k")})
	now := time.Now()

	v := g.sign(Identity{Login: "octocat", Emails: []string{"o|c@example.com", "b@example.com"}}, now.Add(time.Hour))
	if id, ok := g.verify(v, now); !ok || id.Login != "octocat" || strings.Join(id.Emails, ",") != "o|c@example.com,b@example.com" {
		t.Fatalf("verify(valid) = %+v, %v", id, ok)
	}
	if _, ok := g.verify(v, now.Add(2*time.Hour)); ok {
		t.Error("verify accepted an expired session")
	}
	if _, ok := g.verify(v+"x", now); ok {
		t.Error("verify accepted a tampered signature")
	}
	payload, sig, _ := strings.Cut(v, ".")
	forged := g.sign(Identity{Login: "admin"}, now.Add(time.Hour))
	forgedPayload, _, _ := strings.Cut(forged, ".")
	if _, ok := g.verify(forgedPayload+"."+sig, now); ok {
		t.Error("verify accepted a swapped payload")
	}

	other, _ := NewGitHub(Config{ClientID: "id", ClientSecret: "secret", SessionSecret: []byte("other")})
	if _, ok := other.verify(payload+"."+sig, now); ok {
		t.Error("verify accepted a session signed with another secret")
	}
}

// fakeGitHub serves the OAuth and API endpoints used by the login flow.
// When emailsStatus is not 200, GET /user/emails fails with that status.
func fakeGitHub(t *testing.T, login string, orgs []string, emailsStatus int) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /login/oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		http.Redirect(w, r, q.Get("redirect_uri")+"?code=the-code&state="+url.QueryEscape(q.Get("state")), http.StatusFound)
	})
	mux.HandleFunc("POST /login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("code") != "the-code" || r.Form.Get("client_secret") != "secret" {
			json.NewEncoder(w).Encode(map[string]string{"error": "bad_verification_code"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"access_token": "tok"})
	})
	mux.HandleFunc("GET /user", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"login": login, "email": "public@example.com"})
	})
	mux.HandleFunc("GET /user/emails", func(w http.ResponseWriter, r *http.Request) {
		if emailsStatus != http.StatusOK {
			w.WriteHeader(emailsStatus)
			return
		}
		json.NewEncoder(w).Encode([]map[string]any{
			{"email": "work@example.com", "primary": false, "verified": true},
			{"email": "unverified@example.com", "primary": false, "verified": false},
			{"email": "octocat@example.com", "primary": true, "verified": true},
		})
	})
	mux.HandleFunc("GET /user/orgs", func(w http.ResponseWriter, r *http.Request) {
		var out []map[string]string
		for _, o := range orgs {
			out = append(out, map[string]string{"login": o})
		}
		json.NewEncoder(w).Encode(out)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestLoginFlow(t *testing.T) {
	tests := []struct {
		name         string
		allowedOrgs  []string
		allowedUsers []string
		userOrgs     []string
		emailsStatus int
		wantStatus   int
		wantBody     string
	}{
		{name: "no allowlist", wantStatus: http.StatusOK, wantBody: "hello octocat <octocat@example.com,work@example.com>"},
		{name: "allowed user", allowedUsers: []string{"OctoCat"}, wantStatus: http.StatusOK, wantBody: "hello octocat <octocat@example.com,work@example.com>"},
		{name: "allowed org", allowedOrgs: []string{"acme"}, userOrgs: []string{"other", "ACME"}, wantStatus: http.StatusOK, wantBody: "hello octocat <octocat@example.com,work@example.com>"},
		{name: "emails endpoint forbidden falls back to profile email", emailsStatus: http.StatusForbidden, wantStatus: http.StatusOK, wantBody: "hello octocat <public@example.com>"},
		{name: "not in org", allowedOrgs: []string{"acme"}, userOrgs: []string{"other"}, wantStatus: http.StatusForbidden},
		{name: "not allowed user", allowedUsers: []string{"someone"}, wantStatus: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.emailsStatus == 0 {
				tt.emailsStatus = http.StatusOK
			}
			gh := fakeGitHub(t, "octocat", tt.userOrgs, tt.emailsStatus)
			g, err := NewGitHub(Config{
				ClientID: "id", ClientSecret: "secret",
				AllowedOrgs: tt.allowedOrgs, AllowedUsers: tt.allowedUsers,
				AuthorizeURL: gh.URL + "/login/oauth/authorize",
				TokenURL:     gh.URL + "/login/oauth/access_token",
				APIURL:       gh.URL,
				Logger:       log.New(io.Discard, "", 0),
			})
			if err != nil {
				t.Fatal(err)
			}

			mux := http.NewServeMux()
			g.Register(mux)
			mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
				if !g.RequirePage(w, r) {
					return
				}
				id, _ := g.User(r)
				io.WriteString(w, "hello "+id.Login+" <"+strings.Join(id.Emails, ",")+">")
			})
			app := httptest.NewServer(mux)
			defer app.Close()

			jar, _ := cookiejar.New(nil)
			client := &http.Client{Jar: jar}

			// Unauthenticated: / -> /auth/login -> GitHub -> /auth/callback -> /
			resp, err := client.Get(app.URL + "/")
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %q)", resp.StatusCode, tt.wantStatus, body)
			}
			if tt.wantBody != "" && string(body) != tt.wantBody {
				t.Fatalf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestCallbackRejectsBadState(t *testing.T) {
	g, _ := NewGitHub(Config{ClientID: "id", ClientSecret: "secret", Logger: log.New(io.Discard, "", 0)})
	req := httptest.NewRequest("GET", "/auth/callback?code=x&state=attacker", nil)
	req.AddCookie(&http.Cookie{Name: stateCookie, Value: "real"})
	rec := httptest.NewRecorder()
	g.callback(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestRequireAPI(t *testing.T) {
	g, _ := NewGitHub(Config{ClientID: "id", ClientSecret: "secret"})
	h := g.RequireAPI(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) })

	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest("GET", "/api/routes", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated status = %d, want 401", rec.Code)
	}

	req := httptest.NewRequest("GET", "/api/routes", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: g.sign(Identity{Login: "octocat"}, time.Now().Add(time.Hour))})
	rec = httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusTeapot {
		t.Errorf("authenticated status = %d, want 418", rec.Code)
	}
}
