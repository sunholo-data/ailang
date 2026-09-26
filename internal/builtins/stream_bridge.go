package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-SERVEAPI-WS-BRIDGE M2: _stream_bridge, the verdict fold over two
// WebSocket connections. The Go loop lives in internal/effects/stream_bridge.go.

func init() {
	registerStreamBridge()
}

func registerStreamBridge() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/stream/bridge",
		Name:    "_stream_bridge",
		NumArgs: 4,
		IsPure:  false,
		Effect:  "Stream",
		Type:    makeStreamBridgeType,
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			return effects.Call(ctx, "Stream", "bridge", args)
		},
		Metadata: &BuiltinMetadata{
			Description: "Relay frames between two WebSocket connections with a per-frame verdict",
			LongDesc: `Dequeues one frame at a time from either connection and calls step(state, frame).
The step returns the next state and a Verdict: Forward re-sends the original bytes to the
other side, Drop sends nothing, ReplaceText/ReplaceBin send a replacement, CloseBridge(code,
reason) closes the client with that code. Returns the final state and a BridgeEnd. Every
data frame charges Stream.recv; every frame sent charges Stream.send.`,
			Params: []ParamDoc{
				{Name: "client", Description: "The browser leg (a WS route's StreamConn)"},
				{Name: "up", Description: "The upstream leg (from connect)"},
				{Name: "init", Description: "Initial policy state"},
				{Name: "step", Description: "Function: (state, BridgeFrame) -> (state, Verdict)"},
			},
			Returns:   "(s, BridgeEnd)",
			Since:     "v0.44.0",
			Stability: StabilityExperimental,
			Tags:      []string{"stream", "websocket", "bridge", "relay", "serve-api"},
			Category:  "stream",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _stream_bridge: %v", err))
	}
}

// makeStreamBridgeType builds
// forall s. (StreamConn, StreamConn, s, (s, BridgeFrame) -> (s, Verdict)) -> (s, BridgeEnd) ! {Stream}
func makeStreamBridgeType() types.Type {
	T := types.NewBuilder()
	s := T.Var("s")
	stepOut := &types.TTuple{Elements: []types.Type{s, T.Con("Verdict")}}
	step := T.Func(s, T.Con("BridgeFrame")).Returns(stepOut).Build()
	return T.Func(
		T.Con("StreamConn"),
		T.Con("StreamConn"),
		s,
		step,
	).Returns(
		&types.TTuple{Elements: []types.Type{s, T.Con("BridgeEnd")}},
	).Effects("Stream")
}
