package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TST-RC1-16
func TestDocsSearch_StdlibFlagAndOfflineFallback(t *testing.T) {
	// Pure unit check on embedded stdlib search (deterministic, path-normalized).
	results, stats, err := searchEmbeddedStdlib("regex", 10)
	if err != nil {
		t.Fatalf("searchEmbeddedStdlib(regex): %v", err)
	}
	if stats.TotalDocs == 0 {
		t.Fatalf("expected non-zero TotalDocs from embedded stdlib, got 0")
	}
	if len(results) == 0 {
		t.Fatalf("expected matches for query %q in embedded stdlib", "regex")
	}
	foundRegex := false
	for _, r := range results {
		norm := filepath.ToSlash(r.Path)
		if strings.HasPrefix(norm, "std/regex") {
			foundRegex = true
			break
		}
	}
	if !foundRegex {
		t.Fatalf("expected std/regex hit in %+v", results)
	}

	// Function-level match on std/string.split.
	splitRes, _, err := searchEmbeddedStdlib("split", 10)
	if err != nil {
		t.Fatalf("searchEmbeddedStdlib(split): %v", err)
	}
	foundSplit := false
	for _, r := range splitRes {
		norm := filepath.ToSlash(r.Path)
		if strings.Contains(norm, "std/string") && strings.Contains(strings.ToLower(r.Path+" "+r.Title), "split") {
			foundSplit = true
			break
		}
	}
	if !foundSplit {
		t.Fatalf("expected std/string split hit in %+v", splitRes)
	}
}

// TST-RC1-16
func TestDocsSearch_CLIStdlibAndOfflineFallback(t *testing.T) {
	bin := cliTestBin(t)
	home := t.TempDir()
	work := t.TempDir()

	// Clear GitHub / confinement env vars so we test unconfined offline CLI.
	for _, k := range []string{"GITHUB_TOKEN", "AILANG_AGENT_POLICY", "AILANG_NO_GITHUB_SEARCH", "AILANG_STDLIB_PATH"} {
		t.Setenv(k, "")
	}
	// Empty PATH ensures `gh` is unavailable so GitHub backend fails cleanly.
	t.Setenv("PATH", t.TempDir())

	// 1. Explicit --stdlib --json works offline outside repo.
	res := runCLIIn(t, bin, home, work, "docs", "search", "--stdlib", "--json", "regex")
	if res.exitCode != 0 {
		t.Fatalf("docs search --stdlib --json regex failed (exit %d): stdout=%q stderr=%q", res.exitCode, res.stdout, res.stderr)
	}
	var payload struct {
		Results []struct {
			Path  string  `json:"path"`
			Title string  `json:"title"`
			Score float64 `json:"score"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(res.stdout), &payload); err != nil {
		t.Fatalf("invalid JSON from docs search --stdlib --json: %v\nstdout=%s", err, res.stdout)
	}
	if len(payload.Results) == 0 {
		t.Fatalf("expected non-empty results from --stdlib regex, got %+v", payload)
	}

	// 2. Unconfined out-of-repo offline fallback finds stdlib matches for "split".
	fb := runCLIIn(t, bin, home, work, "docs", "search", "--json", "split")
	if fb.exitCode != 0 {
		t.Fatalf("offline unconfined fallback for 'split' failed (exit %d): stdout=%q stderr=%q", fb.exitCode, fb.stdout, fb.stderr)
	}
	var fbPayload struct {
		Results []struct {
			Path  string  `json:"path"`
			Title string  `json:"title"`
			Score float64 `json:"score"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(fb.stdout), &fbPayload); err != nil {
		t.Fatalf("invalid JSON from offline fallback: %v\nstdout=%s", err, fb.stdout)
	}
	if len(fbPayload.Results) == 0 {
		t.Fatalf("expected stdlib fallback results for 'split', got empty")
	}

	// 3. Respects --no-github (must NOT silently fall back when --no-github is set).
	noGH := runCLIIn(t, bin, home, work, "docs", "search", "--no-github", "split")
	if noGH.exitCode == 0 {
		t.Fatalf("--no-github without local docs must fail, got exit 0: stdout=%q", noGH.stdout)
	}

	// 4. Respects AILANG_AGENT_POLICY confinement (must NOT silently fall back).
	t.Setenv("AILANG_AGENT_POLICY", "private-policy")
	conf := runCLIIn(t, bin, home, work, "docs", "search", "split")
	if conf.exitCode == 0 || !strings.Contains(conf.stderr, "confined by AILANG_AGENT_POLICY") {
		t.Fatalf("confined docs search must refuse with policy error: exit=%d stderr=%q", conf.exitCode, conf.stderr)
	}
	_ = os.ErrNotExist
}
