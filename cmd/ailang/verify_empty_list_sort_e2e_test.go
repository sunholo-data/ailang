package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
)

// verifyJSON is the subset of `ailang verify --json` this file asserts on.
type verifyJSON struct {
	Results []struct {
		Function string `json:"function"`
		Status   string `json:"status"`
	} `json:"results"`
}

// TestVerify_EmptyListTakesElementSortFromContext is the end-to-end acceptance
// test for ailang#689.
//
// An empty list literal carries no element type of its own, and SMT-LIB is
// monomorphically sorted, so encoding `[]` as `(Seq Int)` regardless of context
// emitted an ill-sorted term wherever the expected element type was anything
// but int. Z3 rejected the whole query with a raw sort error, which reads like
// the contract FAILED rather than like the tool declining to try.
//
// Every function in the fixture is an ERROR before the fix and VERIFIED after,
// so each is its own arm: the fixture reds if any single context regresses.
func TestVerify_EmptyListTakesElementSortFromContext(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — verify e2e needs the solver")
	}
	bin := buildAilang(t)
	stdout, stderr, _ := runAilangBin(t, bin, "verify", "--json",
		"examples/runnable/contracts/empty_list_sort_verify.ail")

	// A sort mismatch surfaces as a raw Z3 error, never as a verdict.
	combined := stdout + stderr
	for _, bad := range []string{"unknown constant", "unknown sort", "sort mismatch", "are incompatible"} {
		if strings.Contains(combined, bad) {
			t.Fatalf("verify leaked a Z3 sort error %q; output:\n%s", bad, combined)
		}
	}

	var payload verifyJSON
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("verify --json did not emit valid JSON: %v\nstdout:\n%s", err, stdout)
	}

	// Assert per-function, not on an aggregate: an aggregate over a SHORT result
	// list is vacuously green, so the expected set is named and its size checked.
	want := map[string]bool{
		"emptyDoc":    false, // record construction, four distinct element sorts
		"clearLines":  false, // record update
		"noLines":     false, // direct return, list[string]
		"noBlocks":    false, // direct return, list[ADT]
		"single":      false, // cons onto an empty tail
		"oneEmptyRow": false, // empty literal nested inside a non-empty one
		"maybeLines":  false, // if-branch
	}
	for _, r := range payload.Results {
		if _, expected := want[r.Function]; !expected {
			continue
		}
		if r.Status != "verified" {
			t.Errorf("%s: status = %q, want \"verified\"", r.Function, r.Status)
		}
		want[r.Function] = true
	}
	for fn, seen := range want {
		if !seen {
			t.Errorf("%s: absent from verify output — the fixture no longer covers this context", fn)
		}
	}
}

// TestVerify_EmptyListContractsStillFail is the discriminating half of the arm
// above. Making a shape ENCODABLE is only correct if a false contract over that
// same shape still fails: a status of "verified" must mean Z3 checked it, not
// that the encoder waved it through. Each case below is the fixture's shape with
// its contract negated, and each must report a violation.
func TestVerify_EmptyListContractsStillFail(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — verify e2e needs the solver")
	}
	bin := buildAilang(t)

	cases := []struct {
		name string
		fn   string
		src  string
	}{
		{
			name: "direct return",
			fn:   "f",
			src: `export func f() -> list[string] ! {}
ensures { listLength(result) == 1 } {
  []
}`,
		},
		{
			name: "record construction",
			fn:   "mk",
			src: `export type S = { items: list[string], tag: string }
export func mk() -> S ! {}
ensures { listLength(result.items) > 0 } {
  { items: [], tag: "x" }
}`,
		},
		{
			name: "record update",
			fn:   "clear",
			src: `export type S = { items: list[string], tag: string }
export func clear(s: S) -> S ! {}
ensures { listLength(result.items) > 0 } {
  { s | items: [] }
}`,
		},
		{
			name: "cons onto empty",
			fn:   "f",
			src: `export func f(x: string) -> list[string] ! {}
ensures { listLength(result) == 2 } {
  x :: []
}`,
		},
		{
			name: "nested empty",
			fn:   "f",
			src: `export func f() -> list[list[string]] ! {}
ensures { listLength(result) == 0 } {
  [[]]
}`,
		},
	}

	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("project root: %v", err)
	}
	// The file must live under the project root, not os.TempDir(): CWD-relative
	// path resolution behaves differently from a temp-shaped directory, so a
	// /tmp fixture fails for its LOCATION rather than for the code under test.
	dir, err := os.MkdirTemp(projectRoot, "verify_neg_")
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	rel, err := filepath.Rel(projectRoot, dir)
	if err != nil {
		t.Fatalf("rel: %v", err)
	}
	modBase := filepath.ToSlash(rel)

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name := "neg"
			path := filepath.Join(dir, name+".ail")
			src := "module " + modBase + "/" + name + "\n\n" +
				"import std/list (length as listLength)\n\n" + tc.src +
				"\n\nexport func main() -> int ! {} { 0 }\n"
			if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
				t.Fatalf("write: %v", err)
			}
			relPath := filepath.ToSlash(filepath.Join(modBase, name+".ail"))
			stdout, stderr, _ := runAilangBin(t, bin, "verify", "--json", relPath)

			var payload verifyJSON
			if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
				t.Fatalf("case %d: verify --json did not emit valid JSON: %v\nstdout:\n%s\nstderr:\n%s",
					i, err, stdout, stderr)
			}
			var found bool
			for _, r := range payload.Results {
				if r.Function != tc.fn {
					continue
				}
				found = true
				// "violated" is the honest outcome. "verified" would mean the
				// encoding made a false contract provable; "error" would mean
				// the shape is still unencodable and the positive arm above is
				// passing for the wrong reason.
				if r.Status == "verified" || r.Status == "error" {
					t.Fatalf("%s (%s): status = %q, want a violation; stdout:\n%s",
						tc.fn, tc.name, r.Status, stdout)
				}
			}
			if !found {
				t.Fatalf("%s (%s): absent from verify output; stdout:\n%s", tc.fn, tc.name, stdout)
			}
		})
	}
}
