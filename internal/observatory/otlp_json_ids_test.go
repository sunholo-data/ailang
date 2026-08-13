package observatory

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
)

// M-OPENROUTER-BROADCAST-INGEST M1.
//
// These tests cover the OTLP *decode* path, which had NO coverage at all before
// this milestone: every existing test in otlp_receiver_test.go hands convertSpan
// a raw []byte trace ID and never exercises protojson or the HTTP handler. That
// gap is why the hex/base64 defect shipped and survived.
//
// The defect: protojson follows the proto3 JSON mapping and decodes `bytes`
// fields as base64. The OTLP/JSON spec overrides this — traceId and spanId are
// HEX strings. So a correct 32-hex-char trace ID was base64-decoded into 24
// bytes of garbage and stored, while the endpoint still answered
// 200 {"partialSuccess":{}}.

// knownTraceIDHex and knownSpanIDHex are the exact values measured against the
// production receiver on 2026-08-13.
const (
	knownTraceIDHex = "5b8aa5a2d2c872e8321cf37308d69df2" // 16 bytes
	knownSpanIDHex  = "051581bf3cb55c13"                 // 8 bytes
)

// newTestReceiver builds an OTLP receiver over an in-memory SQLite backend.
func newTestReceiver(t *testing.T) (*OTLPReceiver, Backend) {
	t.Helper()
	backend, err := NewSQLiteBackendFromPath(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteBackendFromPath: %v", err)
	}
	return NewOTLPReceiver(backend), backend
}

// otlpJSONTracePayload builds a minimal OTLP/JSON ExportTraceServiceRequest.
func otlpJSONTracePayload(traceID, spanID string) []byte {
	now := time.Now().UnixNano()
	return fmt.Appendf(nil, `{
	  "resourceSpans": [{
	    "resource": {"attributes": [
	      {"key": "service.name", "value": {"stringValue": "m1-decode-test"}}
	    ]},
	    "scopeSpans": [{
	      "spans": [{
	        "traceId": %q,
	        "spanId": %q,
	        "name": "decode.probe",
	        "kind": 1,
	        "startTimeUnixNano": "%d",
	        "endTimeUnixNano": "%d"
	      }]
	    }]
	  }]
	}`, traceID, spanID, now, now)
}

// protobufTracePayload builds the same logical span as otlpJSONTracePayload but
// wire-encoded as protobuf, where trace/span IDs are raw bytes rather than hex.
// This is the control: the protobuf path was always correct, so it defines what
// the JSON path must agree with.
func protobufTracePayload(t *testing.T, traceIDHex, spanIDHex string) []byte {
	t.Helper()
	traceID, err := hex.DecodeString(traceIDHex)
	if err != nil {
		t.Fatalf("decode traceID: %v", err)
	}
	spanID, err := hex.DecodeString(spanIDHex)
	if err != nil {
		t.Fatalf("decode spanID: %v", err)
	}
	now := uint64(time.Now().UnixNano())

	req := &coltracepb.ExportTraceServiceRequest{
		ResourceSpans: []*tracepb.ResourceSpans{{
			ScopeSpans: []*tracepb.ScopeSpans{{
				Spans: []*tracepb.Span{{
					TraceId:           traceID,
					SpanId:            spanID,
					Name:              "decode.probe",
					Kind:              tracepb.Span_SPAN_KIND_INTERNAL,
					StartTimeUnixNano: now,
					EndTimeUnixNano:   now,
				}},
			}},
		}},
	}
	body, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("marshal protobuf: %v", err)
	}
	return body
}

// postTraces sends body to handleTraces with the given content type.
func postTraces(r *OTLPReceiver, contentType string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/traces", bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	r.handleTraces(w, req)
	return w
}

func storedSpans(t *testing.T, backend Backend) []*Span {
	t.Helper()
	spans, err := backend.ListSpans(context.Background(), SpanListOptions{Limit: 100})
	if err != nil {
		t.Fatalf("ListSpans: %v", err)
	}
	return spans
}

// TestOTLPJSON_TraceIDRoundTrip is the headline regression: a hex trace ID sent
// as OTLP/JSON must be stored byte-identically.
//
// Before the fix this stored the base64-decoding of the hex STRING — 24 bytes,
// rendering as 48 hex chars — instead of the 16 bytes the ID actually denotes.
func TestOTLPJSON_TraceIDRoundTrip(t *testing.T) {
	r, backend := newTestReceiver(t)

	w := postTraces(r, "application/json", otlpJSONTracePayload(knownTraceIDHex, knownSpanIDHex))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	spans := storedSpans(t, backend)
	if len(spans) != 1 {
		t.Fatalf("stored %d spans, want 1", len(spans))
	}

	got := spans[0].TraceID
	if len(got) != 32 {
		t.Errorf("trace_id has %d hex chars (%d bytes), want 32 (16 bytes): %s",
			len(got), len(got)/2, got)
	}
	if got != knownTraceIDHex {
		t.Errorf("trace_id = %s, want %s", got, knownTraceIDHex)
	}
}

// TestOTLPJSON_MalformedIDRejected pins the Design Freeze decision: a trace ID
// that is not valid hex is REJECTED with a typed 400 and NO row is written.
//
// Asserting the span count is the point — a 400 that still writes a row would
// pass a status-code-only check.
func TestOTLPJSON_MalformedIDRejected(t *testing.T) {
	r, backend := newTestReceiver(t)

	before := len(storedSpans(t, backend))

	w := postTraces(r, "application/json", otlpJSONTracePayload("not-hex", knownSpanIDHex))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for a malformed traceId; body = %s", w.Code, w.Body.String())
	}

	if after := len(storedSpans(t, backend)); after != before {
		t.Errorf("span count went %d -> %d; a rejected payload must write no rows", before, after)
	}
}

