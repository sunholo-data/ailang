package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// newAgentSessionWorkspace creates the directory one agent-mode run works in, and
// guarantees no other run in this or any process gets the same one.
//
// It used to be <tmp>/ailang_eval/<bench>_<model>_<YYYYMMDD_HHMMSS>_<pid>: no trial,
// no condition, one-second resolution. Trials of one benchmark that the scheduler
// starts in the same second (every --parallel slot fills at once when a suite
// starts) therefore shared a directory. Each overwrote the others' solution file,
// and when the first finished, its deferred RemoveAll + proctree.ReapWorkspace
// deleted the directory and SIGKILLed every sibling whose cwd was inside it. They
// banked as api_error "signal: killed" with empty code. Measured 2026-11-02
// (eval_results/cp_climb/step0_4c2566568): 3 of 18 Haiku runs, all started at
// 17:25:26, each killed within 0.2s of a sibling finishing.
//
// The prefix keeps the old <bench>_<model>_ shape: wire-parity attribution
// (internal/eval_analysis/wire_parity.go) matches an arm by "_<model>_" in the
// workdir path the agent was told. os.MkdirTemp supplies the uniqueness that the
// timestamp only approximated.
func newAgentSessionWorkspace(benchmarkID, model, condition string, trial int) (string, error) {
	root := filepath.Join(os.TempDir(), "ailang_eval")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("create eval workspace root: %w", err)
	}
	prefix := fmt.Sprintf("%s_%s_%s_%d_t%d_", benchmarkID, model,
		time.Now().Format("20060102_150405"), os.Getpid(), trial)
	if condition != "" {
		prefix += condition + "_"
	}
	dir, err := os.MkdirTemp(root, prefix)
	if err != nil {
		return "", fmt.Errorf("create eval workspace: %w", err)
	}
	return dir, nil
}
