// Package ui embeds the built frontend into the binary. The frontend lives in
// web/ and "npm run build" writes it to internal/ui/dist.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the built frontend, rooted at its index.html.
func Dist() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}
