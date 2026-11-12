package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #2210 review mutant M10: an open record pattern's row, once a caller binds
// it to the record's other fields, must also refill TRecord.Fields
// (safeSubstitute). The bytecode VM lays a record out from Fields: without the
// refill, `f` below was typed {name | ρ} with ρ := {id: int} carried only in
// the row, and `--strict-bytecode` read g.id from the wrong slot
// (`_concat_String: arg 0 must be string, got Int`). The evaluator is dynamic
// and printed the right answer either way.
func TestCLI_RunStrictBytecode_OpenPatternRowFieldsReachTheVM(t *testing.T) {
	src := filepath.Join(t.TempDir(), "m10b.ail")
	prog := `module m10b

export func main() -> () ! {IO} {
  let r = match {name: "Grace", id: 7} { {name, ...} => {name: name, id: 1} };
  let f = \obj. match obj { {name, ...} => obj };
  let g = f({name: "Ada", id: 9});
  println("${g.name} ${show(g.id)} ${r.name}")
}
`
	if err := os.WriteFile(src, []byte(prog), 0o644); err != nil {
		t.Fatal(err)
	}
	const want = "Ada 9 Grace"
	for _, mode := range [][]string{{"run"}, {"run", "--bytecode", "--strict-bytecode"}} {
		args := append(append([]string{}, mode...), "--relax-modules", "--caps", "IO", "--entry", "main", src)
		out, stderr, exit := runCLI(t, args...)
		if exit != 0 || strings.TrimSpace(out) != want {
			t.Fatalf("%v: exit %d, got %q, want %q\nstderr=%s", mode, exit, out, want, stderr)
		}
	}
}
