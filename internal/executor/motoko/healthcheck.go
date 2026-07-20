package motoko

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// HealthCheck verifies the motoko binary exists, is executable, and (when
// available) reports its version + git rev.
//
// HISTORY: motoko had no `--version` mode prior to M-MOTOKO-EVAL-HARNESS-
// HARDENING (v0.18.1) — every flag was treated as task input by the agent
// loop and would spawn an LLM call (and hang waiting for the TUI). M2c
// added `motoko --version` which now exits 0 with structured key=value
// output (tui_version, git_rev, ailang_built, motoko_repo).
//
// HealthCheck:
//  1. Verifies binary existence + executability (always required)
//  2. Verifies OPENROUTER_API_KEY is set (wrapper pre-flight requirement)
//  3. Calls `motoko --version` with a 5s timeout — if it succeeds, the
//     version + git_rev are stashed in MotokoExecutor for telemetry.
//     Failure here is NON-FATAL (older motoko binaries pre-M2c hang on
//     any flag; we degrade to "version unknown" rather than refusing
//     the executor).
func (e *MotokoExecutor) HealthCheck(ctx context.Context) error {
	// M-MOTOKO-PARALLEL-EXECUTION-ISOLATION (v0.18.2): cache the result
	// once-per-executor. Pre-cache, the eval harness called HealthCheck
	// per-task, which spawned `motoko --version` per-task — at parallel-N,
	// N concurrent bun startups raced on shared node_modules + .bun/cache,
	// causing N-1 of them to die with exit 1 silently (the dur=0 pattern
	// in Phase 1 captures). Caching means we pay the bun-startup cost
	// once and parallel siblings see the cached result.
	e.healthCheckOnce.Do(func() {
		e.healthCheckErr = e.runHealthCheck(ctx)
	})
	return e.healthCheckErr
}

// runHealthCheck performs the actual one-time validation. Called from
// HealthCheck under sync.Once so it runs exactly once per executor lifetime.
func (e *MotokoExecutor) runHealthCheck(ctx context.Context) error {
	motokoPath := e.motokoPath
	resolvedPath := motokoPath
	if abs, err := exec.LookPath(motokoPath); err == nil {
		resolvedPath = abs
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return fmt.Errorf("motoko CLI not found at %q: %w (build from sunholo-data/motoko_agent or set MotokoPath)", motokoPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("motoko path %q is a directory, expected an executable", motokoPath)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("motoko binary at %q is not executable (chmod +x)", motokoPath)
	}
	if os.Getenv("OPENROUTER_API_KEY") == "" {
		return fmt.Errorf("OPENROUTER_API_KEY not set — motoko routes ALL models via OpenRouter; set this env var or expect every Execute to fail at the wrapper's pre-flight check")
	}

	// Best-effort version query (M-MOTOKO-EVAL-HARNESS-HARDENING M2c). Older
	// motoko binaries (pre-M2c) treat --version as task input and hang. The
	// 5s timeout caps that worst case; on timeout/error we leave version
	// fields at their default ("unknown") and proceed.
	versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, vErr := exec.CommandContext(versionCtx, motokoPath, "--version").Output()
	if vErr == nil {
		e.parseVersionOutput(string(out))
	}

	// M-MOTOKO-PARALLEL-EXECUTION-ISOLATION (v0.18.2) M4-M5: warn loudly
	// about operational gotchas that wasted hours of debugging this sprint.
	// Both are warnings (stderr), NOT errors — they don't block execution.
	// The user can ignore them at their own risk.
	e.warnIfStaleBunProcesses()
	if e.motokoRepo != "" {
		e.warnIfStaleAilangLock(e.motokoRepo)
	}

	return nil
}

// warnIfStaleBunProcesses scans for lingering bun processes that hold ports
// in the wrapper's pick_free_port range (18080-18099). These are typically
// orphaned from interrupted prior runs; they cause new motoko spawns to hit
// EADDRINUSE silently if the wrapper's lsof probe races with them. This
// sprint's investigation lost ~2 hours to exactly this scenario before the
// stale processes were noticed and killed. Fix: warn the operator with a
// one-liner remediation command.
func (e *MotokoExecutor) warnIfStaleBunProcesses() {
	out, err := exec.Command("lsof", "-i", "-P").Output()
	if err != nil {
		return // lsof not available; not critical
	}
	// Look for TCP LISTEN sockets in the 18080-18099 port range (motoko
	// wrapper's reserved range for env-server).
	bunPids := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "bun") || !strings.Contains(line, "(LISTEN)") {
			continue
		}
		// Match :180XX where XX is 80-99
		if !strings.Contains(line, ":1808") && !strings.Contains(line, ":1809") {
			continue
		}
		// Extract PID (column 2 in lsof output).
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			bunPids[fields[1]] = true
		}
	}
	if len(bunPids) > 0 {
		pids := make([]string, 0, len(bunPids))
		for pid := range bunPids {
			pids = append(pids, pid)
		}
		fmt.Fprintf(os.Stderr, "[motoko/healthcheck] WARNING: %d stale bun process(es) hold ports in motoko's range (18080-18099): PIDs %s\n",
			len(bunPids), strings.Join(pids, ", "))
		fmt.Fprintf(os.Stderr, "[motoko/healthcheck]          These can cause parallel motoko spawns to hit EADDRINUSE. Cleanup: pkill -9 -f 'bun.*src/tui'\n")
	}
}

