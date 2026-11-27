package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/prompt"
)

// TST-RC1-20
func TestLatestCompactVersionUsesSemverOrder(t *testing.T) {
	// ASCII order would pick v0.7.4-compact ('7' > '1'); semver must not.
	got := latestCompactVersion([]string{"v0.7.4-compact", "v0.16.21-compact", "v0.16.9-compact", "v0.16.3-compact"})
	if got != "v0.16.21-compact" {
		t.Fatalf("latestCompactVersion = %q, want v0.16.21-compact", got)
	}
	if got := latestCompactVersion([]string{"v0.7.4-compact", "v0.8.0-compact"}); got != "v0.8.0-compact" {
		t.Fatalf("latestCompactVersion = %q, want v0.8.0-compact", got)
	}
	if got := latestCompactVersion([]string{"weird-compact", "v0.1.0-compact"}); got != "v0.1.0-compact" {
		t.Fatalf("unparsable names must sort below parsed versions, got %q", got)
	}
}

// TST-RC1-20
func TestActivePromptIsV01621WithCompactSibling(t *testing.T) {
	active, err := prompt.GetActiveVersion()
	if err != nil {
		t.Fatalf("GetActiveVersion: %v", err)
	}
	if active != "v0.16.21" {
		t.Fatalf("active prompt = %q, want v0.16.21", active)
	}
	resolved, err := resolveCompactPromptVersion("latest")
	if err != nil {
		t.Fatalf("resolveCompactPromptVersion: %v", err)
	}
	if resolved != "v0.16.21-compact" {
		t.Fatalf("--compact resolved to %q, want v0.16.21-compact", resolved)
	}
}

// TST-RC1-20
func TestCompactPromptV01621Hygiene(t *testing.T) {
	body, err := prompt.LoadPrompt("v0.16.21-compact")
	if err != nil {
		t.Fatalf("LoadPrompt(v0.16.21-compact): %v", err)
	}
	if n := len(body); n < 3000 || n > 4300 {
		t.Errorf("compact prompt is %d bytes, want ~3.2-4.0 KB", n)
	}
	for _, bad := range []string{"**", "++", "abs_Int", "head_or", "rand_int", "printErr", "readFileBytes", "unsafeGet", "_xml_"} {
		if strings.Contains(body, bad) {
			t.Errorf("compact prompt must not contain %q", bad)
		}
	}
	for _, want := range []string{"@limit=5", "requires", "ensures", "tests [", ", _)", "ailang docs", "ailang check --fix", "match"} {
		if !strings.Contains(body, want) {
			t.Errorf("compact prompt missing %q", want)
		}
	}
}

// TST-RC1-20
func TestPromptV01621FixesAuditedContradictions(t *testing.T) {
	body, err := prompt.LoadPrompt("v0.16.21")
	if err != nil {
		t.Fatalf("LoadPrompt(v0.16.21): %v", err)
	}
	for _, bad := range []string{"head_or", "abs_Int", "rand_int", "printErr", "readFileBytes", "unsafeGet", "_xml_", "_zip_", "sunholo/gcp-auth"} {
		if strings.Contains(body, bad) {
			t.Errorf("v0.16.21 must not teach %q", bad)
		}
	}
	if re := regexp.MustCompile(`(?m)match [^\n{]*\bwith\s*$`); re.MatchString(body) {
		t.Errorf("v0.16.21 must not teach brace-less `match ... with`")
	}
	for _, want := range []string{
		"@limit=5", "import ./sibling", "filter(\\x. x > 0, _)", "tests [((", "\\acc x.",
		"std/xml", "ailang pkg add", "ailang pkg install", "ailang pkg lock", "ailang pkg search", "ailang pkg docs",
		"sunholo/gcp_auth", "mapResult", "clampInt", "ailang docs",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("v0.16.21 missing %q", want)
		}
	}
	if !strings.Contains(body, "Library modules") {
		t.Errorf("v0.16.21 must explain that library modules import Option/Result explicitly")
	}
}

// TST-RC1-20
func TestPromptV01621MirroredIntoEmbedDir(t *testing.T) {
	for _, name := range []string{"v0.16.21.md", "v0.16.21-compact.md", "versions.json"} {
		src, err := os.ReadFile(filepath.Join("..", "..", "prompts", name))
		if err != nil {
			t.Fatalf("read prompts/%s: %v", name, err)
		}
		dst, err := os.ReadFile(filepath.Join("prompts", name))
		if err != nil {
			t.Fatalf("read cmd/ailang/prompts/%s: %v", name, err)
		}
		if string(src) != string(dst) {
			t.Errorf("cmd/ailang/prompts/%s differs from prompts/%s", name, name)
		}
	}
}
