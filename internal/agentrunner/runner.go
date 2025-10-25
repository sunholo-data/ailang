package agentrunner

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sunholo/ailang/internal/agentprotocol"
)

// AgentConfig configures an agent runner.
type AgentConfig struct {
	AgentID       string
	StateDir      string
	PollInterval  time.Duration
	LeaseDuration int // seconds
	Handler       MessageHandler
	OnError       func(error)
}

// MessageHandler processes a message and returns a response.
type MessageHandler interface {
	// HandleMessage processes the message and returns a response payload.
	// If error is returned, the message will be retried.
	HandleMessage(msg *agentprotocol.Envelope) (map[string]interface{}, error)
}

// Runner is the main agent polling loop.
type Runner struct {
	config *AgentConfig
	db     *agentprotocol.DB
	writer *agentprotocol.MessageWriter
	reader *agentprotocol.MessageReader
	stop   chan struct{}
}

// NewRunner creates a new agent runner.
func NewRunner(config *AgentConfig) (*Runner, error) {
	// Open database
	db, err := agentprotocol.NewDB(config.StateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Register agent
	err = db.RegisterAgent(&agentprotocol.AgentInfo{
		AgentID:       config.AgentID,
		InboxPath:     fmt.Sprintf("%s/messages/%s", config.StateDir, config.AgentID),
		Status:        "active",
		ProtocolCaps:  `["v1.0"]`,
		LastHeartbeat: time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	return &Runner{
		config: config,
		db:     db,
		writer: agentprotocol.NewMessageWriter(config.StateDir),
		reader: agentprotocol.NewMessageReader(config.StateDir),
		stop:   make(chan struct{}),
	}, nil
}

// Run starts the agent polling loop. Blocks until Stop() is called or signal received.
func (r *Runner) Run() error {
	log.Printf("[%s] Agent runner started (poll interval: %v)", r.config.AgentID, r.config.PollInterval)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(r.config.PollInterval)
	defer ticker.Stop()

	// Initial scan
	if err := r.poll(); err != nil {
		log.Printf("[%s] Poll error: %v", r.config.AgentID, err)
	}

	for {
		select {
		case <-ticker.C:
			if err := r.poll(); err != nil {
				log.Printf("[%s] Poll error: %v", r.config.AgentID, err)
			}
		case <-sigChan:
			log.Printf("[%s] Received shutdown signal", r.config.AgentID)
			r.Stop()
			return nil
		case <-r.stop:
			log.Printf("[%s] Agent runner stopped", r.config.AgentID)
			return nil
		}
	}
}

// RunOnce processes pending messages once and exits.
func (r *Runner) RunOnce() error {
	log.Printf("[%s] Running once", r.config.AgentID)
	return r.poll()
}

// poll scans for messages and processes them.
func (r *Runner) poll() error {
	// Update heartbeat
	if err := r.db.UpdateAgentStatus(r.config.AgentID, "active"); err != nil {
		log.Printf("[%s] WARNING: Failed to update agent status: %v", r.config.AgentID, err)
	}

	// Scan for pending messages
	pending, err := r.reader.ScanPendingMessages(r.config.AgentID)
	if err != nil {
		return r.handleError(fmt.Errorf("scan failed: %w", err))
	}

	if len(pending) == 0 {
		log.Printf("[%s] No pending messages", r.config.AgentID)
		return nil
	}

	log.Printf("[%s] Found %d pending message(s)", r.config.AgentID, len(pending))

	for _, msgPath := range pending {
		if err := r.processMessage(msgPath); err != nil {
			if handleErr := r.handleError(fmt.Errorf("process message failed: %w", err)); handleErr != nil {
				log.Printf("[%s] WARNING: Failed to handle error: %v", r.config.AgentID, handleErr)
			}
			// Continue to next message (don't let one failure block others)
		}
	}

	return nil
}

// processMessage handles a single message.
func (r *Runner) processMessage(msgPath string) error {
	// Read message
	msg, err := r.reader.ReadMessage(msgPath)
	if err != nil {
		return fmt.Errorf("read message failed: %w", err)
	}

	if msg == nil {
		// Already processed (idempotency)
		return nil
	}

	log.Printf("[%s] Processing message %s from %s", r.config.AgentID, msg.MessageID, msg.FromAgent)

	// Check if already in database (cross-process deduplication)
	exists, err := r.db.MessageExists(msg.MessageID)
	if err != nil {
		return fmt.Errorf("check exists failed: %w", err)
	}
	if exists {
		log.Printf("[%s] Message %s already processed (database deduplication)", r.config.AgentID, msg.MessageID)
		return nil
	}

	// Acquire lease (crash safety)
	acquired, err := r.db.AcquireLease(msgPath, r.config.AgentID, r.config.LeaseDuration)
	if err != nil {
		return fmt.Errorf("acquire lease failed: %w", err)
	}
	if !acquired {
		log.Printf("[%s] Message %s already locked by another agent", r.config.AgentID, msg.MessageID)
		return nil
	}

	// Ensure lease is released on exit
	defer func() {
		if err := r.db.ReleaseLease(msgPath); err != nil {
			log.Printf("[%s] WARNING: Failed to release lease: %v", r.config.AgentID, err)
		}
	}()

	// Record message in database
	err = r.db.RecordMessage(&agentprotocol.MessageRecord{
		MessageID:     msg.MessageID,
		CorrelationID: msg.CorrelationID,
		TraceID:       msg.TraceID,
		FromAgent:     msg.FromAgent,
		ToAgent:       msg.ToAgent,
		MessageType:   msg.MessageType,
		Status:        "processing",
		CreatedAt:     time.Now().UTC(),
		RetryCount:    0,
	})
	if err != nil {
		return fmt.Errorf("record message failed: %w", err)
	}

	// Log event
	if logErr := r.db.LogEvent(r.config.AgentID, msg.MessageID, "message_received", fmt.Sprintf(`{"from": "%s"}`, msg.FromAgent)); logErr != nil {
		log.Printf("[%s] WARNING: Failed to log event: %v", r.config.AgentID, logErr)
	}

	// Process message with handler
	startTime := time.Now()
	responsePayload, err := r.config.Handler.HandleMessage(msg)
	duration := time.Since(startTime)

	if err != nil {
		// Mark as failed
		if updateErr := r.db.UpdateMessageStatus(msg.MessageID, "failed"); updateErr != nil {
			log.Printf("[%s] WARNING: Failed to update message status: %v", r.config.AgentID, updateErr)
		}
		if logErr := r.db.LogEvent(r.config.AgentID, msg.MessageID, "processing_failed", fmt.Sprintf(`{"error": "%s"}`, err.Error())); logErr != nil {
			log.Printf("[%s] WARNING: Failed to log event: %v", r.config.AgentID, logErr)
		}
		return fmt.Errorf("handler failed: %w", err)
	}

	// Record metrics
	if metricErr := r.db.RecordMetric(r.config.AgentID, "processing_latency_ms", float64(duration.Milliseconds())); metricErr != nil {
		log.Printf("[%s] WARNING: Failed to record metric: %v", r.config.AgentID, metricErr)
	}

	// Mark as completed
	if markErr := r.db.MarkMessageProcessed(msg.MessageID); markErr != nil {
		log.Printf("[%s] WARNING: Failed to mark message as processed: %v", r.config.AgentID, markErr)
	}
	if logErr := r.db.LogEvent(r.config.AgentID, msg.MessageID, "processing_completed", ""); logErr != nil {
		log.Printf("[%s] WARNING: Failed to log event: %v", r.config.AgentID, logErr)
	}

	// Send response (if request type and response payload provided)
	if msg.MessageType == "request" && responsePayload != nil {
		if err := r.sendResponse(msg, responsePayload); err != nil {
			return fmt.Errorf("send response failed: %w", err)
		}
	}

	log.Printf("[%s] Completed message %s in %v", r.config.AgentID, msg.MessageID, duration)

	return nil
}

// sendResponse sends a response message back to the original sender.
func (r *Runner) sendResponse(originalMsg *agentprotocol.Envelope, payload map[string]interface{}) error {
	responseEnv := &agentprotocol.Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       agentprotocol.GenerateMessageID(),
		CorrelationID:   originalMsg.CorrelationID,
		TraceID:         originalMsg.TraceID,
		ParentMessageID: &originalMsg.MessageID,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       r.config.AgentID,
		ToAgent:         originalMsg.FromAgent, // Send back to sender
		MessageType:     "response",
		PayloadSchema:   "generic.v1",
		Payload:         payload,
		DeclaredEffects: []string{},
	}

	_, err := r.writer.WriteMessage(responseEnv)
	if err != nil {
		return err
	}

	// Record response in database
	if recErr := r.db.RecordMessage(&agentprotocol.MessageRecord{
		MessageID:     responseEnv.MessageID,
		CorrelationID: responseEnv.CorrelationID,
		TraceID:       responseEnv.TraceID,
		FromAgent:     responseEnv.FromAgent,
		ToAgent:       responseEnv.ToAgent,
		MessageType:   responseEnv.MessageType,
		Status:        "pending",
		CreatedAt:     time.Now().UTC(),
		RetryCount:    0,
	}); recErr != nil {
		log.Printf("[%s] WARNING: Failed to record response message: %v", r.config.AgentID, recErr)
	}

	log.Printf("[%s] Sent response %s to %s", r.config.AgentID, responseEnv.MessageID, responseEnv.ToAgent)

	return nil
}

// Stop gracefully stops the agent runner.
func (r *Runner) Stop() {
	log.Printf("[%s] Stopping agent runner...", r.config.AgentID)

	// Update status to idle
	if err := r.db.UpdateAgentStatus(r.config.AgentID, "idle"); err != nil {
		log.Printf("[%s] WARNING: Failed to update agent status to idle: %v", r.config.AgentID, err)
	}

	// Signal stop
	close(r.stop)

	// Close database
	if err := r.db.Close(); err != nil {
		log.Printf("[%s] WARNING: Failed to close database: %v", r.config.AgentID, err)
	}
}

// handleError handles errors with the configured error handler.
func (r *Runner) handleError(err error) error {
	if r.config.OnError != nil {
		r.config.OnError(err)
	} else {
		log.Printf("[%s] ERROR: %v", r.config.AgentID, err)
	}
	return err
}
