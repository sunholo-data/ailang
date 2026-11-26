package format

import (
	"strings"
	"testing"
)

func TestUnbracedBranchesNormalizeToBracedBlocks(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "match_arm_newline_let_chain",
			src: `module m

export pure func f(xs: [int]) -> int =
  match xs {
    [] => 0,
    x :: rest =>
      let y = x * 2
      let z = y + 1
      z
  }
`,
		},
		{
			name: "match_arm_semicolon_sequence",
			src: `module m

export pure func f(x: int) -> int =
  match x {
    0 => let a = 1; let b = a + 2; b,
    _ => 0
  }
`,
		},
		{
			name: "if_then_else_unbraced_let_chains",
			src: `module m

export pure func f(c: bool) -> int {
  if c then
    let x = 1;
    x + 1
  else
    let y = 2
    y * 3
}
`,
		},
		{
			name: "nested_match_in_unbraced_arm",
			src: `module m

export pure func f(xs: [int]) -> int =
  match xs {
    x :: rest =>
      let y = x + 1
      match y {
        0 =>
          let z = 10
          z
        n => n * 2
      }
    [] => 0
  }
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := assertIdempotentAndRoundTrips(t, tc.src, "test://"+tc.name)
			// Canonical formatter output must normalize unbraced multi-statement
			// arms/branches into braced `{ ... }` blocks.
			if !strings.Contains(out, "{\n") {
				t.Errorf("expected normalized braced block in formatted output, got:\n%s", out)
			}
		})
	}
}

func TestUnbracedBranchesPreserveComments(t *testing.T) {
	src := `module m

export pure func f(opt: Option[int]) -> int =
  match opt {
    -- empty case
    None => 0,
    -- non-empty case
    Some(x) =>
      let y = x * 2
      -- compute final value
      y + 1
  }
`
	prog := parseProg(t, src, "test://comments")
	out, err := SourceWithComments(prog, []byte(src), Options{})
	if err != nil {
		t.Fatalf("Source with comments failed: %v", err)
	}
	s := string(out)
	for _, want := range []string{"-- empty case", "-- non-empty case", "-- compute final value"} {
		if !strings.Contains(s, want) {
			t.Errorf("formatted output lost comment %q:\n%s", want, s)
		}
	}
	// Verify `-- non-empty case` stayed BEFORE `Some(x) =>`, not moved inside `{ ... }`.
	idxComment := strings.Index(s, "-- non-empty case")
	idxArm := strings.Index(s, "Some(x) =>")
	if idxComment > idxArm {
		t.Errorf("comment '-- non-empty case' was moved after arm pattern:\n%s", s)
	}
}
