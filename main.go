package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/alileza/bridge/audit"
	"github.com/alileza/bridge/auth"
	"github.com/alileza/bridge/httpredirector"
	"github.com/alileza/bridge/portal"
	"github.com/alileza/bridge/storage"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	var (
		listenAddress  string
		storageDir     string
		proxyEnabled   bool
		proxyURL       string
		metricsEnabled bool
		metricsAddress string

		githubClientID     string
		githubClientSecret string
		githubAllowedOrgs  string
		githubAllowedUsers string
		sessionSecret      string
		githubURL          string
	)

	stringFlag(&listenAddress, []string{"listen-address", "l", "listen"}, "LISTEN_ADDRESS", "0.0.0.0:80", "HTTP listen address, e.g. 0.0.0.0:80")
	stringFlag(&storageDir, []string{"storage-dir", "s", "storage"}, "", "./bridgedata", "storage dir")
	boolFlag(&proxyEnabled, []string{"proxy-enabled", "p", "proxy"}, "PROXY_ENABLED", false, "enable proxy mode, it's useful for development UI")
	stringFlag(&proxyURL, []string{"proxy-url", "u", "url"}, "PROXY_URL", "http://localhost:5173", "proxy URL for UI, e.g. http://localhost:5173")
	boolFlag(&metricsEnabled, []string{"metrics", "m"}, "METRICS_ENABLED", false, "expose Prometheus metrics at /metrics")
	stringFlag(&metricsAddress, []string{"metrics-address"}, "METRICS_ADDRESS", "", "serve /metrics on a separate address, e.g. 0.0.0.0:9090 (implies --metrics)")
	stringFlag(&githubClientID, []string{"github-client-id"}, "GITHUB_CLIENT_ID", "", "GitHub OAuth App client ID; enables GitHub login for the portal")
	stringFlag(&githubClientSecret, []string{"github-client-secret"}, "GITHUB_CLIENT_SECRET", "", "GitHub OAuth App client secret")
	stringFlag(&githubAllowedOrgs, []string{"github-allowed-orgs"}, "GITHUB_ALLOWED_ORGS", "", "comma-separated GitHub orgs whose members may log in")
	stringFlag(&githubAllowedUsers, []string{"github-allowed-users"}, "GITHUB_ALLOWED_USERS", "", "comma-separated GitHub usernames who may log in")
	stringFlag(&githubURL, []string{"github-url"}, "GITHUB_URL", "https://github.com", "GitHub base URL; set for GitHub Enterprise Server")
	stringFlag(&sessionSecret, []string{"session-secret"}, "SESSION_SECRET", "", "secret for signing login sessions (random per start when empty)")
	flag.Parse()

	opts := &portal.Options{
		ListenAddress:  listenAddress,
		UIProxyEnabled: proxyEnabled,
		UIProxyURL:     proxyURL,
		Version:        version,
		MetricsEnabled: metricsEnabled,
		MetricsAddress: metricsAddress,
	}
	if githubClientID != "" {
		cfg := auth.Config{
			ClientID:      githubClientID,
			ClientSecret:  githubClientSecret,
			AllowedOrgs:   splitList(githubAllowedOrgs),
			AllowedUsers:  splitList(githubAllowedUsers),
			SessionSecret: []byte(sessionSecret),
		}
		if base := strings.TrimSuffix(githubURL, "/"); base != "https://github.com" {
			cfg.AuthorizeURL = base + "/login/oauth/authorize"
			cfg.TokenURL = base + "/login/oauth/access_token"
			cfg.APIURL = base + "/api/v3"
		}
		gh, err := auth.NewGitHub(cfg)
		if err != nil {
			log.Fatal(err)
		}
		if githubAllowedOrgs == "" && githubAllowedUsers == "" {
			log.Println("warning: GitHub login is enabled without --github-allowed-orgs/--github-allowed-users; any GitHub user can log in")
		}
		if sessionSecret == "" {
			log.Println("warning: --session-secret is not set; logins will not survive a restart")
		}
		opts.Auth = gh
	}

	if err := run(storageDir, opts); err != nil {
		log.Fatal(err)
	}
}

func run(storageDir string, opts *portal.Options) error {
	os.MkdirAll(storageDir, 0755)

	store, err := storage.NewJSONFileStorage(storageDir)
	if err != nil {
		return fmt.Errorf("error initializing storage: %s", err)
	}

	auditLog, err := audit.Open(strings.TrimSuffix(storageDir, ".json") + ".audit.jsonl")
	if err != nil {
		return err
	}
	defer auditLog.Close()
	opts.Audit = auditLog

	opts.Redirector = &httpredirector.HTTPRedirector{
		Storage: store,
	}

	return portal.NewServer(opts).Start()
}

func splitList(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// stringFlag registers a string flag under all names, defaulting to the env var when set.
func stringFlag(p *string, names []string, env, value, usage string) {
	if v, ok := os.LookupEnv(env); ok && env != "" {
		value = v
	}
	for _, name := range names {
		flag.StringVar(p, name, value, usage)
	}
}

// boolFlag registers a bool flag under all names, defaulting to the env var when set.
func boolFlag(p *bool, names []string, env string, value bool, usage string) {
	if v, ok := os.LookupEnv(env); ok && env != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			value = b
		}
	}
	for _, name := range names {
		flag.BoolVar(p, name, value, usage)
	}
}
