package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// Stream effect builtins for AILANG
// These provide bidirectional WebSocket streaming (M-STREAM-BIDI)

func init() {
	registerStreamConnect()
	registerStreamSSEConnect()
	registerStreamSSEPost()
	registerStreamSend()
	registerStreamTransmitBinary()
	registerStreamOnEvent()
	registerStreamRunEventLoop()
	registerStreamClose()
	registerStreamGetStatus()
}

// ============================================================================
// Stream Builtins
// ============================================================================

// registerStreamConnect registers the _stream_connect builtin
func registerStreamConnect() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/stream",
		Name:    "_stream_connect",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "Stream",
		Type:    makeStreamConnectType,
		Impl:    effects.StreamConnect,

		Metadata: &BuiltinMetadata{
			Description: "Open a WebSocket connection to a URL",
			LongDesc: `Establishes a WebSocket connection to the given URL with optional
configuration (headers, subprotocols). Returns a Result containing either a
StreamConn handle or a StreamError. The Stream capability must be granted.

Security: By default only wss:// is allowed. Use AllowHTTP for ws://.
Connections are limited to MaxConnections (default: 4).`,
			Params: []ParamDoc{
				{Name: "url", Description: "WebSocket URL (wss:// or ws:// if AllowHTTP)"},
				{Name: "config", Description: "Configuration record with optional headers and subprotocols"},
			},
			Returns:   "Result[StreamConn, StreamErrorKind]",
			Since:     "v0.8.1",
			Stability: StabilityExperimental,
			Tags:      []string{"stream", "websocket", "connect", "network"},
			Category:  "stream",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _stream_connect: %v", err))
	}
}

// streamConfigType builds the record type for stream config parameter:
// {headers: [{name: string, value: string}]}
// This matches what the Go runtime expects in stream.go and stream_sse.go.
func streamConfigType() types.Type {
	T := types.NewBuilder()
	headerEntry := T.Record(
		types.Field("name", T.String()),
		types.Field("value", T.String()),
	)
	return T.Record(
		types.Field("headers", T.List(headerEntry)),
	)
}

// makeStreamConnectType builds the type signature for _stream_connect
// Type: (string, {headers: [{name: string, value: string}]}) -> Result[StreamConn, StreamErrorKind] ! {Stream}
func makeStreamConnectType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.String(),         // url
		streamConfigType(), // config record with headers
	).Returns(
		T.App("Result", T.Con("StreamConn"), T.Con("StreamErrorKind")),
	).Effects("Stream")
}

// registerStreamSSEConnect registers the _stream_sse_connect builtin
func registerStreamSSEConnect() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/stream",
		Name:    "_stream_sse_connect",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "Stream",
		Type:    makeStreamSSEConnectType,
		Impl:    effects.StreamSSEConnect,

		Metadata: &BuiltinMetadata{
			Description: "Open an SSE (Server-Sent Events) connection to a URL",
			LongDesc: `Establishes a read-only SSE connection via HTTP GET with Accept: text/event-stream.
Used for consuming AI API streaming responses (Anthropic, OpenAI, Gemini).
Returns a Result containing either a StreamConn handle or a StreamError.
The connection shares onEvent/runEventLoop/disconnect with WebSocket connections.

Custom headers (e.g. Authorization: Bearer) are supported via the config record.
SSE connections are read-only — calling transmit() returns Err(ProtocolError).`,
			Params: []ParamDoc{
				{Name: "url", Description: "SSE endpoint URL (https:// or http:// if AllowHTTP)"},
				{Name: "config", Description: "Configuration record with optional headers list"},
			},
			Returns:   "Result[StreamConn, StreamErrorKind]",
			Since:     "v0.8.1",
			Stability: StabilityExperimental,
			Tags:      []string{"stream", "sse", "connect", "network", "ai"},
			Category:  "stream",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _stream_sse_connect: %v", err))
	}
}

// makeStreamSSEConnectType builds the type signature for _stream_sse_connect
// Type: (string, {headers: [{name: string, value: string}]}) -> Result[StreamConn, StreamErrorKind] ! {Stream}
func makeStreamSSEConnectType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.String(),         // url
		streamConfigType(), // config record with headers
	).Returns(
		T.App("Result", T.Con("StreamConn"), T.Con("StreamErrorKind")),
	).Effects("Stream")
}

// registerStreamSend registers the _stream_send builtin
func registerStreamSend() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/stream",
		Name:    "_stream_send",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "Stream",
		Type:    makeStreamSendType,
		Impl:    effects.StreamSend,

		Metadata: &BuiltinMetadata{
			Description: "Send a message on a WebSocket connection",
			LongDesc:    "Sends a text or binary message on an open WebSocket connection. Returns Ok(()) on success or Err(StreamError) on failure. Messages exceeding MaxMessageSize are rejected.",
			Params: []ParamDoc{
				{Name: "conn", Description: "StreamConn handle from connect"},
				{Name: "msg", Description: "StreamMessage: Text(string) or Bin(bytes)"},
			},
			Returns:   "Result[unit, StreamErrorKind]",
			Since:     "v0.8.1",
			Stability: StabilityExperimental,
			Tags:      []string{"stream", "websocket", "send", "message"},
			Category:  "stream",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _stream_send: %v", err))
	}
}

