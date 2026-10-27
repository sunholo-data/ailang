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
