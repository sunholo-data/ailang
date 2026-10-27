package codex

import (
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/executor/nativesession"
)

func bankNativeSession(result *executor.Result) {
	if result == nil {
		return
	}
	root := config.CodexHome()
	if root == "" {
		var err error
		root, err = os.UserHomeDir()
		if err != nil {
			return
		}
		root = filepath.Join(root, ".codex")
	}
	path := nativesession.Find(filepath.Join(root, "sessions"), "codex", result.SessionID, config.CloudRunExecution() != "" && config.CodexHome() != "")
	nativesession.Bank(result, "codex", path)
}
