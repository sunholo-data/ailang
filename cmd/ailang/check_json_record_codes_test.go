package main

import (
	"os"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/types"
)

// #2210/#2221: the record-unification and occurs-check diagnostics are
// reported by `check --json` as their codes, not the generic "ERROR".
func TestCheckJSON_RecordRowCodesAppearAsThemselves(t *testing.T) {
	cases := map[string]string{
		types.TC_REC_001: "module m\nexport func main() -> () ! {IO} {\n  let r = {a: 1};\n  println(show(r.bogus))\n}\n",
		types.TC_OCCURS:  "module m\nexport func main() -> () ! {IO} {\n  let h = \\x. {self: x} == x;\n  println(\"x\")\n}\n",
		types.TC_REC_004: "module m\nfunc f(p: {y: int | r}, q: {x: int | r}) -> {x: int | r} = q\nexport pure func run() -> int = f({y: 1, x: \"s\"}, {x: 2}).x\n",
		types.TC_REC_003: "module m\nexport pure func f(a: {x: int | r}, b: {y: int | r}) -> {x: int | r} = if true then a else b\n",
	}
	for want, src := range cases {
		t.Run(want, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if err := os.WriteFile("m.ail", []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true}, pipeline.Source{Filename: "m.ail"})
			if err == nil {
				t.Fatalf("expected %s", want)
			}
			if got := errorToCheckJSONError(err, "m.ail").Code; got != want {
				t.Fatalf("--json code = %q, want %q (message: %v)", got, want, err)
			}
		})
	}
	for _, code := range types.RecordRowCodes() {
		if _, ok := cases[code]; !ok {
			t.Errorf("code %s has no --json case", code)
		}
	}
}
