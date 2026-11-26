package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

func TestComputeCheckEdits_MechanicalFixes(t *testing.T) {
	t.Run("PAR018_let_rec", func(t *testing.T) {
		src := "module m\nexport pure func f(n: int) -> int {\n  let rec go = \\x. x;\n  go(n)\n}\n"
		edits := computeCheckEdits(checkJSONError{Code: "PAR018", Line: 3, Column: 7}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := applyCheckEdits(src, edits)
		if n != 1 || !strings.Contains(got, "letrec go =") {
			t.Fatalf("applied=%d got:\n%s", n, got)
		}
	})

	t.Run("PAR_HASH_COMMENT", func(t *testing.T) {
		src := "module m\n# Python comment\nexport pure func f() -> int = 1\n"
		edits := computeCheckEdits(checkJSONError{Code: "PAR_HASH_COMMENT", Line: 2, Column: 1}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := applyCheckEdits(src, edits)
		if n != 1 || !strings.Contains(got, "-- Python comment") {
			t.Fatalf("applied=%d got:\n%s", n, got)
		}
	})

	t.Run("IMP010_rename_and_builtin", func(t *testing.T) {
		src := "module m\nimport std/string (nthChar, show)\nexport pure func f(s: string) -> string = s\n"
		e1 := computeCheckEdits(checkJSONError{
			Code:    "IMP010",
			Message: "symbol 'nthChar' not exported by 'std/string' — did you mean 'charAt'? (exported by 'std/string')",
		}, "m.ail", src)
		got1, n1 := applyCheckEdits(src, e1)
		if n1 != 1 || !strings.Contains(got1, "import std/string (charAt, show)") {
			t.Fatalf("rename failed: n=%d got:\n%s", n1, got1)
		}
		e2 := computeCheckEdits(checkJSONError{
			Code:    "IMP010",
			Message: "symbol 'show' not exported by 'std/string' — 'show' is a builtin available in every module; remove it from the import list",
		}, "m.ail", got1)
		got2, n2 := applyCheckEdits(got1, e2)
		if n2 != 1 || !strings.Contains(got2, "import std/string (charAt)") {
			t.Fatalf("builtin removal failed: n=%d got:\n%s", n2, got2)
		}
	})

	t.Run("PAR017_equals_before_brace", func(t *testing.T) {
		src := "module m\nexport pure func f(x: int) -> int = {\n  let y = x + 1;\n  y\n}\n"
		edits := computeCheckEdits(checkJSONError{Code: "PAR017", Line: 3, Column: 16}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := applyCheckEdits(src, edits)
		if n != 1 || !strings.Contains(got, "func f(x: int) -> int {") {
			t.Fatalf("PAR017 fix failed: n=%d got:\n%s", n, got)
		}
	})

	t.Run("missing_import_disambiguated_by_existing_import", func(t *testing.T) {
		src := "module m\nimport std/list (filterMap)\nexport pure func f(xs: [int]) -> [int] = map(\\x. x + 1, xs)\n"
		edits := computeCheckEdits(checkJSONError{
			Code:    "ERROR",
			Message: "undefined variable: map at m.ail:3:42 — `map` is exported by std/list, std/option, std/result",
		}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := applyCheckEdits(src, edits)
		if n != 1 || !strings.Contains(got, "import std/list (filterMap, map)") {
			t.Fatalf("import disambiguation failed: n=%d got:\n%s", n, got)
		}
	})

	t.Run("TC_MATCH_002_missing_arm_stub", func(t *testing.T) {
		src := "module m\ntype Color = Red | Green | Blue\nexport pure func f(c: Color) -> int =\n  match c {\n    Red => 1,\n    Green => 2\n  }\n"
		edits := computeCheckEdits(checkJSONError{
			Code:    "TC_MATCH_002",
			Message: "non-exhaustive match on Color — no arm matches: Blue",
			Line:    4,
			Column:  3,
		}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := applyCheckEdits(src, edits)
		if n != 1 || !strings.Contains(got, "Blue => _?") {
			t.Fatalf("TC_MATCH_002 stub failed: n=%d got:\n%s", n, got)
		}
	})
}

func TestRunCheckFix_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixme.ail")
	src := `module fixme
# Python-style comment
import std/string (show)

export pure func sumAll(xs: [int]) -> int =
  foldl(\acc x. acc + x, 0, xs)
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := pipeline.Config{DryLink: true, RelaxModules: true}
	applied, err := runCheckFix(path, cfg)
	if err != nil {
		t.Fatalf("runCheckFix error: %v", err)
	}
	if applied < 2 {
		t.Fatalf("expected at least 2 mechanical fixes applied, got %d", applied)
	}
	fixedBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	res, err := pipeline.Run(cfg, pipeline.Source{Code: string(fixedBytes), Filename: path})
	if err != nil || len(res.Errors) > 0 {
		t.Fatalf("fixed file still fails check: err=%v res.Errors=%v\nfile:\n%s", err, res.Errors, string(fixedBytes))
	}
}
