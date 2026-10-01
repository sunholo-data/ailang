package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaterializeRunPolicy turns an operator policy FILE into the policy one run
// executes under: `${WORKSPACE}` is replaced by that run's workspace, and the
// result is written read-only OUTSIDE the workspace, at a path unique to the
// workspace (so parallel runs never share or overwrite one file).
//
// The lane's tools and AILANG's FS effect resolve relative paths against the
// policy's SANDBOX ROOT, not the agent's cwd. A policy whose fs_sandbox is a
// directory shared by many runs (the eval root) therefore sends a relative
// `benchmark/solution.ail` to one stray file at that root: the 2026-10-01 lane
// A/B graded the untouched template on 8 of its 12 wrong-answer rows, and
// concurrent runs read each other's solutions. The coordinator already
// materialises per job (MaterializeAgentPolicy); this is the eval path's
// equivalent, and it refuses a lane policy that cannot be per-run.
func MaterializeRunPolicy(policyPath, workspace string) (string, error) {
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		return "", fmt.Errorf("run policy: %w", err)
	}
	toml := string(raw)
	if !strings.Contains(toml, "${WORKSPACE}") {
		return "", fmt.Errorf("run policy %s does not use ${WORKSPACE}: a lane run's fs_sandbox must be its own workspace (fs_sandbox = \"${WORKSPACE}\"), or relative paths resolve outside it and parallel runs share files", policyPath)
	}
	ws, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("run policy: workspace: %w", err)
	}
	if real, rerr := filepath.EvalSymlinks(ws); rerr == nil {
		ws = real
	}
	toml = strings.ReplaceAll(toml, "${WORKSPACE}", ws)

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("run policy: home: %w", err)
	}
	sum := sha256.Sum256([]byte(ws))
	dir := filepath.Join(home, ".ailang", "agent-policy", "runs", hex.EncodeToString(sum[:8]))
	if rel, rerr := filepath.Rel(ws, dir); rerr == nil && !strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("run policy: %s would lie inside the workspace %s", dir, ws)
	}
	_ = os.Chmod(dir, 0o755) // a re-run of the same workspace finds it 0555
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("run policy: %w", err)
	}
	path := filepath.Join(dir, "agent-policy.toml")
	_ = os.Remove(path)
	if err := os.WriteFile(path, []byte(toml), 0o444); err != nil {
		return "", fmt.Errorf("run policy: %w", err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		return "", fmt.Errorf("run policy: %w", err)
	}
	return path, nil
}
