package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"go.opentelemetry.io/otel/attribute"
)

// emitEventSpan creates an OTEL span for eval events so they appear in the ExecHierarchy.
// This bridges inbox messages (collaboration.db) with spans (observatory.db) by creating
// a span whenever we create an inbox message. The span name follows the pattern
// "eval.event.{eventType}" and includes attributes linking to the task/message.
func emitEventSpan(ctx context.Context, eventType, taskID string, msg *messaging.InboxMessage) {
	_, span := evalTracer.Start(ctx, "eval.event."+eventType)
	span.SetAttributes(
		attribute.String("event.type", eventType),
		attribute.String("event.title", msg.Title),
		attribute.String("event.message_id", msg.MessageID),
		attribute.String("ailang.task_id", taskID),
		attribute.String("ailang.category", "eval"),
	)
	// End immediately (event is instantaneous)
	span.End()
}

// broadcastEvalEvent sends an eval event to the dashboard server for real-time updates.
// This is non-blocking and best-effort - failures are silently ignored.
func broadcastEvalEvent(msg *messaging.InboxMessage) {
	// Create a simplified event payload for the dashboard
	event := map[string]interface{}{
		"type":      "eval_event",
		"message":   msg.Title,
		"category":  msg.Category,
		"from":      msg.FromAgent,
		"task_id":   msg.CorrelationID,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	// POST to dashboard server (non-blocking, best-effort)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(
		"http://127.0.0.1:1957/api/coordinator/events",
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return // Silently ignore - dashboard may not be running
	}
	defer resp.Body.Close()
}
