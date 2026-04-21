package coordinator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/websocket"
)

// HTTPBroadcaster sends task events to the Collaboration Hub server via HTTP.
// This enables real-time streaming when daemon and server run as separate processes.
type HTTPBroadcaster struct {
	serverURL  string
	httpClient *http.Client
	logger     *log.Logger

	// Event buffering for retries
	mu          sync.Mutex
	failedQueue []*websocket.TaskStreamEvent
	maxQueue    int
}

// NewHTTPBroadcaster creates a broadcaster that POSTs events to the server.
func NewHTTPBroadcaster(serverURL string, logger *log.Logger) *HTTPBroadcaster {
	return &HTTPBroadcaster{
		serverURL: serverURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger:      logger,
		failedQueue: make([]*websocket.TaskStreamEvent, 0),
		maxQueue:    100,
	}
}

// Broadcast sends an event to the server. Implements EventBroadcaster.
func (h *HTTPBroadcaster) Broadcast(event *websocket.TaskStreamEvent) {
	// Build request body
	body := map[string]interface{}{
		"task_id":        event.TaskID,
		"thread_id":      event.ThreadID,
		"stream_type":    string(event.StreamType),
		"turn_num":       event.TurnNum,
		"text":           event.Text,
		"tool_name":      event.ToolName,
		"tool_input":     event.ToolInput,
		"tool_output":    event.ToolOutput,
		"status":         event.Status,
		"tokens_in":      event.TokensIn,
		"tokens_out":     event.TokensOut,
		"cost":           event.Cost,
		"duration_sec":   event.DurationSec,
		"error_msg":      event.ErrorMsg,
		"workspace":      event.Workspace,
		"directive":      event.Directive,
		"directive_full": event.DirectiveFull,
		"agent_id":       event.AgentID,
		"source_type":    event.SourceType,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		h.logError("Failed to marshal event", err)
		return
	}

	// Log event being sent (debug)
	if h.logger != nil {
		h.logger.Printf("HTTPBroadcaster: Sending %s event for task %s", event.StreamType, event.TaskID)
	}

	// Send to server
	url := fmt.Sprintf("%s/api/coordinator/events", h.serverURL)
	resp, err := h.httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		h.logError("Failed to send event to server", err)
		h.queueFailedEvent(event)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.logError(fmt.Sprintf("Server returned status %d", resp.StatusCode), nil)
		h.queueFailedEvent(event)
		return
	}

	// Log success
	if h.logger != nil {
		h.logger.Printf("HTTPBroadcaster: Event sent successfully")
	}

	// Try to flush failed queue on success
	h.flushFailedQueue()
}

// BroadcastFunc returns the EventBroadcaster function.
func (h *HTTPBroadcaster) BroadcastFunc() EventBroadcaster {
	return h.Broadcast
}

// queueFailedEvent adds a failed event to the retry queue
func (h *HTTPBroadcaster) queueFailedEvent(event *websocket.TaskStreamEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Only queue important events (status changes, errors)
	// Note: StreamType is TaskStreamEventType which compares as string
	st := string(event.StreamType)
	if st != "status" && st != "error" && st != "turn_start" && st != "turn_end" {
		return
	}

	if len(h.failedQueue) >= h.maxQueue {
		// Drop oldest
		h.failedQueue = h.failedQueue[1:]
	}
	h.failedQueue = append(h.failedQueue, event)
}

// flushFailedQueue attempts to resend failed events
func (h *HTTPBroadcaster) flushFailedQueue() {
	h.mu.Lock()
	if len(h.failedQueue) == 0 {
		h.mu.Unlock()
		return
	}

	// Take up to 10 events to retry
	batch := h.failedQueue
	if len(batch) > 10 {
		batch = h.failedQueue[:10]
		h.failedQueue = h.failedQueue[10:]
	} else {
		h.failedQueue = make([]*websocket.TaskStreamEvent, 0)
	}
	h.mu.Unlock()

	// Retry each event (non-blocking)
	for _, event := range batch {
		h.Broadcast(event)
	}
}

func (h *HTTPBroadcaster) logError(msg string, err error) {
	if h.logger == nil {
		return
	}
	if err != nil {
		h.logger.Printf("HTTPBroadcaster: %s: %v", msg, err)
	} else {
		h.logger.Printf("HTTPBroadcaster: %s", msg)
	}
}

// CheckServerAvailable checks if the server is reachable
func (h *HTTPBroadcaster) CheckServerAvailable() bool {
	url := fmt.Sprintf("%s/health", h.serverURL)
	resp, err := h.httpClient.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// DefaultServerURL returns the default Collaboration Hub server URL
// Note: Use 127.0.0.1 instead of localhost to avoid IPv6 resolution issues
// (Go's HTTP client tries IPv6 first, but server may only bind to IPv4)
func DefaultServerURL() string {
	return "http://127.0.0.1:1957"
}
