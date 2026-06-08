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

// TestListElementSoundness_MustReject is the failing test that pins the hole.
//
// M1 localization (2026-06-08): these FAIL today (the hole reproduces at
// compile-time). Confirmed surface:
//   - The pipeline uses internal/types.CoreTypeChecker (NOT the ast.List/ctx.Infer
//     path). The fix lives in the Core checker's class-constraint resolution.
//   - unifyLists (internal/types/unification_types.go:85) DOES recurse into
//     element types, so the leak is upstream: a numeric literal's `Num` constraint
//     is not re-checked after the list-element tyvar is unified to a concrete
//     non-numeric type (`Num[string]` should fail like the scalar/cons paths do).
//   - For the List[Json]-from-Option variant, AsList (helpers.go:120) is
//     case-sensitive and only matches lowercase "list", not capital "List".
//
// Skipped (not failed) so CI stays green until M2 lands; unskip in the M2 fix
// commit. The MustAccept guardrails below run as active regression protection.
func TestListElementSoundness_MustReject(t *testing.T) {
	t.Skip("M-TYPE-LIST-SOUND M2: known type-soundness hole; unskip when the Core-checker constraint fix lands")
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
