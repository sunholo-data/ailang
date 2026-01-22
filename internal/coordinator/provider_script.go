// Package coordinator provides task execution and orchestration for the AILANG daemon.
package coordinator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/telemetry"
)

// ScriptProvider executes shell scripts for deterministic workflow tasks.
// Unlike AI providers, scripts run locally with predictable output.
//
// Usage in agent config:
//
//	invoke:
//	  type: script
//	  command: ./scripts/run-eval.sh
//	  env_from_payload: true
//	  timeout: 30m
type ScriptProvider struct {
	defaultShell   string
	defaultTimeout time.Duration
}

// NewScriptProvider creates a new script execution provider.
func NewScriptProvider() *ScriptProvider {
	return &ScriptProvider{
		defaultShell:   "/bin/sh",
		defaultTimeout: 5 * time.Minute,
	}
}

// Name returns the provider identifier.
func (p *ScriptProvider) Name() string {
	return "script"
}

// CanHandle returns true for tasks with script invoke type.
// Note: Script tasks are explicitly routed via InvokeConfig, not auto-detected.
func (p *ScriptProvider) CanHandle(task *AnalyzedTask) bool {
	// Script tasks are determined by agent config, not task content
	return false
}

// Execute runs a shell script with environment variables from the task payload.
func (p *ScriptProvider) Execute(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
	startTime := time.Now()

	// Validate we have invoke config
	if opts.InvokeConfig == nil || opts.InvokeConfig.Type != "script" {
		return nil, fmt.Errorf("ScriptProvider requires invoke config with type 'script'")
	}

	invoke := opts.InvokeConfig
	if invoke.Command == "" {
		return nil, fmt.Errorf("script invoke config missing 'command' field")
	}

	// Determine shell
	shell := invoke.Shell
	if shell == "" {
		shell = p.defaultShell
	}

	// Parse timeout
	timeout := p.defaultTimeout
	if invoke.Timeout != "" {
		if parsed, err := time.ParseDuration(invoke.Timeout); err == nil {
			timeout = parsed
		}
	}

	// Apply timeout from options if set
	if opts.Timeout > 0 && opts.Timeout < timeout {
		timeout = opts.Timeout
	}

	// Build environment
	env := os.Environ()

	// Inject trace context for distributed tracing (M-TRACE-HIERARCHY)
	// This enables script child processes (ailang eval-suite) to link to coordinator's trace
	env = telemetry.InjectTraceContext(ctx, env)

	// Add AILANG context variables
	// AILANG_PARENT_TASK_ID enables hierarchy tracking: when the script calls
	// `ailang exec`, the exec command will link back to this coordinator task
	env = append(env,
		fmt.Sprintf("AILANG_TASK_ID=%s", task.Task.ID),
		fmt.Sprintf("AILANG_PARENT_TASK_ID=%s", task.Task.ID), // Script's children link to this task
		fmt.Sprintf("AILANG_MESSAGE_ID=%s", task.Task.MessageID),
		fmt.Sprintf("AILANG_WORKSPACE=%s", opts.Workspace),
		"AILANG_PROVIDER=script",
		fmt.Sprintf("AILANG_PAYLOAD=%s", task.Task.Content), // Raw JSON payload
	)

	// Parse JSON payload to environment variables if enabled
	if invoke.EnvFromPayload && task.Task.Content != "" {
		payloadEnv, err := ParsePayloadToEnv(task.Task.Content)
		if err != nil {
			// Log warning but don't fail - content might not be JSON
			// This allows scripts to receive plain text content too
		} else {
			env = append(env, payloadEnv...)
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, shell, "-c", invoke.Command)
	cmd.Env = env

	// Set working directory
	workDir := opts.Workspace
	if invoke.WorkingDir != "" {
		// Support simple template substitution
		workDir = strings.ReplaceAll(invoke.WorkingDir, "{{.Workspace}}", opts.Workspace)
	}
	if workDir != "" {
		cmd.Dir = workDir
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer

	// If event handler is available, stream output
	if opts.EventHandler != nil {
		stdoutWriter := &streamWriter{
			handler: opts.EventHandler,
			stream:  "stdout",
			buffer:  &stdout,
		}
		stderrWriter := &streamWriter{
			handler: opts.EventHandler,
			stream:  "stderr",
			buffer:  &stderr,
		}
		cmd.Stdout = stdoutWriter
		cmd.Stderr = stderrWriter
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	// Execute the command
	err := cmd.Run()
	duration := time.Since(startTime)

	// Build result
	result := &ExecuteResult{
		Provider:     "script",
		Duration:     duration,
		Output:       stdout.String(),
		Cost:         0.0, // Scripts are free!
		TokensUsed:   0,
		InputTokens:  0,
		OutputTokens: 0,
	}

	// Check for errors
	if ctx.Err() == context.DeadlineExceeded {
		result.Success = false
		result.Error = fmt.Sprintf("script timed out after %v", timeout)
		return result, nil
	}

	if err != nil {
		result.Success = false
		errMsg := err.Error()
		if stderr.Len() > 0 {
			errMsg = fmt.Sprintf("%v\nstderr: %s", err, stderr.String())
		}
		result.Error = errMsg
		return result, nil
	}

	// Check exit code
	if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() != 0 {
		result.Success = false
		result.Error = fmt.Sprintf("exit code %d\nstderr: %s", cmd.ProcessState.ExitCode(), stderr.String())
		return result, nil
	}

	result.Success = true
	return result, nil
}

// ParsePayloadToEnv converts a JSON payload to environment variables.
// Keys are converted to UPPER_SNAKE_CASE.
// Nested objects are flattened: {"db": {"host": "x"}} → DB_HOST=x
// Arrays become comma-separated: ["a", "b"] → VALUE=a,b
func ParsePayloadToEnv(payload string) ([]string, error) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return nil, nil
	}

	// Try to parse as JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return nil, fmt.Errorf("payload is not valid JSON: %w", err)
	}

	var env []string
	flattenToEnv("", data, &env)
	return env, nil
}

