package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// makePathDepManifest creates a dep package directory with an ailang.toml
// and returns the directory path.
func makePathDepManifest(t *testing.T, name, version string) string {
	t.Helper()
	dir := t.TempDir()
	content := "[package]\nname = \"" + name + "\"\nversion = \"" + version + "\"\nedition = \"1\"\n"
	if err := os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestRewritePathDeps_Whitespace verifies that rewritePathDepsForPublish handles
// all TOML whitespace formatting variants, not just single-space.
func TestRewritePathDeps_Whitespace(t *testing.T) {
	depDir := makePathDepManifest(t, "sunholo/myext", "0.2.0")

	cases := []struct {
		name    string
		toml    string
		wantRew bool
	}{
		{
			name:    "single-space (canonical)",
			toml:    "[package]\nname = \"myorg/host\"\nversion = \"1.0.0\"\nedition = \"1\"\n\n[dependencies]\n\"sunholo/myext\" = { path = \"" + filepath.ToSlash(depDir) + "\" }\n",
			wantRew: true,
		},
		{
			name:    "aligned padding around equals",
			toml:    "[package]\nname = \"myorg/host\"\nversion = \"1.0.0\"\nedition = \"1\"\n\n[dependencies]\n\"sunholo/myext\"   = { path = \"" + filepath.ToSlash(depDir) + "\" }\n",
			wantRew: true,
		},
		{
			name:    "extra spaces inside braces",
			toml:    "[package]\nname = \"myorg/host\"\nversion = \"1.0.0\"\nedition = \"1\"\n\n[dependencies]\n\"sunholo/myext\" = {  path = \"" + filepath.ToSlash(depDir) + "\"  }\n",
			wantRew: true,
		},
		{
			name:    "no path dep — no rewrite",
			toml:    "[package]\nname = \"myorg/host\"\nversion = \"1.0.0\"\nedition = \"1\"\n\n[dependencies]\n\"sunholo/myext\" = \"0.1.0\"\n",
			wantRew: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tomlPath := filepath.Join(dir, "ailang.toml")
			if err := os.WriteFile(tomlPath, []byte(tc.toml), 0600); err != nil {
				t.Fatal(err)
			}

			manifest, err := pkg.LoadManifest(dir)
			if err != nil {
				t.Fatalf("LoadManifest: %v", err)
			}

			rewritten, err := rewritePathDepsForPublish(dir, manifest)
			if err != nil {
				t.Fatalf("rewritePathDepsForPublish error: %v", err)
			}
			if rewritten != tc.wantRew {
				t.Errorf("rewritten = %v, want %v", rewritten, tc.wantRew)
			}

			if tc.wantRew {
				result, _ := os.ReadFile(tomlPath)
				if strings.Contains(string(result), `path = "`) {
					t.Errorf("path dep not rewritten; file still contains path ref:\n%s", result)
				}
				if !strings.Contains(string(result), `"sunholo/myext" = "0.2.0"`) {
					t.Errorf("expected registry dep %q in output, got:\n%s", `"sunholo/myext" = "0.2.0"`, result)
				}
			}
		})
	}
}

// #636: publish --dry-run is the last reversible moment before an immutable
// publish, so every digest it shows must be the full 256-bit value, not a
// 68-bit prefix that only looks like something an operator can verify.
func TestPrintPublishDigests_Full(t *testing.T) {
	tarball := pkg.TarballHash([]byte("tarball bytes"))
	content := "sha256:" + strings.Repeat("c", 64)
	iface := "sha256:" + strings.Repeat("1", 64)
	v2 := "sha256:ifacev2:" + strings.Repeat("2", 64)

	var buf bytes.Buffer
	printPublishDigests(&buf, 1234, tarball, content, iface, v2)
	out := buf.String()
	for _, want := range []string{"1234 bytes", tarball, content, iface, v2} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing full value %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "...") {
		t.Errorf("digests must not be truncated:\n%s", out)
	}

	buf.Reset()
	printPublishDigests(&buf, 1, tarball, content, iface, "") // v2 shadow-mode failure
	if strings.Contains(buf.String(), "(v2)") {
		t.Errorf("an absent v2 hash must not print an empty line:\n%s", buf.String())
	}
}
