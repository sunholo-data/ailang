package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/policy"
)

// EnvFlags contains all environment-related command-line flags.
type EnvFlags struct {
	AllowEnv         string
	AllowEnvFile     string
	Env              string
	EnvSnapshot      string
	WriteEnvSnapshot string
}

// EnvFlagError is a malformed or unreadable --env* flag. The single-file run
// path prints it and exits 1; the batch path treats it as fatal for the whole
// run rather than as a per-item failure (it is the same flag for every item).
type EnvFlagError struct {
	Msg string
}

func (e *EnvFlagError) Error() string { return e.Msg }

// SetupEnvContext configures the effect context with environment variable
// settings. Returns exit=true if the program should stop (after writing a
// snapshot with --write-env-snapshot); a non-nil error is an *EnvFlagError.
func SetupEnvContext(effCtx *effects.EffContext, flags EnvFlags) (exit bool, err error) {
	// 1. Override env vars with --env KEY=VALUE,FOO=bar
	if flags.Env != "" {
		for _, pair := range strings.Split(flags.Env, ",") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				effCtx.EnvSnapshot[key] = value
			} else {
				return false, &EnvFlagError{Msg: fmt.Sprintf("invalid --env format '%s' (expected KEY=VALUE)", pair)}
			}
		}
	}

	// 2. Load snapshot from JSON file with --env-snapshot
	if flags.EnvSnapshot != "" {
		snapshotData, err := os.ReadFile(flags.EnvSnapshot)
		if err != nil {
			return false, &EnvFlagError{Msg: fmt.Sprintf("cannot read snapshot file '%s': %v", flags.EnvSnapshot, err)}
		}
		var snapshot map[string]string
		if err := json.Unmarshal(snapshotData, &snapshot); err != nil {
			return false, &EnvFlagError{Msg: fmt.Sprintf("invalid snapshot JSON in '%s': %v", flags.EnvSnapshot, err)}
		}
		// Replace snapshot with loaded data
		effCtx.EnvSnapshot = snapshot
	}

	// 3. Set allowlist with --allow-env KEY1,KEY2
	if flags.AllowEnv != "" {
		allowlist := []string{}
		for _, key := range strings.Split(flags.AllowEnv, ",") {
			key = strings.TrimSpace(key)
			if key != "" {
				allowlist = append(allowlist, key)
			}
		}
		effCtx.EnvAllowlist = allowlist
	}

	// 4. Load allowlist from file with --allow-env-file
	if flags.AllowEnvFile != "" {
		allowlistData, err := os.ReadFile(flags.AllowEnvFile)
		if err != nil {
			return false, &EnvFlagError{Msg: fmt.Sprintf("cannot read allowlist file '%s': %v", flags.AllowEnvFile, err)}
		}
		allowlist := []string{}
		for _, line := range strings.Split(string(allowlistData), "\n") {
			line = strings.TrimSpace(line)
			// Skip empty lines and comments
			if line != "" && !strings.HasPrefix(line, "#") {
				allowlist = append(allowlist, line)
			}
		}
		effCtx.EnvAllowlist = allowlist
	}

	// 5. Write snapshot and exit with --write-env-snapshot
	if flags.WriteEnvSnapshot != "" {
		snapshotJSON, err := json.MarshalIndent(effCtx.EnvSnapshot, "", "  ")
		if err != nil {
			return false, &EnvFlagError{Msg: fmt.Sprintf("cannot marshal snapshot: %v", err)}
		}
		if err := os.WriteFile(flags.WriteEnvSnapshot, snapshotJSON, 0644); err != nil {
			return false, &EnvFlagError{Msg: fmt.Sprintf("cannot write snapshot file '%s': %v", flags.WriteEnvSnapshot, err)}
		}
		fmt.Printf("%s Environment snapshot written to %s\n", green("✓"), flags.WriteEnvSnapshot)
		return true, nil // Signal to exit
	}

	return false, nil // Continue execution
}

// NetOptions are the `--net-*` flags.
type NetOptions struct {
	AllowHTTP      bool
	AllowDomains   string
	AllowLocalhost bool
	AllowMetadata  bool
	Timeout        string
}

// StreamOptions are the `--stream-*` flags.
type StreamOptions struct {
	AllowHTTP      bool
	AllowDomains   string
	AllowLocalhost bool
}

// SetupFSLimit resolves the FS read cap: the --fs-max-bytes flag text, else
// AILANG_FS_MAX_BYTES, else unbounded (D-C). A malformed value is an error —
// a safety cap never falls back (M-V1-MEMORY-FOOTPRINT M3).
func SetupFSLimit(effCtx *effects.EffContext, flag string) error {
	if flag != "" {
		n, err := config.ParseByteSize(flag)
		if err != nil {
			return fmt.Errorf("--fs-max-bytes: %w", err)
		}
		effCtx.Env.FSMaxBytes = n
		return nil
	}
	n, _, err := config.FSMaxBytes()
	if err != nil {
		return err
	}
	effCtx.Env.FSMaxBytes = n
	return nil
}

// ProcessOptions are the `--process-*` flags.
type ProcessOptions struct {
	Timeout   string
	Allowlist string
	MaxOutput int64
}

// SetupSharedMemHandler initializes the SharedMem effect context if the capability is granted.
// SharedMem provides shared memory caching for semantic caching (M-DX15).
func SetupSharedMemHandler(effCtx *effects.EffContext) {
	if effCtx.HasCap("SharedMem") {
		// Initialize with in-memory cache (default)
		// Future: could add flags for Redis, memcached, etc.
		effCtx.SharedMem = effects.NewSharedMemContext(nil)
	}
}