// flattenToEnv recursively flattens a nested map to environment variables.
func flattenToEnv(prefix string, data map[string]interface{}, env *[]string) {
	for key, value := range data {
		envKey := toEnvKey(prefix, key)

		switch v := value.(type) {
		case map[string]interface{}:
			// Recurse into nested objects
			flattenToEnv(envKey, v, env)
		case []interface{}:
			// Convert arrays to comma-separated strings
			parts := make([]string, len(v))
			for i, item := range v {
				parts[i] = fmt.Sprintf("%v", item)
			}
			*env = append(*env, fmt.Sprintf("%s=%s", envKey, strings.Join(parts, ",")))
		case bool:
			*env = append(*env, fmt.Sprintf("%s=%t", envKey, v))
		case float64:
			// JSON numbers are float64
			if v == float64(int(v)) {
				*env = append(*env, fmt.Sprintf("%s=%d", envKey, int(v)))
			} else {
				*env = append(*env, fmt.Sprintf("%s=%g", envKey, v))
			}
		case string:
			*env = append(*env, fmt.Sprintf("%s=%s", envKey, v))
		case nil:
			*env = append(*env, fmt.Sprintf("%s=", envKey))
		default:
			*env = append(*env, fmt.Sprintf("%s=%v", envKey, v))
		}
	}
}

// toEnvKey converts a JSON key to an environment variable name.
// Uses UPPER_SNAKE_CASE and sanitizes invalid characters.
func toEnvKey(prefix, key string) string {
	// Convert to uppercase
	key = strings.ToUpper(key)

	// Replace invalid characters with underscore
	// Only allow A-Z, 0-9, and underscore
	re := regexp.MustCompile(`[^A-Z0-9_]`)
	key = re.ReplaceAllString(key, "_")

	// Combine with prefix
	if prefix == "" {
		return key
	}
	return prefix + "_" + key
}

// streamWriter wraps an io.Writer to also send output to an event handler.
type streamWriter struct {
	handler interface {
		OnText(text string)
	}
	stream string
	buffer *bytes.Buffer
}

func (w *streamWriter) Write(p []byte) (n int, err error) {
	// Write to buffer
	n, err = w.buffer.Write(p)
	if err != nil {
		return n, err
	}

	// Send to event handler
	if w.handler != nil {
		w.handler.OnText(string(p))
	}

	return n, nil
}

// Ensure streamWriter implements io.Writer
var _ io.Writer = (*streamWriter)(nil)
