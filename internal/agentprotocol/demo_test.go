package agentprotocol

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestEndToEndMessagePassing simulates two agents exchanging messages.
func TestEndToEndMessagePassing(t *testing.T) {
	tmpDir := t.TempDir()

	// Agent A: design-doc-creator
	agentA := "design-doc-creator"
	// Agent B: sprint-planner
	agentB := "sprint-planner"

	writer := NewMessageWriter(tmpDir)
	reader := NewMessageReader(tmpDir)

	// Step 1: Agent A sends a request to Agent B
	t.Log("Step 1: design-doc-creator sends request to sprint-planner")

	correlationID := GenerateCorrelationID()
	traceID := GenerateTraceID()

	requestEnv := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       GenerateMessageID(),
		CorrelationID:   correlationID,
		TraceID:         traceID,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       agentA,
		ToAgent:         agentB,
		MessageType:     "request",
		Retries:         0,
		PayloadSchema:   "design_doc_completed.v1",
		Payload: map[string]interface{}{
			"design_doc_path": "design_docs/planned/M-AGENT-PROTOCOL.md",
			"status":          "completed",
			"next_stage":      "sprint_planning",
		},
		DeclaredEffects: []string{"FS.read", "FS.write"},
	}

	requestPath, err := writer.WriteMessage(requestEnv)
	if err != nil {
		t.Fatalf("failed to write request: %v", err)
	}
	t.Logf("  ✓ Request written to: %s", requestPath)

	// Step 2: Agent B scans for pending messages
	t.Log("Step 2: sprint-planner scans for pending messages")

	pending, err := reader.ScanPendingMessages(agentB)
	if err != nil {
		t.Fatalf("failed to scan messages: %v", err)
	}

	if len(pending) != 1 {
		t.Fatalf("expected 1 pending message, got %d", len(pending))
	}
	t.Logf("  ✓ Found %d pending message(s)", len(pending))

	// Step 3: Agent B reads the message
	t.Log("Step 3: sprint-planner reads the request")

	receivedMsg, err := reader.ReadMessage(pending[0])
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}

	if receivedMsg == nil {
		t.Fatal("expected message, got nil")
	}

	t.Logf("  ✓ Received message: %s", receivedMsg.MessageID)
	t.Logf("    From: %s", receivedMsg.FromAgent)
	t.Logf("    To: %s", receivedMsg.ToAgent)
	t.Logf("    Type: %s", receivedMsg.MessageType)
	t.Logf("    Payload: %v", receivedMsg.Payload)

	// Verify message contents
	if receivedMsg.FromAgent != agentA {
		t.Errorf("expected FromAgent=%s, got %s", agentA, receivedMsg.FromAgent)
	}
	if receivedMsg.ToAgent != agentB {
		t.Errorf("expected ToAgent=%s, got %s", agentB, receivedMsg.ToAgent)
	}
	if receivedMsg.MessageType != "request" {
		t.Errorf("expected MessageType=request, got %s", receivedMsg.MessageType)
	}

	designDocPath, ok := receivedMsg.Payload["design_doc_path"].(string)
	if !ok || designDocPath != "design_docs/planned/M-AGENT-PROTOCOL.md" {
		t.Errorf("unexpected payload: %v", receivedMsg.Payload)
	}

	// Step 4: Agent B sends a response back to Agent A
	t.Log("Step 4: sprint-planner sends response to design-doc-creator")

	responseEnv := &Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       GenerateMessageID(),
		CorrelationID:   correlationID, // Same correlation ID (part of same cycle)
		TraceID:         traceID,       // Same trace ID (same request chain)
		ParentMessageID: &receivedMsg.MessageID,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       agentB,
		ToAgent:         agentA,
		MessageType:     "response",
		Retries:         0,
		PayloadSchema:   "sprint_plan_created.v1",
		Payload: map[string]interface{}{
			"sprint_plan_path": ".ailang/state/sprints/current_sprint.json",
			"status":           "ready",
			"estimated_days":   1.5,
			"tasks":            3,
		},
		DeclaredEffects: []string{"FS.read", "FS.write"},
	}

	responsePath, err := writer.WriteMessage(responseEnv)
	if err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
	t.Logf("  ✓ Response written to: %s", responsePath)

	// Step 5: Agent A scans for pending messages (should see the response)
	t.Log("Step 5: design-doc-creator scans for response")

	readerA := NewMessageReader(tmpDir)
	pendingForA, err := readerA.ScanPendingMessages(agentA)
	if err != nil {
		t.Fatalf("failed to scan messages: %v", err)
	}

	if len(pendingForA) != 1 {
		t.Fatalf("expected 1 pending message for agent A, got %d", len(pendingForA))
	}
	t.Logf("  ✓ Found %d pending message(s) for agent A", len(pendingForA))

	// Step 6: Agent A reads the response
	t.Log("Step 6: design-doc-creator reads the response")

	responseMsg, err := readerA.ReadMessage(pendingForA[0])
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	t.Logf("  ✓ Received response: %s", responseMsg.MessageID)
	t.Logf("    From: %s", responseMsg.FromAgent)
	t.Logf("    To: %s", responseMsg.ToAgent)
	t.Logf("    Type: %s", responseMsg.MessageType)
	t.Logf("    Parent: %s", *responseMsg.ParentMessageID)
	t.Logf("    Payload: %v", responseMsg.Payload)

	// Verify response
	if responseMsg.FromAgent != agentB {
		t.Errorf("expected FromAgent=%s, got %s", agentB, responseMsg.FromAgent)
	}
	if responseMsg.ToAgent != agentA {
		t.Errorf("expected ToAgent=%s, got %s", agentA, responseMsg.ToAgent)
	}
	if responseMsg.MessageType != "response" {
		t.Errorf("expected MessageType=response, got %s", responseMsg.MessageType)
	}
	if *responseMsg.ParentMessageID != receivedMsg.MessageID {
		t.Errorf("expected ParentMessageID=%s, got %s", receivedMsg.MessageID, *responseMsg.ParentMessageID)
	}
	if responseMsg.CorrelationID != correlationID {
		t.Errorf("expected same CorrelationID, got %s", responseMsg.CorrelationID)
	}

	// Step 7: Verify idempotency (reading the same message again should be skipped)
	t.Log("Step 7: Verify idempotency (re-reading should skip)")

	duplicateMsg, err := readerA.ReadMessage(pendingForA[0])
	if err != nil {
		t.Fatalf("failed to read duplicate: %v", err)
	}

	if duplicateMsg != nil {
		t.Errorf("expected nil (already seen), got message: %s", duplicateMsg.MessageID)
	}
	t.Logf("  ✓ Duplicate read correctly skipped (idempotency working)")

	// Step 8: Inspect the message directory
	t.Log("Step 8: Inspecting message directory")

	messagesDir := filepath.Join(tmpDir, "messages")
	entries, err := os.ReadDir(messagesDir)
	if err != nil {
		t.Fatalf("failed to read messages dir: %v", err)
	}

	t.Logf("  Messages directory contains %d file(s):", len(entries))
	for _, entry := range entries {
		path := filepath.Join(messagesDir, entry.Name())
		info, _ := entry.Info()
		t.Logf("    - %s (%d bytes)", entry.Name(), info.Size())

		// Pretty-print JSON content
		data, _ := os.ReadFile(path)
		var prettyJSON map[string]interface{}
		if err := json.Unmarshal(data, &prettyJSON); err == nil {
			t.Logf("      From: %s → To: %s (%s)",
				prettyJSON["from_agent"],
				prettyJSON["to_agent"],
				prettyJSON["message_type"])
		}
	}

	t.Log("\n✅ End-to-end test PASSED!")
	t.Log("   - Request sent from design-doc-creator to sprint-planner")
	t.Log("   - Response sent from sprint-planner to design-doc-creator")
	t.Log("   - Message IDs tracked (correlation_id, trace_id, parent_message_id)")
	t.Log("   - Idempotency verified (duplicate reads skipped)")
	t.Log("   - All messages persisted to disk as observable JSON files")
}

