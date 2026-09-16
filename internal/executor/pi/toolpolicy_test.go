package pi

import (
	"context"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// M-AGENT-AILANG-ONLY-EXECUTION M1: the EFFECTIVE tool policy is banked on
// every row, before any policy changes. A nil AllowedTools means the caller
// let pi's defaults apply — banked as a sentinel, never as an empty list
// (which would read as "no tools").

func runFakeWithTask(t *testing.T, task *executor.Task) *executor.Result {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()
	_ = writeFakePi(t, dir, loadFixtureLines(t, "v0_85_1/fizzbuzz.ndjson"))
	e, _ := New(&executor.Config{PiPath: filepath.Join(dir, "pi"), PiModel: "anthropic/claude-haiku-4-5", TimeoutSeconds: 10})
	task.Workspace = dir
	task.Timeout = 10 * time.Second
	res, err := e.ExecuteStreaming(context.Background(), task, &executor.NoOpEventHandler{})
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	return res
}

func TestToolPolicy_NilBanksSentinel(t *testing.T) {
	res := runFakeWithTask(t, &executor.Task{ID: "tp", Directive: "x"})
	if !reflect.DeepEqual(res.ToolPolicy, []string{executor.ToolPolicyCLIDefault}) {
		t.Fatalf("ToolPolicy = %v, want [%q]", res.ToolPolicy, executor.ToolPolicyCLIDefault)
	}
	if res.PolicyDigest != "" {
		t.Fatalf("PolicyDigest = %q, want empty (no policy passed)", res.PolicyDigest)
	}
}

func TestToolPolicy_ExplicitListBankedVerbatim(t *testing.T) {
	res := runFakeWithTask(t, &executor.Task{ID: "tp", Directive: "x", AllowedTools: []string{"Read", "Write"}})
	if !reflect.DeepEqual(res.ToolPolicy, []string{"Read", "Write"}) {
		t.Fatalf("ToolPolicy = %v, want [Read Write]", res.ToolPolicy)
	}
}

func TestToolPolicy_EmptyListBanksEmpty(t *testing.T) {
	// --no-tools: an explicitly empty policy is a real, different thing from nil.
	res := runFakeWithTask(t, &executor.Task{ID: "tp", Directive: "x", AllowedTools: []string{}})
	if res.ToolPolicy == nil || len(res.ToolPolicy) != 0 {
		t.Fatalf("ToolPolicy = %v, want an empty non-nil list", res.ToolPolicy)
	}
}
