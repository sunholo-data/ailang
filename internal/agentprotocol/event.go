package agentprotocol

import (
	"time"
)

// InteractiveEvent represents a provider-agnostic event from an interactive
// environment (Claude Code, VSCode, etc.) that can trigger agent workflows.
//
// This abstraction allows the agent protocol to remain independent of any
// specific interactive tool's event format.
type InteractiveEvent struct {
	// SessionID uniquely identifies the interactive session
	// Example: Claude session UUID
	SessionID string `json:"session_id"`

	// UserID identifies who triggered the event
	UserID string `json:"user_id"`

	// Event describes what happened
	// Examples: "Stop", "TaskComplete", "SessionStart", "Notification"
	Event string `json:"event"`

	// Timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// Artifacts are content-addressed references to files/blobs
	// This avoids embedding large content directly in events
	Artifacts []ArtifactRef `json:"artifacts,omitempty"`

	// Notes provides free-form context about the event
	// Examples: "User said 'looks good'", "Session timeout"
	Notes string `json:"notes,omitempty"`

	// Provider identifies the source of the event
	// Examples: "claude-code", "vscode", "headless"
	Provider string `json:"provider,omitempty"`
}

// ArtifactRef is a content-addressed reference to a file or blob.
// The hash ensures immutability and allows deduplication.
type ArtifactRef struct {
	// Path is the original file path (for human readability)
	// Example: "design_docs/planned/M-FIX-123.md"
	Path string `json:"path"`

	// Hash is the SHA256 hash of the content (content-addressed)
	// Format: "sha256:abc123..."
	Hash string `json:"hash"`

	// MimeType describes the content type
	// Examples: "text/markdown", "application/json", "text/plain"
	MimeType string `json:"mime_type"`

	// Size is the content size in bytes
	Size int64 `json:"size,omitempty"`
}

// EventToMessage converts an InteractiveEvent to an agent protocol Envelope.
// This is the bridge from interactive sessions to autonomous agent workflows.
//
// Parameters:
//   - event: The interactive event to convert
//   - toAgent: The target agent ID
//   - correlationID: Optional correlation ID (generated if empty)
//
// Returns an Envelope ready to be written to the agent's inbox.
func EventToMessage(event *InteractiveEvent, toAgent string, correlationID string) *Envelope {
	if correlationID == "" {
		correlationID = GenerateCorrelationID()
	}

	// Build payload from event
	payload := map[string]interface{}{
		"event":      event.Event,
		"session_id": event.SessionID,
		"user_id":    event.UserID,
		"provider":   event.Provider,
		"notes":      event.Notes,
	}

	// Add artifacts if present
	if len(event.Artifacts) > 0 {
		artifacts := make([]map[string]interface{}, len(event.Artifacts))
		for i, art := range event.Artifacts {
			artifacts[i] = map[string]interface{}{
				"path":      art.Path,
				"hash":      art.Hash,
				"mime_type": art.MimeType,
				"size":      art.Size,
			}
		}
		payload["artifacts"] = artifacts
	}

	return &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       GenerateMessageID(),
		CorrelationID:   correlationID,
		TraceID:         GenerateTraceID(),
		Timestamp:       event.Timestamp.UTC().Format(time.RFC3339),
		TTLSeconds:      3600, // 1 hour default TTL
		FromAgent:       "interactive",
		ToAgent:         toAgent,
		MessageType:     "notification",
		PayloadSchema:   "interactive_event.v1",
		Payload:         payload,
	}
}