// SetupNetHandler configures Net effect security settings if the capability is granted.
func SetupNetHandler(effCtx *effects.EffContext, allowHTTP bool, allowDomains string, allowLocalhost bool, allowMetadata bool, timeout string) error {
	if effCtx.HasCap("Net") {
		// --net-timeout mirrors --process-timeout: the 30s default was a hard
		// ceiling with no flag, so a local 27B model answering in 45s looked
		// identical to a dead one (Daneel, 2026-09-11).
		if timeout != "" {
			d, err := time.ParseDuration(timeout)
			if err != nil || d <= 0 {
				return fmt.Errorf("invalid --net-timeout %q: want a positive Go duration such as 300s or 5m", timeout)
			}
			effCtx.Net.Timeout = d
		}
		effCtx.Net.AllowHTTP = allowHTTP
		effCtx.Net.AllowLocalhost = allowLocalhost
		effCtx.Net.AllowMetadata = allowMetadata
		if allowDomains != "" {
			for _, d := range strings.Split(allowDomains, ",") {
				d = strings.TrimSpace(d)
				if d != "" {
					effCtx.Net.AllowedDomains = append(effCtx.Net.AllowedDomains, d)
				}
			}
		}
	}
	return nil
}

// SetupStreamHandler initializes the Stream effect context if the capability is granted.
// Stream provides bidirectional WebSocket connections (M-STREAM-BIDI).
func SetupStreamHandler(effCtx *effects.EffContext, allowHTTP bool, allowDomains string, allowLocalhost bool) {
	if effCtx.HasCap("Stream") {
		effCtx.Stream = effects.NewStreamContext()
		effCtx.Stream.AllowHTTP = allowHTTP
		effCtx.Stream.AllowLocalhost = allowLocalhost
		if allowDomains != "" {
			for _, d := range strings.Split(allowDomains, ",") {
				d = strings.TrimSpace(d)
				if d != "" {
					effCtx.Stream.AllowedDomains = append(effCtx.Stream.AllowedDomains, d)
				}
			}
		}
	}
}

// SetupProcessHandler initializes the Process effect context if the capability is granted.
// Process provides external command execution (M-PROCESS).
func SetupProcessHandler(effCtx *effects.EffContext, timeout string, allowlist string, maxOutput int64) error {
	if !effCtx.HasCap("Process") {
		return nil
	}

	pc := effects.NewProcessContext()

	// Parse timeout duration
	if timeout != "" {
		d, err := time.ParseDuration(timeout)
		if err != nil {
			return fmt.Errorf("invalid --process-timeout %q: %w", timeout, err)
		}
		pc.Timeout = d
	}

	// Set max output
	if maxOutput > 0 {
		pc.MaxOutput = maxOutput
	}

	// Resolve allowlist (path-pinned at startup)
	if err := pc.ResolveAllowlist(allowlist); err != nil {
		return fmt.Errorf("invalid --process-allowlist: %w", err)
	}

	effCtx.Process = pc
	return nil
}

// SetupSharedIndexHandler initializes the SharedIndex effect context if the capability is granted.
// SharedIndex provides similarity-based semantic retrieval (M-DX16).
func SetupSharedIndexHandler(effCtx *effects.EffContext) {
	if effCtx.HasCap("SharedIndex") {
		// Initialize with in-memory index (default)
		// Future: could add flags for external index backends
		effCtx.SharedIndex = effects.NewSharedIndexContext(nil)
	}
}

// FlushDebugOutput collects Debug effect logs and prints them to stderr via
// the shared effects.DebugSink. minLevel is the --log-level threshold
// (0=DEBUG/TRACE, 1=INFO, 2=WARNING, 3=ERROR, 4=NONE). A non-empty label
// prefixes UNSTRUCTURED lines so batch output remains attributable even under
// --quiet; a structured (JSON-object) line is written verbatim so log
// consumers can parse it (M-DEBUG-SINK-STRUCTURED-LINES).
func FlushDebugOutput(effCtx *effects.EffContext, minLevel int, label string) {
	if effCtx == nil {
		return
	}
	effects.DebugSink{W: os.Stderr, MinLevel: minLevel, Label: label}.Flush(effCtx.Debug)
}

// ApplyPolicy binds an effect context to a resolved operator policy
// (M-EXECUTOR-POLICY-HARDENING M3). It runs AFTER the flag-driven handler
// setup and can only tighten what that produced:
//
//   - restricted mode refuses a configured proxy (Net.RefuseProxy) rather
//     than claiming destination-IP enforcement behind one;
//   - max_fs_transfer_bytes caps every FS read and write;
//   - the operator budgets (M4) become the shared ceiling.
//
// nil is a no-op: every non-policy run passes through unchanged.
func ApplyPolicy(effCtx *effects.EffContext, res *policy.Resolved) {
	if res == nil {
		return
	}
	if res.Restricted() {
		if effCtx.Net != nil {
			effCtx.Net.RefuseProxy = true
		}
		// M6: Process runs through the confined adapter (read-only git with
		// a hardened invocation), and `.git/` is read-only to the program so
		// the repo config the adapter trusts stays the launcher's.
		if effCtx.Process != nil {
			effCtx.Process.Confined = true
		}
		effCtx.Env.ProtectGitDir = true
	}
	if res.MaxFSTransferBytes > 0 {
		if effCtx.Env.FSMaxBytes == 0 || res.MaxFSTransferBytes < effCtx.Env.FSMaxBytes {
			effCtx.Env.FSMaxBytes = res.MaxFSTransferBytes
		}
	}
	if len(res.Budgets) > 0 {
		effCtx.SetOperatorBudget(effects.NewOperatorBudget(res.Budgets))
	}
}
