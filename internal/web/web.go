// Package web embeds the static frontend assets so the server ships as a single binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:static
var assets embed.FS

// Static returns a filesystem rooted at the static/ directory.
func Static() fs.FS {
	sub, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	return sub
}
