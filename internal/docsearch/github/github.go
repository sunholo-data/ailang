// Package github implements authenticated GitHub code search.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/docsearch"
	"github.com/sunholo-data/ailang/internal/gitutil"
)

const defaultAPI = "https://api.github.com"

// Backend searches the repository's Markdown files through GitHub.
type Backend struct {
	Repo   string
	Token  string
	client *http.Client
	apiURL string
}

// NewBackend resolves a token and the current repository.
func NewBackend(ctx context.Context) (*Backend, error) {
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		cmd := exec.CommandContext(ctx, "gh", "auth", "token")
		if out, err := cmd.Output(); err == nil {
			token = strings.TrimSpace(string(out))
		}
	}
	if token == "" {
		return nil, fmt.Errorf("GitHub search requires authentication: set GITHUB_TOKEN or run `gh auth login`")
	}
	owner, repo, err := gitutil.GitHubOwnerRepo(ctx, ".")
	if err != nil {
		return nil, fmt.Errorf("detecting GitHub repository: %w", err)
	}
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("current directory has no GitHub origin; use --path from a local checkout")
	}
	return &Backend{Repo: owner + "/" + repo, Token: token, client: &http.Client{Timeout: 10 * time.Second}, apiURL: defaultAPI}, nil
}

// NewBackendForTest constructs a backend without reading credentials or git.
func NewBackendForTest(apiURL, repo, token string, client *http.Client) *Backend {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Backend{Repo: repo, Token: token, client: client, apiURL: strings.TrimRight(apiURL, "/")}
}

func (b *Backend) Search(ctx context.Context, opts docsearch.SearchOptions) ([]docsearch.SearchResult, docsearch.SearchStats, error) {
	stats := docsearch.SearchStats{}
	if strings.TrimSpace(b.Token) == "" {
		return nil, stats, fmt.Errorf("GitHub search requires an authentication token")
	}
	if !validRepo(b.Repo) {
		return nil, stats, fmt.Errorf("invalid GitHub repository %q", b.Repo)
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	perPage := limit
	if perPage > 100 {
		perPage = 100
	}
	page := 1
	var results []docsearch.SearchResult
	for len(results) < limit {
		u := strings.TrimRight(b.apiURL, "/") + "/search/code"
		q := opts.Query + " repo:" + b.Repo
		if opts.Subdir != "" {
			q += " path:" + opts.Subdir
		}
		v := url.Values{"q": {q}, "per_page": {strconv.Itoa(perPage)}, "page": {strconv.Itoa(page)}}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u+"?"+v.Encode(), nil)
		if err != nil {
			return nil, stats, fmt.Errorf("creating GitHub search request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+b.Token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		resp, err := b.client.Do(req)
		if err != nil {
			return nil, stats, fmt.Errorf("GitHub search request failed: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, stats, fmt.Errorf("reading GitHub search response: %w", readErr)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, stats, statusError(resp, body)
		}
		var payload struct {
			Total int `json:"total_count"`
			Items []struct {
				Path    string  `json:"path"`
				HTMLURL string  `json:"html_url"`
				Score   float64 `json:"score"`
			} `json:"items"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, stats, fmt.Errorf("decoding GitHub search response: %w", err)
		}
		for _, item := range payload.Items {
			results = append(results, docsearch.SearchResult{Path: item.HTMLURL, Title: item.Path, Score: item.Score})
			if len(results) == limit {
				break
			}
		}
		stats.TotalDocs = payload.Total
		if len(payload.Items) < perPage || len(results) >= limit {
			break
		}
		page++
	}
	stats.SearchTimeMs = 0
	return results, stats, nil
}

func validRepo(repo string) bool {
	parts := strings.Split(repo, "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] != "" && !strings.ContainsAny(repo, " ?#")
}

func statusError(resp *http.Response, body []byte) error {
	message := strings.TrimSpace(string(body))
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("GitHub authentication rejected (401); check GITHUB_TOKEN or `gh auth status`")
	case http.StatusForbidden, http.StatusTooManyRequests:
		reset := resp.Header.Get("X-RateLimit-Reset")
		if reset != "" {
			return fmt.Errorf("GitHub code-search rate limit exceeded; reset at Unix time %s", reset)
		}
		return fmt.Errorf("GitHub code-search rate limit exceeded")
	case http.StatusNotFound:
		return fmt.Errorf("GitHub repository %s was not found or is inaccessible", message)
	default:
		return fmt.Errorf("GitHub code-search API returned HTTP %d: %s", resp.StatusCode, message)
	}
}
