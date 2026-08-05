package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed webdist/*
var webFS embed.FS

func webHandler() http.Handler {
	sub, err := fs.Sub(webFS, "webdist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(sub, path); err != nil {
			r.URL.Path = "/index.html"
		} else {
			r.URL.Path = "/" + path
		}
		fileServer.ServeHTTP(w, r)
	})
}
