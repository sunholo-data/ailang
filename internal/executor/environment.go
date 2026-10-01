// Package executor provides environment setup utilities shared across all AI executors.
package executor

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/stdlibroot"
	"github.com/sunholo-data/ailang/internal/telemetry"
)

// Embed the Claude settings and hook script from the repo
//
//go:embed claude_settings.json
var embeddedClaudeSettings []byte

//go:embed claude_telemetry.sh
var embeddedHookScript []byte

// EnvironmentOptions configures the environment building process.
type EnvironmentOptions struct {
	// Task is the task being executed
	Task *Task

	// SessionID is the unique session identifier
	SessionID string

	// Context is used for trace context extraction
	Context context.Context

	// EnableClaudeTelemetry enables Claude Code specific telemetry vars
	EnableClaudeTelemetry bool

	// EnableGeminiTelemetry enables Gemini CLI specific telemetry vars
	EnableGeminiTelemetry bool

	// GCPProject overrides GOOGLE_CLOUD_PROJECT for this subprocess.
	// When empty, falls back to the shell environment value.
	GCPProject string

	// GCPLocation overrides GOOGLE_CLOUD_LOCATION for this subprocess.
	// When empty, falls back to the shell environment value.
	GCPLocation string

	// ExecutorEnv is the executor's own required set ("NAME=value"), applied
	// last (e.g. motoko's MODEL, MOTOKO_CONFIG, ENV_PORT). It is code-owned,
	// never task-supplied, so it is not validated like Task.ExtraEnv.
	ExecutorEnv []string
}

// BuildEnvironment builds the environment for an AI executor's model-facing
// child process (M-EXECUTOR-ENV-HARDENING D4). It constructs the env from
// layers through one de-duplicating builder (envSet), so every key appears
// once and precedence is fixed:
//
//  1. inherited — the host environment (minus CLAUDECODE)
//  2. harness-injected — values the harness owns:
//     AILANG_RIG_LEASE, AILANG_STDLIB_PATH, PWD, TRACEPARENT/TRACESTATE,
//     AILANG_TASK_ID/SESSION_ID/PARENT_TASK_ID, AILANG_CHAIN_ID/STAGE_ID/
//     MESSAGE_ID, AILANG_AGENT_POLICY (from Task.PolicyPath),
//     OTEL_RESOURCE_ATTRIBUTES, OTEL_EXPORTER_OTLP_ENDPOINT/PROTOCOL,
//     GOOGLE_CLOUD_PROJECT/LOCATION, and the Claude/Gemini telemetry switches
//  3. Task.ExtraEnv — validated first (ValidateExtraEnv); a refused name is an
//     error, never a silent drop
//  4. EnvironmentOptions.ExecutorEnv — the executor's own required set
//
// Later layers win. The error is non-nil only when Task.ExtraEnv is refused.
func BuildEnvironment(opts EnvironmentOptions) ([]string, error) {
	if opts.Task != nil {
		if err := ValidateExtraEnv(opts.Task.ExtraEnv); err != nil {
			return nil, err
		}
	}

	env := newEnvSet()

	// Layer 1: inherited.
	env.setEntries(os.Environ())
	// Strip CLAUDECODE to prevent "Cannot be launched inside another Claude
	// Code session" errors. Applies to ALL executors — any of them may shell
	// out to Claude Code.
	env.unset("CLAUDECODE")

	// Layer 2: harness-injected.
	injectHarnessEnv(env, opts)

	// Layer 3: per-task extra env (benchmark agent_env such as
	// MOTOKO_AST_AUTOREAD, mission-stage pins, browser-lane endpoints).
	// Validated above; applied in sorted order so the result is deterministic.
	if opts.Task != nil {
		env.setMapSorted(opts.Task.ExtraEnv)
	}

	// Layer 4: the executor's own required set.
	env.setEntries(opts.ExecutorEnv)

	return env.environ(), nil
}