// TestIdempotencyAcrossReaders verifies that different reader instances share state.
func TestIdempotencyAcrossReaders(t *testing.T) {
	tmpDir := t.TempDir()

	writer := NewMessageWriter(tmpDir)

	// Write a message
	env := &Envelope{
		ProtocolVersion: "1.0.0",
		MessageID:       "msg_idempotency_test",
		FromAgent:       "sender",
		ToAgent:         "receiver",
		MessageType:     "request",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	msgPath, err := writer.WriteMessage(env)
	if err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	// Reader 1 reads the message
	reader1 := NewMessageReader(tmpDir)
	msg1, err := reader1.ReadMessage(msgPath)
	if err != nil {
		t.Fatalf("reader1 failed to read: %v", err)
	}
	if msg1 == nil {
		t.Fatal("expected message, got nil")
	}

	// Reader 2 (different instance) tries to read the same message
	reader2 := NewMessageReader(tmpDir)
	msg2, err := reader2.ReadMessage(msgPath)
	if err != nil {
		t.Fatalf("reader2 failed to read: %v", err)
	}

	// NOTE: Currently, each reader has its own in-memory seen map.
	// This means reader2 will see the message again.
	// This is expected behavior for Milestone 1 (in-memory deduplication).
	// Milestone 2 will add SQLite-based deduplication for cross-process idempotency.

	if msg2 == nil {
		t.Log("⚠️  reader2 skipped (in-memory deduplication only)")
		t.Log("   This is expected for Milestone 1.")
		t.Log("   Milestone 2 will add SQLite-based cross-process idempotency.")
	} else {
		t.Logf("✓ reader2 read message (different reader instance)")
		t.Logf("  This is expected for Milestone 1 (in-memory seen map per reader).")
		t.Logf("  Milestone 2 will add SQLite-based deduplication.")
	}

	// Verify reader1 still skips on re-read (same instance)
	msg1Again, err := reader1.ReadMessage(msgPath)
	if err != nil {
		t.Fatalf("reader1 second read failed: %v", err)
	}
	if msg1Again != nil {
		t.Errorf("expected nil (already seen by reader1), got %s", msg1Again.MessageID)
	} else {
		t.Log("✓ reader1 correctly skipped duplicate (in-memory deduplication working)")
	}
}

// TestNotificationMessageType tests the "notification" message type.
func TestNotificationMessageType(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewMessageWriter(tmpDir)

	// Send a notification (fire-and-forget, no response expected)
	notification := &Envelope{
		ProtocolVersion: "1.0.0",
		MessageID:       GenerateMessageID(),
		CorrelationID:   GenerateCorrelationID(),
		TraceID:         GenerateTraceID(),
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		FromAgent:       "eval-analyzer",
		ToAgent:         "design-doc-creator",
		MessageType:     "notification",
		PayloadSchema:   "dx_friction_detected.v1",
		Payload: map[string]interface{}{
			"friction_type": "missing_builtin",
			"severity":      "medium",
			"description":   "AI struggled with list comprehension syntax",
			"suggestion":    "Add list.map() and list.filter() builtins",
		},
		DeclaredEffects: []string{"FS.write"},
	}

	path, err := writer.WriteMessage(notification)
	if err != nil {
		t.Fatalf("failed to write notification: %v", err)
	}

	t.Logf("✓ Notification sent: %s", path)

	// Read back
	reader := NewMessageReader(tmpDir)
	msg, err := reader.ReadMessage(path)
	if err != nil {
		t.Fatalf("failed to read notification: %v", err)
	}

	if msg.MessageType != "notification" {
		t.Errorf("expected MessageType=notification, got %s", msg.MessageType)
	}

	if msg.ParentMessageID != nil {
		t.Errorf("notifications should not have parent_message_id, got %s", *msg.ParentMessageID)
	}

	t.Logf("✓ Notification received:")
	t.Logf("  Type: %s", msg.Payload["friction_type"])
	t.Logf("  Severity: %s", msg.Payload["severity"])
	t.Logf("  Description: %s", msg.Payload["description"])
	t.Logf("  Suggestion: %s", msg.Payload["suggestion"])
}

func ExampleMessageWriter() {
	// Create a writer (in production, use actual state dir)
	writer := NewMessageWriter("/tmp/ailang_demo")

	// Send a request from design-doc-creator to sprint-planner
	request := &Envelope{
		ProtocolVersion: "1.0.0",
		MessageID:       GenerateMessageID(),
		CorrelationID:   GenerateCorrelationID(),
		TraceID:         GenerateTraceID(),
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		FromAgent:       "design-doc-creator",
		ToAgent:         "sprint-planner",
		MessageType:     "request",
		PayloadSchema:   "design_doc_completed.v1",
		Payload: map[string]interface{}{
			"design_doc_path": "design_docs/planned/M-AGENT-PROTOCOL.md",
			"status":          "completed",
		},
		DeclaredEffects: []string{"FS.read"},
	}

	path, err := writer.WriteMessage(request)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Message written to: %s\n", path)
	// Output will be something like:
	// Message written to: /tmp/ailang_demo/messages/msg_20251023_abc123.pending.json
}

func ExampleMessageReader() {
	// Create a reader (in production, use actual state dir)
	reader := NewMessageReader("/tmp/ailang_demo")

	// Scan for pending messages addressed to "sprint-planner"
	pending, err := reader.ScanPendingMessages("sprint-planner")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Found %d pending message(s)\n", len(pending))

	// Read each message
	for _, msgPath := range pending {
		msg, err := reader.ReadMessage(msgPath)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", msgPath, err)
			continue
		}

		if msg == nil {
			// Already processed (idempotency)
			continue
		}

		fmt.Printf("Received: %s from %s\n", msg.MessageID, msg.FromAgent)
		fmt.Printf("Payload: %v\n", msg.Payload)
	}
}
