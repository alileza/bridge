package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/urfave/cli/v2"

	"bridge/httpredirector"
	"bridge/portal"
	"bridge/storage/localstorage"
	"bridge/storage/s3storage"
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
			&cli.BoolFlag{
				Name:    "enable-opengraph",
				Usage:   "This will enable opengraph support for the redirector. This will make the redirector fetch the target URL and parse the opengraph tags to use as the redirector's title and description. This will make the redirector slightly slower.",
				EnvVars: []string{"ENABLE_OPENGRAPH"},
				Aliases: []string{"o", "opengraph", "og"},
			},
			&cli.StringFlag{
				Name:    "ui-static-path",
				Aliases: []string{"ui", "static"},
				Value:   "/app/portal/dist",
				Usage:   "path to the static files",
				EnvVars: []string{"STATIC_PATH"},
			},
			&cli.StringFlag{
				Name:    "storage-path",
				Aliases: []string{"s", "storage"},
				Value:   "file://./routes.json",
				Usage:   "storage path (file://<bucket name>/<path>, s3://<file path>), if not set, it will use in-memory storage",
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
			enableOpengraph := c.Bool("enable-opengraph")
			storagePath := c.String("storage-path")

			var store httpredirector.Storage
			if storagePath == "" {
				store = &sync.Map{}
			} else {
				ss, err := url.Parse(storagePath)
				if err != nil {
					return fmt.Errorf("error parsing storage path: %s", err)
				}
				switch ss.Scheme {
				case "file":
					store = localstorage.NewLocalStorage(strings.ReplaceAll(storagePath, "file://", ""))
				case "s3":
					reg := ss.Query().Get("region")
					if reg == "" {
						return fmt.Errorf("region is required for S3 storage, put it on the query string: ?region=us-west-1")
					}
					ls, err := s3storage.NewS3Storage(ss.Host, reg)
					if err != nil {
						return fmt.Errorf("error creating S3 storage: %s", err)
					}
					store = ls
				default:
					return fmt.Errorf("unsupported storage scheme: %s", ss.Scheme)
				}
			}

			prtl := portal.NewServer(&portal.Options{
				ListenAddress: listenAddress,

				UIStaticFilepath: staticPath,
				UIProxyEnabled:   proxyEnabled,
				UIProxyURL:       proxyURL,

				Redirector: &httpredirector.HTTPRedirector{
					EnableOpengraph: enableOpengraph,
					Storage:         store,
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