// injectHarnessEnv writes the values the harness owns (layer 2).
func injectHarnessEnv(env *envSet, opts EnvironmentOptions) {
	// Always define the rig lease (M-RIG-GPU-ADMISSION-GATEWAY): the held lock's
	// token, or "none". A pi provider that templates it into a header refuses to
	// start when the variable is unset (measured 2026-09-27), so an agent must
	// never inherit an environment without it.
	env.set(config.EnvRigLease, config.RigLease())

	// AILANG stdlib path.
	// Priority: workspace/std (cloud: cloned repo has stdlib) > cwd/std (local: running from repo root).
	// Only a directory that IS a stdlib is exported: an explicit AILANG_STDLIB_PATH
	// that holds none is an error in the child (M-STDLIB-ROOT-RESOLUTION), and with
	// nothing exported the child uses the stdlib built into its binary.
	if stdlibPath := childStdlibPath(opts); stdlibPath != "" {
		env.set("AILANG_STDLIB_PATH", stdlibPath)
	}

	if opts.Task != nil && opts.Task.Workspace != "" {
		env.set("PWD", opts.Task.Workspace)
	}

	// The program policy the ailang_only lane's tools are gated by. Derived
	// from Task.PolicyPath — never from ExtraEnv, which may not name it.
	if opts.Task != nil && opts.Task.PolicyPath != "" {
		env.set(config.EnvAgentPolicy, opts.Task.PolicyPath)
	}

	// W3C trace context, so `ailang run` commands spawned by the agent link back
	// to this trace.
	if opts.Context != nil {
		env.setEntries(telemetry.InjectTraceContext(opts.Context, nil))
	}

	// Correlation IDs for fallback linking.
	taskID, parentTaskID := "", ""
	if opts.Task != nil {
		taskID = opts.Task.ID
		parentTaskID = opts.Task.ParentTaskID
	}
	env.setEntries(telemetry.InjectCorrelationIDs(nil, taskID, opts.SessionID))

	// Parent task ID for hierarchy tracking (M-TASK-HIERARCHY): an explicit
	// parent (nested exec calls), else this task, so the agent's own child
	// ailang commands link back to it.
	effectiveParentID := parentTaskID
	if effectiveParentID == "" && taskID != "" {
		effectiveParentID = taskID
	}
	if effectiveParentID != "" {
		env.set("AILANG_PARENT_TASK_ID", effectiveParentID)
	}

	// Chain context (M-CHAINS-SIMPLIFY), passed via Task.Metadata.
	if opts.Task != nil && opts.Task.Metadata != nil {
		if chainID := opts.Task.Metadata["chain_id"]; chainID != "" {
			env.set("AILANG_CHAIN_ID", chainID)
		}
		if stageID := opts.Task.Metadata["stage_id"]; stageID != "" {
			env.set("AILANG_STAGE_ID", stageID)
		}
		if messageID := opts.Task.Metadata["message_id"]; messageID != "" {
			env.set("AILANG_MESSAGE_ID", messageID)
		}
	}

	// Resource attributes for trace linking (M-TASK-HIERARCHY).
	env.set("OTEL_RESOURCE_ATTRIBUTES", BuildResourceAttributes(opts.Task, opts.SessionID))

	// OTEL exporter. Priority: parent env > the local observatory server.
	endpoint := config.OTLPEndpoint()
	if endpoint == "" {
		endpoint = "http://localhost:1957"
	}
	env.set("OTEL_EXPORTER_OTLP_ENDPOINT", endpoint)
	if protocol := config.OTLPProtocol(); protocol != "" {
		env.set("OTEL_EXPORTER_OTLP_PROTOCOL", protocol)
	}

	// GCP project for export. Priority: EnvironmentOptions override >
	// OTLP_GOOGLE_CLOUD_PROJECT > config.CloudProject (no project is fine here,
	// the exporter then stays unconfigured).
	project := opts.GCPProject
	if project == "" {
		project = config.TraceProjectOverride()
	}
	if project == "" {
		project, _ = config.CloudProject(context.Background())
	}
	if project != "" {
		env.set("GOOGLE_CLOUD_PROJECT", project)
		env.set("OTLP_GOOGLE_CLOUD_PROJECT", project)
	}

	location := opts.GCPLocation
	if location == "" {
		location = config.GoogleCloudLocation()
	}
	if location != "" {
		env.set("GOOGLE_CLOUD_LOCATION", location)
	}

	if opts.EnableClaudeTelemetry {
		env.set("CLAUDE_CODE_ENABLE_TELEMETRY", "1")
		env.set("OTEL_METRICS_EXPORTER", "otlp")
		env.set("OTEL_LOGS_EXPORTER", "otlp")
	}

	if opts.EnableGeminiTelemetry {
		env.set("GEMINI_TELEMETRY_ENABLED", "true")
		if target := config.GeminiTelemetryTarget(); target != "" {
			env.set("GEMINI_TELEMETRY_TARGET", target)
		} else if project != "" {
			env.set("GEMINI_TELEMETRY_TARGET", "gcp")
		}
	}
}

