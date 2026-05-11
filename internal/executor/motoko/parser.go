package motoko

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// motokoStateDir is the per-repo state subdirectory motoko writes to:
// session JSONL, profile config, etc. Used by findSessionJSONL when
// searching multiple candidate roots (workspace, MOTOKO_REPO, discovered).
const motokoStateDir = ".motoko"

// Schema-v1 events emitted by motoko's session JSONL. Every event carries
// `schema_version: "1"`, `session_id`, and `type`. Documented in motoko_agent
// design_docs/implemented/motoko_agent/m-motoko-eval-instrumentation.md.
//
// We use json.RawMessage for fields whose shape varies by event type so a
// single Go struct can cover all event types — narrowed by `type` switch in
// the consumer.
type motokoEvent struct {
	SchemaVersion string `json:"schema_version,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
	Type          string `json:"type"`

	// Shared fields (present on most events)
	Step  *int   `json:"step,omitempty"`
	Model string `json:"model,omitempty"`

	// Per-step usage (thinking, native_tool_calls, native_tool_results)
	InputTokens              *int     `json:"input_tokens,omitempty"`
	OutputTokens             *int     `json:"output_tokens,omitempty"`
	CostUSD                  *float64 `json:"cost_usd,omitempty"`
	CacheReadInputTokens     *int     `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens *int     `json:"cache_creation_input_tokens,omitempty"`

	// thinking event
	FinishReason string `json:"finish_reason,omitempty"`
	Text         string `json:"text,omitempty"`
	// ToolCallsRaw decodes the union: an int (count, in thinking events) or
	// an array of ToolCall objects (in native_tool_calls events). The
	// disambiguation happens in parseSessionLine using ev.Type — we do
	// NOT json-tag this as `tool_calls` because the field's shape varies.
	ToolCallsRaw json.RawMessage `json:"tool_calls,omitempty"`

	// session_start
	Task          string `json:"task,omitempty"`
	MotokoCommit  string `json:"motoko_commit,omitempty"`
	BrainVersion  string `json:"brainVersion,omitempty"`
	ConfigProfile string `json:"config_profile,omitempty"`

	// done event (per-step success)
	Output string `json:"output,omitempty"`
	Source string `json:"source,omitempty"`

	// run_summary (terminal totals)
	StepsExecuted       *int             `json:"steps_executed,omitempty"`
	Usage               *runSummaryUsage `json:"usage,omitempty"`
	TotalCostUSD        *float64         `json:"total_cost_usd,omitempty"`
	TotalCostMillicents *int             `json:"total_cost_millicents,omitempty"`
	DurationMS          *int64           `json:"duration_ms,omitempty"`
	Error               string           `json:"error,omitempty"`

	// dp7_verifier_rejected
	Errors string `json:"errors,omitempty"`

	// cost_warning — Threshold is unique to this event; total_cost_millicents
	// is shared with run_summary above (TotalCostMillicents field).
	Threshold     string `json:"threshold,omitempty"`
	CapMillicents *int   `json:"cap_millicents,omitempty"`

	// NativeToolCalls is populated by parseSessionLine from ToolCallsRaw
	// when ev.Type == "native_tool_calls" (where the field is an array).
	// Not json-tagged — set by the parser, not the decoder.
	NativeToolCalls []json.RawMessage `json:"-"`
}

