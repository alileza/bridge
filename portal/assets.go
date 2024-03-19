package portal

import "embed"

//go:embed ui/dist/index.html
//go:embed ui/dist/bridge.png
//go:embed ui/dist/assets/*
var assets embed.FS