// clearStalePort8080 kills any orphaned motoko host (bun running src/tui) still
// LISTENing on the FIXED env-server port 8080 before we spawn. The eval adapter
// pins ENV_PORT=8080 and the rig runs --parallel 1, so a holder at spawn time is
// never a legitimate concurrent run — it is an orphan from a crashed/SIGKILLed
// prior run. Left in place it makes THIS run silently crash with "no run_summary"
// (the 10h rig-wedge of 2026-06-29). Best-effort: if lsof/ps are absent (Windows)
// or nothing holds 8080, this is a no-op. It only kills a confirmed motoko host —
// an unrelated service on 8080 is warned about, never killed.
func (e *MotokoExecutor) clearStalePort8080() {
	out, err := exec.Command("lsof", "-nP", "-iTCP:8080", "-sTCP:LISTEN", "-t").Output()
	if err != nil || len(out) == 0 {
		return
	}
	for _, pidStr := range strings.Fields(string(out)) {
		pid, perr := strconv.Atoi(pidStr)
		if perr != nil {
			continue
		}
		desc := ""
		if cmdOut, cerr := exec.Command("ps", "-o", "command=", "-p", pidStr).Output(); cerr == nil {
			desc = strings.TrimSpace(string(cmdOut))
		}
		if !strings.Contains(desc, "bun") || !strings.Contains(desc, "src/tui") {
			fmt.Fprintf(os.Stderr, "[motoko/healthcheck] WARNING: port 8080 held by non-motoko PID %d (%s) — not killing; this run may fail to bind its env-server\n", pid, desc)
			continue
		}
		fmt.Fprintf(os.Stderr, "[motoko/healthcheck] LOUD: killing STALE motoko env-server PID %d squatting port 8080 (orphan from a crashed/hung run; would otherwise crash this run with 'no run_summary')\n", pid)
		if kerr := killProcessGroup(pid); kerr != nil {
			_ = killProcess(pid)
		}
	}
}

// warnIfStaleAilangLock checks whether motoko's ailang.lock matches the
// current disk state of its dependencies. When the operator publishes a new
// version of an extension package (e.g. ran `ailang publish` for v0.1.1
// while motoko's lock still records v0.1.0), the lock-vs-disk drift causes
// AILANG type-checking to fail with cryptic effect-row mismatches in
// registry_generated.ail. This sprint's investigation lost ~1 hour to
// exactly this scenario.
func (e *MotokoExecutor) warnIfStaleAilangLock(motokoRepo string) {
	lockPath := filepath.Join(motokoRepo, "ailang.lock")
	if _, err := os.Stat(lockPath); err != nil {
		return // no lock file, nothing to compare
	}
	// Don't try to recompute hashes here — too expensive for HealthCheck.
	// Instead, check the lock's mtime against the package source mtimes;
	// if any package source is newer than the lock, the lock is stale.
	lockInfo, err := os.Stat(lockPath)
	if err != nil {
		return
	}
	packagesDir := filepath.Dir(filepath.Dir(motokoRepo)) + "/ailang-packages/packages"
	if _, err := os.Stat(packagesDir); err != nil {
		return // ailang-packages not at the canonical sibling path; skip
	}
	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		tomlPath := filepath.Join(packagesDir, entry.Name(), "ailang.toml")
		tomlInfo, err := os.Stat(tomlPath)
		if err != nil {
			continue
		}
		if tomlInfo.ModTime().After(lockInfo.ModTime()) {
			fmt.Fprintf(os.Stderr, "[motoko/healthcheck] WARNING: %s/ailang.toml is newer than %s\n",
				entry.Name(), lockPath)
			fmt.Fprintf(os.Stderr, "[motoko/healthcheck]          Stale lock causes effect-row mismatch errors. Fix: cd %s && ailang lock && ailang generate-extension-registry\n",
				motokoRepo)
			return // one warning is enough; don't spam per-package
		}
	}
}

// parseVersionOutput populates the executor's version fields from the
// `motoko --version` key=value output. Lines that don't match the expected
// format are silently ignored.
func (e *MotokoExecutor) parseVersionOutput(out string) {
	for _, line := range strings.Split(out, "\n") {
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key, val := line[:idx], strings.TrimSpace(line[idx+1:])
		switch key {
		case "tui_version":
			e.tuiVersion = val
		case "git_rev":
			e.gitRev = val
		case "ailang_built":
			e.ailangBuilt = val
		case "motoko_repo":
			e.motokoRepo = val
		}
	}
}