// makeStreamSendType builds the type signature for _stream_send
// Type: (StreamConn, string) -> Result[unit, StreamErrorKind] ! {Stream}
func makeStreamSendType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.Con("StreamConn"), // conn
		T.String(),          // msg (auto-wrapped as Text in Go runtime)
	).Returns(
		T.App("Result", T.Unit(), T.Con("StreamErrorKind")),
	).Effects("Stream")
}

// registerStreamOnEvent registers the _stream_onEvent builtin
func registerStreamOnEvent() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/stream",
		Name:    "_stream_onEvent",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "Stream",
		Type:    makeStreamOnEventType,
		Impl:    effects.StreamOnEvent,

		Metadata: &BuiltinMetadata{
			Description: "Register an event handler for a WebSocket connection",
			LongDesc:    "Registers a callback function to handle WebSocket events. The handler receives a StreamEvent and returns bool (true=continue, false=stop). Must be called before runEventLoop.",
			Params: []ParamDoc{
				{Name: "conn", Description: "StreamConn handle from connect"},
				{Name: "handler", Description: "Function: StreamEvent -> bool"},
			},
			Returns:   "unit",
			Since:     "v0.8.1",
			Stability: StabilityExperimental,
			Tags:      []string{"stream", "websocket", "event", "handler"},
			Category:  "stream",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _stream_onEvent: %v", err))
	}
}

// makeStreamOnEventType builds the type signature for _stream_onEvent
// Type: (StreamConn, (StreamEvent -> bool)) -> unit ! {Stream}
func makeStreamOnEventType() types.Type {
	T := types.NewBuilder()
	handlerType := T.Func(T.Con("StreamEvent")).Returns(T.Bool()).Build()
	return T.Func(
		T.Con("StreamConn"), // conn
		handlerType,         // handler: (StreamEvent -> bool)
	).Returns(
		T.Unit(),
	).Effects("Stream")
}

// registerStreamRunEventLoop registers the _stream_runEventLoop builtin
func registerStreamRunEventLoop() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/stream",
		Name:    "_stream_runEventLoop",
		NumArgs: 1,
		IsPure:  false,
		Effect:  "Stream",
		Type:    makeStreamRunEventLoopType,
		Impl:    effects.StreamRunEventLoop,

		Metadata: &BuiltinMetadata{
			Description: "Run the event loop, dispatching events to the registered handler",
			LongDesc: `Blocks the current thread, reading events from the WebSocket connection
and dispatching them to the registered handler (set via onEvent). Stops when:
- The handler returns false
- The idle timeout expires (no messages for IdleTimeout)
- The max duration ceiling is reached
- A handler panic is recovered and an Error event delivered`,
			Params: []ParamDoc{
				{Name: "conn", Description: "StreamConn handle with a registered handler"},
			},
			Returns:   "unit",
			Since:     "v0.8.1",
			Stability: StabilityExperimental,
			Tags:      []string{"stream", "websocket", "event", "loop", "blocking"},
			Category:  "stream",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _stream_runEventLoop: %v", err))
	}
}

// makeStreamRunEventLoopType builds the type signature for _stream_runEventLoop
// Type: StreamConn -> unit ! {Stream}
func makeStreamRunEventLoopType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.Con("StreamConn"), // conn
	).Returns(
		T.Unit(),
	).Effects("Stream")
}

// registerStreamClose registers the _stream_close builtin
func registerStreamClose() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/stream",
		Name:    "_stream_close",
		NumArgs: 1,
		IsPure:  false,
		Effect:  "Stream",
		Type:    makeStreamCloseType,
		Impl:    effects.StreamClose,

		Metadata: &BuiltinMetadata{
			Description: "Close a WebSocket connection gracefully",
			LongDesc:    "Sends a close frame and shuts down the connection. Safe to call multiple times (idempotent). Releases the connection slot.",
			Params: []ParamDoc{
				{Name: "conn", Description: "StreamConn handle to close"},
			},
			Returns:   "unit",
			Since:     "v0.8.1",
			Stability: StabilityExperimental,
			Tags:      []string{"stream", "websocket", "close"},
			Category:  "stream",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _stream_close: %v", err))
	}
}

// makeStreamCloseType builds the type signature for _stream_close
// Type: StreamConn -> unit ! {Stream}
func makeStreamCloseType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.Con("StreamConn"), // conn
	).Returns(
		T.Unit(),
	).Effects("Stream")
}

