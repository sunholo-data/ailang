package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/elaborate"
)

// TST-RC1-12
func TestStrictFallbacksStructuralListBaseCase(t *testing.T) {
	dir := t.TempDir()
	checkFallbacks := func(t *testing.T, name, src string) []elaborate.Warning {
		t.Helper()
		p := filepath.Join(dir, name+".ail")
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		res, err := Run(Config{Mode: ModeCheck, RelaxModules: true, NoCache: true}, Source{
			Code:     src,
			Filename: p,
		})
		if err != nil {
			t.Fatalf("Run(%s) unexpected error: %v", filepath.ToSlash(p), err)
		}
		if len(res.Errors) > 0 {
			t.Fatalf("Run(%s) unexpected result errors: %v", filepath.ToSlash(p), res.Errors)
		}
		var out []elaborate.Warning
		for _, w := range res.Warnings {
			if sf, ok := w.(*StrictFallbackWarning); ok {
				out = append(out, sf)
			}
		}
		return out
	}

	t.Run("cons_structural_recursion_exempted", func(t *testing.T) {
		src := `module sf_cons_ok
pure func checkPos(x: int) -> Result[int, string] =
  if x > 0 then Ok(x) else Err("non-positive")

export pure func traversePos(xs: [int]) -> Result[[int], string] =
  match xs {
    [] => Ok([]),
    x :: rest => match checkPos(x) {
      Ok(v) => match traversePos(rest) {
        Ok(vs) => Ok(v :: vs),
        Err(e) => Err(e)
      },
      Err(e) => Err(e)
    }
  }
`
		if ws := checkFallbacks(t, "sf_cons_ok", src); len(ws) != 0 {
			t.Fatalf("expected 0 STRICT_FALLBACK_001 warnings on [] => Ok([]) with x :: rest sibling, got %d: %v", len(ws), ws)
		}
	})

	t.Run("list_spread_sibling_exempted", func(t *testing.T) {
		src := `module sf_spread_ok
export pure func copyList(xs: [int]) -> Result[[int], string] =
  match xs {
    [] => Ok([]),
    [x, ...rest] => match copyList(rest) {
      Ok(vs) => Ok(x :: vs),
      Err(e) => Err(e)
    }
  }
`
		if ws := checkFallbacks(t, "sf_spread_ok", src); len(ws) != 0 {
			t.Fatalf("expected 0 STRICT_FALLBACK_001 warnings on [] => Ok([]) with [x, ...rest] sibling, got %d: %v", len(ws), ws)
		}
	})

	t.Run("swallowers_still_flagged", func(t *testing.T) {
		cases := []struct {
			name string
			src  string
		}{
			{
				name: "err_to_empty_ok",
				src: `module sf_err_bad
export pure func swallowErr(r: Result[[int], string]) -> Result[[int], string] =
  match r {
    Ok(xs) => Ok(xs),
    Err(_) => Ok([])
  }
`,
			},
			{
				name: "none_to_empty_ok",
				src: `module sf_none_bad
export pure func swallowNone(o: Option[[int]]) -> Result[[int], string] =
  match o {
    Some(xs) => Ok(xs),
    None => Ok([])
  }
`,
			},
			{
				name: "wildcard_to_empty_ok",
				src: `module sf_wild_bad
export pure func swallowWild(xs: [int]) -> Result[[int], string] =
  match xs {
    _ => Ok([])
  }
`,
			},
			{
				name: "empty_list_without_nonempty_list_sibling",
				src: `module sf_no_list_sibling
export pure func notRecursion(xs: [int]) -> Result[[int], string] =
  match xs {
    [] => Ok([]),
    _ => Ok([1])
  }
`,
			},
			{
				name: "empty_list_arm_returning_empty_string_still_flagged",
				src: `module sf_empty_str_bad
export pure func wrongEmptyKind(xs: [string]) -> Result[string, string] =
  match xs {
    [] => Ok(""),
    x :: _ => Ok(x)
  }
`,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				ws := checkFallbacks(t, tc.name, tc.src)
				if len(ws) == 0 {
					t.Fatalf("%s: expected STRICT_FALLBACK_001 warning, got 0", tc.name)
				}
			})
		}
	})
}

// TST-RC1-14
func TestWarningSpansAndWarningFile(t *testing.T) {
	type warningFiler interface{ WarningFile() string }
	type warningPositioner interface{ Position() (int, int) }

	dir := t.TempDir()
	t.Chdir(dir)
	depPath := filepath.Join(dir, "dep.ail")
	depSrc := `module dep
import std/string (split)
import std/list (take, flatMap, length)

func expand(x: int) -> [int] { [x, x] }

export pure func runTraps(name: string) -> int {
  let parts = split("/", name);
  let items = take(2, flatMap(expand, [1, 2, 3]));
  length(parts) + length(items)
}
`
	mainPath := filepath.Join(dir, "main.ail")
	mainSrc := `module main
import dep (runTraps)
export pure func main() -> int = runTraps("a/b")
`
	if err := os.WriteFile(depPath, []byte(depSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Run(Config{Mode: ModeCheck, RelaxModules: true, NoCache: true}, Source{
		Code:     mainSrc,
		Filename: mainPath,
	})
	if err != nil || len(res.Errors) > 0 {
		t.Fatalf("Run failed: err=%v errors=%v", err, res.Errors)
	}

	var sawSplit, sawTakeFlatMap bool
	for _, w := range res.Warnings {
		switch w.(type) {
		case *ArgOrderWarning:
			sawSplit = true
		case *TakeAfterFlatMapWarning:
			sawTakeFlatMap = true
		default:
			continue
		}
		filer, ok := w.(warningFiler)
		if !ok {
			t.Fatalf("%T does not implement WarningFile() string", w)
		}
		wf := filepath.ToSlash(filer.WarningFile())
		if wf == "" || !(wf == "dep.ail" || strings.HasSuffix(wf, "/dep.ail")) {
			t.Errorf("%T WarningFile() = %q, want non-empty path for dep.ail", w, wf)
		}
		pos, ok := w.(warningPositioner)
		if !ok {
			t.Fatalf("%T does not implement Position()", w)
		}
		line, col := pos.Position()
		if line <= 0 || col <= 0 {
			t.Errorf("%T Position() = %d:%d, want >0", w, line, col)
		}
		rendered := filepath.ToSlash(w.String())
		if !strings.Contains(rendered, "dep.ail:") {
			t.Errorf("%T String() missing source file:line span:\n%s", w, rendered)
		}
	}
	if !sawSplit {
		t.Fatal("expected ArgOrderWarning in res.Warnings, got none")
	}
	if !sawTakeFlatMap {
		t.Fatal("expected TakeAfterFlatMapWarning in res.Warnings, got none")
	}
}
