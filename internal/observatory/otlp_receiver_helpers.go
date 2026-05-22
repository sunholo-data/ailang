// Package observatory provides helper functions for the OTLP receiver.
package observatory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
)

// anyValueToGo converts an OTLP AnyValue to a Go value.
func anyValueToGo(v *commonpb.AnyValue) any {
	if v == nil {
		return nil
	}
	switch val := v.Value.(type) {
	case *commonpb.AnyValue_StringValue:
		return val.StringValue
	case *commonpb.AnyValue_IntValue:
		return val.IntValue
	case *commonpb.AnyValue_DoubleValue:
		return val.DoubleValue
	case *commonpb.AnyValue_BoolValue:
		return val.BoolValue
	case *commonpb.AnyValue_ArrayValue:
		arr := make([]any, len(val.ArrayValue.Values))
		for i, elem := range val.ArrayValue.Values {
			arr[i] = anyValueToGo(elem)
		}
		return arr
	case *commonpb.AnyValue_KvlistValue:
		m := make(map[string]any)
		for _, kv := range val.KvlistValue.Values {
			m[kv.Key] = anyValueToGo(kv.Value)
		}
		return m
	default:
		return nil
	}
}

// generateSpanID generates a random 16-character hex span ID.
func generateSpanID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateTraceID generates a random 32-character hex trace ID.
func generateTraceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// matchesPattern checks if a span name matches a single filter pattern,
// optionally scoped to a specific service.
func matchesPattern(name string, resourceAttrs map[string]any, p FilterPattern) bool {
	// Check service scope first
	if p.Service != "" {
		serviceName, _ := resourceAttrs["service.name"].(string)
		if serviceName != p.Service {
			return false
		}
	}

	switch p.Type {
	case "prefix":
		return strings.HasPrefix(name, p.Pattern)
	case "suffix":
		return strings.HasSuffix(name, p.Pattern)
	case "exact":
		return name == p.Pattern
	default:
		return name == p.Pattern
	}
}

// shouldFilterSpan returns true if the span should be filtered out (not stored).
// Uses the receiver's SpanFilterConfig: allow-list takes priority over deny-list.
func (r *OTLPReceiver) shouldFilterSpan(name string, resourceAttrs map[string]any) bool {
	if r.filterConfig.DisableAll {
		return false
	}

	// Allow-list takes priority: if any allow pattern matches, keep the span
	for _, p := range r.filterConfig.AllowPatterns {
		if matchesPattern(name, resourceAttrs, p) {
			return false
		}
	}

	// Deny-list: if any deny pattern matches, filter the span
	for _, p := range r.filterConfig.DenyPatterns {
		if matchesPattern(name, resourceAttrs, p) {
			return true
		}
	}

	// Default: keep the span
	return false
}

// validateTaskHierarchy validates that task_id and assignment_id references
// exist, and CLEARS them if they don't.
//
// M-EVAL-LOCAL-OBSERVABILITY (v0.22.0): Previously this only logged warnings
// and let the INSERT downstream fail on FOREIGN KEY constraint, which dropped
// ALL spans for eval-suite-driven runs (eval-suite generates task_id attributes
// like "eval-<timestamp>" without inserting parent rows in the tasks table).
//
// Now: when a referenced task or assignment is missing, the field is set to ""
// (which downstream code translates to SQL NULL via the interface{} pattern).
// The span is still stored — just with a NULL parent reference — so live
// monitoring can see it. This is the right model: eval-suite tasks are
// ephemeral; coordinator tasks are durable; both should produce queryable
// spans regardless of whether the parent row exists.
//
// Cleared FK references mean:
//   - Aggregation updates (UpdateTaskAggregates / UpdateAgentAssignmentAggregates)
//     short-circuit gracefully because they already check for empty IDs.
//   - GetTaskHierarchy / GetTaskSpanSummary queries don't return this span
//     under a particular task — correct, because it has no task parent.
//   - The span is still queryable by trace_id, name, span_id, etc.
func (r *OTLPReceiver) validateTaskHierarchy(ctx context.Context, span *Span) {
	// Validate task_id if present; clear if parent missing
	if span.TaskID != "" {
		task, err := r.backend.GetTask(ctx, span.TaskID)
		if err != nil || task == nil {
			fmt.Printf("observatory: span %s has task_id=%s but task not found — storing with task_id=NULL\n", span.ID, span.TaskID)
			span.TaskID = ""
		}
	}

	// Validate assignment_id if present; clear if parent missing
	if span.AgentAssignmentID != "" {
		assignment, err := r.backend.GetAgentAssignment(ctx, span.AgentAssignmentID)
		if err != nil || assignment == nil {
			fmt.Printf("observatory: span %s has assignment_id=%s but assignment not found — storing with assignment_id=NULL\n", span.ID, span.AgentAssignmentID)
			span.AgentAssignmentID = ""
		}
	}
}

