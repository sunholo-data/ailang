package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDocsSearch_ConfinementActualCLI(t *testing.T) {
	f := newDocsConfinementFixture(t)
	f.proveInstruments(t)
	helpers, err := os.ReadFile(f.helperLog)
	if err != nil {
		t.Fatal(err)
	}
	home, work := docsState(t, f.home), docsState(t, f.work)
	count := f.requests.Load()
	t.Setenv("AILANG_AGENT_POLICY", "private-policy")
	for _, token := range []string{"", "dummy-docs-test-token"} {
		t.Setenv("GITHUB_TOKEN", token)
		r := f.run(t, "docs", "search", "--json", "needle")
		assertDocsRefusal(t, r)
		f.assertUntouched(t, home, work, count, string(helpers))
	}
}

func assertDocsRefusal(t *testing.T, r cliResult) {
	t.Helper()
	if r.exitCode == 0 || !strings.Contains(r.stderr, "confined by AILANG_AGENT_POLICY") || !strings.Contains(r.stderr, "--path") {
		t.Fatalf("want canonical docs refusal: %+v", r)
	}
	for _, unsafe := range []string{"set GITHUB_TOKEN", "gh auth login", "unset --no-github", "unset AILANG"} {
		if strings.Contains(r.stderr, unsafe) {
			t.Fatalf("unsafe hint %q: %s", unsafe, r.stderr)
		}
	}
}

func TestDocsSearch_ConfinementLocalAndDisabled(t *testing.T) {
	f := newDocsConfinementFixture(t)
	t.Setenv("AILANG_AGENT_POLICY", "private-policy")
	for _, args := range [][]string{{"--no-github"}, {"--path", filepath.Join(f.work, "absent")}} {
		r := f.run(t, append([]string{"docs", "search"}, append(args, "needle")...)...)
		if r.exitCode == 0 || strings.Contains(r.stderr, "gh auth login") || strings.Contains(r.stderr, "unset") {
			t.Fatalf("local-only failure/hint: %+v", r)
		}
	}
	t.Setenv("AILANG_NO_GITHUB_SEARCH", "1")
	r := f.run(t, "docs", "search", "needle")
	if r.exitCode == 0 || strings.Contains(r.stderr, "unset") {
		t.Fatalf("env-disabled failure/hint: %+v", r)
	}
	t.Setenv("AILANG_NO_GITHUB_SEARCH", "")
	corpus := filepath.Join(f.work, "design_docs")
	if err := os.Mkdir(corpus, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "needle.md"), []byte("# Needle contracts\nneedle typed contracts and effects\n"), 0600); err != nil {
		t.Fatal(err)
	}
	home, work := docsState(t, f.home), docsState(t, f.work)
	explicit := f.run(t, "docs", "search", "--json", "--path", corpus, "needle")
	auto := f.run(t, "docs", "search", "--json", "needle")
	a, b := docsStableResults(t, explicit), docsStableResults(t, auto)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("explicit/auto results differ: %v %v", a, b)
	}
	f.assertUntouched(t, home, work, 0, "")
}

func docsStableResults(t *testing.T, r cliResult) any {
	t.Helper()
	var result struct {
		Results []struct {
			Path, Title string
			Score       float64
		} `json:"results"`
	}
	if r.exitCode != 0 || json.Unmarshal([]byte(r.stdout), &result) != nil || len(result.Results) != 1 {
		t.Fatalf("local JSON result: %+v", r)
	}
	if result.Results[0].Title != "Needle contracts" || !strings.HasSuffix(filepath.ToSlash(result.Results[0].Path), "/needle.md") {
		t.Fatalf("local semantics: %+v", result)
	}
	// Explicit paths preserve their spelling; auto discovery resolves the child CWD.
	// Compare the same real file, including macOS /var -> /private/var aliases.
	canonical, err := filepath.EvalSymlinks(result.Results[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	result.Results[0].Path = filepath.ToSlash(canonical)
	return result.Results
}