// BuildResourceAttributes creates OTEL_RESOURCE_ATTRIBUTES value.
// Merges existing attributes from environment with task-specific attributes.
// Priority: existing env attrs + task Metadata + default attrs.
func BuildResourceAttributes(task *Task, sessionID string) string {
	attrs := make(map[string]string)

	// 1. Start with existing environment attributes (preserve user settings)
	if existing := config.OTELResourceAttributes(); existing != "" {
		for _, pair := range strings.Split(existing, ",") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				attrs[parts[0]] = parts[1]
			}
		}
	}

	// 2. Add task Metadata attributes (from Observatory context via coordinator)
	if task != nil && task.Metadata != nil {
		for k, v := range task.Metadata {
			if strings.HasPrefix(k, "ailang.") && v != "" {
				attrs[k] = v
			}
		}
	}

	// 3. Add task-specific attributes (high priority for coordinator-spawned tasks)
	// ailang.source MUST override user defaults for proper cost attribution
	if task != nil {
		if _, exists := attrs["ailang.task_id"]; !exists && task.ID != "" {
			attrs["ailang.task_id"] = task.ID
		}
		// Add chain context from Task.Metadata (M-CHAINS-SIMPLIFY)
		if task.Metadata != nil {
			if chainID := task.Metadata["chain_id"]; chainID != "" {
				attrs["ailang.chain_id"] = chainID
			}
			if stageID := task.Metadata["stage_id"]; stageID != "" {
				attrs["ailang.stage_id"] = stageID
			}
		}
	}
	if _, exists := attrs["ailang.session_id"]; !exists && sessionID != "" {
		attrs["ailang.session_id"] = sessionID
	}
	// ALWAYS set source to coordinator when spawning from executor
	// This overrides any user default (e.g., ailang.source=user in shell env)
	// Critical for proper cost attribution: GitHub → Coordinator → Claude Code
	attrs["ailang.source"] = "coordinator"

	// Build final attribute string, sorted by key: map iteration order is
	// randomized, and the child env must be a deterministic function of the
	// task (M-EXECUTOR-ENV-HARDENING A1).
	parts := make([]string, 0, len(attrs))
	for k, v := range attrs {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

// UpdateEnvVar updates or appends an environment variable in the given slice.
func UpdateEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	for i, v := range env {
		if strings.HasPrefix(v, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// FindNVMBinary scans ~/.nvm/versions/node/ for a binary by name, trying the
// newest Node version first. Returns the full path if found, or empty string.
// This avoids hardcoding a specific Node version (e.g., v22.20.0) that breaks
// when NVM upgrades.
func FindNVMBinary(binaryName string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	nvmDir := filepath.Join(homeDir, ".nvm", "versions", "node")
	entries, err := os.ReadDir(nvmDir)
	if err != nil {
		return ""
	}

	// Collect version directories, sort newest first using proper semver comparison
	var versions []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "v") {
			versions = append(versions, entry.Name())
		}
	}
	sort.Slice(versions, func(i, j int) bool {
		mi, ni, pi := parseSemver(versions[i])
		mj, nj, pj := parseSemver(versions[j])
		if mi != mj {
			return mi > mj
		}
		if ni != nj {
			return ni > nj
		}
		return pi > pj
	})

	// Return the first version that has the binary
	for _, ver := range versions {
		binPath := filepath.Join(nvmDir, ver, "bin", binaryName)
		if _, err := os.Stat(binPath); err == nil {
			return binPath
		}
	}
	return ""
}

// FindNVMNodeBinDir returns the bin/ directory for the newest NVM Node version
// that contains the given binary. Useful for adding to PATH so all Node tools
// in that version are available.
func FindNVMNodeBinDir(binaryName string) string {
	path := FindNVMBinary(binaryName)
	if path == "" {
		return ""
	}
	return filepath.Dir(path)
}

// FindNativeBinary looks for a native (non-Node.js) Claude Code binary.
// Returns the absolute path, or empty string. The native binary is a
// Mach-O/ELF executable that does not require Node, making it the preferred
// option when available.
//
// The native installer's ~/.local/bin/<name> is checked FIRST: it self-updates,
// while the copy bundled in a VSCode extension only moves when VSCode updates
// extensions. Measured 2026-09-29: the bundled copy was 2.1.259 against an
// installed 2.1.284, rejected claude-sonnet-5-5 as unrecognized_model, and
// resolved the "sonnet" alias to claude-sonnet-5 — so every agent eval ran on
// whatever CLI VSCode last happened to ship.
func FindNativeBinary(binaryName string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	installed := filepath.Join(homeDir, ".local", "bin", binaryName)
	if info, err := os.Stat(installed); err == nil && !info.IsDir() {
		return installed
	}

	// Map Go arch names to VSCode extension arch names
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	platform := runtime.GOOS + "-" + arch

	// Pattern: ~/.vscode/extensions/anthropic.claude-code-*-<platform>/resources/native-binary/claude
	pattern := filepath.Join(homeDir, ".vscode", "extensions",
		fmt.Sprintf("anthropic.%s-code-*-%s", binaryName, platform),
		"resources", "native-binary", binaryName)

	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}

	// If multiple versions installed, pick the last (highest version by dir name sort)
	sort.Strings(matches)
	newest := matches[len(matches)-1]

	// Verify it exists and is a file
	info, err := os.Stat(newest)
	if err != nil || info.IsDir() {
		return ""
	}
	return newest
}

