package eval_harness

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// TestGradeInWorkspace guards M-EVAL-RELIABLE-GRADING: the grader must run a harness-owned probe
// against the module the agent implemented (kept via solution_files) in its preserved workspace —
// PASS when the implementation is correct, FAIL when it is wrong. The legacy path discarded the
// implementation entirely, so a correct multi-file solution could not pass.
func TestGradeInWorkspace(t *testing.T) {
	// The grader invokes the bare "ailang" on PATH — require it fresh.
	testutil.RequireAilangOnPath(t)

	// Probe (harness-owned) imports the agent's implementation module and prints its result.
	const probe = "module main\n" +
		"import std/io (println)\n" +
		"import lib/impl (answer)\n" +
		"export func main() -> () ! {IO} { println(answer()) }\n"
	const stub = "module lib/impl\nexport func answer() -> string = \"\"\n"

	spec := &BenchmarkSpec{
		Caps:            []string{"IO"},
		ExpectedOut:     "42",
		GradeEntrypoint: "main.ail",
		SolutionFiles:   []string{"lib/impl.ail"},
		InputFiles: map[string]string{
			"main.ail":     probe,
			"lib/impl.ail": stub,
		},
	}

	cases := []struct {
		name string
		impl string
		want bool
	}{
		{"correct_impl_passes", "module lib/impl\nexport func answer() -> string = \"42\"\n", true},
		{"wrong_impl_fails", "module lib/impl\nexport func answer() -> string = \"oops\"\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ws := t.TempDir()
			// Simulate the agent having implemented the solution_file in its workspace.
			if err := os.MkdirAll(filepath.Join(ws, "lib"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(ws, "lib", "impl.ail"), []byte(tc.impl), 0644); err != nil {
				t.Fatal(err)
			}
			res := gradeInWorkspace(spec, ws)
			if res.StdoutOk != tc.want {
				t.Errorf("StdoutOk=%v want %v (compileOk=%v runtimeOk=%v stdout=%q stderr=%q)",
					res.StdoutOk, tc.want, res.CompileOk, res.RuntimeOk, res.Stdout, res.Stderr)
			}
		})
	}
}
