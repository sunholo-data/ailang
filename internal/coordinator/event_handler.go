// Package coordinator provides the event handler for streaming executor events.
package coordinator

import (
	"fmt"
	"sync"
	"time"

	"github.com/sunholo/ailang/internal/websocket"
)

// EventBroadcaster is a function that broadcasts task stream events.
// This allows the event handler to be decoupled from the WebSocket server.
type EventBroadcaster func(*websocket.TaskStreamEvent)

// EventStorer is a function that stores task stream events to database.
// This allows events to be persisted for historical replay.
type EventStorer func(*TaskEventRecord) error

// CoordinatorEventHandler implements executor.EventHandler to capture
// and broadcast executor events via WebSocket.
type CoordinatorEventHandler struct {
	taskID    string
	threadID  string
	broadcast EventBroadcaster
	store     EventStorer

	// Rate limiting
	mu              sync.Mutex
	lastEventTime   time.Time
	eventCount      int
	maxEventsPerSec int
	throttled       bool

	// Event buffering for replay
	eventBuffer   []*websocket.TaskStreamEvent
	maxBufferSize int
	bufferMu      sync.RWMutex

	// Metrics tracking
	currentTurn int
	tokensIn    int
	tokensOut   int
	cost        float64
	startTime   time.Time
}

// NewCoordinatorEventHandler creates a new event handler for a task.
func NewCoordinatorEventHandler(taskID, threadID string, broadcast EventBroadcaster) *CoordinatorEventHandler {
	return &CoordinatorEventHandler{
		taskID:          taskID,
		threadID:        threadID,
		broadcast:       broadcast,
		maxEventsPerSec: 10,  // Rate limit: max 10 events per second
		maxBufferSize:   100, // Keep last 100 events for replay
		eventBuffer:     make([]*websocket.TaskStreamEvent, 0, 100),
		startTime:       time.Now(),
	}
}

// SetEventStorer sets the database storage function for persisting events.
func (h *CoordinatorEventHandler) SetEventStorer(storer EventStorer) {
	h.store = storer
}

// OnTurnStart is called when a new turn starts
func (h *CoordinatorEventHandler) OnTurnStart(turnNum int) {
	h.mu.Lock()
	h.currentTurn = turnNum
	h.mu.Unlock()

	h.emitEvent(&websocket.TaskStreamEvent{
		TaskID:     h.taskID,
		ThreadID:   h.threadID,
		StreamType: websocket.TaskStreamTurnStart,
		TurnNum:    turnNum,
	})
}

// OnText is called when text is generated
func (h *CoordinatorEventHandler) OnText(text string) {
	event := &websocket.TaskStreamEvent{
		TaskID:     h.taskID,
		ThreadID:   h.threadID,
		StreamType: websocket.TaskStreamText,
		TurnNum:    h.getCurrentTurn(),
		Text:       text, // Store full text, don't truncate
	}

	// Always store events (for historical replay)
	// Only apply rate limiting to WebSocket broadcast
	h.storeEvent(event)

	// Rate limit broadcasting only
	if h.checkRateLimit() {
		h.broadcastEvent(event)
	}
}

// OnToolUse is called when a tool is invoked
func (h *CoordinatorEventHandler) OnToolUse(toolName string, input string) {
	h.emitEvent(&websocket.TaskStreamEvent{
		TaskID:     h.taskID,
		ThreadID:   h.threadID,
		StreamType: websocket.TaskStreamToolUse,
		TurnNum:    h.getCurrentTurn(),
		ToolName:   toolName,
		ToolInput:  input, // Store full input, truncate only on broadcast
	})
}

// OnToolResult is called when a tool returns a result
func (h *CoordinatorEventHandler) OnToolResult(toolName string, output string) {
	h.emitEvent(&websocket.TaskStreamEvent{
		TaskID:     h.taskID,
		ThreadID:   h.threadID,
		StreamType: websocket.TaskStreamToolResult,
		TurnNum:    h.getCurrentTurn(),
		ToolName:   toolName,
		ToolOutput: output, // Store full output, truncate only on broadcast
	})
}

// OnTurnEnd is called when a turn ends
func (h *CoordinatorEventHandler) OnTurnEnd(turnNum int) {
	h.emitEvent(&websocket.TaskStreamEvent{
		TaskID:     h.taskID,
		ThreadID:   h.threadID,
		StreamType: websocket.TaskStreamTurnEnd,
		TurnNum:    turnNum,
	})
}

