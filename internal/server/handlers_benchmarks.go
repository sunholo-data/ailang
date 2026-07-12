package server

import (
	"context"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
)

// M-EVAL-DATA-HOSTING-DECOUPLE: serve the benchmark JSONs from a private GCS bucket
// so the docs site fetches them at RUNTIME instead of having them baked into the
// Docusaurus build. A rig republish (gsutil cp, every ~45 min) then shows within
// ~1 min with no site rebuild / GitHub Pages deploy — killing the mixed-build /
// stale-cache class of bug. Public + read-only; CORS is applied by the global
// middleware; the bucket stays private (this service's SA has objectViewer).

var (
	benchBucketOnce    sync.Once
	benchStorageClient *storage.Client
	benchStorageErr    error
	// Safe object paths only: json files, alphanumerics + / _ . - (no traversal).
	benchPathRe = regexp.MustCompile(`^[a-zA-Z0-9_./-]+\.json$`)
)

func benchBucketName() string {
	if b := os.Getenv("BENCHMARKS_BUCKET"); b != "" {
		return b
	}
	return "ailang-multivac-dev-benchmarks"
}

func benchClient(ctx context.Context) (*storage.Client, error) {
	benchBucketOnce.Do(func() {
		benchStorageClient, benchStorageErr = storage.NewClient(ctx)
	})
	return benchStorageClient, benchStorageErr
}

// handleBenchmarks streams gs://<bucket>/<path> for GET /benchmarks/<path>.
func (s *Server) handleBenchmarks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	obj := strings.TrimPrefix(r.URL.Path, "/benchmarks/")
	if obj == "" || strings.Contains(obj, "..") || strings.HasPrefix(obj, "/") || !benchPathRe.MatchString(obj) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	ctx := r.Context()
	cl, err := benchClient(ctx)
	if err != nil {
		http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
		return
	}
	rc, err := cl.Bucket(benchBucketName()).Object(obj).NewReader(ctx)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	_, _ = io.Copy(w, rc)
}
