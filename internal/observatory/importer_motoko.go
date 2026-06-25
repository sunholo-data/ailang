package observatory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// motokoNS is the stable namespace for deterministic motoko-import IDs, so re-importing
// the same run is idempotent (same chain/stage/session IDs -> delete-and-replace).
var motokoNS = uuid.MustParse("a1c0c0de-0000-4000-8000-000000000001")

var motokoStampRe = regexp.MustCompile(`(\d{8})-(\d{6})`)

// motokoEvent is the union of fields we read from a .motoko/logfile/*.jsonl event stream.
type motokoEvent struct {
	Type         string `json:"type"`
	Step         *int   `json:"step"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	Model        string `json:"model"`
	// tool_calls is overloaded across event types: an ARRAY in native_tool_calls events but
	// an INT count in thinking events. Keep it raw and decode only where it's an array.
	ToolCalls json.RawMessage `json:"tool_calls"`
	Results   []motokoResult  `json:"results"`
	// run_summary fields
	StepsExecuted int     `json:"steps_executed"`
	Error         string  `json:"error"`
	DurationMs    int64   `json:"duration_ms"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
}

type motokoCall struct {
	ID        string          `json:"id"`
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
}

type motokoResult struct {
	ToolCallID string          `json:"tool_call_id"`
	ExitCode   int             `json:"exit_code"`
	Stdout     string          `json:"stdout"`
	Stderr     string          `json:"stderr"`
	Payload    json.RawMessage `json:"payload"`
}

// Anthropic-style content blocks, the shape the chains chat renderer parses.
type motokoBlock struct {
	Type       string            `json:"type"`
	Text       string            `json:"text,omitempty"`
	ToolUse    *motokoToolUse    `json:"tool_use,omitempty"`
	ToolResult *motokoToolResult `json:"tool_result,omitempty"`
}
type motokoToolUse struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}
type motokoToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
}

// MotokoImportResult summarizes a completed import.
type MotokoImportResult struct {
	ChainID      string
	SessionLabel string
	Status       string
	FinishReason string
	Error        string
	Steps        int
	ToolCalls    int
	TokensIn     int
	TokensOut    int
	PeakInput    int
}

