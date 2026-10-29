package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// #1797: a module whose main is declared but not exported used to exit 0 with
// no output on both engines. It must fail and say to add `export`; a module
// that exports nothing at all runs nothing, so it fails too and points at
// `ailang check`.
func TestRunUnexportedEntryIsAnError(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	nomain := filepath.Join(dir, "nomain.ail")
	writeFile(t, nomain, "module nomain\nimport std/io (println)\nfunc main() -> () ! {IO} { println(\"hi\") }\n")
	for _, bytecode := range []bool{false, true} {
		args := []string{"run", "--relax-modules", "--caps", "IO", "--entry", "main"}
		if bytecode {
			args = append(args, "--bytecode")
		}
		_, stderr, code := runWithStdin(t, bin, "", append(args, nomain)...)
		if code == 0 || !strings.Contains(stderr, "not exported") {
			t.Errorf("bytecode=%v: exit=%d stderr=%q; want a 'not exported' error", bytecode, code, stderr)
		}
	}

	helpers := filepath.Join(dir, "helpers.ail")
	writeFile(t, helpers, "module helpers\npure func two() -> int = 2\n")
	_, stderr, code := runWithStdin(t, bin, "", "run", "--relax-modules", helpers)
	if code == 0 || !strings.Contains(stderr, "exports no functions") || !strings.Contains(stderr, "ailang check") {
		t.Errorf("export-less module: exit=%d stderr=%q; want exit 1 and an 'exports no functions' error naming 'ailang check'", code, stderr)
	}
}
