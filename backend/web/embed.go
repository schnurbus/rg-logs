// Package web embeds the Vite production build served by the API process.
//
// Local `go run` without a frontend build only has .gitkeep; SPA routes 404.
// The Docker multi-stage build copies frontend/dist/* into this directory
// before compiling the binary.
package web

import "embed"

//go:embed all:*
var FS embed.FS
