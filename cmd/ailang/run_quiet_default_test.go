package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// `ailang run` stdout is the program's output and nothing else. The progress
// lines ("→ Type checking...", "→ Effect checking...", "✓ Running ...") used to
// go to stdout by default, so any benchmark whose output is compared byte for
// byte (a quine above all) saw them mixed into its own output. Measured
// 2026-09-30: a local motoko quine run spent several steps grepping --help for a
// quiet flag. Progress is now opt-in (--verbose) and goes to stderr.
func TestRun_StdoutIsProgramOutputOnly(t *testing.T) {
	bin := testutil.FindAilangBinary(t)
	repoRoot := findRepoRootForTest(t)

	dir := t.TempDir()
	fp := filepath.Join(dir, "quietprobe.ail")
	prog := "module quietprobe\n\nimport std/io (print)\n\nexport func main() -> () ! {IO} {\n  print(\"X\")\n}\n"
	if err := os.WriteFile(fp, []byte(prog), 0o644); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) (string, string) {
		t.Helper()
		cmd := exec.Command(bin, append([]string{"run"}, append(args, "--entry", "main", "--caps", "IO", fp)...)...)
		cmd.Env = append(os.Environ(), "AILANG_STDLIB_PATH="+filepath.Join(repoRoot, "std"))
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("ailang run %v: %v\nstderr:\n%s", args, err, stderr.String())
		}
		return stdout.String(), stderr.String()
	}

	for _, tc := range []struct {
		name         string
		args         []string
		wantProgress bool
	}{
		{"default", nil, false},
		{"--quiet", []string{"--quiet"}, false},
		{"--verbose", []string{"--verbose"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr := run(tc.args...)
			if stdout != "X" {
				t.Errorf("stdout = %q, want exactly the program's output %q", stdout, "X")
			}
			if got := strings.Contains(stderr, "Type checking"); got != tc.wantProgress {
				t.Errorf("progress on stderr = %v, want %v; stderr:\n%s", got, tc.wantProgress, stderr)
			}
		})
	}
}
