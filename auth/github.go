// Package auth implements GitHub OAuth login with signed session cookies.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	sessionCookie = "bridge_session"
	stateCookie   = "bridge_oauth_state"
)

type Config struct {
	ClientID     string
	ClientSecret string

	// AllowedOrgs and AllowedUsers restrict who may log in. A user is allowed
	// when they match either list. When both are empty, any GitHub user may log in.
	AllowedOrgs  []string
	AllowedUsers []string

	// SessionSecret signs session cookies. When empty a random secret is
	// generated, which means sessions do not survive a restart.
	SessionSecret []byte
	SessionTTL    time.Duration

	// Endpoints; overridable for tests or GitHub Enterprise.
	AuthorizeURL string
	TokenURL     string
	APIURL       string
	HTTPClient   *http.Client
	Logger       *log.Logger
}

type GitHub struct {
	c Config
}

// Identity is the logged-in GitHub user.
type Identity struct {
	Login string `json:"l"`
	// Emails are the user's verified emails, primary first; empty if GitHub didn't share any.
	Emails []string `json:"e,omitempty"`
}

// maxEmails caps how many emails are kept, to bound the session cookie size.
const maxEmails = 10

func NewGitHub(c Config) (*GitHub, error) {
	if c.ClientID == "" || c.ClientSecret == "" {
		return nil, errors.New("github auth: client id and client secret are required")
	}
	if len(c.SessionSecret) == 0 {
		c.SessionSecret = make([]byte, 32)
		if _, err := rand.Read(c.SessionSecret); err != nil {
			return nil, err
		}
	}
	if c.SessionTTL == 0 {
		c.SessionTTL = 7 * 24 * time.Hour
	}
	if c.AuthorizeURL == "" {
		c.AuthorizeURL = "https://github.com/login/oauth/authorize"
	}
	if c.TokenURL == "" {
		c.TokenURL = "https://github.com/login/oauth/access_token"
	}
	if c.APIURL == "" {
		c.APIURL = "https://api.github.com"
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if c.Logger == nil {
		c.Logger = log.Default()
	}
	return &GitHub{c: c}, nil
}

// Register adds the login, callback and logout endpoints.
func (g *GitHub) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/login", g.login)
	mux.HandleFunc("GET /auth/callback", g.callback)
	mux.HandleFunc("POST /auth/logout", g.logout)
}

// User returns the GitHub identity of the request's session, if any.
func (g *GitHub) User(r *http.Request) (Identity, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return Identity{}, false
	}
	return g.verify(c.Value, time.Now())
}

// RequireAPI rejects unauthenticated requests with 401.
func (g *GitHub) RequireAPI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := g.User(r); !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "login required"})
			return
		}
		next(w, r)
	}
}

// RequirePage sends unauthenticated requests to the GitHub login.
func (g *GitHub) RequirePage(w http.ResponseWriter, r *http.Request) bool {
	if _, ok := g.User(r); ok {
		return true
	}
	http.Redirect(w, r, "/auth/login", http.StatusFound)
	return false
}

func (g *GitHub) login(w http.ResponseWriter, r *http.Request) {
	state := randomString(16)
	http.SetCookie(w, &http.Cookie{
		Name: stateCookie, Value: state, Path: "/auth", MaxAge: 600,
		HttpOnly: true, Secure: isHTTPS(r), SameSite: http.SameSiteLaxMode,
	})

	scope := "read:user user:email"
	if len(g.c.AllowedOrgs) > 0 {
		scope += " read:org"
	}
	q := url.Values{
		"client_id":    {g.c.ClientID},
		"redirect_uri": {baseURL(r) + "/auth/callback"},
		"scope":        {scope},
		"state":        {state},
		"allow_signup": {"false"},
	}
	http.Redirect(w, r, g.c.AuthorizeURL+"?"+q.Encode(), http.StatusFound)
}

