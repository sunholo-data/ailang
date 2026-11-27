package quickfix_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/importhint"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/quickfix"
	"github.com/sunholo-data/ailang/internal/stdlibindex"
)

// TST-RC1-13
func TestIMP010CrossModuleMove(t *testing.T) {
	importhint.Locator = stdlibindex.Modules
	importhint.SymbolsOf = stdlibindex.SymbolsOf
	importhint.ModuleLocator = stdlibindex.AllModules

	t.Run("multi_symbol_import_moves_to_new_import_line", func(t *testing.T) {
		src := `module m
import std/list (map, join)

export pure func f(xs: [string]) -> string =
  join(",", map(\s. s, xs))
`
		edits := quickfix.ComputeEdits(quickfix.Diagnostic{
			Code:    "IMP010",
			Message: "symbol 'join' not exported by 'std/list' — 'join' is exported by std/string; import it from there, not 'std/list'",
		}, "m.ail", src)
		if len(edits) == 0 {
			t.Fatal("expected IMP010 cross-module move edits, got none")
		}
		got, applied := quickfix.ApplyEdits(src, edits)
		if applied == 0 {
			t.Fatalf("expected edits to apply, got applied=0\nsrc:\n%s", src)
		}
		if !strings.Contains(got, "import std/list (map)\n") {
			t.Errorf("expected import std/list (map), got:\n%s", got)
		}
		if !strings.Contains(got, "import std/string (join)\n") {
			t.Errorf("expected import std/string (join), got:\n%s", got)
		}
	})

	t.Run("single_symbol_import_replaces_or_deletes_cleanly", func(t *testing.T) {
		src := `module m
import std/list (join)

export pure func f(xs: [string]) -> string = join(",", xs)
`
		edits := quickfix.ComputeEdits(quickfix.Diagnostic{
			Code:    "IMP010",
			Message: "symbol 'join' not exported by 'std/list' — 'join' is exported by std/string; import it from there, not 'std/list'",
		}, "m.ail", src)
		if len(edits) == 0 {
			t.Fatal("expected edits for single-symbol IMP010 move, got none")
		}
		got, applied := quickfix.ApplyEdits(src, edits)
		if applied == 0 {
			t.Fatal("expected edits to apply")
		}
		if strings.Contains(got, "import std/list") {
			t.Errorf("expected offending single-symbol std/list import to be removed, got:\n%s", got)
		}
		if !strings.Contains(got, "import std/string (join)") {
			t.Errorf("expected import std/string (join), got:\n%s", got)
		}
	})

	t.Run("single_symbol_merges_into_existing_target_import", func(t *testing.T) {
		src := `module m
import std/string (split)
import std/list (join)

export pure func f(s: string) -> string = join("-", split(s, ","))
`
		edits := quickfix.ComputeEdits(quickfix.Diagnostic{
			Code:    "IMP010",
			Message: "symbol 'join' not exported by 'std/list' — 'join' is exported by std/string; import it from there, not 'std/list'",
		}, "m.ail", src)
		if len(edits) == 0 {
			t.Fatal("expected edits, got none")
		}
		got, applied := quickfix.ApplyEdits(src, edits)
		if applied == 0 {
			t.Fatal("expected edits to apply")
		}
		if strings.Contains(got, "import std/list") {
			t.Errorf("expected std/list import line deleted, got:\n%s", got)
		}
		if !strings.Contains(got, "import std/string (split, join)") {
			t.Errorf("expected merged import std/string (split, join), got:\n%s", got)
		}
	})

	t.Run("ambiguous_multi_exporter_not_auto_moved", func(t *testing.T) {
		src := `module m
import std/io (length)
export pure func f(xs: [int]) -> int = length(xs)
`
		edits := quickfix.ComputeEdits(quickfix.Diagnostic{
			Code:    "IMP010",
			Message: "symbol 'length' not exported by 'std/io' — 'length' is exported by std/array, std/bytes, std/list, std/string; import it from there, not 'std/io'",
		}, "m.ail", src)
		if len(edits) != 0 {
			t.Fatalf("expected 0 edits on ambiguous multi-module export, got %d: %+v", len(edits), edits)
		}
	})

	t.Run("end_to_end_pipeline_recheck_zero_errors", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "fix_imp010.ail")
		src := `module fix_imp010
import std/list (map, join)

export pure func render(xs: [string]) -> string =
  join(", ", map(\s. s, xs))
`
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		cfg := pipeline.Config{DryLink: true, RelaxModules: true, NoCache: true}
		_, runErr := pipeline.Run(cfg, pipeline.Source{Code: src, Filename: path})
		if runErr == nil {
			t.Fatal("expected initial IMP010 failure, got nil")
		}
		rep, ok := ailerrors.AsReport(runErr)
		if !ok || rep.Code != "IMP010" {
			t.Fatalf("expected structured IMP010 report, got ok=%v err=%v", ok, runErr)
		}
		edits := quickfix.ComputeEdits(quickfix.Diagnostic{
			Code:    rep.Code,
			Message: rep.Message,
			File:    filepath.ToSlash(path),
		}, filepath.ToSlash(path), src)
		if len(edits) == 0 {
			t.Fatalf("expected ComputeEdits to produce edits for %q, got none", rep.Message)
		}
		fixed, n := quickfix.ApplyEdits(src, edits)
		if n == 0 {
			t.Fatal("expected ApplyEdits to apply >0 edits")
		}
		if err := os.WriteFile(path, []byte(fixed), 0o644); err != nil {
			t.Fatal(err)
		}
		res2, err2 := pipeline.Run(cfg, pipeline.Source{Code: fixed, Filename: path})
		if err2 != nil || len(res2.Errors) > 0 {
			t.Fatalf("re-check after fix failed: err=%v errors=%v\nfixed:\n%s", err2, res2.Errors, fixed)
		}
	})
}
