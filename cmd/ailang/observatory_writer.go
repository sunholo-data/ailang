package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/sunholo-data/ailang/internal/observatory"
)

// ObservatoryWriter captures structured tool calls and conversation data
// during agent eval runs. It implements executor.EventHandler and writes
// tool calls to observatory.db session_tools table in real-time.
//
// When nil, eval streaming works exactly as before — no behavior change.
type ObservatoryWriter struct {
	Store     *observatory.Store
	SessionID string
	StageID   string
	ChainID   string

	mu              sync.Mutex
	currentToolID   string // tracks active tool call for result correlation
	currentToolName string
}

// NewObservatoryWriter creates a writer and initializes the session in observatory.
// Returns nil if store is nil (graceful no-op).
func NewObservatoryWriter(store *observatory.Store, chainID, stageID, workspace string) *ObservatoryWriter {
	if store == nil {
		return nil
	}

	sessionID := uuid.New().String()

	ctx := context.Background()
	corr := &observatory.SessionCorrelation{
		ChainID: chainID,
		StageID: stageID,
	}
	_ = store.UpsertSessionWithCorrelation(ctx, sessionID, workspace, "", "eval-agent", corr)

	return &ObservatoryWriter{
		Store:     store,
		SessionID: sessionID,
		StageID:   stageID,
		ChainID:   chainID,
	}
}

// OnTurnStart is called at the beginning of each conversation turn.
func (w *ObservatoryWriter) OnTurnStart(turnNum int) {
	// No-op for observatory; turn data flows via stage metrics
}

// OnText is called for streaming text tokens.
func (w *ObservatoryWriter) OnText(text string) {
	// No-op for observatory; text is captured in transcript
}

// OnToolUse records a tool call start in observatory.
func (w *ObservatoryWriter) OnToolUse(toolName string, input string) {
	if w == nil || w.Store == nil {
		return
	}

	toolUseID := uuid.New().String()

	w.mu.Lock()
	w.currentToolID = toolUseID
	w.currentToolName = toolName
	w.mu.Unlock()

	ctx := context.Background()
	// Truncate input to prevent bloating the DB
	if len(input) > 2000 {
		input = input[:2000] + "...(truncated)"
	}
	_ = w.Store.InsertToolStart(ctx, w.SessionID, toolUseID, toolName, input)
}

// OnToolResult records tool call completion in observatory.
func (w *ObservatoryWriter) OnToolResult(toolName string, output string) {
	if w == nil || w.Store == nil {
		return
	}

	w.mu.Lock()
	toolUseID := w.currentToolID
	w.currentToolID = ""
	w.currentToolName = ""
	w.mu.Unlock()

	if toolUseID == "" {
		// Try to find the latest unfinished tool call
		ctx := context.Background()
		var err error
		toolUseID, err = w.Store.FindLatestUnfinishedTool(ctx, w.SessionID, toolName)
		if err != nil {
			return // Can't correlate, skip
		}
	}

	// Truncate output to prevent bloating the DB
	if len(output) > 2000 {
		output = output[:2000] + "...(truncated)"
	}

	ctx := context.Background()
	_ = w.Store.UpdateToolEnd(ctx, toolUseID, output, true)
}

// OnTurnEnd is called at the end of each conversation turn.
func (w *ObservatoryWriter) OnTurnEnd(turnNum int) {
	// No-op for observatory; turn metrics flow via stage
}

// OnError records an error event.
func (w *ObservatoryWriter) OnError(err error) {
	if w == nil || w.Store == nil {
		return
	}

	// If there's an active tool call, mark it as failed
	w.mu.Lock()
	toolUseID := w.currentToolID
	w.currentToolID = ""
	w.mu.Unlock()

	if toolUseID != "" {
		ctx := context.Background()
		_ = w.Store.UpdateToolEnd(ctx, toolUseID, fmt.Sprintf("error: %v", err), false)
	}
}

// LinkToStage updates the chain stage with this session ID.
func (w *ObservatoryWriter) LinkToStage() {
	if w == nil || w.Store == nil || w.StageID == "" {
		return
	}
	ctx := context.Background()
	_ = w.Store.UpdateStageSession(ctx, w.StageID, w.SessionID)
}
