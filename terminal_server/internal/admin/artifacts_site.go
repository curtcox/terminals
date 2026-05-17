package admin

import (
	"net/http"
)

// newArtifactHandler serves local validation artifacts (frames, video, JSON
// manifests) written to the repo's artifacts/ directory by make usecase-validate.
// The server always runs from the repo root, so "artifacts" is a relative path.
// If the directory does not exist, all requests return 404.
func newArtifactHandler() http.Handler {
	fileServer := http.StripPrefix("/artifacts/", http.FileServer(http.Dir("artifacts")))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=60")
		fileServer.ServeHTTP(w, r)
	})
}
