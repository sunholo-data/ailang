package effects

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval"
)

// StreamNDJSONPost opens a streaming HTTP POST that returns newline-delimited
// JSON (NDJSON). Used by APIs like Ollama that stream one JSON object per line
// instead of using the SSE `data:` framing that Anthropic/OpenAI/Gemini use.
//
// Differs from StreamSSEPost in three ways:
//  1. No `Accept: text/event-stream` header sent — caller controls Accept via
//     config.headers
//  2. No response Content-Type check — caller has explicitly chosen NDJSON
//     parsing by calling this builtin, so any 200-OK response body is treated
//     as line-delimited
//  3. Each non-empty line is emitted as a single sse_data event (no `data:`
//     prefix, no blank-line dispatch) — the StreamEvent.SSEData variant on
//     the AILANG side carries the raw JSON line in its data field
//
// Reuses StreamConnection, eventBuffer, makeStreamErr, makeStreamOk so the
// AILANG-side type surface (StreamConn, StreamEvent::SSEData/Closed/StreamError)
// is identical to ssePost.
//
// Args: [url: string, body: string, config: record{headers}]
// Returns: Result[StreamConn(int), StreamError]
func StreamNDJSONPost(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("_stream_ndjson_post: expected 3 arguments, got %d", len(args))
	}

	urlVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_stream_ndjson_post: expected String for url, got %T", args[0])
	}

	bodyVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_stream_ndjson_post: expected String for body, got %T", args[1])
	}

	if ctx.Stream == nil {
		return nil, fmt.Errorf("E_STREAM_NO_CONTEXT: Stream effect not configured (missing --caps Stream)")
	}

	if err := ctx.RequireCapWithBudget("Stream", "stream.ndjson_post"); err != nil {
		return makeStreamErr("BudgetExhausted", err.Error()), nil
	}

	if err := ctx.Stream.ValidateURL(urlVal.Value); err != nil {
		return makeStreamErr("ConnectionFailed", err.Error()), nil
	}

	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")

	if configRec, ok := args[2].(*eval.RecordValue); ok {
		if hdrs, ok := configRec.Fields["headers"]; ok {
			if hdrList, ok := hdrs.(*eval.ListValue); ok {
				for _, hdr := range hdrList.Elements {
					if hdrRec, ok := hdr.(*eval.RecordValue); ok {
						nameVal, _ := hdrRec.Fields["name"].(*eval.StringValue)
						valVal, _ := hdrRec.Fields["value"].(*eval.StringValue)
						if nameVal != nil && valVal != nil {
							headers.Set(nameVal.Value, valVal.Value)
						}
					}
				}
			}
		}
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: ctx.Stream.ConnectTimeout,
		}).DialContext,
		TLSHandshakeTimeout:   ctx.Stream.ConnectTimeout,
		ResponseHeaderTimeout: ctx.Stream.ConnectTimeout,
	}
	client := &http.Client{Transport: transport}

	req, err := http.NewRequestWithContext(context.Background(), "POST", urlVal.Value, strings.NewReader(bodyVal.Value))
	if err != nil {
		return makeStreamErr("ConnectionFailed", fmt.Sprintf("NDJSON POST request creation failed: %s", err.Error())), nil
	}
	req.Header = headers

	resp, err := client.Do(req)
	if err != nil {
		return makeStreamErr("ConnectionFailed", fmt.Sprintf("NDJSON POST connection failed: %s", err.Error())), nil
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return makeStreamErr("ConnectionFailed",
			fmt.Sprintf("NDJSON POST HTTP error: %d %s", resp.StatusCode, resp.Status)), nil
	}

	conn := &StreamConnection{
		protocol:    "NDJSON",
		httpResp:    resp,
		status:      StreamStatusOpen,
		eventBuffer: make(chan streamEvent, ctx.Stream.EventBufferSize),
		done:        make(chan struct{}),
		idleTimeout: ctx.Stream.IdleTimeout,
		maxDuration: ctx.Stream.MaxDuration,
	}

	id, err := ctx.Stream.AcquireConnection(conn)
	if err != nil {
		resp.Body.Close()
		return makeStreamErr("ConnectionFailed", err.Error()), nil
	}

	go conn.ndjsonReadLoop()

	conn.eventBuffer <- streamEvent{
		kind: "opened",
		text: "NDJSON-POST",
	}

	return makeStreamOk(makeStreamConn(id)), nil
}

// ndjsonReadLoop reads newline-delimited JSON from the HTTP response body and
// emits one sse_data event per non-empty line.
//
// Unlike SSE, NDJSON has no per-event framing: every line is a complete event.
// Blank lines are ignored (some servers emit them as keep-alives). EOF →
// Closed event, read error → ProtocolError event.
func (sc *StreamConnection) ndjsonReadLoop() {
	defer func() {
		sc.mu.Lock()
		status := sc.status
		sc.mu.Unlock()
		if status != StreamStatusClosed && status != StreamStatusClosing {
			sc.eventBuffer <- streamEvent{kind: "closed", code: 0, reason: "NDJSON stream ended"}
		}
	}()

	scanner := bufio.NewScanner(sc.httpResp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for {
		select {
		case <-sc.done:
			return
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				sc.eventBuffer <- streamEvent{
					kind:    "error",
					errType: "ProtocolError",
					text:    fmt.Sprintf("NDJSON read error: %s", err.Error()),
				}
			}
			return
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		sc.mu.Lock()
		sc.messagesRecv++
		sc.bytesRecv += int64(len(line))
		sc.mu.Unlock()

		sc.eventBuffer <- streamEvent{
			kind:         "sse_data",
			text:         line,
			sseEventType: "message",
		}
	}
}

func init() {
	RegisterOp("Stream", "ndjson_post", StreamNDJSONPost)
}
