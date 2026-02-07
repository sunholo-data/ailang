// Package executor provides environment setup utilities shared across all AI executors.
package executor

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/telemetry"
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

	// Set up AILANG stdlib path
	cwd, _ := os.Getwd()
	stdlibPath := filepath.Join(cwd, "std")
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

	// For GCP export, set the project
	// Check OTLP_GOOGLE_CLOUD_PROJECT first (Gemini CLI standard), fallback to GOOGLE_CLOUD_PROJECT
	project := os.Getenv("OTLP_GOOGLE_CLOUD_PROJECT")
	if project == "" {
		project = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if project != "" {
		env = append(env, fmt.Sprintf("GOOGLE_CLOUD_PROJECT=%s", project))
		env = append(env, fmt.Sprintf("OTLP_GOOGLE_CLOUD_PROJECT=%s", project))
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

	// Collect version directories, sort newest first
	var versions []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "v") {
			versions = append(versions, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))

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
