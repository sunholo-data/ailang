package codex

import (
	"github.com/sunholo-data/ailang/internal/executor"
	"os"
	"path/filepath"
	"testing"
)

func TestBankNativeSession(t *testing.T) {
	t.Setenv("CLOUD_RUN_EXECUTION", "")
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	path := filepath.Join(root, filepath.FromSlash("sessions/2026/10/27/rollout-stamp-id.jsonl"))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("native"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, success := range []bool{true, false} {
		result := &executor.Result{SessionID: "id", Success: success, ProviderData: map[string]any{"existing": "preserved"}}
		bankNativeSession(result)
		if result.ProviderData["codex_session_jsonl"] != path || result.ProviderData["existing"] != "preserved" {
			t.Fatal(result.ProviderData)
		}
	}
	result := &executor.Result{SessionID: "missing"}
	bankNativeSession(result)
	if len(result.ProviderData) != 0 {
		t.Fatal(result.ProviderData)
	}
	t.Setenv("CLOUD_RUN_EXECUTION", "execution")
	bankNativeSession(result)
	if result.ProviderData["codex_session_jsonl"] != path {
		t.Fatal(result.ProviderData)
	}
	bankNativeSession(nil)
}
