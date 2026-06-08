package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-TYPE-LIST-ELEMENT-SOUNDNESS
//
// A list whose element type does not match the expected element type must be
// rejected at COMPILE time. Today `needStrs([42])` and `Json`-into-`[string]`
// type-check and only fail at runtime in _str_join ("expected string, got
// tagged value"). These tests pin the hole (must-reject) and guard the
// neighbouring sound cases (must-accept) so the fix can't over-tighten.

// checkListSoundnessSource type-checks a single-module source (ModeCheck, no
// run) from an isolated temp dir and returns the resulting error (nil = compiles).
func checkListSoundnessSource(t *testing.T, src string) error {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "ailang-list-soundness")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.ail")
	if err := os.WriteFile(testFile, []byte(src), 0644); err != nil {
		t.Fatalf("write test.ail: %v", err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(orig)

	_, runErr := Run(Config{Mode: ModeCheck}, Source{Filename: "test.ail"})
	return runErr
}

// TestListElementSoundness_MustReject pins the hole — now CLOSED by the C3 fix.
//
// Root cause (found 2026-06-08): Scheme.Instantiate silently dropped the
// scheme's class constraints, so a let-bound `[42]` generalized to
// `forall a. Num a => [a]` lost its `Num` obligation at the use site — unifying
// `[a'] ~ [string]` accepted `needStrs([42])`. Fix: InstantiateWithConstraints
// re-emits the freshened constraints, so the existing ground-constraint resolver
// rejects `Num[string]` (same path the scalar `needStr(42)` already used). The
// annotated case exposed a second bug: `let x: T` annotations were dropped during
// elaboration entirely (`let x: int = "hello"` compiled) — now enforced via a
// recorded annotation + `valueType ~ annot` in inferLet. Both subcases now pass.
func TestListElementSoundness_MustReject(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			// THE hole: a numeric-literal list into a [string] parameter.
			name: "numeric literal list into [string] param",
			src: `module test
pure func needStrs(xs: [string]) -> string = "ok"
export pure func main() -> string = needStrs([42])
`,
		},
		{
			// Same hole via an explicit annotation.
			name: "annotated [string] bound to [42]",
			src: `module test
export pure func main() -> string {
  let xs: [string] = [42];
  "ok"
}
`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkListSoundnessSource(t, tc.src)
			if err == nil {
				t.Fatalf("SOUNDNESS HOLE: expected a compile-time type error (int element vs [string]), got none")
			}
			// The error must be AILANG-level, not a Go-internal leak.
			if strings.Contains(err.Error(), "*types.") || strings.Contains(err.Error(), "tagged value") {
				t.Errorf("error leaks Go internals / is a runtime message: %s", err.Error())
			}
		})
	}
}

// TestListElementSoundness_ConstructorPatternBind pins round 2 — CLOSED.
//
// Root cause: in checkPattern (internal/types/typechecker_patterns.go) the
// ConstructorPattern arg loop bound each pattern variable to an orphaned fresh
// type var, never tied to the constructor's field type or the scrutinee's
// resolved type arg. So a destructured value — `Some(b)` from `Option[Box]`, or
// even `Wrap(n)` from a monomorphic local ADT — lost its concrete type, and
// `[b]` unified freely with `[string]`. This is the json_parse shape (getObject
// returns Option[Json]). Fix: instantiate the constructor's factory scheme
// (`$adt.make_<T>_<C>`) so field/result type-param vars are shared, constrain
// `scrutinee ~ result`, and bind each arg to the real field type.
func TestListElementSoundness_ConstructorPatternBind(t *testing.T) {
	cases := []struct {
		name, src string
	}{
		{"imported Option[Box] extracted into [string] (json_parse shape)", `module test
import std/option (Option, Some, None)
type Box = Wrap(int)
pure func needStrs(xs: [string]) -> string = "ok"
pure func getBox() -> Option[Box] = Some(Wrap(1))
export func main() -> () ! {IO} { match getBox() { Some(b) => { let _ = needStrs([b]); () }, None => () } }
`},
		{"monomorphic local ADT destructured directly", `module test
type Box = Wrap(int)
pure func needStrs(xs: [string]) -> string = "ok"
export func main() -> () ! {IO} { match Wrap(1) { Wrap(n) => { let _ = needStrs([n]); () } } }
`},
		{"multi-param Result Ok-extraction into [string]", `module test
import std/result (Result, Ok, Err)
pure func needStrs(xs: [string]) -> string = "ok"
pure func get() -> Result[int, string] = Ok(1)
export func main() -> () ! {IO} { match get() { Ok(n) => { let _ = needStrs([n]); () }, Err(_) => () } }
`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkListSoundnessSource(t, tc.src)
			if err == nil {
				t.Fatalf("SOUNDNESS HOLE: destructured non-string value into [string] should be rejected")
			}
			if strings.Contains(err.Error(), "*types.") || strings.Contains(err.Error(), "tagged value") {
				t.Errorf("error leaks Go internals / is a runtime message: %s", err.Error())
			}
		})
	}
}

// TestListElementSoundness_ConstructorPatternBind_NoOvertighten guards that the
// round-2 fix links arg<->field type WITHOUT forcing wrong concreteness: a value
// extracted from a polymorphic constructor and used at its genuine type must
// still compile.
func TestListElementSoundness_ConstructorPatternBind_NoOvertighten(t *testing.T) {
	cases := []struct {
		name, src string
	}{
		{"polymorphic Option resolved to int, used as int", `module test
import std/option (Option, Some, None)
pure func ident(o: Option[int]) -> Option[int] = o
export func main() -> () ! {IO} { let r = match ident(Some(7)) { Some(x) => x + 1, None => 0 }; let _ = r; () }
`},
		{"extracted string used correctly in [string]", `module test
import std/option (Option, Some, None)
pure func needStrs(xs: [string]) -> string = "ok"
pure func getS() -> Option[string] = Some("hi")
export func main() -> () ! {IO} { match getS() { Some(s) => { let _ = needStrs([s]); () }, None => () } }
`},
		{"Result Ok value used at its real type", `module test
import std/result (Result, Ok, Err)
pure func get() -> Result[int, string] = Ok(5)
export func main() -> () ! {IO} { let r = match get() { Ok(n) => n + 1, Err(_) => 0 }; let _ = r; () }
`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkListSoundnessSource(t, tc.src); err != nil {
				t.Errorf("expected this valid program to compile, got: %v", err)
			}
		})
	}
}

// TestListElementSoundness_MustAccept guards the neighbouring sound cases so the
// fix does not over-tighten (empty list, numeric defaulting, nesting, typed).
func TestListElementSoundness_MustAccept(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "homogeneous string list into [string]",
			src: `module test
pure func needStrs(xs: [string]) -> string = "ok"
export pure func main() -> string = needStrs(["a", "b"])
`,
		},
		{
			name: "numeric list defaults to [int]",
			src: `module test
pure func sumLen(xs: [int]) -> int = 3
export pure func main() -> int = sumLen([1, 2, 3])
`,
		},
		{
			name: "empty list unifies with [string]",
			src: `module test
pure func needStrs(xs: [string]) -> string = "ok"
export pure func main() -> string = needStrs([])
`,
		},
		{
			name: "nested int lists",
			src: `module test
pure func needNested(xs: [[int]]) -> int = 0
export pure func main() -> int = needNested([[1], [2, 3]])
`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkListSoundnessSource(t, tc.src); err != nil {
				t.Errorf("expected this valid program to compile, got: %v", err)
			}
		})
	}
}
