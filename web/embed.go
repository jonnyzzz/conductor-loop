package web

import (
	"embed"
	"io/fs"
	"net/http"
)

// embeddedAssets contains the fallback web UI shipped with the binary.
//
//go:embed src
var embeddedAssets embed.FS

// FileSystem returns an http.FileSystem rooted at web/src.
func FileSystem() (http.FileSystem, error) {
	sub, err := fs.Sub(embeddedAssets, "src")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}
