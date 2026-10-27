package claude

import (
	"github.com/sunholo-data/ailang/internal/executor"
	"os"
	"path/filepath"
	"testing"
)

func TestBankNativeSession(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", root)
	path := filepath.Join(root, filepath.FromSlash("projects/work/id.jsonl"))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("native"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, success := range []bool{true, false} {
		result := &executor.Result{SessionID: "id", Success: success, ProviderData: map[string]any{"existing": "preserved"}}
		bankNativeSession(result)
		if result.ProviderData["claude_session_jsonl"] != path || result.ProviderData["existing"] != "preserved" {
			t.Fatal(result.ProviderData)
		}
	}
	result := &executor.Result{SessionID: "missing"}
	bankNativeSession(result)
	if len(result.ProviderData) != 0 {
		t.Fatal(result.ProviderData)
	}
	bankNativeSession(nil)
}

func TestNativeSessionIdentity(t *testing.T) {
	if got := nativeSessionID("generated", "actual"); got != "actual" {
		t.Fatal(got)
	}
	if got := nativeSessionID("generated", ""); got != "generated" {
		t.Fatal(got)
	}
	task := &executor.Task{Iteration: 2, ResumeSessionID: "resumed"}
	if got := nativeSessionID("generated", resumedSessionID(task)); got != "resumed" {
		t.Fatal(got)
	}
	task.Iteration = 1
	if got := resumedSessionID(task); got != "" {
		t.Fatal(got)
	}
}