// RemoveEnvVar removes all entries for the given environment variable key.
func RemoveEnvVar(env []string, key string) []string {
	prefix := key + "="
	result := make([]string, 0, len(env))
	for _, v := range env {
		if !strings.HasPrefix(v, prefix) {
			result = append(result, v)
		}
	}
	return result
}

// parseSemver extracts major, minor, patch from a version string like "v25.5.0".
func parseSemver(v string) (int, int, int) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 3 {
		return 0, 0, 0
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])
	return major, minor, patch
}

// GetClaudeSettingsPath returns the path to the AILANG-specific Claude settings file.
// Creates the settings file with hooks configuration if it doesn't exist.
// The settings and hook script are embedded in the binary from scripts/hooks/.
func GetClaudeSettingsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Ensure directories exist
	claudeDir := filepath.Join(homeDir, ".ailang", "claude")
	hooksDir := filepath.Join(homeDir, ".ailang", "hooks")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create claude dir: %w", err)
	}
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create hooks dir: %w", err)
	}

	// Paths
	settingsPath := filepath.Join(claudeDir, "settings.json")
	hookScriptPath := filepath.Join(hooksDir, "claude_telemetry.sh")

	// Create hook script from embedded content (overwrite to ensure latest version)
	if err := os.WriteFile(hookScriptPath, embeddedHookScript, 0755); err != nil {
		return "", fmt.Errorf("failed to create hook script: %w", err)
	}

	// Create settings file from embedded content (overwrite to ensure latest version)
	if err := os.WriteFile(settingsPath, embeddedClaudeSettings, 0644); err != nil {
		return "", fmt.Errorf("failed to create settings file: %w", err)
	}

	return settingsPath, nil
}

// childStdlibPath picks the stdlib root to export to an agent child process:
// <workspace>/std, else <cwd>/std, and only if it holds the stdlib this binary
// was built with; "" otherwise (the child then uses its own embedded stdlib).
func childStdlibPath(opts EnvironmentOptions) string {
	var candidates []string
	if opts.Task != nil && opts.Task.Workspace != "" {
		candidates = append(candidates, filepath.Join(opts.Task.Workspace, "std"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "std"))
	}
	for _, c := range candidates {
		if !stdlibroot.IsStdlibDir(c) {
			continue
		}
		// Only a stdlib identical to the one built into this binary is exported
		// (see stdlibMatchesBinary). The child then runs the same stdlib whether
		// or not the checkout it was launched from is mid-edit or a release ahead.
		ok, why := stdlibMatchesBinary(c)
		if ok {
			return c
		}
		warnStdlibMismatchOnce(c, why)
	}
	return ""
}
