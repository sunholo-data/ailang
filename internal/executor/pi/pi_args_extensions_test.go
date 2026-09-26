package pi

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// IsolateFromProjectExtensions must emit BOTH flags, and only when asked.
//
// --approve is not decoration. --no-extensions also disables the globally installed
// workspace-trust extension, which is what grants project trust headlessly; without trust pi
// ignores project-local .agents/skills. Measured 2026-09-14: with --no-extensions alone the
// gate was gone but `sprint-evaluator` was NOT in the model's skills list — and that skill is
// the evaluator's terminator (100-point rubric, threshold 70). Dropping it trades a visible
// deadlock for an invisibly unqualified judge.
func TestBuildPiArgs_ProjectExtensionIsolation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		isolate bool
		want    bool
	}{
		{"isolated evaluator", true, true},
		{"author role", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args, err := buildPiArgs("m", &executor.Task{IsolateFromProjectExtensions: tc.isolate}, "do it")
			if err != nil {
				t.Fatal(err)
			}
			joined := strings.Join(args, " ")
			for _, flag := range []string{"--no-extensions", "--approve"} {
				if got := strings.Contains(joined, flag); got != tc.want {
					t.Errorf("%s present = %v, want %v (args: %s)", flag, got, tc.want, joined)
				}
			}
			// Skills must never be disabled by this path.
			if strings.Contains(joined, "--no-skills") {
				t.Error("--no-skills must never be emitted: the evaluator's rubric IS its terminator")
			}
		})
	}
}
