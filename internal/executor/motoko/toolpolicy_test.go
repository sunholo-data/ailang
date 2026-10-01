package motoko

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// motoko always exposes Bash/Read/Write/Edit/Grep, so a task that forbids any
// of them must be refused, never run with wider tools than it asked for.
func TestCheckToolPolicy(t *testing.T) {
	ailangOnly, err := executor.ProfileTools("ailang_only")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		task    executor.Task
		wantErr string
	}{
		{"nil means executor default", executor.Task{}, ""},
		// What the eval harness passes for every agent run (agent_runner.go).
		{"eval harness list", executor.Task{AllowedTools: []string{"Bash", "Read", "Write", "Edit", "Grep"}}, ""},
		{"superset is fine", executor.Task{AllowedTools: []string{"Bash", "Read", "Write", "Edit", "Grep", "Glob", "WebFetch"}}, ""},
		{"ailang_only lane", executor.Task{AllowedTools: ailangOnly}, "forbids Bash, Read, Write, Edit, Grep"},
		// The coordinator's read-only list for a question task (questionTools).
		{"question task", executor.Task{AllowedTools: []string{"Read", "Grep", "Glob", "WebFetch", "WebSearch"}}, "forbids Bash, Write, Edit"},
		{"empty list = no tools", executor.Task{AllowedTools: []string{}}, "forbids Bash, Read, Write, Edit, Grep"},
		{"program policy", executor.Task{PolicyPath: "/tmp/agent-policy.toml"}, "cannot run under a program policy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkToolPolicy(&tc.task)
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("unexpected refusal: %v", err)
			case tc.wantErr != "" && err == nil:
				t.Fatalf("want a refusal containing %q, got none", tc.wantErr)
			case tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr):
				t.Fatalf("refusal %q does not contain %q", err, tc.wantErr)
			}
		})
	}
}

// The refusal happens before motoko is started, and an accepted run records the
// tool surface it actually had instead of leaving ToolPolicy nil ("unmeasured").
func TestExecute_ToolPolicyRefusedBeforeSpawn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash mock binary requires POSIX shell")
	}
	tmp := t.TempDir()
	marker := filepath.Join(tmp, "spawned")
	fixture, err := filepath.Abs(filepath.Join("testdata", "session_success.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	mock := filepath.Join(tmp, "motoko")
	script := "#!/bin/bash\nset -e\ntouch " + marker + "\nLOGDIR=\"$WORKDIR/.motoko/logfile\"\nmkdir -p \"$LOGDIR\"\nSESSION=\"${MOTOKO_SESSION_ID:-session_unknown}\"\nsed \"s/session_test-success/$SESSION/g\" \"" + fixture + "\" > \"$LOGDIR/$SESSION.jsonl\"\n"
	if err := os.WriteFile(mock, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	ws := filepath.Join(tmp, "ws")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	exec, err := New(&executor.Config{MotokoPath: mock, MotokoModel: "ollama/qwen3.6:35b-a3b-mxfp8", MotokoProfile: "dogfood"})
	if err != nil {
		t.Fatal(err)
	}
	ailangOnly, _ := executor.ProfileTools("ailang_only")

	if _, err := exec.Execute(context.Background(), &executor.Task{Workspace: ws, Directive: "x", AllowedTools: ailangOnly}); err == nil {
		t.Fatal("ailang_only task was accepted; want a refusal")
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("motoko was started for a task it refused")
	}

	res, err := exec.Execute(context.Background(), &executor.Task{Workspace: ws, Directive: "x", AllowedTools: []string{"Bash", "Read", "Write", "Edit", "Grep"}})
	if err != nil {
		t.Fatalf("eval-harness tool list refused: %v", err)
	}
	if !slices.Equal(res.ToolPolicy, nativeTools) {
		t.Errorf("Result.ToolPolicy = %v, want the native surface %v", res.ToolPolicy, nativeTools)
	}
}