func (g *GitHub) callback(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(stateCookie)
	state := r.URL.Query().Get("state")
	if err != nil || state == "" || subtle.ConstantTimeCompare([]byte(c.Value), []byte(state)) != 1 {
		http.Error(w, "invalid oauth state, please try logging in again", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: stateCookie, Path: "/auth", MaxAge: -1})

	token, err := g.exchange(r.URL.Query().Get("code"), baseURL(r)+"/auth/callback")
	if err != nil {
		g.c.Logger.Println("github auth: token exchange failed:", err)
		http.Error(w, "github login failed", http.StatusBadGateway)
		return
	}

	id, err := g.fetchIdentity(token)
	if err != nil {
		g.c.Logger.Println("github auth: fetching user failed:", err)
		http.Error(w, "github login failed", http.StatusBadGateway)
		return
	}

	login := id.Login
	allowed, err := g.allowed(token, login)
	if err != nil {
		g.c.Logger.Println("github auth: checking org membership failed:", err)
		http.Error(w, "github login failed", http.StatusBadGateway)
		return
	}
	if !allowed {
		g.c.Logger.Printf("github auth: denied login for %s", login)
		http.Error(w, fmt.Sprintf("@%s is not allowed to use this bridge", login), http.StatusForbidden)
		return
	}

	g.c.Logger.Printf("github auth: %s logged in", login)
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: g.sign(id, time.Now().Add(g.c.SessionTTL)), Path: "/",
		MaxAge: int(g.c.SessionTTL.Seconds()), HttpOnly: true, Secure: isHTTPS(r), SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (g *GitHub) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Path: "/", MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

func (g *GitHub) exchange(code, redirectURI string) (string, error) {
	if code == "" {
		return "", errors.New("missing code")
	}
	form := url.Values{
		"client_id":     {g.c.ClientID},
		"client_secret": {g.c.ClientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}
	req, err := http.NewRequest(http.MethodPost, g.c.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	var body struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := g.do(req, &body); err != nil {
		return "", err
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("no access token (%s)", body.Error)
	}
	return body.AccessToken, nil
}

func (g *GitHub) fetchIdentity(token string) (Identity, error) {
	var user struct {
		Login string `json:"login"`
		Email string `json:"email"`
	}
	if err := g.api(token, "/user", &user); err != nil {
		return Identity{}, err
	}
	if user.Login == "" {
		return Identity{}, errors.New("empty login")
	}
	id := Identity{Login: user.Login}
	if user.Email != "" {
		id.Emails = []string{user.Email}
	}

	// The profile email is only set when public; ask for all verified ones.
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := g.api(token, "/user/emails", &emails); err != nil {
		g.c.Logger.Printf("github auth: could not read emails for %s: %v", user.Login, err)
		return id, nil
	}
	var verified []string
	for _, e := range emails {
		if !e.Verified {
			continue
		}
		if e.Primary {
			verified = append([]string{e.Email}, verified...)
		} else {
			verified = append(verified, e.Email)
		}
	}
	if len(verified) > maxEmails {
		verified = verified[:maxEmails]
	}
	id.Emails = verified
	return id, nil
}

func (g *GitHub) allowed(token, login string) (bool, error) {
	if len(g.c.AllowedOrgs) == 0 && len(g.c.AllowedUsers) == 0 {
		return true, nil
	}
	if slices.ContainsFunc(g.c.AllowedUsers, func(u string) bool { return strings.EqualFold(u, login) }) {
		return true, nil
	}
	if len(g.c.AllowedOrgs) == 0 {
		return false, nil
	}
	var orgs []struct {
		Login string `json:"login"`
	}
	if err := g.api(token, "/user/orgs?per_page=100", &orgs); err != nil {
		return false, err
	}
	for _, o := range orgs {
		if slices.ContainsFunc(g.c.AllowedOrgs, func(a string) bool { return strings.EqualFold(a, o.Login) }) {
			return true, nil
		}
	}
	return false, nil
}

func (g *GitHub) api(token, path string, v any) error {
	req, err := http.NewRequest(http.MethodGet, g.c.APIURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	return g.do(req, v)
}

func (g *GitHub) do(req *http.Request, v any) error {
	resp, err := g.c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s %s: %s", req.Method, req.URL.Path, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

type session struct {
	Identity
	Expires int64 `json:"x"`
}

// sign produces "<base64url JSON session>.<base64url HMAC>".
func (g *GitHub) sign(id Identity, expires time.Time) string {
	b, _ := json.Marshal(session{Identity: id, Expires: expires.Unix()})
	payload := base64.RawURLEncoding.EncodeToString(b)
	return payload + "." + g.mac(payload)
}

func (g *GitHub) verify(value string, now time.Time) (Identity, bool) {
	payload, sig, ok := strings.Cut(value, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(g.mac(payload))) {
		return Identity{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return Identity{}, false
	}
	var s session
	if err := json.Unmarshal(raw, &s); err != nil || s.Login == "" || now.Unix() > s.Expires {
		return Identity{}, false
	}
	return s.Identity, true
}

func (g *GitHub) mac(payload string) string {
	h := hmac.New(sha256.New, g.c.SessionSecret)
	h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func randomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func baseURL(r *http.Request) string {
	scheme := "http"
	if isHTTPS(r) {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
