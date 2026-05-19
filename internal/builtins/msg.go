package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Msg effect builtins for AILANG (M-COG-RUNTIME, v0.21.x).
//
// These wire the AILANG-callable surface to the effects-registered ops
// in internal/effects/msg.go. AILANG bindings live in std/cognition.ail.
//
// makeCognitionErrorType helper is shared with dom.go (same package).

func init() {
	registerMsgSend()
	registerMsgRecv()
	registerMsgSendResult()
	registerMsgRecvResult()
}

// ============================================================================
// _msg_send — bare op
// ============================================================================

func registerMsgSend() {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "Msg", "send", args)
	}
	typ := func() types.Type {
		T := types.NewBuilder()
		sendResultRec := T.Record(
			types.Field("msg_id", T.String()),
			types.Field("clock", T.Int()),
			types.Field("budget_remaining", T.Int()),
		)
		return T.Func(T.String(), T.String()).Returns(sendResultRec).Effects("Msg")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/cognition", Name: "_msg_send", NumArgs: 2,
		IsPure: false, Effect: "Msg", Type: typ, Impl: impl,
		Metadata: &BuiltinMetadata{
			Description: "Send a payload to a mailbox; returns assigned msg_id, Lamport clock, and remaining budget",
			Params: []ParamDoc{
				{Name: "to", Description: "Mailbox name (NodeID or topic)"},
				{Name: "payload", Description: "Message body as string"},
			},
			Returns:   "Record {msg_id: string, clock: int, budget_remaining: int}",
			Since:     "v0.21.0",
			Stability: StabilityExperimental,
			Tags:      []string{"msg", "cognitive-os", "send"},
			Category:  "cognitive-os",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _msg_send: %v", err))
	}
}

// ============================================================================
// _msg_recv — bare op
// ============================================================================

func registerMsgRecv() {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "Msg", "recv", args)
	}
	typ := func() types.Type {
		T := types.NewBuilder()
		msgRec := T.Record(
			types.Field("msg_id", T.String()),
			types.Field("from", T.String()),
			types.Field("to", T.String()),
			types.Field("payload", T.String()),
			types.Field("clock", T.Int()),
		)
		return T.Func(T.String()).Returns(msgRec).Effects("Msg")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/cognition", Name: "_msg_recv", NumArgs: 1,
		IsPure: false, Effect: "Msg", Type: typ, Impl: impl,
		Metadata: &BuiltinMetadata{
			Description: "Receive the next message from a mailbox",
			Params: []ParamDoc{
				{Name: "mailbox", Description: "Mailbox name to read from"},
			},
			Returns:   "Record {msg_id, from, to, payload, clock}",
			Since:     "v0.21.0",
			Stability: StabilityExperimental,
			Tags:      []string{"msg", "cognitive-os", "recv"},
			Category:  "cognitive-os",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _msg_recv: %v", err))
	}
}

// ============================================================================
// _msg_send_result — Result-returning variant
// ============================================================================

func registerMsgSendResult() {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "Msg", "sendResult", args)
	}
	typ := func() types.Type {
		T := types.NewBuilder()
		sendResultRec := T.Record(
			types.Field("msg_id", T.String()),
			types.Field("clock", T.Int()),
			types.Field("budget_remaining", T.Int()),
		)
		errRec := makeCognitionErrorType(T)
		resultType := T.App("Result", sendResultRec, errRec)
		return T.Func(T.String(), T.String()).Returns(resultType).Effects("Msg")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/cognition", Name: "_msg_send_result", NumArgs: 2,
		IsPure: false, Effect: "Msg", Type: typ, Impl: impl,
		Metadata: &BuiltinMetadata{
			Description: "Result-returning variant of send — never panics; surfaces failures as Err({code, message})",
			Params: []ParamDoc{
				{Name: "to", Description: "Mailbox name (NodeID or topic)"},
				{Name: "payload", Description: "Message body as string"},
			},
			Returns:   "Result[{msg_id, clock, budget_remaining}, {code, message}]",
			Since:     "v0.21.0",
			Stability: StabilityExperimental,
			Tags:      []string{"msg", "cognitive-os", "result"},
			Category:  "cognitive-os",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _msg_send_result: %v", err))
	}
}

// ============================================================================
// _msg_recv_result — Result-returning variant
// ============================================================================

func registerMsgRecvResult() {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "Msg", "recvResult", args)
	}
	typ := func() types.Type {
		T := types.NewBuilder()
		msgRec := T.Record(
			types.Field("msg_id", T.String()),
			types.Field("from", T.String()),
			types.Field("to", T.String()),
			types.Field("payload", T.String()),
			types.Field("clock", T.Int()),
		)
		errRec := makeCognitionErrorType(T)
		resultType := T.App("Result", msgRec, errRec)
		return T.Func(T.String()).Returns(resultType).Effects("Msg")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/cognition", Name: "_msg_recv_result", NumArgs: 1,
		IsPure: false, Effect: "Msg", Type: typ, Impl: impl,
		Metadata: &BuiltinMetadata{
			Description: "Result-returning variant of recv — empty mailbox (stub) and transport-close surface as Err",
			Params: []ParamDoc{
				{Name: "mailbox", Description: "Mailbox name to read from"},
			},
			Returns:   "Result[{msg_id, from, to, payload, clock}, {code, message}]",
			Since:     "v0.21.0",
			Stability: StabilityExperimental,
			Tags:      []string{"msg", "cognitive-os", "result", "recv"},
			Category:  "cognitive-os",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _msg_recv_result: %v", err))
	}
}
