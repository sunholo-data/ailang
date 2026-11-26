package elaborate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/elaborate"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/pipeline"
)

func runCheck(t *testing.T, dir, name, src string) error {
	t.Helper()
	path := filepath.Join(dir, name+".ail")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, RelaxModules: true}, pipeline.Source{
		Code:     src,
		Filename: path,
	})
	return err
}

// TST-TL-03: `?` outside a function returning Result[T, E] or Option[T] fails
// with TC_TRY_RETURN_TYPE and a concrete fix suggestion.
func TestTryLet_RejectsInvalidReturnTypes(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name         string
		src          string
		wantContains string
	}{
		{
			name: "returns_int",
			src: `module bad_ret_int
pure func step(x: int) -> Result[int, string] = Ok(x)
export pure func bad(x: int) -> int {
  let y = step(x)?
  y + 1
}
`,
			wantContains: "returning int",
		},
		{
			name: "unannotated_function",
			src: `module bad_ret_unannot
pure func step(x: int) -> Option[int] = Some(x)
export pure func bad(x: int) {
  let y = step(x)?
  y + 1
}
`,
			wantContains: "without a declared return type",
		},
		{
			name: "nested_backslash_lambda_inside_result_fn",
			src: `module bad_nested_lam
import std/list (map)
pure func step(x: int) -> Result[int, string] = Ok(x)
export pure func outer(xs: [int]) -> Result[[int], string] {
  let ys = map(\x. { let v = step(x)?; v + 1 }, xs)
  Ok(ys)
}
`,
			wantContains: "lambda",
		},
		{
			name: "module_level_let",
			src: `module bad_mod_let
let top = Ok(1)?
export pure func f() -> int = top
`,
			wantContains: "outside a function",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runCheck(t, dir, tc.name, tc.src)
			if err == nil {
				t.Fatalf("expected %s error, got nil", elaborate.CodeTryReturnType)
			}
			rep, ok := ailerrors.AsReport(err)
			if !ok {
				if !strings.Contains(err.Error(), elaborate.CodeTryReturnType) {
					t.Fatalf("expected code %s, got %T: %v", elaborate.CodeTryReturnType, err, err)
				}
				return
			}
			if rep.Code != elaborate.CodeTryReturnType {
				t.Fatalf("code = %q, want %q (msg=%s)", rep.Code, elaborate.CodeTryReturnType, rep.Message)
			}
			if !strings.Contains(rep.Message, tc.wantContains) {
				t.Errorf("message %q missing %q", rep.Message, tc.wantContains)
			}
			if rep.Fix == nil || rep.Fix.Suggestion == "" {
				t.Errorf("expected non-empty Fix.Suggestion on %s", rep.Code)
			}
		})
	}
}

// TST-TL-03: `?` inside a non-spine position (such as a `let` RHS block, call
// argument block, `if` condition, or discarded statement) fails with
// TC_TRY_NON_SPINE so `Ok(Err(_))` can never occur.
func TestTryLet_RejectsNonSpinePositions(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "non_spine_let_rhs_block",
			src: `module non_spine_1
pure func step() -> Result[int, string] = Err("boom")
export pure func bad() -> Result[int, string] =
  let x = {
    let y = step()?
    y + 1
  } in
  Ok(x)
`,
		},
		{
			name: "non_spine_inside_ok_call_arg",
			src: `module non_spine_2
pure func step() -> Result[int, string] = Err("boom")
export pure func bad() -> Result[int, string] =
  Ok({
    let y = step()?
    y + 1
  })
`,
		},
		{
			name: "non_spine_if_condition",
			src: `module non_spine_3
pure func check() -> Option[bool] = None
export pure func bad() -> Option[int] =
  if { let b = check()?; b } then Some(1) else Some(0)
`,
		},
		{
			name: "non_spine_discarded_block_branch",
			src: `module non_spine_4
pure func step() -> Result[int, string] = Err("boom")
export pure func bad(flag: bool) -> Result[int, string] {
  if flag then {
    let _ = step()?
    ()
  } else ();
  Ok(42)
}
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := runCheck(t, dir, tc.name, tc.src)
			if err == nil {
				t.Fatalf("expected %s error, got nil", elaborate.CodeTryNonSpine)
			}
			rep, ok := ailerrors.AsReport(err)
			if !ok {
				t.Fatalf("expected structured *errors.Report, got %T: %v", err, err)
			}
			if rep.Code != elaborate.CodeTryNonSpine {
				t.Fatalf("code = %q, want %q (msg=%s)", rep.Code, elaborate.CodeTryNonSpine, rep.Message)
			}
			if rep.Fix == nil || !strings.Contains(strings.ToLower(rep.Fix.Suggestion), "hoist") {
				t.Errorf("expected hoist suggestion in Fix, got %+v", rep.Fix)
			}
		})
	}
}

// Positive check: spine positions (block statements, let..in chains, tail if/match
// arms, annotated FuncLit callbacks, and Result/Option type aliases) type-check cleanly.
func TestTryLet_AcceptsValidSpineAndFuncLitAndTypeAlias(t *testing.T) {
	dir := t.TempDir()
	src := `module valid_spine
import std/list (map)

type AppResult = Result[int, string]

pure func parseNonNeg(n: int) -> Result[int, string] =
  if n >= 0 then Ok(n) else Err("negative")

export pure func viaAlias(a: int, b: int) -> AppResult {
  let x = parseNonNeg(a)?
  if x == 0 then
    Ok(0)
  else {
    let y = parseNonNeg(b)?
    Ok(x + y)
  }
}

export pure func viaFuncLit(xs: [int]) -> [Result[int, string]] =
  map(func(n: int) -> Result[int, string] {
    let v = parseNonNeg(n)?
    Ok(v * 10)
  }, xs)
`
	if err := runCheck(t, dir, "valid_spine", src); err != nil {
		t.Fatalf("expected valid_spine to type-check cleanly, got: %v", err)
	}
}
