package observatory

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// M-MISSION-LOOP-UNIFIED-TELEMETRY M1.
//
// OpenRouter Broadcast delivers our chain id as `session.id` — verified live on
// the v0.33.1 probe span, which carried session.id correctly and chain_id
// absent. convertSpan reads chain_id only from ailang.chain_id / chain_id, so
// the identifier arrives and is then ignored.
//
// The fix REUSES the linkage that already exists rather than adding a parallel
// map: chain_stages has a session_id column, and sessions carries chain_id +
// stage_id (added in migrate_v8). Resolution mirrors the existing
// LookupTaskBySessionID sibling.
//
// The load-bearing test here is the NEGATIVE CONTROL: `session.id` belongs to
// the Claude Code path today, and widening it must not change how those spans
// resolve.

// seedSessionWithChain registers a session bound to a chain and stage, which is
// what an OpenRouter dispatch will do for its correlation id.
func seedSessionWithChain(t *testing.T, backend Backend, sessionID, chainID, stageID string) {
	t.Helper()
	db := backend.(*SQLiteBackend).DB()
	_, err := db.Exec(
		`INSERT INTO sessions (session_id, workspace, source, started_at, chain_id, stage_id)
		 VALUES (?, ?, 'otel', ?, ?, ?)`,
		sessionID, "/tmp/ws", time.Now().Format(time.RFC3339Nano), chainID, stageID)
	if err != nil {
		t.Fatalf("seed session %q: %v", sessionID, err)
	}
}

// otlpJSONSpanWithSessionID builds an OTLP/JSON payload shaped like the one
// OpenRouter Broadcast actually sends: the correlation id arrives as a
// `session.id` SPAN attribute.
func otlpJSONSpanWithSessionID(traceID, spanID, sessionID string, extraAttrs string) []byte {
	now := time.Now().UnixNano()
	return fmt.Appendf(nil, `{
	  "resourceSpans": [{
	    "resource": {"attributes": [
	      {"key": "service.name", "value": {"stringValue": "openrouter"}}
	    ]},
	    "scopeSpans": [{
	      "spans": [{
	        "traceId": %q,
	        "spanId": %q,
	        "name": "LLM Generation",
	        "kind": 1,
	        "startTimeUnixNano": "%d",
	        "endTimeUnixNano": "%d",
	        "attributes": [
	          {"key": "session.id", "value": {"stringValue": %q}}%s
	        ]
	      }]
	    }]
	  }]
	}`, traceID, spanID, now, now, sessionID, extraAttrs)
}

// TestSessionChainLinkage_BroadcastSpanResolvesToChain is the headline: a span
// carrying only session.id lands with its chain and stage populated.
func TestSessionChainLinkage_BroadcastSpanResolvesToChain(t *testing.T) {
	r, backend := newTestReceiver(t)
	seedSessionWithChain(t, backend, "sess-mission-iter-191", "chain-abc", "stage-xyz")

	body := otlpJSONSpanWithSessionID(knownTraceIDHex, knownSpanIDHex, "sess-mission-iter-191", "")
	if w := postTraces(r, "application/json", body); w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	spans := storedSpans(t, backend)
	if len(spans) != 1 {
		t.Fatalf("stored %d spans, want 1", len(spans))
	}
	if spans[0].ChainID != "chain-abc" {
		t.Errorf("chain_id = %q, want %q — the session.id linkage did not resolve", spans[0].ChainID, "chain-abc")
	}
	if spans[0].StageID != "stage-xyz" {
		t.Errorf("stage_id = %q, want %q", spans[0].StageID, "stage-xyz")
	}
}

// TestSessionChainLinkage_ClaudeCodePathUnchanged is the NEGATIVE CONTROL and the
// most important test in this milestone.
//
// `session.id` belongs to the Claude Code path today. A Claude Code span whose
// session has no chain must resolve exactly as it does now: no chain_id, no
// error, and no interference with the workspace enrichment that path relies on.
func TestSessionChainLinkage_ClaudeCodePathUnchanged(t *testing.T) {
	r, backend := newTestReceiver(t)

	// A Claude Code session: workspace set, NO chain/stage.
	db := backend.(*SQLiteBackend).DB()
	_, err := db.Exec(
		`INSERT INTO sessions (session_id, workspace, source, started_at) VALUES (?, ?, 'hook', ?)`,
		"claude-sess-1", "/Users/dev/project", time.Now().Format(time.RFC3339Nano))
	if err != nil {
		t.Fatalf("seed claude session: %v", err)
	}

	body := otlpJSONSpanWithSessionID(knownTraceIDHex, knownSpanIDHex, "claude-sess-1", "")
	if w := postTraces(r, "application/json", body); w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	spans := storedSpans(t, backend)
	if len(spans) != 1 {
		t.Fatalf("stored %d spans, want 1", len(spans))
	}
	if spans[0].ChainID != "" {
		t.Errorf("chain_id = %q, want empty — a Claude Code session with no chain must not acquire one", spans[0].ChainID)
	}
}

// TestSessionChainLinkage_ExplicitChainIDWins pins precedence. An explicit
// ailang.chain_id is a direct assertion by the producer; a session lookup is an
// inference. The assertion must win.
func TestSessionChainLinkage_ExplicitChainIDWins(t *testing.T) {
	r, backend := newTestReceiver(t)
	seedSessionWithChain(t, backend, "sess-conflict", "chain-from-session", "stage-from-session")

	extra := `,{"key":"chain_id","value":{"stringValue":"chain-explicit"}}`
	body := otlpJSONSpanWithSessionID(knownTraceIDHex, knownSpanIDHex, "sess-conflict", extra)
	if w := postTraces(r, "application/json", body); w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	spans := storedSpans(t, backend)
	if len(spans) != 1 {
		t.Fatalf("stored %d spans, want 1", len(spans))
	}
	if spans[0].ChainID != "chain-explicit" {
		t.Errorf("chain_id = %q, want %q — an explicit chain_id must beat a session lookup",
			spans[0].ChainID, "chain-explicit")
	}
}

// TestSessionChainLinkage_UnknownSessionIsNotAnError: an unrecognised session is
// ordinary (any Claude Code session predating this feature), not a failure.
func TestSessionChainLinkage_UnknownSessionIsNotAnError(t *testing.T) {
	r, backend := newTestReceiver(t)

	body := otlpJSONSpanWithSessionID(knownTraceIDHex, knownSpanIDHex, "sess-never-seen", "")
	if w := postTraces(r, "application/json", body); w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — an unknown session must not fail ingest", w.Code)
	}

	spans := storedSpans(t, backend)
	if len(spans) != 1 {
		t.Fatalf("stored %d spans, want 1", len(spans))
	}
	if spans[0].ChainID != "" {
		t.Errorf("chain_id = %q, want empty", spans[0].ChainID)
	}
}

// TestLookupChainBySessionID_Direct exercises the backend method itself.
func TestLookupChainBySessionID_Direct(t *testing.T) {
	_, backend := newTestReceiver(t)
	seedSessionWithChain(t, backend, "sess-direct", "chain-1", "stage-1")

	chainID, stageID := backend.LookupChainBySessionID(context.Background(), "sess-direct")
	if chainID != "chain-1" || stageID != "stage-1" {
		t.Errorf("got (%q, %q), want (chain-1, stage-1)", chainID, stageID)
	}

	// Unknown session returns empties, not an error — mirrors the existing
	// LookupTaskBySessionID contract.
	if c, s := backend.LookupChainBySessionID(context.Background(), "nope"); c != "" || s != "" {
		t.Errorf("unknown session gave (%q, %q), want empties", c, s)
	}
}
