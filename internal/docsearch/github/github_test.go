package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/docsearch"
)

func TestNewBackendDefaultsOutsideGitRepository(t *testing.T) {
	dir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	t.Setenv("GITHUB_TOKEN", "test-token")

	b, err := NewBackend(context.Background())
	if err != nil {
		t.Fatalf("NewBackend() error = %v", err)
	}
	if b.Repo != defaultRepo {
		t.Fatalf("Repo = %q, want %q", b.Repo, defaultRepo)
	}
}

func TestNewBackendRejectsUnrelatedGitHubRepository(t *testing.T) {
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if out, err := exec.Command("git", "-C", dir, "remote", "add", "origin", "https://github.com/acme/other.git").CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %v: %s", err, out)
	}
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })
	t.Setenv("GITHUB_TOKEN", "test-token")

	_, err = NewBackend(context.Background())
	if err == nil || !strings.Contains(err.Error(), "acme/other") || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("NewBackend() error = %v, want explicit unrelated-repository error", err)
	}
}

func TestBackendSearchAuthenticatedRequestAndPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("missing auth header")
		}
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("missing accept header")
		}
		if r.URL.Query().Get("q") != "contracts repo:acme/docs path:guides" {
			t.Errorf("query = %q", r.URL.Query().Get("q"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"total_count":1,"items":[{"path":"guides/a.md","html_url":"https://github.com/acme/docs/blob/main/guides/a.md","score":0.8}]}`))
	}))
	defer server.Close()
	b := NewBackendForTest(server.URL, "acme/docs", "secret", server.Client())
	results, stats, err := b.Search(context.Background(), docOptions("contracts", "guides"))
	if err != nil || len(results) != 1 || stats.TotalDocs != 1 {
		t.Fatalf("results=%v stats=%+v err=%v", results, stats, err)
	}
}

func TestBackendErrorsNeverEchoToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"bad token"}`, http.StatusUnauthorized)
	}))
	defer server.Close()
	b := NewBackendForTest(server.URL, "acme/docs", "secret", server.Client())
	_, _, err := b.Search(context.Background(), docOptions("x", ""))
	if err == nil || strings.Contains(err.Error(), "secret") || !strings.Contains(err.Error(), "authentication rejected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackendRejectsInvalidRepoAndMissingToken(t *testing.T) {
	b := NewBackendForTest("http://invalid", "bad repo", "secret", nil)
	if _, _, err := b.Search(context.Background(), docOptions("x", "")); err == nil {
		t.Fatal("expected repo error")
	}
	b = NewBackendForTest("http://invalid", "acme/docs", "", nil)
	if _, _, err := b.Search(context.Background(), docOptions("x", "")); err == nil {
		t.Fatal("expected token error")
	}
}

func TestStatusErrorOnlyClassifiesZeroRemainingAsRateLimit(t *testing.T) {
	for _, tt := range []struct {
		name      string
		status    int
		remaining string
		wantRate  bool
	}{
		{"permission denied", http.StatusForbidden, "1", false},
		{"forbidden rate limit", http.StatusForbidden, "0", true},
		{"too many requests rate limit", http.StatusTooManyRequests, "0", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.status, Header: make(http.Header)}
			resp.Header.Set("X-RateLimit-Remaining", tt.remaining)
			err := statusError(resp, []byte(`{"message":"denied"}`))
			gotRate := strings.Contains(err.Error(), "rate limit")
			if gotRate != tt.wantRate {
				t.Fatalf("error = %v, rate-limit = %v, want %v", err, gotRate, tt.wantRate)
			}
		})
	}
}

func TestCacheHitMissExpiryIsolationAndNoTokenPersistence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir() reads USERPROFILE, not HOME, on Windows
	results := []docsearch.SearchResult{{Path: "a.md", Title: "a", Score: 1}}
	stats := docsearch.SearchStats{TotalDocs: 1}
	now := time.Now().UTC()
	if _, _, ok := readCache("acme/docs", "q", "guides", 1, now); ok {
		t.Fatal("unexpected cache hit before write")
	}
	if err := writeCache("acme/docs", "q", "guides", 1, results, stats, now); err != nil {
		t.Fatal(err)
	}
	got, gotStats, ok := readCache("acme/docs", "q", "guides", 1, now.Add(time.Minute))
	if !ok || len(got) != 1 || gotStats.TotalDocs != 1 {
		t.Fatalf("cache hit = %v, %v, %v", got, gotStats, ok)
	}
	if _, _, ok := readCache("other/docs", "q", "guides", 1, now.Add(time.Minute)); ok {
		t.Fatal("cache key leaked across repositories")
	}
	if _, _, ok := readCache("acme/docs", "q", "guides", 1, now.Add(cacheTTL)); ok {
		t.Fatal("expired cache entry returned a hit")
	}
	onDiskPath, err := cachePath("acme/docs", "q", "guides", 1)
	if err != nil {
		t.Fatalf("cachePath: %v", err)
	}
	data, err := os.ReadFile(onDiskPath)
	if err != nil || strings.Contains(string(data), "test-token") {
		t.Fatalf("cache contains credential or could not be read: %v", err)
	}
}

func docOptions(query, subdir string) docsearch.SearchOptions {
	return docsearch.SearchOptions{Query: query, Subdir: subdir, Limit: 10}
}
