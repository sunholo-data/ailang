package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/policy"
)

// MaterializeAgentPolicy writes a program policy delivered by CONTENT
// (AILANG_AGENT_POLICY_TOML) to a read-only file OUTSIDE the workspace and
// returns its path, or "" when no content was given (default-deny: the pi tool
// then refuses with a named reason). The file is 0444 in a 0555 directory
// under the user's home, never under the workspace — the policy's own
// fs_sandbox must not contain it (D4), and pi's write tool, which writes as
// this user, cannot open it for writing. M-AGENT-AILANG-ONLY-EXECUTION.
//
// `${WORKSPACE}` in the TOML is replaced by the workspace path: a Job's clone
// lives at a per-task directory the policy author cannot know, and AILANG's FS
// and Process effects resolve relative paths against the SANDBOX ROOT, not the
// program's cwd — a sandbox one level above the clone made `git log` run in
// /workspace and fail with "not a git repository" (456 turns, 2026-09-16).
func MaterializeAgentPolicy(toml, workspace string) (string, error) {
	if strings.TrimSpace(toml) == "" {
		return "", nil
	}
	if workspace != "" {
		abs, err := filepath.Abs(workspace)
		if err == nil {
			toml = strings.ReplaceAll(toml, "${WORKSPACE}", abs)
		}
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

// CheckLanePolicy is the rule that keeps the ailang_only lane honest
// (M-EXECUTOR-POLICY-HARDENING D3): an agent whose ONLY route to execution is
// the policy gate must run a RESTRICTED policy. A trusted_host policy behind
// the lane would carry the lane's "no shell" claim with a host-integration
// grant (Process, AI, Env…) underneath it. The full profile is free to use
// either mode — it has a shell anyway and makes no confinement claim.
//
// policyTOML is the policy CONTENT (what the dispatcher ships); an empty
// content with a non-full profile is already refused upstream. Errors name
// the agent's migration: change tool_policy to full, or the policy's mode.
func CheckLanePolicy(toolPolicy string, policyTOML []byte) error {
	if strings.TrimSpace(toolPolicy) == "" || strings.TrimSpace(toolPolicy) == ToolProfileFull || len(policyTOML) == 0 {
		return nil
	}
	pol, err := policy.Decode(policyTOML)
	if err != nil {
		return fmt.Errorf("tool_policy %q: %w", toolPolicy, err)
	}
	res, err := policy.Resolve(pol, policy.DigestBytes(policyTOML))
	if err != nil {
		return fmt.Errorf("tool_policy %q: %w", toolPolicy, err)
	}
	if !res.Restricted() {
		return fmt.Errorf("tool_policy %q requires a restricted policy, but the policy sets security_mode = %q — the lane's execution boundary would be claimed over a host-integration grant; declare tool_policy: full for this agent, or drop security_mode/host effects from the policy", toolPolicy, res.Mode)
	}
	return nil
}
