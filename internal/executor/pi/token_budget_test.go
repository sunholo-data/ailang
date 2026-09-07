package pi

import (
	"context"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

func TestPiTokenBudgetIsEnforced(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	for _, limit := range []int{1, 100000} {
		t.Run(strconv.Itoa(limit), func(t *testing.T) {
			dir := t.TempDir()
			writeFakePi(t, dir, loadFixtureLines(t, "fizzbuzz.ndjson"))
			e, err := New(&executor.Config{PiPath: filepath.Join(dir, "pi"), PiModel: "anthropic/claude-haiku-4-5"})
			if err != nil {
				t.Fatal(err)
			}
			result, err := e.ExecuteStreaming(context.Background(), &executor.Task{ID: "token-test", Directive: "test", Workspace: dir, Timeout: 5 * time.Second, MaxTokensPerBench: limit}, &collectingHandler{})
			if err != nil || result == nil {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if limit == 1 {
				if result.Success || result.FinishReason != executor.FinishThrashAborted || result.ThrashKilledAt <= limit {
					t.Fatalf("token budget ignored: success=%v finish=%s killed=%d", result.Success, result.FinishReason, result.ThrashKilledAt)
				}
			} else if !result.Success || result.ThrashKilledAt != 0 {
				t.Fatalf("within-budget control failed: %+v", result)
			}
		})
	}
}
