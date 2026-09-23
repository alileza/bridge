package portal

import "embed"

//go:embed ui/dist/index.html
//go:embed ui/dist/bridge.png ui/dist/favicon.png ui/dist/apple-touch-icon.png
//go:embed ui/dist/assets/*
var assets embed.FS
