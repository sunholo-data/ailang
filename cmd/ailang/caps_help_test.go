package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Through the real binary: the typo is a non-zero exit at invocation, not a
// program that runs with a capability quietly missing.
func TestRun_UnknownCapabilityIsFatal(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join(t.TempDir(), "p.ail")
	if err := os.WriteFile(src, []byte("module p\nimport std/io (println)\nexport func main() -> () ! {IO} { println(\"x\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runAilangBin(t, bin, "run", "--relax-modules", "--caps", "IO,FS,Nett", "--entry", "main", src)
	if code == 0 {
		t.Fatalf("--caps IO,FS,Nett exited 0\nstdout:\n%s", stdout)
	}
	if strings.Contains(stdout, "\nx\n") || strings.HasPrefix(stdout, "x\n") {
		t.Errorf("program ran despite the rejected --caps\nstdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, `unknown capability "Nett"`) || !strings.Contains(stderr, "did you mean Net") {
		t.Errorf("stderr should name the entry and the suggestion:\n%s", stderr)
	}
}
