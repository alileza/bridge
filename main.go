package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

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
	)

	stringFlag(&listenAddress, []string{"listen-address", "l", "listen"}, "LISTEN_ADDRESS", "0.0.0.0:80", "HTTP listen address, e.g. 0.0.0.0:80")
	stringFlag(&storageDir, []string{"storage-dir", "s", "storage"}, "", "./bridgedata", "storage dir")
	boolFlag(&proxyEnabled, []string{"proxy-enabled", "p", "proxy"}, "PROXY_ENABLED", false, "enable proxy mode, it's useful for development UI")
	stringFlag(&proxyURL, []string{"proxy-url", "u", "url"}, "PROXY_URL", "http://localhost:5173", "proxy URL for UI, e.g. http://localhost:5173")
	boolFlag(&metricsEnabled, []string{"metrics", "m"}, "METRICS_ENABLED", false, "expose Prometheus metrics at /metrics")
	stringFlag(&metricsAddress, []string{"metrics-address"}, "METRICS_ADDRESS", "", "serve /metrics on a separate address, e.g. 0.0.0.0:9090 (implies --metrics)")
	flag.Parse()

	opts := &portal.Options{
		ListenAddress:  listenAddress,
		UIProxyEnabled: proxyEnabled,
		UIProxyURL:     proxyURL,
		Version:        version,
		MetricsEnabled: metricsEnabled,
		MetricsAddress: metricsAddress,
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

	opts.Redirector = &httpredirector.HTTPRedirector{
		Storage: store,
	}

	return portal.NewServer(opts).Start()
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
