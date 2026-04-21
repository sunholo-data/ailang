// Package server provides HTTP handlers for the collaboration hub.
package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/observatory"
)

// ClaudeHookPayload matches Claude Code's native hook JSON schema.
// HTTP hooks POST the full JSON — we store it without field subsetting.
type ClaudeHookPayload struct {
	// Common fields (always present in all hook events)
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	PermissionMode string `json:"permission_mode"`
	HookEventName  string `json:"hook_event_name"`

	// Agent fields (present in --agent mode or inside subagents)
	AgentID   string `json:"agent_id,omitempty"`
	AgentType string `json:"agent_type,omitempty"`

	// Tool fields (PreToolUse, PostToolUse, PostToolUseFailure)
	ToolName     string          `json:"tool_name,omitempty"`
	ToolInput    json.RawMessage `json:"tool_input,omitempty"`
	ToolResponse json.RawMessage `json:"tool_response,omitempty"`
	ToolUseID    string          `json:"tool_use_id,omitempty"`

	// SessionStart fields
	Source string `json:"source,omitempty"`
	Model  string `json:"model,omitempty"`

	// SubagentStart/SubagentStop fields (agent_type is reused above)

	// AILANG correlation IDs (extracted from HTTP headers, not JSON body)
	TaskID    string `json:"-"`
	ChainID   string `json:"-"`
	StageID   string `json:"-"`
	MessageID string `json:"-"`
}

