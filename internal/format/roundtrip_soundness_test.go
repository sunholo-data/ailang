package format

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// roundtrip_soundness_test.go pins the FORMATTER SOUNDNESS property:
//
//	if the printer emits output for a parse-clean file, that output must re-parse.
//
// This is weaker than the corpus comment gate (it says nothing about comment
// attachment or idempotence) and deliberately so: it is the one property whose
// violation means `ailang fmt` produced source the compiler cannot read.
//
// It exists because `std/cognition.ail` violated it undetected. Every formatter
// corpus test walked `../../examples` ONLY, so std/'s 46 files — including the
// sole round-trip offender in the whole repo — were invisible to CI by
// construction. Widening the existing comment gate instead was rejected: that
// gate carries an acceptance ceiling calibrated on examples/ alone
// (inlineInteriorRefusalCeiling), so adding std/ would have failed it for
// comment-attachment reasons belonging to a different work item.

// soundnessRoots are the two trees that ship AILANG source. A root that does not
// exist is a hard failure, not a skip: a mistyped root returns zero files and
// reads exactly like a clean sweep.
var soundnessRoots = []string{"../../std", "../../examples"}

// TestFormatterOutputAlwaysReParses walks every shipped .ail file. For each file
// that parses cleanly, the printer's output (with comments, the real CLI path)
// must itself parse. A refusal is fine — fail-closed is the designed behaviour —
// but SUCCESSFUL output that cannot be re-read is a soundness defect.
func TestFormatterOutputAlwaysReParses(t *testing.T) {
	perRoot := map[string]int{}
	var emitted, refused int
	var defects []string

	for _, root := range soundnessRoots {
		if _, err := os.Stat(root); err != nil {
			t.Fatalf("corpus root %s is missing: %v (a root that does not exist "+
				"enumerates zero files and reads like a clean sweep)", root, err)
		}
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".ail") {
				return nil
			}
			data, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			p := parser.New(lexer.New(string(data), path))
			prog := p.Parse()
			if len(p.Errors()) > 0 || prog == nil || prog.File == nil {
				return nil // not formatter-eligible (negative fixtures, mid-edit files)
			}
			perRoot[root]++

			out, ferr := SourceWithComments(prog, data, Options{})
			if ferr != nil {
				refused++ // fail-closed refusal: not a soundness violation
				return nil
			}
			emitted++

			rp := parser.New(lexer.New(string(out), path))
			if reprog := rp.Parse(); len(rp.Errors()) > 0 || reprog == nil || reprog.File == nil {
				msg := "re-parse produced no file"
				if len(rp.Errors()) > 0 {
					msg = rp.Errors()[0].Error()
				}
				defects = append(defects, path+": "+msg)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}

	t.Logf("SOUNDNESS: emitted=%d refused=%d defects=%d (per-root parse-valid: %v)",
		emitted, refused, len(defects), perRoot)

	// Anti-vacuity floor. Each root must contribute, or the sweep is not covering
	// what its name claims — the exact failure that hid this defect.
	for _, root := range soundnessRoots {
		if perRoot[root] == 0 {
			t.Fatalf("corpus root %s contributed ZERO parse-valid files — instrument failure, not a pass", root)
		}
	}
	if emitted == 0 {
		t.Fatal("printer emitted output for ZERO files — instrument failure, not a pass")
	}
	for _, d := range defects {
		t.Errorf("FORMATTER SOUNDNESS DEFECT: %s", d)
	}
}

// TestBareArrowSoundnessByConstruct pins the construct class directly: a sole
// parameter of a function type may drop its parentheses only when the result
// re-parses to the same thing. The record case is the one that regressed.
func TestBareArrowSoundnessByConstruct(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"record_param", "module t\nexport func f(cb: ({a: string}) -> ()) -> () { g(cb) }\n"},
		{"record_param_with_effects", "module t\nexport func f(cb: ({a: string}) -> () ! {Msg}) -> () ! {Msg} { g(cb) }\n"},
		{"open_record_param", "module t\nexport func f(cb: ({a: string, ...}) -> ()) -> () { g(cb) }\n"},
		{"func_param", "module t\nexport func f(cb: ((int) -> int) -> int) -> int { g(cb) }\n"},
		{"tuple_param", "module t\nexport func f(cb: ((int, string)) -> bool) -> bool { g(cb) }\n"},
		{"simple_param", "module t\nexport func f(cb: (string) -> ()) -> () { g(cb) }\n"},
		{"list_param", "module t\nexport func f(cb: ([int]) -> ()) -> () { g(cb) }\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := parser.New(lexer.New(tc.src, tc.name))
			prog := p.Parse()
			if len(p.Errors()) > 0 || prog == nil || prog.File == nil {
				t.Fatalf("fixture does not parse: %v", p.Errors())
			}
			out, err := Source(prog, Options{})
			if err != nil {
				t.Fatalf("Source: %v", err)
			}
			rp := parser.New(lexer.New(string(out), tc.name))
			if reprog := rp.Parse(); len(rp.Errors()) > 0 || reprog == nil {
				t.Errorf("formatted output failed to re-parse: %v\n--- emitted ---\n%s",
					rp.Errors(), out)
			}
		})
	}
}
