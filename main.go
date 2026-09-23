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

func main() {
	var (
		listenAddress string
		storageDir    string
		proxyEnabled  bool
		proxyURL      string
	)

	stringFlag(&listenAddress, []string{"listen-address", "l", "listen"}, "LISTEN_ADDRESS", "0.0.0.0:80", "HTTP listen address, e.g. 0.0.0.0:80")
	stringFlag(&storageDir, []string{"storage-dir", "s", "storage"}, "", "./bridgedata", "storage dir")
	boolFlag(&proxyEnabled, []string{"proxy-enabled", "p", "proxy"}, "PROXY_ENABLED", false, "enable proxy mode, it's useful for development UI")
	stringFlag(&proxyURL, []string{"proxy-url", "u", "url"}, "PROXY_URL", "http://localhost:5173", "proxy URL for UI, e.g. http://localhost:5173")
	flag.Parse()

	if err := run(listenAddress, storageDir, proxyEnabled, proxyURL); err != nil {
		log.Fatal(err)
	}
}

func run(listenAddress, storageDir string, proxyEnabled bool, proxyURL string) error {
	os.MkdirAll(storageDir, 0755)

	store, err := storage.NewJSONFileStorage(storageDir)
	if err != nil {
		return fmt.Errorf("error initializing storage: %s", err)
	}

	prtl := portal.NewServer(&portal.Options{
		ListenAddress: listenAddress,

		UIProxyEnabled: proxyEnabled,
		UIProxyURL:     proxyURL,

		Redirector: &httpredirector.HTTPRedirector{
			Storage: store,
		},
	})

	return prtl.Start()
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