// ImportMotokoSession reads a motoko run log (.motoko/logfile/session_*.jsonl) and writes it
// into the observatory DB as a first-class chain (chain + stage + session + per-turn
// chat_messages), so `ailang chains view/chat/tree/diagnose` work on it. Idempotent.
func (s *Store) ImportMotokoSession(ctx context.Context, logPath string) (*MotokoImportResult, error) {
	raw, err := os.ReadFile(logPath)
	if err != nil {
		return nil, fmt.Errorf("read motoko log: %w", err)
	}

	var events []motokoEvent
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var ev motokoEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue // skip malformed lines rather than abort the import
		}
		events = append(events, ev)
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("no events in %s", logPath)
	}

	label := strings.TrimSuffix(filepath.Base(logPath), ".jsonl")
	started := motokoStartedAt(label)

	// Roll up per-step events.
	var summary *motokoEvent
	thinkByStep := map[int]motokoEvent{}
	callsByStep := map[int][]motokoCall{}
	resByStep := map[int][]motokoResult{}
	maxStep := 0
	for i := range events {
		ev := events[i]
		switch ev.Type {
		case "run_summary":
			cp := ev
			summary = &cp
		case "thinking":
			if ev.Step != nil {
				thinkByStep[*ev.Step] = ev
				if *ev.Step > maxStep {
					maxStep = *ev.Step
				}
			}
		case "native_tool_calls":
			if ev.Step != nil {
				var calls []motokoCall
				_ = json.Unmarshal(ev.ToolCalls, &calls)
				callsByStep[*ev.Step] = calls
			}
		case "native_tool_results":
			if ev.Step != nil {
				resByStep[*ev.Step] = ev.Results
			}
		}
	}

	res := &MotokoImportResult{SessionLabel: label}
	res.FinishReason = "unknown"
	if summary != nil {
		res.FinishReason = summary.FinishReason
		res.Error = summary.Error
		res.Steps = summary.StepsExecuted
	}
	if res.Steps == 0 {
		res.Steps = maxStep + 1
	}
	res.Status = "completed"
	if res.FinishReason != "stop" || res.Error != "" {
		res.Status = "failed"
	}
	var cost float64
	var model string
	var durationMs int64
	if summary != nil {
		cost = summary.TotalCostUSD
		durationMs = summary.DurationMs
	}
	for _, t := range thinkByStep {
		res.TokensIn += t.InputTokens
		res.TokensOut += t.OutputTokens
		if t.InputTokens > res.PeakInput {
			res.PeakInput = t.InputTokens
		}
		if model == "" && t.Model != "" {
			model = t.Model
		}
	}
	for _, c := range callsByStep {
		res.ToolCalls += len(c)
	}

	chainID := uuid.NewSHA1(motokoNS, []byte(label)).String()
	stageID := uuid.NewSHA1(motokoNS, []byte(label+":stage")).String()
	sessID := uuid.NewSHA1(motokoNS, []byte(label+":session")).String()
	res.ChainID = chainID
	workdir := filepath.Dir(logPath)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Idempotent: clear any prior import of this run.
	for _, stmt := range []struct {
		sql string
		arg string
	}{
		{`DELETE FROM chat_messages WHERE session_id = ?`, sessID},
		{`DELETE FROM chain_stages WHERE id = ?`, stageID},
		{`DELETE FROM sessions WHERE session_id = ?`, sessID},
		{`DELETE FROM execution_chains WHERE id = ?`, chainID},
	} {
		if _, err := tx.ExecContext(ctx, stmt.sql, stmt.arg); err != nil {
			return nil, fmt.Errorf("clear prior import: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO execution_chains
		(id, source_type, source_ref, github_repo, github_issue_number, status, current_stage,
		 workspace_id, workspace_path, created_at, updated_at, completed_at,
		 total_cost, total_tokens, total_turns, stages_completed)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		chainID, "motoko", label, "", 0, res.Status, 1, "", workdir, started, started, started,
		cost, res.TokensIn+res.TokensOut, res.Steps, 1); err != nil {
		return nil, fmt.Errorf("insert chain: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO chain_stages
		(id, chain_id, stage_number, agent_id, provider, session_id, status, started_at,
		 completed_at, error_message, cost, tokens_in, tokens_out, turns, tool_calls, duration_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		stageID, chainID, 1, "motoko-agent", "ollama", sessID, res.Status, started, started,
		res.Error, cost, res.TokensIn, res.TokensOut, res.Steps, res.ToolCalls, durationMs); err != nil {
		return nil, fmt.Errorf("insert stage: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sessions (session_id, workspace, source, started_at, turn_count, chain_id, stage_id)
		VALUES (?,?,?,?,?,?,?)`,
		sessID, workdir, "motoko", started, res.Steps, chainID, stageID); err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	// Per step: an assistant turn (thinking text + tool_use blocks) then, if any, a user turn
	// carrying the tool_result blocks.
	steps := sortedSteps(thinkByStep, callsByStep)
	turn := 0
	for _, st := range steps {
		t := thinkByStep[st]
		text := strings.TrimSpace(t.Text)
		var blocks []motokoBlock
		if text != "" {
			blocks = append(blocks, motokoBlock{Type: "text", Text: text})
		}
		var summaryParts []string
		if text != "" {
			summaryParts = append(summaryParts, text)
		}
		for _, c := range callsByStep[st] {
			input := c.Arguments
			if len(input) == 0 {
				input = json.RawMessage("{}")
			}
			blocks = append(blocks, motokoBlock{Type: "tool_use",
				ToolUse: &motokoToolUse{ID: c.ID, Name: c.Tool, Input: input}})
			summaryParts = append(summaryParts, motokoToolLine(c))
		}
		if t.FinishReason != "" && t.FinishReason != "tool_calls" {
			blocks = append(blocks, motokoBlock{Type: "text",
				Text: fmt.Sprintf("[finish_reason=%s]", t.FinishReason)})
		}
		if len(blocks) == 0 {
			blocks = append(blocks, motokoBlock{Type: "text", Text: fmt.Sprintf("(step %d)", st)})
		}
		blocksJSON, _ := json.Marshal(blocks)
		summaryText := strings.TrimSpace(strings.Join(summaryParts, " "))
		if summaryText == "" {
			summaryText = fmt.Sprintf("(step %d)", st)
		}
		turn++
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO chat_messages
			(id, session_id, turn_number, role, content_text, content_thinking, content_json,
			 tokens_in, tokens_out, model, timestamp)
			VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			uuid.New().String(), sessID, turn, "assistant", summaryText, text, string(blocksJSON),
			t.InputTokens, t.OutputTokens, modelOr(t.Model, model), started); err != nil {
			return nil, fmt.Errorf("insert chat (assistant): %w", err)
		}

		results := resByStep[st]
		if len(results) > 0 {
			var rblocks []motokoBlock
			for _, r := range results {
				content := string(r.Payload)
				if content == "" {
					content = r.Stdout + r.Stderr
				}
				if len(content) > 1500 {
					content = content[:1500]
				}
				rblocks = append(rblocks, motokoBlock{Type: "tool_result",
					ToolResult: &motokoToolResult{ToolUseID: r.ToolCallID, Content: content,
						IsError: r.ExitCode != 0}})
			}
			rJSON, _ := json.Marshal(rblocks)
			turn++
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO chat_messages
				(id, session_id, turn_number, role, content_text, content_json,
				 tokens_in, tokens_out, model, timestamp)
				VALUES (?,?,?,?,?,?,?,?,?,?)`,
				uuid.New().String(), sessID, turn, "user", "tool results", string(rJSON),
				0, 0, modelOr(t.Model, model), started); err != nil {
				return nil, fmt.Errorf("insert chat (results): %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return res, nil
}

func motokoStartedAt(label string) time.Time {
	if m := motokoStampRe.FindStringSubmatch(label); m != nil {
		if t, err := time.ParseInLocation("20060102 150405", m[1]+" "+m[2], time.Local); err == nil {
			return t
		}
	}
	return time.Now()
}

func motokoToolLine(c motokoCall) string {
	var args map[string]interface{}
	_ = json.Unmarshal(c.Arguments, &args)
	tgt := ""
	for _, k := range []string{"path", "file", "cmd", "command"} {
		if v, ok := args[k]; ok {
			tgt = fmt.Sprintf("%v", v)
			break
		}
	}
	if len(tgt) > 90 {
		tgt = tgt[:90]
	}
	return fmt.Sprintf("→ %s(%s)", c.Tool, tgt)
}

func modelOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func sortedSteps(a map[int]motokoEvent, b map[int][]motokoCall) []int {
	seen := map[int]bool{}
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	out := make([]int, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	// simple insertion sort (step counts are small)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}
