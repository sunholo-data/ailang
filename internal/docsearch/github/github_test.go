package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/docsearch"
)

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

func docOptions(query, subdir string) docsearch.SearchOptions {
	return docsearch.SearchOptions{Query: query, Subdir: subdir, Limit: 10}
}
