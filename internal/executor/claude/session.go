package claude

import (
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/executor/nativesession"
	"os"
	"path/filepath"
)

func bankNativeSession(result *executor.Result) {
	if result == nil {
		return
	}
	root := config.ClaudeConfigDir()
	if root == "" {
		root, _ = os.UserHomeDir()
		root = filepath.Join(root, ".claude")
	}
	path := nativesession.Find(filepath.Join(root, "projects"), "claude", result.SessionID, false)
	nativesession.Bank(result, "claude", path)
}
