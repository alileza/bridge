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
				Value:   "0.0.0.0:8080",
				Usage:   "address to listen on",
				EnvVars: []string{"LISTEN_ADDRESS"},
			},
			&cli.StringFlag{
				Name:    "ui-static-path",
				Aliases: []string{"ui", "static"},
				Value:   "/app/portal/dist",
				Usage:   "path to the static files",
				EnvVars: []string{"STATIC_PATH"},
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
			staticPath := c.String("static-path")
			proxyEnabled := c.Bool("proxy-enabled")
			proxyURL := c.String("proxy-url")
			storageDir := c.String("storage-dir")

			os.MkdirAll(storageDir, 0755)
			store := storage.NewStorage(storageDir + "/routes.json")

			prtl := portal.NewServer(&portal.Options{
				ListenAddress: listenAddress,

				UIStaticFilepath: staticPath,
				UIProxyEnabled:   proxyEnabled,
				UIProxyURL:       proxyURL,

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
