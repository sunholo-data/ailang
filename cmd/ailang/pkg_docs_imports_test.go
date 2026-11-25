package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `ailang pkg docs` ends with one canonical pkg/ import line per exported
// module, mapped through module_prefix (M-PKG-PARALLEL-AGENTS M6). The bare
// prefix spelling also resolves, but only pkg/ names exactly one package.
func TestPrintAgentDoc_CanonicalImportLines(t *testing.T) {
	dir := t.TempDir()
	manifest := "[package]\nname = \"sunholo/ailang_parse\"\nversion = \"0.60.0\"\nedition = \"1\"\nmodule_prefix = \"docparse\"\n" +
		"[exports]\nmodules = [\"docparse/services/markdown_parser\", \"docparse/types/document\"]\n[effects]\nmax = []\n"
	if err := os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	printAgentDoc(&out, []byte("# guide"), dir)
	got := out.String()
	for _, want := range []string{
		"# guide",
		"import pkg/sunholo/ailang_parse/services/markdown_parser\n",
		"import pkg/sunholo/ailang_parse/types/document\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output lacks %q:\n%s", want, got)
		}
	}

	out.Reset()
	printAgentDoc(&out, []byte("# guide"), t.TempDir()) // no manifest
	if strings.Contains(out.String(), "import ") || !strings.Contains(out.String(), "# guide") {
		t.Errorf("without a manifest only the guide prints: %q", out.String())
	}
}
