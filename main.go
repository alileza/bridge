package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"bridge/httpredirector"
	"bridge/portal"
	"bridge/storage"
)

func main() {
	var err error
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "listen-address",
				Aliases: []string{"l", "listen"},
				Value:   "0.0.0.0:80",
				Usage:   "HTTP listen address, e.g. 0.0.0.0:80",
				EnvVars: []string{"LISTEN_ADDRESS"},
			},
			&cli.StringFlag{
				Name:    "storage-dir",
				Aliases: []string{"s", "storage"},
				Value:   "./bridgedata",
				Usage:   "storage dir (default: ./bridgedata)",
			},
			&cli.BoolFlag{
				Name:    "proxy-enabled",
				Aliases: []string{"p", "proxy"},
				Value:   false,
				Usage:   "enable proxy mode",
				EnvVars: []string{"PROXY_ENABLED"},
				Hidden:  true,
			},
			&cli.StringFlag{
				Name:    "proxy-url",
				Aliases: []string{"u", "url"},
				Value:   "http://localhost:5173",
				Usage:   "proxy URL",
				EnvVars: []string{"PROXY_URL"},
				Hidden:  true,
			},
		},
		Action: func(c *cli.Context) error {
			listenAddress := c.String("listen-address")
			proxyEnabled := c.Bool("proxy-enabled")
			proxyURL := c.String("proxy-url")
			storageDir := c.String("storage-dir")

			os.MkdirAll(storageDir, 0755)
			store := storage.NewStorage(storageDir + "/routes.json")

			prtl := portal.NewServer(&portal.Options{
				ListenAddress: listenAddress,

				UIProxyEnabled: proxyEnabled,
				UIProxyURL:     proxyURL,

				Redirector: &httpredirector.HTTPRedirector{
					Storage: store,
				},
			})

			return prtl.Start()
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
