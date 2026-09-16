package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaterializeAgentPolicy writes a program policy delivered by CONTENT
// (AILANG_AGENT_POLICY_TOML) to a read-only file OUTSIDE the workspace and
// returns its path, or "" when no content was given (default-deny: the pi tool
// then refuses with a named reason). The file is 0444 in a 0555 directory
// under the user's home, never under the workspace — the policy's own
// fs_sandbox must not contain it (D4), and pi's write tool, which writes as
// this user, cannot open it for writing. M-AGENT-AILANG-ONLY-EXECUTION.
func MaterializeAgentPolicy(toml, workspace string) (string, error) {
	if strings.TrimSpace(toml) == "" {
		return "", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("agent policy: home: %w", err)
	}
	dir := filepath.Join(home, ".ailang", "agent-policy")
	if workspace != "" {
		ws, _ := filepath.Abs(workspace)
		if rel, rerr := filepath.Rel(ws, dir); rerr == nil && !strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("agent policy: %s would lie inside the workspace %s (D4)", dir, ws)
		}
	}
	// Re-materialising must work: an earlier run leaves the dir 0555.
	_ = os.Chmod(dir, 0o755)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("agent policy: %w", err)
	}
	path := filepath.Join(dir, "agent-policy.toml")
	_ = os.Remove(path)
	if err := os.WriteFile(path, []byte(toml), 0o444); err != nil {
		return "", fmt.Errorf("agent policy: %w", err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		return "", fmt.Errorf("agent policy: %w", err)
	}
	return path, nil
}
