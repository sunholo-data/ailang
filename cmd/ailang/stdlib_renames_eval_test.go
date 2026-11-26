package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/importhint"
)

// Post-removal contract (M-V1-CLOSEOUT-SWARM Slice 1):
//  1. Importing any of the 16 removed @deprecated stdlib aliases fails with
//     IMP010 naming its live replacement (and Note where present).
//  2. The canonical camelCase replacements evaluate identically across the
//     tree-walking evaluator and the bytecode VM.
func TestStdlibRenamesEvaluatorTranscript(t *testing.T) {
	bin := buildAilang(t)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	stdPath := filepath.Join(root, "std")
	dir := t.TempDir()

	for i, r := range importhint.StdlibRenames {
		t.Run("removed_"+r.Module+"."+r.Old, func(t *testing.T) {
			modName := fmt.Sprintf("removed_%d", i)
			p := filepath.Join(dir, modName+".ail")
			src := fmt.Sprintf("module %s\nimport %s (%s)\nexport pure func one() -> int = 1\n", modName, r.Module, r.Old)
			if err := os.WriteFile(p, []byte(src), 0o600); err != nil {
				t.Fatal(err)
			}
			_, stderr, rc := runAilangBin(t, bin, "run", "--stdlib-path", stdPath, "--entry", "one", p)
			if rc == 0 {
				t.Fatalf("importing removed %s.%s succeeded; want IMP010", r.Module, r.Old)
			}
			wantRepl := fmt.Sprintf("use '%s' from '%s'", r.New, r.Module)
			if !strings.Contains(stderr, "IMP010") || !strings.Contains(stderr, wantRepl) {
				t.Errorf("import %s (%s): want IMP010 containing %q, got:\n%s", r.Module, r.Old, wantRepl, stderr)
			}
			if r.Note != "" && !strings.Contains(stderr, r.Note) {
				t.Errorf("import %s (%s): want IMP010 note %q, got:\n%s", r.Module, r.Old, r.Note, stderr)
			}
		})
	}

	probe := filepath.Join(dir, "renameprobe.ail")
	code := `module renameprobe
import std/list (contains, headOr, lastOr, nthOr)
import std/string (charAtOr)
import std/bytes (fromString, toString, sliceLen)
import std/math (absFloat, absInt)
import std/array (fromList, get)
import std/option (Option, Some, None)
pure func showOpt(o: Option[bytes]) -> string = match o { Some(b) => toString(b), None => "none" }
export pure func summary() -> string = "${show(contains([1, 2, 3], 2))},${show(contains([1, 2, 3], 5))},${show(headOr([10, 20], 0))},${show(lastOr([10, 20], 0))},${show(nthOr([10, 20], 1, 0))},${charAtOr("abc", 1, "?")},${showOpt(sliceLen(fromString("hello"), 1, 3))},${show(absFloat(0.0 - 2.5))},${show(absInt(0 - 4))},${show(get(fromList([7, 8, 9]), 2))}"
`
	if err := os.WriteFile(probe, []byte(code), 0o600); err != nil {
		t.Fatal(err)
	}
	const want = "true,false,10,20,20,b,ell,2.5,4,9"
	for _, eng := range []struct {
		name string
		args []string
	}{
		{"evaluator", []string{"run", "--stdlib-path", stdPath, "--entry", "summary", probe}},
		{"vm", []string{"run", "--bytecode", "--stdlib-path", stdPath, "--entry", "summary", probe}},
	} {
		got, stderr, rc := runAilangBin(t, bin, eng.args...)
		if rc != 0 {
			t.Fatalf("%s: exit %d\n%s", eng.name, rc, stderr)
		}
		if strings.Contains(stderr, "TC_DEPRECATED_001") {
			t.Errorf("%s: unexpected deprecation warning on stderr:\n%s", eng.name, stderr)
		}
		if got = strings.TrimSpace(got); got != want {
			t.Fatalf("%s transcript = %q, want %q", eng.name, got, want)
		}
	}
}
