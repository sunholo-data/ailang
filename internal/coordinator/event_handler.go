// Package coordinator provides the event handler for streaming executor events.
package coordinator

import (
	"sync"
	"time"

	"github.com/sunholo/ailang/internal/websocket"
)

// EventBroadcaster is a function that broadcasts task stream events.
// This allows the event handler to be decoupled from the WebSocket server.
type EventBroadcaster func(*websocket.TaskStreamEvent)

// CoordinatorEventHandler implements executor.EventHandler to capture
// and broadcast executor events via WebSocket.
type CoordinatorEventHandler struct {
	taskID    string
	threadID  string
	broadcast EventBroadcaster

	// Rate limiting
	mu              sync.Mutex
	lastEventTime   time.Time
	eventCount      int
	maxEventsPerSec int
	throttled       bool

	// Event buffering for replay
	eventBuffer     []*websocket.TaskStreamEvent
	maxBufferSize   int
	bufferMu        sync.RWMutex

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
		maxEventsPerSec: 10,      // Rate limit: max 10 events per second
		maxBufferSize:   100,     // Keep last 100 events for replay
		eventBuffer:     make([]*websocket.TaskStreamEvent, 0, 100),
		startTime:       time.Now(),
	}
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
	// Text events can be high-frequency, apply rate limiting
	if !h.checkRateLimit() {
		return
	}

	h.emitEvent(&websocket.TaskStreamEvent{
		TaskID:     h.taskID,
		ThreadID:   h.threadID,
		StreamType: websocket.TaskStreamText,
		TurnNum:    h.getCurrentTurn(),
		Text:       truncateString(text, 2000), // Limit text size
	})
}

// OnToolUse is called when a tool is invoked
func (h *CoordinatorEventHandler) OnToolUse(toolName string, input string) {
	h.emitEvent(&websocket.TaskStreamEvent{
		TaskID:     h.taskID,
		ThreadID:   h.threadID,
		StreamType: websocket.TaskStreamToolUse,
		TurnNum:    h.getCurrentTurn(),
		ToolName:   toolName,
		ToolInput:  truncateString(input, 1000), // Limit input size
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
		ToolOutput: truncateString(output, 2000), // Limit output size
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

// emitEvent broadcasts an event and buffers it for replay
func (h *CoordinatorEventHandler) emitEvent(event *websocket.TaskStreamEvent) {
	// Buffer the event
	h.bufferMu.Lock()
	if len(h.eventBuffer) >= h.maxBufferSize {
		// Remove oldest event
		h.eventBuffer = h.eventBuffer[1:]
	}
	h.eventBuffer = append(h.eventBuffer, event)
	h.bufferMu.Unlock()

	// Broadcast if we have a broadcaster
	if h.broadcast != nil {
		h.broadcast(event)
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
