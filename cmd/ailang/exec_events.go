// OTEL streaming event handler for `ailang exec` runs.
//
// spanningEventHandler translates executor streaming callbacks (turn/text/tool/error)
// into OTEL child spans for dashboard hierarchy, NDJSON events for --stream-json
// consumers, and coordinator TaskEventRecord rows for chat history.
// Pure move out of exec.go (file-size gate); no behaviour change.
package main

import (
	"context"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// spanningEventHandler creates OTEL child spans from streaming events
// for hierarchical tracing in the dashboard while also emitting NDJSON.
// Optionally stores events to the coordinator database for chat history view.
type spanningEventHandler struct {
	ctx         context.Context
	tracer      trace.Tracer
	taskID      string
	streamJSON  bool
	currentTurn int
	turnSpan    trace.Span
	toolSpans   map[string]trace.Span // tool name -> active span
	// Event storage for chat history (optional)
	eventStore func(*coordinator.TaskEventRecord) error
}

// newSpanningEventHandler creates a handler that creates child spans.
// If eventStore is provided, events will also be stored for chat history.
func newSpanningEventHandler(ctx context.Context, taskID string, streamJSON bool, eventStore func(*coordinator.TaskEventRecord) error) *spanningEventHandler {
	return &spanningEventHandler{
		ctx:        ctx,
		tracer:     execTracer,
		taskID:     taskID,
		streamJSON: streamJSON,
		toolSpans:  make(map[string]trace.Span),
		eventStore: eventStore,
	}
}

// SetContext updates the handler's context for proper span hierarchy.
// Called by the executor after creating its span, so turn/tool spans
// become children of the executor's span rather than siblings.
func (h *spanningEventHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *spanningEventHandler) OnTurnStart(turnNum int) {
	h.currentTurn = turnNum
	// Create child span for this turn
	_, h.turnSpan = h.tracer.Start(h.ctx, "exec.turn",
		trace.WithAttributes(
			attribute.Int("turn.number", turnNum),
			attribute.String("exec.task_id", h.taskID),
		),
	)
	// Store to database for chat history
	if h.eventStore != nil {
		h.eventStore(&coordinator.TaskEventRecord{
			TaskID:     h.taskID,
			StreamType: "turn_start",
			TurnNum:    turnNum,
		})
	}
	// Also emit NDJSON for backward compatibility
	if h.streamJSON {
		emitEvent(ExecEvent{
			Type: "turn_start",
			Turn: turnNum,
		})
	}
}

func (h *spanningEventHandler) OnText(text string) {
	// Text is recorded on turn span as an event, not a separate span
	if h.turnSpan != nil {
		h.turnSpan.AddEvent("text", trace.WithAttributes(
			attribute.String("text.content", truncateString(text, 500)),
		))
	}
	// Store FULL text to database for chat history (not truncated!)
	if h.eventStore != nil {
		h.eventStore(&coordinator.TaskEventRecord{
			TaskID:     h.taskID,
			StreamType: "text",
			TurnNum:    h.currentTurn,
			Text:       text,
		})
	}
	if h.streamJSON {
		emitEvent(ExecEvent{
			Type:    "text",
			Content: text,
		})
	}
}

func (h *spanningEventHandler) OnToolUse(toolName, input string) {
	// Create child span for tool use (child of current turn)
	_, toolSpan := h.tracer.Start(h.ctx, "exec.tool_use",
		trace.WithAttributes(
			attribute.String("tool.name", toolName),
			attribute.String("tool.input", truncateString(input, 1000)),
			attribute.String("exec.task_id", h.taskID),
			attribute.Int("turn.number", h.currentTurn),
		),
	)
	h.toolSpans[toolName] = toolSpan

	// Store to database for chat history
	if h.eventStore != nil {
		h.eventStore(&coordinator.TaskEventRecord{
			TaskID:     h.taskID,
			StreamType: "tool_use",
			TurnNum:    h.currentTurn,
			ToolName:   toolName,
			ToolInput:  input,
		})
	}
	if h.streamJSON {
		emitEvent(ExecEvent{
			Type:  "tool_use",
			Tool:  toolName,
			Input: input,
		})
	}
}

func (h *spanningEventHandler) OnToolResult(toolName, output string) {
	// End the matching tool span with the result
	if span, ok := h.toolSpans[toolName]; ok {
		span.SetAttributes(attribute.String("tool.output", truncateString(output, 1000)))
		span.End()
		delete(h.toolSpans, toolName)
	}
	// Store to database for chat history
	if h.eventStore != nil {
		h.eventStore(&coordinator.TaskEventRecord{
			TaskID:     h.taskID,
			StreamType: "tool_result",
			TurnNum:    h.currentTurn,
			ToolName:   toolName,
			ToolOutput: output,
		})
	}
	if h.streamJSON {
		emitEvent(ExecEvent{
			Type:   "tool_result",
			Tool:   toolName,
			Output: output,
		})
	}
}

func (h *spanningEventHandler) OnTurnEnd(turnNum int) {
	// End any remaining tool spans (shouldn't happen normally)
	for name, span := range h.toolSpans {
		span.SetAttributes(attribute.Bool("tool.incomplete", true))
		span.End()
		delete(h.toolSpans, name)
	}
	// End turn span
	if h.turnSpan != nil {
		h.turnSpan.End()
		h.turnSpan = nil
	}
	// Store to database for chat history
	if h.eventStore != nil {
		h.eventStore(&coordinator.TaskEventRecord{
			TaskID:     h.taskID,
			StreamType: "turn_end",
			TurnNum:    turnNum,
		})
	}
	if h.streamJSON {
		emitEvent(ExecEvent{
			Type: "turn_end",
			Turn: turnNum,
		})
	}
}

func (h *spanningEventHandler) OnError(err error) {
	// Record error on turn span if active
	if h.turnSpan != nil {
		h.turnSpan.RecordError(err)
		h.turnSpan.SetStatus(codes.Error, err.Error())
	}
	// Store to database for chat history
	if h.eventStore != nil {
		h.eventStore(&coordinator.TaskEventRecord{
			TaskID:     h.taskID,
			StreamType: "error",
			TurnNum:    h.currentTurn,
			ErrorMsg:   err.Error(),
		})
	}
	if h.streamJSON {
		emitEvent(ExecEvent{
			Type:  "error",
			Error: err.Error(),
		})
	}
}
