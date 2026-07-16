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

// StreamSSEConnect opens an SSE (Server-Sent Events) connection.
//
// SSE is a unidirectional HTTP streaming protocol used by AI APIs (Anthropic,
// OpenAI, Gemini) for token-by-token response streaming. Reuses the existing
// event dispatch infrastructure (eventBuffer, onEvent, runEventLoop).
//
// Args: [url: string, config: record{headers}]
// Returns: Result[StreamConn(int), StreamError]
func StreamSSEConnect(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("_stream_sse_connect: expected 2 arguments, got %d", len(args))
	}

	urlVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_stream_sse_connect: expected String for url, got %T", args[0])
	}

	if ctx.Stream == nil {
		return nil, fmt.Errorf("E_STREAM_NO_CONTEXT: Stream effect not configured (missing --caps Stream)")
	}

	// Budget check: Stream.connect consumes one budget unit
	if err := ctx.RequireCapWithBudget("Stream", "stream.sse_connect"); err != nil {
		return makeStreamErr("BudgetExhausted", err.Error()), nil
	}

	// Validate URL — ValidateURL already handles https:// (same as wss://)
	if err := ctx.Stream.ValidateURL(urlVal.Value); err != nil {
		return makeStreamErr("ConnectionFailed", err.Error()), nil
	}

	// Parse config for custom headers (critical for Authorization: Bearer)
	headers := make(http.Header)
	headers.Set("Accept", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")

	if configRec, ok := args[1].(*eval.RecordValue); ok {
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

	// HTTP GET with SSE headers
	// Use a transport-level dial timeout so the connect phase is bounded,
	// but body streaming can continue indefinitely (managed by event loop timers).
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: ctx.Stream.ConnectTimeout,
		}).DialContext,
		TLSHandshakeTimeout:   ctx.Stream.ConnectTimeout,
		ResponseHeaderTimeout: ctx.Stream.ConnectTimeout,
	}
	client := &http.Client{
		Transport: transport,
		// Timeout: 0 — no overall timeout; body reads are long-lived
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", urlVal.Value, nil)
	if err != nil {
		return makeStreamErr("ConnectionFailed", fmt.Sprintf("SSE request creation failed: %s", err.Error())), nil
	}
	req.Header = headers

	resp, err := client.Do(req)
	if err != nil {
		return makeStreamErr("ConnectionFailed", fmt.Sprintf("SSE connection failed: %s", err.Error())), nil
	}

	// Verify content type
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		resp.Body.Close()
		return makeStreamErr("ConnectionFailed",
			fmt.Sprintf("SSE expected Content-Type text/event-stream, got %q (HTTP %d)", ct, resp.StatusCode)), nil
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return makeStreamErr("ConnectionFailed",
			fmt.Sprintf("SSE HTTP error: %d %s", resp.StatusCode, resp.Status)), nil
	}

	// Create connection — reuses shared infrastructure
	conn := &StreamConnection{
		protocol:    "SSE",
		httpResp:    resp,
		status:      StreamStatusOpen,
		eventBuffer: make(chan streamEvent, ctx.Stream.EventBufferSize),
		done:        make(chan struct{}),
		idleTimeout: ctx.Stream.IdleTimeout,
		maxDuration: ctx.Stream.MaxDuration,
	}

	// Register connection
	id, err := ctx.Stream.AcquireConnection(conn)
	if err != nil {
		resp.Body.Close()
		return makeStreamErr("ConnectionFailed", err.Error()), nil
	}

	// Deliver Opened BEFORE starting the read goroutine so it is always the
	// first event in the buffer (buffered channel → never blocks); otherwise
	// the reader can race an sse_data event ahead of Opened.
	conn.eventBuffer <- streamEvent{
		kind: "opened",
		text: "SSE",
	}

	// Start SSE read goroutine
	go conn.sseReadLoop()

	return makeStreamOk(makeStreamConn(id)), nil
}

