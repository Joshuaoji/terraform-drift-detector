package api

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

//go:embed all:webdist
var webFS embed.FS

func webHandler() http.Handler {
	sub, err := fs.Sub(webFS, "webdist")
	if err != nil {
		panic(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		reqPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if reqPath == "" || reqPath == "." {
			reqPath = "index.html"
		}

		// SPA fallback: unknown paths serve index.html
		if _, err := fs.Stat(sub, reqPath); err != nil {
			reqPath = "index.html"
		}

		data, err := fs.ReadFile(sub, reqPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		contentType := mime.TypeByExtension(path.Ext(reqPath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_, _ = w.Write(data)
		}
	})
}
