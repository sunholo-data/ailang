package codegen_golden_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// engineCase is one program under engines/: its summary() must print want on
// the evaluator, --bytecode and --strict-bytecode, with no evaluator fallback.
type engineCase struct {
	dir, entry, want string
	// strictBC001 names the issue when --strict-bytecode is known to refuse
	// the program (BC001); the evaluator and --bytecode (which falls back)
	// must still print want. When the VM learns the program, this test fails
	// and the pin comes out.
	strictBC001 string
}

// The coverage the Go emitter's codegen harness gave, kept when the emitter
// was removed (#2389): the 27-call std/list/string/math/option parity program,
// the ADT-alias parity program, the ten known-failure programs (each was a
// Go-only bug; their evaluator answers are the right ones), and the audio
// behaviours that were asserted only through emitted Go.
var engineCases = []engineCase{
	{dir: "engines", entry: "stdlib_parity.ail", want: strings.Join([]string{
		"map=[2, 4, 6]", "filter=[2, 3]", "foldl=10", "reverse=[3, 2, 1]", "take=[5, 6]",
		"drop=[6, 7]", "any=true", "concat=[1, 2, 3]", "length=3", "range=[1, 2, 3]",
		"contains=true", "flatMap=[1, 1, 2, 2]", "dedup=[1, 2]", "trim=[hi]", "upper=ABC",
		"lower=abc", `split=["a", "b", "c"]`, "substring=el", "repeat=ababab",
		`chars=["a", "b", "c"]`, "sqrt=4.0", "pow=1024.0", "floor=2.0", "ceil=3.0",
		"absInt=3", "absFloat=2.5", "option=Some(4),None",
	}, "\n")},
	{dir: "engines", entry: "adt_alias_parity.ail", want: "sw=1,2,3\nchain=1,100,2,3\nopt=42,0\nlight=Amber"},
	{dir: "engines", entry: "contains_records.ail", want: "true"},        // #2241
	{dir: "engines", entry: "curried_lambda.ail", want: "6"},             // #2264
	{dir: "engines", entry: "export_record_return.ail", want: "dog 6.0"}, // #2307
	{dir: "engines", entry: "generic_string_eq.ail", want: "eq=false"},   // #2332
	{dir: "engines", entry: "let_into_helper.ail", want: "true"},         // #2262
	{dir: "engines", entry: "std_helper_headOr.ail", want: "4"},          // #2259
	{dir: "engines", entry: "std_helper_zipWith.ail", want: "[11, 22]"},  // #2259
	{dir: "engines", entry: "string_length.ail", want: "5"},              // #2261
	{dir: "engines", entry: "zip_tuples.ail", want: "[(1, 3), (2, 4)]"},  // #2263
	{dir: "engines/nominal_same_name", entry: "app.ail", want: "1,x", strictBC001: "#2260"},
	// First-class wavHeaderChecked/pcm16ToFloats, packed Array set/get, and
	// user top-level `sin`/`trim` that must not be replaced by builtins.
	{dir: "engines", entry: "audio_probe.ail", want: "0.0 0.5 true true 15 true 46"},
}

func TestEngineParityCorpus(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	// Every program under engines/ is in the table (an unlisted one never runs).
	listed := map[string]bool{}
	for _, c := range engineCases {
		listed[filepath.Join(c.dir, c.entry)] = true
	}
	var unlisted []string
	_ = filepath.Walk("engines", func(p string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(p, ".ail") && !listed[p] && filepath.Dir(p) == "engines" {
			unlisted = append(unlisted, p)
		}
		return nil
	})
	sort.Strings(unlisted)
	if len(unlisted) > 0 {
		t.Errorf("programs under engines/ not in engineCases: %v", unlisted)
	}

	for _, c := range engineCases {
		t.Run(strings.TrimSuffix(c.entry, ".ail")+"@"+filepath.Base(c.dir), func(t *testing.T) {
			dir, err := filepath.Abs(c.dir)
			if err != nil {
				t.Fatal(err)
			}
			run := func(flags ...string) (string, string, error) {
				args := append([]string{"run", "--quiet", "--relax-modules"}, flags...)
				cmd := exec.Command(ailangBin, append(args, "--entry", "summary", c.entry)...)
				cmd.Dir = dir
				var stdout, stderr bytes.Buffer
				cmd.Stdout, cmd.Stderr = &stdout, &stderr
				err := cmd.Run()
				return strings.TrimSpace(stdout.String()), stderr.String(), err
			}
			for _, flags := range [][]string{nil, {"--bytecode"}, {"--strict-bytecode"}} {
				got, stderr, err := run(flags...)
				strict := len(flags) == 1 && flags[0] == "--strict-bytecode"
				if strict && c.strictBC001 != "" {
					if err == nil || !strings.Contains(stderr, "BC001") {
						t.Errorf("--strict-bytecode: want the known BC001 refusal (%s), got err=%v out=%q\n%s"+
							"\nIf the VM runs it now, drop strictBC001 for this case.", c.strictBC001, err, got, stderr)
					}
					continue
				}
				if err != nil || got != c.want {
					t.Errorf("%v: err=%v, got %q, want %q\n%s", flags, err, got, c.want, stderr)
				}
				fellBack := strings.Contains(stderr, "falling back") || strings.Contains(stderr, "evaluator-only")
				if len(flags) == 1 && flags[0] == "--bytecode" && fellBack && c.strictBC001 == "" {
					t.Errorf("--bytecode fell back to the evaluator:\n%s", stderr)
				}
			}
		})
	}
}