// handleClaudeHooks handles POST /api/hooks/claude
// This is the unified endpoint for Claude Code HTTP hooks (type: "http").
// It receives the FULL hook JSON payload directly from Claude Code and
// routes to existing observatory/exec storage based on hook_event_name.
func (s *Server) handleClaudeHooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Hook token authentication: when configured, require Bearer token.
	// When no token is set (local mode), all requests pass through.
	if s.hookToken != "" {
		auth := r.Header.Get("Authorization")
		if auth == "" || auth != "Bearer "+s.hookToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	if s.obsBackend == nil {
		http.Error(w, "Observatory not configured", http.StatusServiceUnavailable)
		return
	}

	var payload ClaudeHookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("claude-hooks: failed to parse event: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Extract AILANG correlation IDs from HTTP headers.
	// These are set by the coordinator via allowedEnvVars + headers in hook config.
	payload.TaskID = r.Header.Get("X-Ailang-Task-Id")
	payload.ChainID = r.Header.Get("X-Ailang-Chain-Id")
	payload.StageID = r.Header.Get("X-Ailang-Stage-Id")
	payload.MessageID = r.Header.Get("X-Ailang-Message-Id")

	if payload.HookEventName == "" {
		log.Printf("claude-hooks: missing hook_event_name")
		http.Error(w, "hook_event_name required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	switch payload.HookEventName {
	case "SessionStart":
		s.handleClaudeSessionStart(ctx, payload)
	case "PreToolUse":
		s.handleClaudePreToolUse(ctx, payload)
	case "PostToolUse":
		s.handleClaudePostToolUse(ctx, payload)
	case "PostToolUseFailure":
		s.handleClaudePostToolUseFailure(ctx, payload)
	case "Stop":
		s.handleClaudeStop(ctx, payload)
	case "SubagentStart":
		s.handleClaudeSubagentEvent(ctx, payload)
	case "SubagentStop":
		s.handleClaudeSubagentEvent(ctx, payload)
	case "TaskCompleted":
		s.handleClaudeTaskCompleted(ctx, payload)
	default:
		log.Printf("claude-hooks: unhandled event type: %s", coordinator.SanitizeLog(payload.HookEventName))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleClaudeSessionStart(ctx context.Context, p ClaudeHookPayload) {
	if p.SessionID == "" || p.Cwd == "" {
		log.Printf("claude-hooks: SessionStart missing session_id or cwd")
		return
	}

	var corr *observatory.SessionCorrelation
	if p.TaskID != "" || p.ChainID != "" || p.StageID != "" || p.MessageID != "" {
		corr = &observatory.SessionCorrelation{
			TaskID:    p.TaskID,
			ChainID:   p.ChainID,
			StageID:   p.StageID,
			MessageID: p.MessageID,
		}
		log.Printf("claude-hooks: SessionStart with correlation task=%s chain=%s stage=%s msg=%s",
			p.TaskID, p.ChainID, p.StageID, p.MessageID)
	}

	// Use model field as version info (SessionStart provides it)
	version := p.Model

	if err := s.obsBackend.UpsertSessionWithCorrelation(ctx, p.SessionID, p.Cwd, version, "http-hook", corr); err != nil {
		log.Printf("claude-hooks: SessionStart upsert error: %v", err)
	} else {
		log.Printf("claude-hooks: SessionStart session=%s workspace=%s model=%s", coordinator.SanitizeLog(p.SessionID), coordinator.SanitizeLog(p.Cwd), coordinator.SanitizeLog(p.Model))
	}

	// Backfill any spans that arrived before this hook (race condition handling)
	backfilled, err := s.obsBackend.BackfillSpansWorkspace(ctx, p.SessionID, p.Cwd)
	if err != nil {
		log.Printf("claude-hooks: SessionStart backfill error: %v", err)
	} else if backfilled > 0 {
		log.Printf("claude-hooks: SessionStart backfilled %d spans with workspace", backfilled)
	}
}

func (s *Server) handleClaudePreToolUse(ctx context.Context, p ClaudeHookPayload) {
	if p.SessionID == "" || p.ToolName == "" {
		log.Printf("claude-hooks: PreToolUse missing session_id or tool_name")
		return
	}

	toolUseID := p.ToolUseID
	if toolUseID == "" {
		toolUseID = uuid.New().String()
	}

	toolInput := ""
	if p.ToolInput != nil {
		toolInput = string(p.ToolInput)
	}

	if err := s.obsBackend.InsertToolStart(ctx, p.SessionID, toolUseID, p.ToolName, toolInput); err != nil {
		log.Printf("claude-hooks: PreToolUse error: %v", err)
	} else {
		log.Printf("claude-hooks: PreToolUse session=%s tool=%s", coordinator.SanitizeLog(p.SessionID), coordinator.SanitizeLog(p.ToolName))
	}
}

func (s *Server) handleClaudePostToolUse(ctx context.Context, p ClaudeHookPayload) {
	toolUseID := p.ToolUseID
	if toolUseID == "" && p.SessionID != "" && p.ToolName != "" {
		if foundID, err := s.obsBackend.FindLatestUnfinishedTool(ctx, p.SessionID, p.ToolName); err == nil && foundID != "" {
			toolUseID = foundID
		}
	}
	if toolUseID == "" {
		toolUseID = uuid.New().String()
	}

	toolResponse := ""
	if p.ToolResponse != nil {
		toolResponse = string(p.ToolResponse)
		if len(toolResponse) > 10000 {
			toolResponse = toolResponse[:10000] + "...[truncated]"
		}
	}

	if err := s.obsBackend.UpdateToolEnd(ctx, toolUseID, toolResponse, true); err != nil {
		log.Printf("claude-hooks: PostToolUse error: %v", err)
	} else {
		log.Printf("claude-hooks: PostToolUse tool_use_id=%s tool=%s", coordinator.SanitizeLog(toolUseID), coordinator.SanitizeLog(p.ToolName))
	}
}

func (s *Server) handleClaudePostToolUseFailure(ctx context.Context, p ClaudeHookPayload) {
	toolUseID := p.ToolUseID
	if toolUseID == "" && p.SessionID != "" && p.ToolName != "" {
		if foundID, err := s.obsBackend.FindLatestUnfinishedTool(ctx, p.SessionID, p.ToolName); err == nil && foundID != "" {
			toolUseID = foundID
		}
	}
	if toolUseID == "" {
		toolUseID = uuid.New().String()
	}

	toolResponse := ""
	if p.ToolResponse != nil {
		toolResponse = string(p.ToolResponse)
		if len(toolResponse) > 10000 {
			toolResponse = toolResponse[:10000] + "...[truncated]"
		}
	}

	// Mark as failure (success=false)
	if err := s.obsBackend.UpdateToolEnd(ctx, toolUseID, toolResponse, false); err != nil {
		log.Printf("claude-hooks: PostToolUseFailure error: %v", err)
	} else {
		log.Printf("claude-hooks: PostToolUseFailure tool_use_id=%s tool=%s", coordinator.SanitizeLog(toolUseID), coordinator.SanitizeLog(p.ToolName))
	}
}

func (s *Server) handleClaudeStop(ctx context.Context, p ClaudeHookPayload) {
	if p.SessionID == "" {
		log.Printf("claude-hooks: Stop missing session_id")
		return
	}

	if err := s.obsBackend.UpdateSessionEnded(ctx, p.SessionID); err != nil {
		log.Printf("claude-hooks: Stop error: %v", err)
	} else {
		log.Printf("claude-hooks: Stop session=%s ended", coordinator.SanitizeLog(p.SessionID))
	}
}

func (s *Server) handleClaudeSubagentEvent(ctx context.Context, p ClaudeHookPayload) {
	// SubagentStart/SubagentStop events track agent spawning.
	// Store as tool calls with a synthetic tool name for observatory visibility.
	if p.SessionID == "" {
		log.Printf("claude-hooks: %s missing session_id", coordinator.SanitizeLog(p.HookEventName))
		return
	}

	agentInfo := p.AgentType
	if agentInfo == "" {
		agentInfo = p.AgentID
	}

	if p.HookEventName == "SubagentStart" {
		toolUseID := p.AgentID
		if toolUseID == "" {
			toolUseID = uuid.New().String()
		}
		input := `{"agent_type":"` + agentInfo + `","event":"SubagentStart"}`
		if err := s.obsBackend.InsertToolStart(ctx, p.SessionID, toolUseID, "Subagent:"+agentInfo, input); err != nil {
			log.Printf("claude-hooks: SubagentStart error: %v", err)
		} else {
			log.Printf("claude-hooks: SubagentStart session=%s agent=%s", coordinator.SanitizeLog(p.SessionID), coordinator.SanitizeLog(agentInfo))
		}
	} else {
		// SubagentStop — try to find matching start
		toolUseID := p.AgentID
		if toolUseID == "" {
			if foundID, err := s.obsBackend.FindLatestUnfinishedTool(ctx, p.SessionID, "Subagent:"+agentInfo); err == nil && foundID != "" {
				toolUseID = foundID
			} else {
				toolUseID = uuid.New().String()
			}
		}
		output := `{"agent_type":"` + agentInfo + `","event":"SubagentStop"}`
		if err := s.obsBackend.UpdateToolEnd(ctx, toolUseID, output, true); err != nil {
			log.Printf("claude-hooks: SubagentStop error: %v", err)
		} else {
			log.Printf("claude-hooks: SubagentStop session=%s agent=%s", coordinator.SanitizeLog(p.SessionID), coordinator.SanitizeLog(agentInfo))
		}
	}
}

func (s *Server) handleClaudeTaskCompleted(ctx context.Context, p ClaudeHookPayload) {
	// TaskCompleted fires when Claude marks a task as done.
	// Log for observability — this helps track end-to-end task duration.
	log.Printf("claude-hooks: TaskCompleted session=%s agent_type=%s task_id=%s",
		coordinator.SanitizeLog(p.SessionID), coordinator.SanitizeLog(p.AgentType), coordinator.SanitizeLog(p.TaskID))
}
