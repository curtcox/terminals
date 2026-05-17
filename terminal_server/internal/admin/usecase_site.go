package admin

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed usecases_site_static/*
var usecaseSiteFiles embed.FS

func redirectUsecaseSiteIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.Redirect(w, r, "/docs/usecases/", http.StatusMovedPermanently)
}

func newUsecaseSiteHandler() http.Handler {
	site, err := fs.Sub(usecaseSiteFiles, "usecases_site_static")
	if err != nil {
		panic(err)
	}
	fileServer := http.StripPrefix("/docs/usecases/", http.FileServer(http.FS(site)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if strings.TrimPrefix(r.URL.Path, "/docs/usecases/") == "" {
			content, err := fs.ReadFile(site, "index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=60")
			_, _ = w.Write(content)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=60")
		fileServer.ServeHTTP(w, r)
	})
}
