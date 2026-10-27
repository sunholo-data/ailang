package codex

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
	root := config.CodexHome()
	if root == "" {
		root, _ = os.UserHomeDir()
		root = filepath.Join(root, ".codex")
	}
	path := nativesession.Find(filepath.Join(root, "sessions"), "codex", result.SessionID, config.CloudRunExecution() != "" && config.CodexHome() != "")
	nativesession.Bank(result, "codex", path)
}
