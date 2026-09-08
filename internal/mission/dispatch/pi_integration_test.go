package dispatch

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/executor/pi"
)

func TestActualPiAdapterDispatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake Pi executable uses POSIX sh; pure dispatch tests cover Windows")
	}
	r, runner, _, _ := fixture(t)
	r.Models = []string{"judge"}
	dir := t.TempDir()
	path := filepath.Join(dir, "pi")
	script := `#!/bin/sh
if [ "$1" = "--version" ]; then echo fixture; exit 0; fi
printf '%s\n' "$@" > args.txt
test "$AILANG_MESSAGES_STORE" = gcp || exit 22
test "$AILANG_MESSAGES_PROJECT" = ailang-multivac || exit 23
cat <<'EVENTS'
{"type":"session","id":"fixture-session"}
{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"Independent verdict"}}
{"type":"message_end","message":{"role":"assistant","stopReason":"stop","usage":{"input":10,"output":5,"cost":{"total":0.00002}}}}
{"type":"agent_end"}
EVENTS
`
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	f := executor.NewFactory(&executor.Config{PiPath: path})
	defer f.Close()
	f.Register("pi", func(c *executor.Config) (executor.Executor, error) { return pi.New(c) })
	runner.Executors = f
	report, err := runner.Run(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if report.Result.Output != "Independent verdict" || report.Result.SessionID != "fixture-session" || report.ArtifactVerified {
		t.Fatalf("incorrect adapter receipt: %+v", report)
	}
	args, err := os.ReadFile(filepath.Join(r.Workspace, "args.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "minimax/model") || !strings.Contains(string(args), r.Instructions) {
		t.Fatal("wire model/directive lost at adapter boundary")
	}
}