// Helper functions
func extractInt(attrs map[string]any, keys ...string) int {
	for _, key := range keys {
		if v, ok := attrs[key]; ok {
			switch val := v.(type) {
			case int:
				return val
			case int64:
				return int(val)
			case float64:
				return int(val)
			case string:
				// Claude Code sends numbers as strings
				if i, err := strconv.Atoi(val); err == nil {
					return i
				}
				// Try parsing as float then convert
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					return int(f)
				}
			}
		}
	}
	return 0
}

func extractFloat(attrs map[string]any, keys ...string) float64 {
	for _, key := range keys {
		if v, ok := attrs[key]; ok {
			switch val := v.(type) {
			case float64:
				return val
			case int:
				return float64(val)
			case int64:
				return float64(val)
			case string:
				// Claude Code sends numbers as strings
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					return f
				}
			}
		}
	}
	return 0
}

func extractString(attrs map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := attrs[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

// extractTaskIDFromCwd extracts task ID from worktree path in process.cwd attribute.
// Claude Code CLI doesn't pass OTEL_RESOURCE_ATTRIBUTES to subprocesses,
// but the worktree path contains the task ID.
func extractTaskIDFromCwd(attrs map[string]any) string {
	cwd := extractString(attrs, "process.cwd")
	if cwd == "" {
		return ""
	}
	return extractTaskIDFromPath(cwd)
}

// extractTaskIDFromPath extracts task ID from a file path.
// Path format: .../worktrees/.../task-XXXXXXXX/...
// Returns empty string if no task ID found.
func extractTaskIDFromPath(path string) string {
	if path == "" {
		return ""
	}

	// Look for task ID pattern in the path
	const taskPrefix = "task-"
	idx := strings.Index(path, "/worktrees/")
	if idx == -1 {
		return ""
	}

	// Find task-XXXXXXXX in the path after /worktrees/
	remainder := path[idx:]
	taskIdx := strings.Index(remainder, taskPrefix)
	if taskIdx == -1 {
		return ""
	}

	// Extract task ID (task-XXXXXXXX format, 8 hex chars after prefix)
	start := taskIdx
	end := start + len(taskPrefix) + 8 // task- + 8 hex chars
	if end > len(remainder) {
		// Try to find next path separator
		nextSlash := strings.Index(remainder[start:], "/")
		if nextSlash > 0 {
			end = start + nextSlash
		} else {
			end = len(remainder)
		}
	}

	taskID := remainder[start:end]
	if strings.HasPrefix(taskID, taskPrefix) {
		return taskID
	}
	return ""
}

// parseFilterPattern parses a single pattern string into a FilterPattern.
// Formats: "name" (exact), "name*" (prefix), "*name" (suffix), "service:name" (service-scoped).
func parseFilterPattern(raw string) FilterPattern {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return FilterPattern{}
	}

	// Check for service-scoped pattern: "service:pattern"
	var service string
	if idx := strings.Index(raw, ":"); idx > 0 && !strings.HasPrefix(raw, "/") {
		service = raw[:idx]
		raw = raw[idx+1:]
	}

	var patternType, pattern string
	switch {
	case strings.HasPrefix(raw, "*") && strings.HasSuffix(raw, "*"):
		// *contains* — treat as prefix+suffix not supported, use prefix for simplicity
		patternType = "prefix"
		pattern = strings.TrimPrefix(raw, "*")
		pattern = strings.TrimSuffix(pattern, "*")
	case strings.HasSuffix(raw, "*"):
		patternType = "prefix"
		pattern = strings.TrimSuffix(raw, "*")
	case strings.HasPrefix(raw, "*"):
		patternType = "suffix"
		pattern = strings.TrimPrefix(raw, "*")
	default:
		patternType = "exact"
		pattern = raw
	}

	return FilterPattern{Type: patternType, Pattern: pattern, Service: service}
}

// LoadSpanFilterConfig creates a SpanFilterConfig from environment variables,
// merging with defaults. Exported for testing.
func LoadSpanFilterConfig() *SpanFilterConfig {
	config := DefaultSpanFilterConfig()

	if os.Getenv("AILANG_SPAN_FILTER_DISABLE") == "true" {
		config.DisableAll = true
	}

	if allowEnv := os.Getenv("AILANG_SPAN_FILTER_ALLOW"); allowEnv != "" {
		for _, raw := range strings.Split(allowEnv, ",") {
			if p := parseFilterPattern(raw); p.Pattern != "" {
				config.AllowPatterns = append(config.AllowPatterns, p)
			}
		}
	}

	if denyEnv := os.Getenv("AILANG_SPAN_FILTER_DENY"); denyEnv != "" {
		for _, raw := range strings.Split(denyEnv, ",") {
			if p := parseFilterPattern(raw); p.Pattern != "" {
				config.DenyPatterns = append(config.DenyPatterns, p)
			}
		}
	}

	fmt.Printf("observatory: span filter config loaded (allow=%d, deny=%d, disable=%v)\n",
		len(config.AllowPatterns), len(config.DenyPatterns), config.DisableAll)
	return config
}