// registerStreamGetStatus registers the _stream_status builtin
func registerStreamGetStatus() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/stream",
		Name:    "_stream_status",
		NumArgs: 1,
		IsPure:  false,
		Effect:  "Stream",
		Type:    makeStreamGetStatusType,
		Impl:    effects.StreamGetStatus,

		Metadata: &BuiltinMetadata{
			Description: "Get the current status of a WebSocket connection",
			LongDesc:    "Returns the connection status as a StreamStatus ADT: Connecting, Open, Closing, or StreamClosed.",
			Params: []ParamDoc{
				{Name: "conn", Description: "StreamConn handle to check"},
			},
			Returns:   "StreamStatus: Connecting | Open | Closing | StreamClosed",
			Since:     "v0.8.1",
			Stability: StabilityExperimental,
			Tags:      []string{"stream", "websocket", "status"},
			Category:  "stream",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _stream_status: %v", err))
	}
}

// makeStreamGetStatusType builds the type signature for _stream_status
// Type: StreamConn -> StreamStatus ! {Stream}
func makeStreamGetStatusType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.Con("StreamConn"), // conn
	).Returns(
		T.Con("StreamStatus"),
	).Effects("Stream")
}

// registerStreamTransmitBinary registers the _stream_transmit_binary builtin
func registerStreamTransmitBinary() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/stream",
		Name:    "_stream_transmit_binary",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "Stream",
		Type:    makeStreamTransmitBinaryType,
		Impl:    streamTransmitBinaryImpl,

		Metadata: &BuiltinMetadata{
			Description: "Send binary data (bytes) on a WebSocket connection",
			LongDesc: `Sends raw binary data as a WebSocket binary frame. Wraps the bytes
in a Bin() StreamMessage ADT and delegates to the existing StreamSend handler.
Use this for PCM audio, images, or other binary data without base64 encoding overhead.`,
			Params: []ParamDoc{
				{Name: "conn", Description: "StreamConn handle from connect"},
				{Name: "data", Description: "Binary data to send"},
			},
			Returns:   "Result[unit, StreamErrorKind]",
			Since:     "v0.8.1",
			Stability: StabilityExperimental,
			Tags:      []string{"stream", "websocket", "binary", "send", "pcm"},
			Category:  "stream",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _stream_transmit_binary: %v", err))
	}
}

// makeStreamTransmitBinaryType builds the type signature for _stream_transmit_binary
// Type: (StreamConn, bytes) -> Result[unit, StreamErrorKind] ! {Stream}
func makeStreamTransmitBinaryType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.Con("StreamConn"), // conn
		T.Bytes(),           // data
	).Returns(
		T.App("Result", T.Unit(), T.Con("StreamErrorKind")),
	).Effects("Stream")
}

// streamTransmitBinaryImpl wraps bytes in Bin() ADT and delegates to StreamSend
func streamTransmitBinaryImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("_stream_transmit_binary: expected 2 arguments, got %d", len(args))
	}

	bytesVal, ok := args[1].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_stream_transmit_binary: expected Bytes for data, got %T", args[1])
	}

	// Wrap bytes in Bin() ADT — StreamSend already handles this variant
	binADT := &eval.TaggedValue{
		CtorName: "Bin",
		Fields:   []eval.Value{bytesVal},
	}

	// Delegate to existing StreamSend which handles Bin(bytes) at line 358
	return effects.StreamSend(ctx, []eval.Value{args[0], binADT})
}

// registerStreamSSEPost registers the _stream_sse_post builtin
func registerStreamSSEPost() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/stream",
		Name:    "_stream_sse_post",
		NumArgs: 3,
		IsPure:  false,
		Effect:  "Stream",
		Type:    makeStreamSSEPostType,
		Impl:    effects.StreamSSEPost,

		Metadata: &BuiltinMetadata{
			Description: "Open an SSE connection via HTTP POST (for AI API streaming)",
			LongDesc: `Sends an HTTP POST request with a JSON body and reads the response as an
SSE stream. This is the standard pattern for AI API streaming: Claude, OpenAI,
and Gemini all use POST+SSE where the request body contains the prompt/config
and the response is streamed back as Server-Sent Events.

The connection shares onEvent/runEventLoop/disconnect with WebSocket and GET-SSE connections.
Custom headers (e.g. Authorization: Bearer) are supported via the config parameter.
Content-Type defaults to application/json.`,
			Params: []ParamDoc{
				{Name: "url", Description: "SSE endpoint URL (https://)"},
				{Name: "body", Description: "Request body (typically JSON)"},
				{Name: "config", Description: "Configuration record with optional headers list"},
			},
			Returns:   "Result[StreamConn, StreamErrorKind]",
			Since:     "v0.8.1",
			Stability: StabilityExperimental,
			Tags:      []string{"stream", "sse", "post", "ai", "anthropic", "openai"},
			Category:  "stream",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _stream_sse_post: %v", err))
	}
}

// makeStreamSSEPostType builds the type signature for _stream_sse_post
// Type: (string, string, {headers: [{name: string, value: string}]}) -> Result[StreamConn, StreamErrorKind] ! {Stream}
func makeStreamSSEPostType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.String(),         // url
		T.String(),         // body
		streamConfigType(), // config record with headers
	).Returns(
		T.App("Result", T.Con("StreamConn"), T.Con("StreamErrorKind")),
	).Effects("Stream")
}