// sseReadLoop reads SSE events from the HTTP response body.
//
// Follows the WHATWG Server-Sent Events spec:
//   - Empty line → dispatch accumulated event
//   - : prefix → comment, skip
//   - data: → append to data buffer (multi-line data joined with \n)
//   - event: → set event type
//   - id: → store last event ID
//   - retry: → reconnection interval (stored but not acted on)
//
// Produces streamEvent{kind: "sse_data", text: data, sseEventType: eventType}
// On EOF: delivers Closed event.
func (sc *StreamConnection) sseReadLoop() {
	defer func() {
		sc.mu.Lock()
		status := sc.status
		sc.mu.Unlock()
		if status != StreamStatusClosed && status != StreamStatusClosing {
			sc.eventBuffer <- streamEvent{kind: "closed", code: 0, reason: "SSE stream ended"}
		}
	}()

	scanner := bufio.NewScanner(sc.httpResp.Body)
	// Allow up to 1MB lines (some AI APIs send large JSON chunks)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var dataLines []string
	var eventType string
	var eventID string

	dispatch := func() {
		if len(dataLines) == 0 {
			return
		}
		data := strings.Join(dataLines, "\n")
		if eventType == "" {
			eventType = "message" // SSE default
		}

		sc.mu.Lock()
		sc.messagesRecv++
		sc.bytesRecv += int64(len(data))
		if eventID != "" {
			sc.lastEventID = eventID
		}
		sc.mu.Unlock()

		sc.eventBuffer <- streamEvent{
			kind:         "sse_data",
			text:         data,
			sseEventType: eventType,
			sseID:        eventID,
		}

		// Reset for next event
		dataLines = nil
		eventType = ""
		eventID = ""
	}

	for {
		select {
		case <-sc.done:
			return
		default:
		}

		if !scanner.Scan() {
			// EOF or read error
			if err := scanner.Err(); err != nil {
				sc.eventBuffer <- streamEvent{
					kind:    "error",
					errType: "ProtocolError",
					text:    fmt.Sprintf("SSE read error: %s", err.Error()),
				}
			}
			return
		}

		line := scanner.Text()

		// Empty line → dispatch event
		if line == "" {
			dispatch()
			continue
		}

		// Comment line — skip
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Parse field
		field, value := parseSSEField(line)
		switch field {
		case "data":
			dataLines = append(dataLines, value)
		case "event":
			eventType = value
		case "id":
			// Per spec: ignore id fields containing NUL
			if !strings.Contains(value, "\x00") {
				eventID = value
			}
		case "retry":
			// Store but don't act on — AILANG determinism axiom (no auto-reconnect)
		default:
			// Unknown field — ignore per spec
		}
	}
}

// parseSSEField splits an SSE line into field name and value.
// Per WHATWG spec:
//   - "field: value" → ("field", "value")
//   - "field:value"  → ("field", "value")
//   - "field:"       → ("field", "")
//   - "field"        → ("field", "")
func parseSSEField(line string) (string, string) {
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return line, ""
	}
	field := line[:idx]
	value := line[idx+1:]
	// Per spec: strip single leading space from value
	value = strings.TrimPrefix(value, " ")
	return field, value
}

// StreamSSEPost opens an SSE connection via HTTP POST.
//
// This is the standard pattern for AI API streaming: Claude, OpenAI, and Gemini
// all use POST+SSE where the request body contains the prompt/config and the
// response is streamed back as Server-Sent Events.
//
// Args: [url: string, body: string, config: record{headers}]
// Returns: Result[StreamConn(int), StreamError]
func StreamSSEPost(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("_stream_sse_post: expected 3 arguments, got %d", len(args))
	}

	urlVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_stream_sse_post: expected String for url, got %T", args[0])
	}

	bodyVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_stream_sse_post: expected String for body, got %T", args[1])
	}

	if ctx.Stream == nil {
		return nil, fmt.Errorf("E_STREAM_NO_CONTEXT: Stream effect not configured (missing --caps Stream)")
	}

	// Budget check
	if err := ctx.RequireCapWithBudget("Stream", "stream.sse_post"); err != nil {
		return makeStreamErr("BudgetExhausted", err.Error()), nil
	}

	// Validate URL
	if err := ctx.Stream.ValidateURL(urlVal.Value); err != nil {
		return makeStreamErr("ConnectionFailed", err.Error()), nil
	}

	// Parse config for custom headers
	headers := make(http.Header)
	headers.Set("Accept", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Content-Type", "application/json") // Default for AI APIs

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

	// HTTP POST with body
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: ctx.Stream.ConnectTimeout,
		}).DialContext,
		TLSHandshakeTimeout:   ctx.Stream.ConnectTimeout,
		ResponseHeaderTimeout: ctx.Stream.ConnectTimeout,
	}
	client := &http.Client{
		Transport: transport,
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", urlVal.Value, strings.NewReader(bodyVal.Value))
	if err != nil {
		return makeStreamErr("ConnectionFailed", fmt.Sprintf("SSE POST request creation failed: %s", err.Error())), nil
	}
	req.Header = headers

	resp, err := client.Do(req)
	if err != nil {
		return makeStreamErr("ConnectionFailed", fmt.Sprintf("SSE POST connection failed: %s", err.Error())), nil
	}

	// Verify content type
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		resp.Body.Close()
		return makeStreamErr("ConnectionFailed",
			fmt.Sprintf("SSE POST expected Content-Type text/event-stream, got %q (HTTP %d)", ct, resp.StatusCode)), nil
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return makeStreamErr("ConnectionFailed",
			fmt.Sprintf("SSE POST HTTP error: %d %s", resp.StatusCode, resp.Status)), nil
	}

	// Create connection — reuses shared SSE infrastructure
	conn := &StreamConnection{
		protocol:    "SSE",
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

	// Deliver Opened BEFORE starting the read goroutine so it is always the
	// first event in the buffer (buffered channel → never blocks); otherwise
	// the reader can race an sse_data event ahead of Opened.
	conn.eventBuffer <- streamEvent{
		kind: "opened",
		text: "SSE-POST",
	}

	// Start SSE read goroutine (reuses sseReadLoop from GET-SSE)
	go conn.sseReadLoop()

	return makeStreamOk(makeStreamConn(id)), nil
}

func init() {
	RegisterOp("Stream", "sse_connect", StreamSSEConnect)
	RegisterOp("Stream", "sse_post", StreamSSEPost)
}
