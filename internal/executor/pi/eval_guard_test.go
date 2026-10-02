package pi

import (
	"os"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// An eval run loads the shell guard; any other run does not (a mission stage legitimately
// runs multi-minute commands).
func TestBuildPiArgs_EvalShellGuardOnlyForEvalRuns(t *testing.T) {
	guarded, err := buildPiArgs("ollama/qwen3.8", &executor.Task{EvalShellGuard: true}, "d")
	if err != nil {
		t.Fatal(err)
	}
	path := ""
	for i, a := range guarded {
		if a == "-e" && i+1 < len(guarded) && strings.HasSuffix(guarded[i+1], "/"+evalShellGuardFile) {
			path = guarded[i+1]
		}
	}
	if path == "" {
		t.Fatalf("eval run args carry no -e %s: %v", evalShellGuardFile, guarded)
	}
	if b, err := os.ReadFile(path); err != nil || !strings.Contains(string(b), "isUnboundedFind") {
		t.Fatalf("materialized guard missing or wrong at %s: %v", path, err)
	}
	if guarded[len(guarded)-1] != "d" {
		t.Errorf("directive is not the trailing argument: %v", guarded)
	}

	plain, err := buildPiArgs("ollama/qwen3.8", &executor.Task{}, "d")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range plain {
		if strings.Contains(a, evalShellGuardFile) {
			t.Fatalf("non-eval run loads the eval guard: %v", plain)
		}
	}
}
