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
}

// BuildEnvironment creates the common environment variables for AI executor processes.
// This consolidates the duplicated setup code from claude.go and gemini.go.
//
// The environment includes:
//   - AILANG_STDLIB_PATH: Path to AILANG standard library
//   - PWD: Working directory (if workspace specified)
//   - TRACEPARENT: W3C trace context for distributed tracing
//   - AILANG_TASK_ID, AILANG_SESSION_ID: Correlation IDs
//   - AILANG_PARENT_TASK_ID: Parent task for hierarchy tracking
//   - AILANG_CHAIN_ID, AILANG_STAGE_ID: Execution chain context (M-CHAINS-SIMPLIFY)
//   - AILANG_MESSAGE_ID: Source message that triggered this chain
//   - OTEL_RESOURCE_ATTRIBUTES: Resource attributes for trace linking
//   - OTEL_EXPORTER_OTLP_ENDPOINT: OTLP endpoint for trace collection
//   - GOOGLE_CLOUD_PROJECT: GCP project for cloud tracing
func BuildEnvironment(opts EnvironmentOptions) []string {
	env := os.Environ()

	// Strip CLAUDECODE env var to prevent "Cannot be launched inside another
	// Claude Code session" errors. This applies to ALL executors — even Gemini
	// may shell out to Claude Code, and future executors shouldn't need to know
	// about this workaround.
	env = RemoveEnvVar(env, "CLAUDECODE")

	// Set up AILANG stdlib path.
	// Priority: workspace/std (cloud: cloned repo has stdlib) > cwd/std (local: running from repo root).
	// This ensures cloud agents (where cwd=/workspace but repo is at /workspace/{taskID})
	// find the stdlib correctly when the cloned repo is an AILANG workspace.
	cwd, _ := os.Getwd()
	stdlibPath := filepath.Join(cwd, "std")
	if opts.Task != nil && opts.Task.Workspace != "" {
		workspaceStd := filepath.Join(opts.Task.Workspace, "std")
		if info, err := os.Stat(workspaceStd); err == nil && info.IsDir() {
			stdlibPath = workspaceStd
		}
	}
	env = append(env, fmt.Sprintf("AILANG_STDLIB_PATH=%s", stdlibPath))

	// Set working directory if specified
	if opts.Task != nil && opts.Task.Workspace != "" {
		env = append(env, fmt.Sprintf("PWD=%s", opts.Task.Workspace))
	}

	// Inject W3C trace context for distributed tracing
	// This enables ailang run commands spawned by the executor to link back to this trace
	if opts.Context != nil {
		env = telemetry.InjectTraceContext(opts.Context, env)
	}

	// Inject correlation IDs for fallback linking
	taskID := ""
	parentTaskID := ""
	if opts.Task != nil {
		taskID = opts.Task.ID
		parentTaskID = opts.Task.ParentTaskID
	}
	env = telemetry.InjectCorrelationIDs(env, taskID, opts.SessionID)

	// Inject parent task ID for hierarchy tracking (M-TASK-HIERARCHY)
	// When the executor spawns AI CLI (Claude/Gemini), any child ailang commands
	// (ailang run, ailang check) should link back to this executor's task ID.
	// - If ParentTaskID is set: propagate the explicit parent (nested exec calls)
	// - Otherwise: use this task's ID as the parent for child commands
	effectiveParentID := parentTaskID
	if effectiveParentID == "" && taskID != "" {
		effectiveParentID = taskID
	}
	if effectiveParentID != "" {
		env = append(env, fmt.Sprintf("AILANG_PARENT_TASK_ID=%s", effectiveParentID))
	}

	// Inject chain context for unified hierarchy tracking (M-CHAINS-SIMPLIFY)
	// Chain IDs are passed via Task.Metadata from the coordinator
	if opts.Task != nil && opts.Task.Metadata != nil {
		if chainID := opts.Task.Metadata["chain_id"]; chainID != "" {
			env = append(env, fmt.Sprintf("AILANG_CHAIN_ID=%s", chainID))
		}
		if stageID := opts.Task.Metadata["stage_id"]; stageID != "" {
			env = append(env, fmt.Sprintf("AILANG_STAGE_ID=%s", stageID))
		}
		if messageID := opts.Task.Metadata["message_id"]; messageID != "" {
			env = append(env, fmt.Sprintf("AILANG_MESSAGE_ID=%s", messageID))
		}
		// M-PKG-INFLIGHT: surface the agent name to child processes so that
		// `ailang publish` can attach X-Ailang-Agent-ID and the validator can
		// link package_builds rows back to the agent that fired them.
		if agentID := opts.Task.Metadata["ailang.agent_id"]; agentID != "" {
			env = append(env, fmt.Sprintf("AILANG_AGENT_ID=%s", agentID))
		}
	}

	// Build resource attributes for trace linking (M-TASK-HIERARCHY)
	// Merges existing attributes from environment with task-specific attributes
	resourceAttrs := BuildResourceAttributes(opts.Task, opts.SessionID)
	env = append(env, fmt.Sprintf("OTEL_RESOURCE_ATTRIBUTES=%s", resourceAttrs))

	// Configure OTEL exporter for trace collection
	// Priority: parent env > default to local observatory server
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		// Default to local observatory for unified trace collection
		endpoint = "http://localhost:1957"
	}
	env = append(env, fmt.Sprintf("OTEL_EXPORTER_OTLP_ENDPOINT=%s", endpoint))

	// Pass through OTEL protocol if set
	if protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"); protocol != "" {
		env = append(env, fmt.Sprintf("OTEL_EXPORTER_OTLP_PROTOCOL=%s", protocol))
	}

	// For GCP export, set the project.
	// Priority: EnvironmentOptions override > OTLP_GOOGLE_CLOUD_PROJECT > GOOGLE_CLOUD_PROJECT
	project := opts.GCPProject
	if project == "" {
		project = os.Getenv("OTLP_GOOGLE_CLOUD_PROJECT")
	}
	if project == "" {
		project = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if project != "" {
		env = UpdateEnvVar(env, "GOOGLE_CLOUD_PROJECT", project)
		env = UpdateEnvVar(env, "OTLP_GOOGLE_CLOUD_PROJECT", project)
	}

	// GCP location override (e.g. us-central1, europe-west1)
	location := opts.GCPLocation
	if location == "" {
		location = os.Getenv("GOOGLE_CLOUD_LOCATION")
	}
	if location != "" {
		env = UpdateEnvVar(env, "GOOGLE_CLOUD_LOCATION", location)
	}

	// Claude-specific telemetry configuration
	if opts.EnableClaudeTelemetry {
		env = append(env, "CLAUDE_CODE_ENABLE_TELEMETRY=1")
		env = append(env, "OTEL_METRICS_EXPORTER=otlp")
		env = append(env, "OTEL_LOGS_EXPORTER=otlp")
	}

	// Gemini-specific telemetry configuration
	if opts.EnableGeminiTelemetry {
		env = append(env, "GEMINI_TELEMETRY_ENABLED=true")

		// Set telemetry target based on available configuration
		if target := os.Getenv("GEMINI_TELEMETRY_TARGET"); target != "" {
			env = append(env, fmt.Sprintf("GEMINI_TELEMETRY_TARGET=%s", target))
		} else if project != "" {
			env = append(env, "GEMINI_TELEMETRY_TARGET=gcp")
		}
	}

	return env
}

// BuildResourceAttributes creates OTEL_RESOURCE_ATTRIBUTES value.
// Merges existing attributes from environment with task-specific attributes.
// Priority: existing env attrs + task Metadata + default attrs.
func BuildResourceAttributes(task *Task, sessionID string) string {
	attrs := make(map[string]string)

	// 1. Start with existing environment attributes (preserve user settings)
	if existing := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); existing != "" {
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

	// Build final attribute string
	var parts []string
	for k, v := range attrs {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
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

// FindNativeBinary looks for a native (non-Node.js) binary installed by the
// VSCode Claude Code extension. Returns the absolute path, or empty string.
// The native binary is a Mach-O/ELF executable that does not require Node,
// making it the preferred option when available.
func FindNativeBinary(binaryName string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
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
