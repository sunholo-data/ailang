package pipeline

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestTypedHoles_ReportsExpectedTypeAndLocals(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name         string
		src          string
		wantContains []string
	}{
		{
			name: "anonymous_return_hole",
			src: `module hole1
export pure func addOne(x: int) -> int = _?
`,
			wantContains: []string{"`_?`", "`int`", "x: int"},
		},
		{
			name: "named_hole_in_call_arg",
			src: `module hole2
import std/list (map)
export pure func demo(xs: [int]) -> [string] = map(?toStr, xs)
`,
			wantContains: []string{"`?toStr`", "int -> string", "xs: list[int]"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".ail")
			if err := os.WriteFile(path, []byte(tc.src), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Run(Config{DryLink: true, RelaxModules: true}, Source{
				Code:     tc.src,
				Filename: path,
			})
			if err == nil {
				t.Fatalf("expected TC_HOLE error, got nil")
			}
			rep, ok := ailerrors.AsReport(err)
			if !ok {
				t.Fatalf("expected *errors.Report, got %T: %v", err, err)
			}
			if rep.Code != types.TC_HOLE {
				t.Fatalf("code = %q, want %q (msg=%s)", rep.Code, types.TC_HOLE, rep.Message)
			}
			combined := rep.Message + " " + rep.Fix.Suggestion
			for _, w := range tc.wantContains {
				if !strings.Contains(combined, w) {
					t.Errorf("TC_HOLE output %q missing %q", combined, w)
				}
			}
		})
	}
}

func TestTypedHoles_ReportsAllHolesInModule(t *testing.T) {
	dir := t.TempDir()
	src := `module multi_hole
import std/list (map)
export pure func f(x: int) -> int = _?
export pure func g(xs: [int]) -> [string] = map(?toStr, xs)
`
	path := filepath.Join(dir, "multi_hole.ail")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Run(Config{DryLink: true, RelaxModules: true}, Source{
		Code:     src,
		Filename: path,
	})
	if err == nil {
		t.Fatal("expected HoleErrors, got nil")
	}
	var holes types.HoleErrors
	if !errors.As(err, &holes) {
		t.Fatalf("expected types.HoleErrors, got %T: %v", err, err)
	}
	if len(holes) != 2 {
		t.Fatalf("got %d holes, want 2: %v", len(holes), holes)
	}
}
