package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// _cog_drain builtin (M-COG-RUNTIME-BROWSER M4).
//
// AILANG signature:
//
//   func drain(timeout_ms: int) -> int ! {Cog}
//
// Pumps the per-EffContext pending callback queue. Each queued
// (closure, event) pair invokes via ctx.FnCaller on the evaluator's
// own goroutine — solves the "AILANG closure from background goroutine"
// problem for Subscribe callbacks.
//
// See internal/effects/cog_drain.go for the queue mechanics.

func init() {
	registerCogDrain()
	registerDOMSubscribe()
	registerMsgSubscribe()
}

func registerCogDrain() {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "Cog", "drain", args)
	}
	typ := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Int()).Returns(T.Int()).Effects("Cog")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/cognition", Name: "_cog_drain", NumArgs: 1,
		IsPure: false, Effect: "Cog", Type: typ, Impl: impl,
		Metadata: &BuiltinMetadata{
			Description: "Pump pending Subscribe callbacks via FnCaller; returns count dispatched",
			Params: []ParamDoc{
				{Name: "timeout_ms", Description: "0=non-blocking; >0=block up to ms; <0=block until at least one event"},
			},
			Returns:   "int — count of callbacks dispatched",
			Since:     "v0.21.0",
			Stability: StabilityExperimental,
			Tags:      []string{"cognitive-os", "subscribe", "drain"},
			Category:  "cognitive-os",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _cog_drain: %v", err))
	}
}

func registerDOMSubscribe() {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "DOM", "subscribe", args)
	}
	typ := func() types.Type {
		T := types.NewBuilder()
		// (region: string, eventTypes: [string], callback: <opaque function>) -> () ! DOM
		// Callback type is intentionally opaque — the AILANG side passes
		// an eval.Value (closure), which the builtin's Impl forwards
		// verbatim. The type checker treats it as `α → ()` (any function
		// returning unit); the actual DOMEvent shape is documented in
		// stdlib/std/dom.ail.
		callbackType := T.Func(T.Con("DOMEvent")).Returns(T.Unit()).Effects("DOM", "Cog")
		return T.Func(T.String(), T.List(T.String()), callbackType).Returns(T.Unit()).Effects("DOM", "Cog")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/dom", Name: "_dom_subscribe", NumArgs: 3,
		IsPure: false, Effect: "DOM", Type: typ, Impl: impl,
		Metadata: &BuiltinMetadata{
			Description: "Register a callback for DOM events in a scoped region (callbacks fire via _cog_drain)",
			Params: []ParamDoc{
				{Name: "region", Description: "Agent-scoped region ID"},
				{Name: "eventTypes", Description: "List of event types to subscribe to (e.g. [\"click\", \"input\"])"},
				{Name: "callback", Description: "AILANG function (DOMEvent) -> () ! {DOM, Cog}"},
			},
			Returns:   "Unit",
			Since:     "v0.21.0",
			Stability: StabilityExperimental,
			Tags:      []string{"cognitive-os", "dom", "subscribe"},
			Category:  "cognitive-os",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _dom_subscribe: %v", err))
	}
}

func registerMsgSubscribe() {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "Msg", "subscribe", args)
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
		callbackType := T.Func(msgRec).Returns(T.Unit()).Effects("Msg", "Cog")
		return T.Func(T.String(), callbackType).Returns(T.Unit()).Effects("Msg", "Cog")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/cognition", Name: "_msg_subscribe", NumArgs: 2,
		IsPure: false, Effect: "Msg", Type: typ, Impl: impl,
		Metadata: &BuiltinMetadata{
			Description: "Register a callback for mailbox arrivals (callbacks fire via _cog_drain)",
			Params: []ParamDoc{
				{Name: "mailbox", Description: "Mailbox to subscribe to"},
				{Name: "callback", Description: "AILANG function (Message) -> () ! {Msg, Cog}"},
			},
			Returns:   "Unit",
			Since:     "v0.21.0",
			Stability: StabilityExperimental,
			Tags:      []string{"cognitive-os", "msg", "subscribe"},
			Category:  "cognitive-os",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _msg_subscribe: %v", err))
	}
}