// runSummaryUsage matches the omit-when-absent semantics of motoko's
// usage block. Cache fields are pointers so we can distinguish absent from 0.
type runSummaryUsage struct {
	InputTokens              int  `json:"input_tokens"`
	OutputTokens             int  `json:"output_tokens"`
	CacheReadInputTokens     *int `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens *int `json:"cache_creation_input_tokens,omitempty"`
	TotalTokens              int  `json:"total_tokens"`
}

// parseSessionLine parses a single JSONL line. Returns (event, raw, err).
// Raw map preserves the full payload for ProviderData round-trip.
func parseSessionLine(line []byte) (*motokoEvent, map[string]any, error) {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" {
		return nil, nil, fmt.Errorf("empty line")
	}
	if trimmed[0] != '{' {
		return nil, nil, fmt.Errorf("non-JSON line")
	}
	var ev motokoEvent
	if err := json.Unmarshal(line, &ev); err != nil {
		return nil, nil, fmt.Errorf("parse: %w", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err != nil {
		// We have ev; raw is best-effort
		raw = nil
	}
	// Disambiguate the union-typed `tool_calls` field by event type:
	//   - thinking event:           tool_calls is int (count)
	//   - native_tool_calls event:  tool_calls is array of {id, tool, arguments}
	if len(ev.ToolCallsRaw) > 0 {
		if ev.Type == "native_tool_calls" {
			var arr []json.RawMessage
			if err := json.Unmarshal(ev.ToolCallsRaw, &arr); err == nil {
				ev.NativeToolCalls = arr
			}
		}
		// For thinking events the count is unused (we increment NumTurns
		// per thinking event regardless); no need to decode the int.
	}
	return &ev, raw, nil
}

// findSessionJSONL locates the session_<id>.jsonl file produced by a motoko
// run. Strategy:
//  1. If MOTOKO_SESSION_ID was set (we always set it), look for that exact
//     filename: ${workspace}/.motoko/logfile/<sessionID>.jsonl
//  2. Fall back to the newest *.jsonl in the same directory (handles cases
//     where the wrapper renames the file or our env-var injection didn't
//     reach the wrapper layer).
//  3. **MOTOKO_REPO fallback** (added after smoke testing 2026-05-08): the
//     current motoko wrapper does `cd "$MOTOKO_REPO"` before exec'ing the
//     agent, so JSONL actually lands in `$MOTOKO_REPO/.motoko/logfile/`
//     — NOT the workspace. Search there too.
//  4. **discoveredRepo fallback** (M-MOTOKO-EVAL-HARNESS-HARDENING M3b,
//     gap #5): when MOTOKO_REPO env is not set but the executor's
//     HealthCheck populated motokoRepo from `motoko --version` output,
//     use that path. Lets the adapter work with zero env-var setup.
//  5. Return ("", err) if none exist — caller treats as "no JSONL" and
//     emits a Result with Error pointing to that fact.
func findSessionJSONL(workspace, sessionID, discoveredRepo string) (string, error) {
	candidates := []string{filepath.Join(workspace, motokoStateDir, "logfile")}
	if motokoRepo := os.Getenv("MOTOKO_REPO"); motokoRepo != "" {
		candidates = append(candidates, filepath.Join(motokoRepo, motokoStateDir, "logfile"))
	} else if discoveredRepo != "" {
		// MOTOKO_REPO env was unset but `motoko --version` reported one.
		// This is the zero-config path: HealthCheck queried the binary,
		// stashed the answer, and now we use it without requiring the
		// caller to plumb an env var through their setup.
		candidates = append(candidates, filepath.Join(discoveredRepo, motokoStateDir, "logfile"))
	}

	var lastErr error
	for _, logDir := range candidates {
		if _, err := os.Stat(logDir); err != nil {
			lastErr = err
			continue
		}
		path, err := findJSONLInDir(logDir, sessionID)
		if err == nil {
			return path, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("motoko session JSONL not found in workspace or MOTOKO_REPO: %w", lastErr)
}

// findJSONLInDir is the single-directory search logic used by findSessionJSONL.
func findJSONLInDir(logDir, sessionID string) (string, error) {

	if sessionID != "" {
		candidate := filepath.Join(logDir, sessionID+".jsonl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// Fallback: newest .jsonl in dir
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return "", fmt.Errorf("read motoko logfile dir: %w", err)
	}
	type fileTime struct {
		path string
		mod  int64
	}
	var jsonls []fileTime
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".jsonl") {
			continue
		}
		info, err := ent.Info()
		if err != nil {
			continue
		}
		jsonls = append(jsonls, fileTime{
			path: filepath.Join(logDir, ent.Name()),
			mod:  info.ModTime().UnixNano(),
		})
	}
	if len(jsonls) == 0 {
		return "", fmt.Errorf("no .jsonl files in %s", logDir)
	}
	sort.Slice(jsonls, func(i, j int) bool { return jsonls[i].mod > jsonls[j].mod })
	return jsonls[0].path, nil
}

// parseSessionJSONL reads a motoko session JSONL file and folds its events
// into an executor.Result. The terminal `run_summary` event is the
// authoritative source for totals when present; otherwise the parser sums
// per-step `thinking` events as a fallback (covers the crash-mid-run case
// where motoko never emitted run_summary).
//
// Ordering: events are processed in file order. State accumulates as we go;
// run_summary OVERRIDES the accumulated totals (it's authoritative).
func parseSessionJSONL(path string) (*executor.Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open session JSONL: %w", err)
	}
	defer func() { _ = f.Close() }()

	res := &executor.Result{
		ProviderData: map[string]any{},
	}
	rawEvents := []map[string]any{}

	// Per-step accumulators (used as fallback when run_summary is absent).
	var (
		sumInputTokens     int
		sumOutputTokens    int
		sumCacheRead       int
		sumCacheCreation   int
		sumCostUSD         float64
		numTurns           int
		numToolCalls       int
		lastDoneOutput     string
		lastFinishReason   string
		gotRunSummary      bool
		gotErrorEvent      bool
		errorEventMessage  string
		motokoCommit       string
		motokoModel        string
		dp7RejectionsCount int
	)

	scanner := bufio.NewScanner(f)
	// motoko events can carry full file contents in tool results — bump the
	// buffer ceiling well above the default 64KB.
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		ev, raw, err := parseSessionLine(line)
		if err != nil {
			// Skip non-JSON / malformed lines silently (preamble chatter etc.)
			continue
		}
		if raw != nil {
			rawEvents = append(rawEvents, raw)
		}

		switch ev.Type {
		case "session_start":
			if ev.SessionID != "" {
				res.SessionID = ev.SessionID
			}
			motokoCommit = ev.MotokoCommit
			motokoModel = ev.Model

		case "thinking":
			numTurns++
			if ev.InputTokens != nil {
				sumInputTokens += *ev.InputTokens
			}
			if ev.OutputTokens != nil {
				sumOutputTokens += *ev.OutputTokens
			}
			if ev.CostUSD != nil {
				sumCostUSD += *ev.CostUSD
			}
			if ev.CacheReadInputTokens != nil {
				sumCacheRead += *ev.CacheReadInputTokens
			}
			if ev.CacheCreationInputTokens != nil {
				sumCacheCreation += *ev.CacheCreationInputTokens
			}
			lastFinishReason = ev.FinishReason

		case "native_tool_calls":
			numToolCalls += len(ev.NativeToolCalls)

		case "done":
			lastDoneOutput = ev.Output

		case "dp7_verifier_rejected":
			dp7RejectionsCount++

		case "error":
			gotErrorEvent = true
			errorEventMessage = ev.Error
			if errorEventMessage == "" {
				// motoko emits {code, message, source} on error events; map raw
				if raw != nil {
					if msg, ok := raw["message"].(string); ok {
						errorEventMessage = msg
					}
				}
			}

		case "run_summary":
			gotRunSummary = true
			res.NumTurns = 0
			if ev.StepsExecuted != nil {
				res.NumTurns = *ev.StepsExecuted
			}
			if ev.Usage != nil {
				res.InputTokens = ev.Usage.InputTokens
				res.OutputTokens = ev.Usage.OutputTokens
				if ev.Usage.CacheReadInputTokens != nil {
					res.CacheReadInputTokens = *ev.Usage.CacheReadInputTokens
				}
				if ev.Usage.CacheCreationInputTokens != nil {
					res.CacheCreationInputTokens = *ev.Usage.CacheCreationInputTokens
				}
			}
			if ev.TotalCostUSD != nil {
				res.CostUSD = *ev.TotalCostUSD
			}
			if ev.DurationMS != nil {
				res.DurationMS = int(*ev.DurationMS)
			}
			res.Success = ev.FinishReason == "stop"
			if !res.Success {
				if ev.Error != "" {
					res.Error = ev.Error
				} else {
					res.Error = "motoko terminated with finish_reason=" + ev.FinishReason
				}
			}
			if motokoModel != "" {
				res.ProviderData["motoko_model"] = motokoModel
			}
			res.ProviderData["motoko_finish_reason"] = ev.FinishReason
			// M-EVAL-SWEET-SPOT: promote the finish_reason to the canonical
			// top-level field so the eval harness can categorize without
			// reaching into ProviderData.
			res.FinishReason = ev.FinishReason
			// M-EVAL-SWEET-SPOT-FOLLOWUP: when motoko stopped because its
			// cost cap fired, populate Result.CostKilledAt so the per-result
			// JSON's cost_killed_at field is non-zero (drives the new
			// budget_blocked sweet-spot bucket).
			if ev.FinishReason == "cost_exhausted" {
				res.CostKilledAt = res.CostUSD
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner: %w", err)
	}

	// If run_summary is missing, fall back to summed totals + infer success
	// from the last thinking event's finish_reason. M-MOTOKO-EVAL-HARNESS-
	// HARDENING M3a (gap #2): pre-M1c the JSONL frequently truncated before
	// run_summary reached disk, but the run had completed successfully. The
	// parser was setting Success=false in that case, mis-attributing
	// successful runs as crashes. Now: prefer run_summary when present;
	// otherwise treat finish_reason="stop" on the last thinking event as
	// success (motoko emitted a clean stop, just lost the trailing summary).
	// finish_reason="error" or absent = treat as crash.
	if !gotRunSummary {
		res.InputTokens = sumInputTokens
		res.OutputTokens = sumOutputTokens
		res.CacheReadInputTokens = sumCacheRead
		res.CacheCreationInputTokens = sumCacheCreation
		res.CostUSD = sumCostUSD
		res.NumTurns = numTurns
		switch {
		case gotErrorEvent:
			res.Success = false
			res.Error = "motoko emitted error event without run_summary: " + errorEventMessage
		case lastFinishReason == "stop":
			res.Success = true
			res.ProviderData["motoko_finish_reason"] = lastFinishReason
			res.ProviderData["motoko_run_summary_missing"] = true
			res.FinishReason = lastFinishReason
		default:
			res.Success = false
			if lastFinishReason != "" {
				res.Error = "motoko terminated with finish_reason=" + lastFinishReason + " and no run_summary"
				res.FinishReason = lastFinishReason
			} else {
				res.Error = "motoko terminated without emitting run_summary (likely crash)"
			}
		}
	}

	// Always populated (regardless of run_summary presence).
	res.ToolCallCount = numToolCalls
	res.Output = lastDoneOutput
	if lastDoneOutput == "" && lastFinishReason != "" {
		res.Output = "(motoko terminated with finish_reason=" + lastFinishReason + ")"
	}

	// ProviderData round-trip — keep the full event stream + key motoko metadata
	// so consumers can extract anything we didn't explicitly map.
	res.ProviderData["motoko_events"] = rawEvents
	if motokoCommit != "" {
		res.ProviderData["motoko_commit"] = motokoCommit
	}
	if dp7RejectionsCount > 0 {
		res.ProviderData["dp7_rejections"] = dp7RejectionsCount
	}

	return res, nil
}
