package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed index.html style.css app.js marked.min.js
var assetsFS embed.FS

// Assets returns an http.FileSystem for the embedded web assets.
func Assets() http.FileSystem {
	fsys, err := fs.Sub(assetsFS, ".")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}

// IndexHandler returns an http.Handler that serves the index.html file.
func IndexHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFileFS(w, r, assetsFS, "index.html")
	})
}
