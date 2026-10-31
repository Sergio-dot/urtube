// Package web embeds the built frontend (web/dist, produced by "npm run build")
// into the binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the built frontend, rooted at web/dist.
func Dist() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}
