package claude

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
	root := config.ClaudeConfigDir()
	if root == "" {
		var err error
		root, err = os.UserHomeDir()
		if err != nil {
			return
		}
		root = filepath.Join(root, ".claude")
	}
	path := nativesession.Find(filepath.Join(root, "projects"), "claude", result.SessionID, false)
	nativesession.Bank(result, "claude", path)
}

// The CLI's reported identity wins; resumed runs must not bank the generated UUID.
func nativeSessionID(fallback, actual string) string {
	if actual != "" {
		return actual
	}
	return fallback
}
func resumedSessionID(task *executor.Task) string {
	if task.Iteration > 1 {
		return task.ResumeSessionID
	}
	return ""
}
