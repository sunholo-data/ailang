package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/websocket"
)

// TelemetryParser captures stdout from a Claude process and broadcasts telemetry updates
type TelemetryParser struct {
	instanceID string
	pid        int
	wsServer   *websocket.Server
	startedAt  time.Time

	// Accumulated telemetry
	mu        sync.RWMutex
	turns     int
	tokensIn  int
	tokensOut int
	cost      float64
	status    string

	// For line-by-line parsing
	buffer []byte
}

// NewTelemetryParser creates a new telemetry parser for a process
func NewTelemetryParser(instanceID string, pid int, wsServer *websocket.Server) *TelemetryParser {
	return &TelemetryParser{
		instanceID: instanceID,
		pid:        pid,
		wsServer:   wsServer,
		startedAt:  time.Now(),
		status:     "running",
	}
}

// Write implements io.Writer to capture stdout from the process
// It parses NDJSON from Claude's stream-json output format
func (t *TelemetryParser) Write(p []byte) (n int, err error) {
	// Append to buffer
	t.buffer = append(t.buffer, p...)

	// Process complete lines
	for {
		// Find newline
		idx := -1
		for i, b := range t.buffer {
			if b == '\n' {
				idx = i
				break
			}
		}
		if idx == -1 {
			break // No complete line yet
		}

		// Extract line
		line := t.buffer[:idx]
		t.buffer = t.buffer[idx+1:]

		// Parse and process the line
		t.processLine(line)
	}

	return len(p), nil
}

// processLine parses a single NDJSON line from Claude's output
func (t *TelemetryParser) processLine(line []byte) {
	if len(line) == 0 {
		return
	}

	// Try to parse as JSON
	var event map[string]interface{}
	if err := json.Unmarshal(line, &event); err != nil {
		// Not JSON, might be regular stdout - ignore
		return
	}

	eventType, _ := event["type"].(string)

	t.mu.Lock()
	defer t.mu.Unlock()

	switch eventType {
	case "stream_event":
		// Parse streaming events for turn counting
		streamEvent, _ := event["event"].(map[string]interface{})
		if streamEvent != nil {
			streamType, _ := streamEvent["type"].(string)
			if streamType == "message_start" {
				t.turns++
				t.broadcastUpdate()
			}
		}

	case "result":
		// Final result with cost and usage
		if cost, ok := event["total_cost_usd"].(float64); ok {
			t.cost = cost
		}

		// Parse usage
		if usage, ok := event["usage"].(map[string]interface{}); ok {
			if inputTokens, ok := usage["input_tokens"].(float64); ok {
				t.tokensIn = int(inputTokens)
			}
			if outputTokens, ok := usage["output_tokens"].(float64); ok {
				t.tokensOut = int(outputTokens)
			}
		}

		// Check if completed
		if subtype, ok := event["subtype"].(string); ok {
			if subtype == "success" {
				t.status = "completed"
			} else if subtype == "error" || subtype == "timeout" {
				t.status = "error"
			}
		}

		t.broadcastUpdate()
	}
}

// broadcastUpdate sends current telemetry to all WebSocket clients
func (t *TelemetryParser) broadcastUpdate() {
	if t.wsServer == nil {
		return
	}

	t.wsServer.BroadcastTelemetry(&websocket.TelemetryEvent{
		InstanceID:  t.instanceID,
		PID:         t.pid,
		Turns:       t.turns,
		TokensIn:    t.tokensIn,
		TokensOut:   t.tokensOut,
		Cost:        t.cost,
		Status:      t.status,
		DurationSec: int(time.Since(t.startedAt).Seconds()),
	})
}

// GetTelemetry returns current accumulated telemetry
func (t *TelemetryParser) GetTelemetry() (turns, tokensIn, tokensOut int, cost float64, status string) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.turns, t.tokensIn, t.tokensOut, t.cost, t.status
}

// MarkComplete marks the process as complete with final status
func (t *TelemetryParser) MarkComplete(exitErr error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if exitErr != nil {
		t.status = "error"
	} else if t.status == "running" {
		t.status = "completed"
	}

	t.broadcastUpdate()
}

// TeeWriter wraps an io.Writer and also writes to a TelemetryParser
type TeeWriter struct {
	original io.Writer
	parser   *TelemetryParser
}

