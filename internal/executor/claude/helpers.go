// Package claude helper utilities extracted from claude.go to keep that file
// under the 800-line code-health gate. Pure, receiver-less helpers only.
package claude

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

func getErrorMessage(result *claudeHeadlessResult) string {
	if result.IsError {
		return result.Result
	}
	if result.Subtype == "error" || result.Subtype == "timeout" {
		return result.Result
	}
	return ""
}

// isCloudWorkspace detects if a workspace path is in a cloud container.
// Cloud paths typically start with /workspace/ (Cloud Run, Google Cloud Container).
// This detection ensures we use appropriate permission handling for cloud environments.
func isCloudWorkspace(workspace string) bool {
	// Cloud container paths typically start with /workspace/
	// This convention is used by Cloud Run, Pub/Sub-triggered containers, and GCP workspaces
	return strings.HasPrefix(workspace, "/workspace/")
}

// isValidUUID checks if a string is a valid UUID format
// Claude Code CLI requires session IDs to be valid UUIDs
func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// intFromAny coerces a JSON-decoded number-ish value to int.
// json.Unmarshal into map[string]interface{} yields float64 for numbers;
// some payloads may also use json.Number or raw int. Returns 0 on miss.
func intFromAny(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}