// TestNormalizeOTLPJSONIDs_Cases covers the codec directly.
func TestNormalizeOTLPJSONIDs_Cases(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		{"valid 16-byte trace id", "traceId", knownTraceIDHex, false},
		{"valid 8-byte span id", "spanId", knownSpanIDHex, false},
		{"valid snake_case trace id", "trace_id", knownTraceIDHex, false},
		{"valid parent span id", "parentSpanId", knownSpanIDHex, false},
		{"trace id too short", "traceId", "5b8aa5a2", true},
		{"trace id too long", "traceId", knownTraceIDHex + "ff", true},
		{"span id given trace-id length", "spanId", knownTraceIDHex, true},
		{"non-hex characters", "traceId", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", true},
		// An empty ID is legal in OTLP — protojson decodes it to nil — so the
		// normalizer must leave it alone rather than reject it.
		{"empty id is left alone", "traceId", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := fmt.Appendf(nil, `{"resourceSpans":[{"scopeSpans":[{"spans":[{%q:%q}]}]}]}`, tt.field, tt.value)
			_, err := normalizeOTLPJSONIDs(body)
			if tt.wantErr && err == nil {
				t.Errorf("normalizeOTLPJSONIDs(%s=%q) = nil error, want an error", tt.field, tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("normalizeOTLPJSONIDs(%s=%q) = %v, want no error", tt.field, tt.value, err)
			}
		})
	}
}

// TestNormalizeOTLPJSONIDs_HexWinsOverBase64 pins the precedence rule.
//
// Every 32-char hex string is ALSO syntactically valid base64, so a decoder that
// sniffs the encoding has no principled way to choose. The OTLP/JSON spec says
// hex, so hex must win unconditionally — this test fails if anyone later
// "improves" the codec by trying base64 first.
func TestNormalizeOTLPJSONIDs_HexWinsOverBase64(t *testing.T) {
	// knownTraceIDHex is simultaneously valid hex (16 bytes) and valid base64
	// (24 bytes — its length is already a multiple of 4, so it needs no
	// padding, and every hex digit is in the base64 alphabet). Confirm the
	// premise, so this test cannot silently go vacuous.
	asBase64, err := base64.StdEncoding.DecodeString(knownTraceIDHex)
	if err != nil {
		t.Fatalf("premise broken: %q is not valid base64, so this test proves nothing", knownTraceIDHex)
	}
	if len(asBase64) != 24 {
		t.Fatalf("premise broken: base64-decoding %q gives %d bytes, want 24", knownTraceIDHex, len(asBase64))
	}
	if _, err := hex.DecodeString(knownTraceIDHex); err != nil {
		t.Fatalf("premise broken: %q is not valid hex", knownTraceIDHex)
	}

	r, backend := newTestReceiver(t)
	if w := postTraces(r, "application/json", otlpJSONTracePayload(knownTraceIDHex, knownSpanIDHex)); w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	spans := storedSpans(t, backend)
	if len(spans) != 1 {
		t.Fatalf("stored %d spans, want 1", len(spans))
	}
	if spans[0].TraceID != knownTraceIDHex {
		t.Errorf("trace_id = %s, want %s (hex must win over base64)", spans[0].TraceID, knownTraceIDHex)
	}
}

// TestOTLPProtobuf_Unchanged is the no-regression control: the protobuf path was
// always correct and must stay that way.
func TestOTLPProtobuf_Unchanged(t *testing.T) {
	r, backend := newTestReceiver(t)

	body := protobufTracePayload(t, knownTraceIDHex, knownSpanIDHex)
	if w := postTraces(r, "application/x-protobuf", body); w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	spans := storedSpans(t, backend)
	if len(spans) != 1 {
		t.Fatalf("stored %d spans, want 1", len(spans))
	}
	if spans[0].TraceID != knownTraceIDHex {
		t.Errorf("trace_id = %s, want %s", spans[0].TraceID, knownTraceIDHex)
	}
	if spans[0].ID != knownSpanIDHex {
		t.Errorf("span_id = %s, want %s", spans[0].ID, knownSpanIDHex)
	}
}

// TestOTLPJSON_MatchesProtobuf is the cross-encoding invariant the defect broke:
// the same logical span must produce the same stored IDs whichever encoding
// carried it.
func TestOTLPJSON_MatchesProtobuf(t *testing.T) {
	rJSON, backendJSON := newTestReceiver(t)
	if w := postTraces(rJSON, "application/json", otlpJSONTracePayload(knownTraceIDHex, knownSpanIDHex)); w.Code != http.StatusOK {
		t.Fatalf("json ingest status = %d", w.Code)
	}
	jsonSpans := storedSpans(t, backendJSON)
	if len(jsonSpans) != 1 {
		t.Fatalf("json path stored %d spans, want 1", len(jsonSpans))
	}

	rPB, backendPB := newTestReceiver(t)
	pbBody := protobufTracePayload(t, knownTraceIDHex, knownSpanIDHex)
	if w := postTraces(rPB, "application/x-protobuf", pbBody); w.Code != http.StatusOK {
		t.Fatalf("protobuf ingest status = %d", w.Code)
	}
	pbSpans := storedSpans(t, backendPB)
	if len(pbSpans) != 1 {
		t.Fatalf("protobuf path stored %d spans, want 1", len(pbSpans))
	}

	if jsonSpans[0].TraceID != pbSpans[0].TraceID {
		t.Errorf("trace_id differs by encoding: json=%s protobuf=%s", jsonSpans[0].TraceID, pbSpans[0].TraceID)
	}
	if jsonSpans[0].ID != pbSpans[0].ID {
		t.Errorf("span_id differs by encoding: json=%s protobuf=%s", jsonSpans[0].ID, pbSpans[0].ID)
	}
}