// NewTeeWriter creates a writer that writes to both original and telemetry parser
func NewTeeWriter(original io.Writer, parser *TelemetryParser) *TeeWriter {
	return &TeeWriter{
		original: original,
		parser:   parser,
	}
}

// Write writes to both the original writer and the telemetry parser
func (t *TeeWriter) Write(p []byte) (n int, err error) {
	// Write to original first
	if t.original != nil {
		n, err = t.original.Write(p)
		if err != nil {
			return n, err
		}
	} else {
		n = len(p)
	}

	// Also write to telemetry parser (best-effort, don't fail on parse errors)
	_, _ = t.parser.Write(p)

	return n, nil
}

// PipeReader creates a pipe that captures output for telemetry while still allowing normal reading
func (t *TelemetryParser) PipeReader(r io.Reader) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Bytes()
			// Process for telemetry
			t.processLine(line)
			// Write to pipe for downstream consumers
			if _, err := pw.Write(append(line, '\n')); err != nil {
				log.Printf("TelemetryParser: pipe write error: %v", err)
				return
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("TelemetryParser: scanner error: %v", err)
		}
	}()

	return pr
}

// TelemetryReportRequest is the JSON body for POST /api/telemetry
type TelemetryReportRequest struct {
	PID        int     `json:"pid"`         // Process ID (required)
	InstanceID string  `json:"instance_id"` // Optional friendly name
	Turns      int     `json:"turns"`
	TokensIn   int     `json:"tokens_in"`
	TokensOut  int     `json:"tokens_out"`
	Cost       float64 `json:"cost"`
	Status     string  `json:"status"` // running, completed, error
}

// handleTelemetryReport handles POST /api/telemetry for external processes to report their telemetry
// This allows eval suite, ailang run, and other processes to self-report their Claude usage
func (s *Server) handleTelemetryReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TelemetryReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// PID is required
	if req.PID == 0 {
		// Use current process PID if not specified
		req.PID = os.Getpid()
	}

	// Generate instance ID if not provided
	if req.InstanceID == "" {
		req.InstanceID = fmt.Sprintf("external_%d", req.PID)
	}

	// Default status
	if req.Status == "" {
		req.Status = "running"
	}

	// Store in external telemetry map
	s.externalTelemetryMu.Lock()
	s.externalTelemetry[req.PID] = &websocket.TelemetryEvent{
		InstanceID:  req.InstanceID,
		PID:         req.PID,
		Turns:       req.Turns,
		TokensIn:    req.TokensIn,
		TokensOut:   req.TokensOut,
		Cost:        req.Cost,
		Status:      req.Status,
		DurationSec: 0, // Will be calculated by monitor from process start time
	}
	s.externalTelemetryMu.Unlock()

	// Broadcast to WebSocket clients
	s.wsServer.BroadcastTelemetry(&websocket.TelemetryEvent{
		InstanceID:  req.InstanceID,
		PID:         req.PID,
		Turns:       req.Turns,
		TokensIn:    req.TokensIn,
		TokensOut:   req.TokensOut,
		Cost:        req.Cost,
		Status:      req.Status,
		DurationSec: 0,
	})

	// Record metrics when a run completes
	// This bridges telemetry reports from evals/agents to the metrics aggregation system
	if req.Status == "completed" && (req.TokensIn > 0 || req.TokensOut > 0) {
		stats := &messaging.MessageExecutionStats{
			DurationMS:   0, // Not tracked in telemetry request
			InputTokens:  req.TokensIn,
			OutputTokens: req.TokensOut,
			CostCents:    int(req.Cost * 100), // Convert dollars to cents
			FilesCreated: nil,
		}
		// Use instance_id as agent_id, create a synthetic thread for telemetry runs
		threadID := fmt.Sprintf("telemetry_%s", req.InstanceID)
		_ = s.store.RecordMetrics(threadID, req.InstanceID, stats)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetExternalTelemetry returns stored telemetry for a PID (used by monitor)
func (s *Server) GetExternalTelemetry(pid int) *websocket.TelemetryEvent {
	s.externalTelemetryMu.RLock()
	defer s.externalTelemetryMu.RUnlock()
	return s.externalTelemetry[pid]
}