// OnError is called when an error occurs
func (h *CoordinatorEventHandler) OnError(err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	h.emitEvent(&websocket.TaskStreamEvent{
		TaskID:     h.taskID,
		ThreadID:   h.threadID,
		StreamType: websocket.TaskStreamError,
		TurnNum:    h.getCurrentTurn(),
		ErrorMsg:   errMsg,
	})
}

// UpdateMetrics updates the token and cost metrics
func (h *CoordinatorEventHandler) UpdateMetrics(tokensIn, tokensOut int, cost float64) {
	h.mu.Lock()
	h.tokensIn = tokensIn
	h.tokensOut = tokensOut
	h.cost = cost
	h.mu.Unlock()
}

// EmitStatus emits a status change event
func (h *CoordinatorEventHandler) EmitStatus(status string) {
	h.mu.Lock()
	tokensIn := h.tokensIn
	tokensOut := h.tokensOut
	cost := h.cost
	duration := int(time.Since(h.startTime).Seconds())
	h.mu.Unlock()

	h.emitEvent(&websocket.TaskStreamEvent{
		TaskID:      h.taskID,
		ThreadID:    h.threadID,
		StreamType:  websocket.TaskStreamStatus,
		Status:      status,
		TokensIn:    tokensIn,
		TokensOut:   tokensOut,
		Cost:        cost,
		DurationSec: duration,
	})
}

// GetEventBuffer returns a copy of the event buffer for replay
func (h *CoordinatorEventHandler) GetEventBuffer() []*websocket.TaskStreamEvent {
	h.bufferMu.RLock()
	defer h.bufferMu.RUnlock()

	result := make([]*websocket.TaskStreamEvent, len(h.eventBuffer))
	copy(result, h.eventBuffer)
	return result
}

// emitEvent broadcasts an event, buffers it for replay, and stores to database
func (h *CoordinatorEventHandler) emitEvent(event *websocket.TaskStreamEvent) {
	h.storeEvent(event)
	h.broadcastEvent(event)
}

// storeEvent stores an event to database for historical replay (always called)
func (h *CoordinatorEventHandler) storeEvent(event *websocket.TaskStreamEvent) {
	// Buffer the event (in-memory for current session)
	h.bufferMu.Lock()
	if len(h.eventBuffer) >= h.maxBufferSize {
		// Remove oldest event
		h.eventBuffer = h.eventBuffer[1:]
	}
	h.eventBuffer = append(h.eventBuffer, event)
	h.bufferMu.Unlock()

	// Store to database for historical replay (async to not block streaming)
	if h.store != nil {
		go func() {
			record := &TaskEventRecord{
				TaskID:      event.TaskID,
				ThreadID:    event.ThreadID,
				StreamType:  string(event.StreamType),
				TurnNum:     event.TurnNum,
				Text:        event.Text,
				ToolName:    event.ToolName,
				ToolInput:   event.ToolInput,
				ToolOutput:  event.ToolOutput,
				ErrorMsg:    event.ErrorMsg,
				Status:      event.Status,
				TokensIn:    event.TokensIn,
				TokensOut:   event.TokensOut,
				Cost:        event.Cost,
				DurationSec: event.DurationSec,
			}
			if err := h.store(record); err != nil {
				fmt.Printf("[DEBUG] EventHandler: failed to store event: %v\n", err)
			}
		}()
	}
}

// broadcastEvent broadcasts an event to WebSocket clients (may be rate-limited)
func (h *CoordinatorEventHandler) broadcastEvent(event *websocket.TaskStreamEvent) {
	if h.broadcast != nil {
		// Truncate for WebSocket broadcast (live streaming)
		broadcastEvent := *event
		broadcastEvent.Text = truncateString(event.Text, 2000)
		broadcastEvent.ToolInput = truncateString(event.ToolInput, 1000)
		broadcastEvent.ToolOutput = truncateString(event.ToolOutput, 2000)
		h.broadcast(&broadcastEvent)
	}
}

// checkRateLimit checks if we're within the rate limit for high-frequency events
func (h *CoordinatorEventHandler) checkRateLimit() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()

	// Reset counter every second
	if now.Sub(h.lastEventTime) >= time.Second {
		h.eventCount = 0
		h.lastEventTime = now
		h.throttled = false
	}

	h.eventCount++

	if h.eventCount > h.maxEventsPerSec {
		h.throttled = true
		return false
	}

	return true
}

// getCurrentTurn returns the current turn number safely
func (h *CoordinatorEventHandler) getCurrentTurn() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.currentTurn
}

// IsThrottled returns whether events are being throttled
func (h *CoordinatorEventHandler) IsThrottled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.throttled
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
