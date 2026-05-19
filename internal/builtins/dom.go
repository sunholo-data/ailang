package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// DOM effect builtins for AILANG (M-COG-RUNTIME, v0.21.x).
//
// These wire the AILANG-callable surface to the effects-registered ops
// in internal/effects/dom.go. Same pattern as internal/builtins/env.go:
// each builtin has a Type function (declaring the AILANG signature) and an
// Impl function (dispatching to the registered op via effects.Call).
//
// AILANG bindings live in std/dom.ail.

func init() {
	registerDOMApplyPatch()
	registerDOMApplyBatch()
	registerDOMApplyPatchResult()
	registerDOMApplyBatchResult()
}

// ============================================================================
// Shared type-builder helpers (DOM + Msg)
// ============================================================================

// makeCognitionErrorType builds the record type {code: string, message: string}
// used as the Err side of every Result-returning Cognitive OS op.
func makeCognitionErrorType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("code", T.String()),
		types.Field("message", T.String()),
	)
}

// makeDOMPatchType returns the opaque T.Con("DOMPatch") used in builtin
// signatures. The actual ADT is declared in std/dom.ail; the type checker
// resolves the name when the stdlib loads.
func makeDOMPatchType(T *types.Builder) types.Type {
	return T.Con("DOMPatch")
}

// ============================================================================
// _dom_apply_patch — bare op
// ============================================================================

func registerDOMApplyPatch() {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "DOM", "applyPatch", args)
	}
	typ := func() types.Type {
		T := types.NewBuilder()
		patchResultRec := T.Record(
			types.Field("node_id", T.String()),
			types.Field("budget_remaining", T.Int()),
		)
		return T.Func(T.String(), makeDOMPatchType(T)).Returns(patchResultRec).Effects("DOM")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/dom", Name: "_dom_apply_patch", NumArgs: 2,
		IsPure: false, Effect: "DOM", Type: typ, Impl: impl,
		Metadata: &BuiltinMetadata{
			Description: "Apply a single DOM patch atomically to an agent's scoped region",
			Params: []ParamDoc{
				{Name: "region", Description: "Agent-scoped region identifier"},
				{Name: "patch", Description: "Structured DOMPatch variant"},
			},
			Returns:   "Record {node_id: string, budget_remaining: int}",
			Since:     "v0.21.0",
			Stability: StabilityExperimental,
			Tags:      []string{"dom", "cognitive-os", "patch"},
			Category:  "cognitive-os",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _dom_apply_patch: %v", err))
	}
}

// ============================================================================
// _dom_apply_batch — bare op
// ============================================================================

func registerDOMApplyBatch() {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "DOM", "applyBatch", args)
	}
	typ := func() types.Type {
		T := types.NewBuilder()
		batchResultRec := T.Record(
			types.Field("node_ids", T.List(T.String())),
			types.Field("budget_remaining", T.Int()),
		)
		return T.Func(T.String(), T.List(makeDOMPatchType(T))).Returns(batchResultRec).Effects("DOM")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/dom", Name: "_dom_apply_batch", NumArgs: 2,
		IsPure: false, Effect: "DOM", Type: typ, Impl: impl,
		Metadata: &BuiltinMetadata{
			Description: "Apply a sequence of DOM patches transactionally — all or nothing",
			Params: []ParamDoc{
				{Name: "region", Description: "Agent-scoped region identifier"},
				{Name: "patches", Description: "List of structured DOMPatch variants"},
			},
			Returns:   "Record {node_ids: [string], budget_remaining: int}",
			Since:     "v0.21.0",
			Stability: StabilityExperimental,
			Tags:      []string{"dom", "cognitive-os", "patch", "batch"},
			Category:  "cognitive-os",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _dom_apply_batch: %v", err))
	}
}

// ============================================================================
// _dom_apply_patch_result — Result-returning variant
// ============================================================================

func registerDOMApplyPatchResult() {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "DOM", "applyPatchResult", args)
	}
	typ := func() types.Type {
		T := types.NewBuilder()
		patchResultRec := T.Record(
			types.Field("node_id", T.String()),
			types.Field("budget_remaining", T.Int()),
		)
		errRec := makeCognitionErrorType(T)
		resultType := T.App("Result", patchResultRec, errRec)
		return T.Func(T.String(), makeDOMPatchType(T)).Returns(resultType).Effects("DOM")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/dom", Name: "_dom_apply_patch_result", NumArgs: 2,
		IsPure: false, Effect: "DOM", Type: typ, Impl: impl,
		Metadata: &BuiltinMetadata{
			Description: "Result-returning variant of applyPatch — never panics; surfaces failures as Err({code, message})",
			Params: []ParamDoc{
				{Name: "region", Description: "Agent-scoped region identifier"},
				{Name: "patch", Description: "Structured DOMPatch variant"},
			},
			Returns:   "Result[{node_id, budget_remaining}, {code, message}]",
			Since:     "v0.21.0",
			Stability: StabilityExperimental,
			Tags:      []string{"dom", "cognitive-os", "result"},
			Category:  "cognitive-os",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _dom_apply_patch_result: %v", err))
	}
}

// ============================================================================
// _dom_apply_batch_result — Result-returning variant
// ============================================================================

func registerDOMApplyBatchResult() {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "DOM", "applyBatchResult", args)
	}
	typ := func() types.Type {
		T := types.NewBuilder()
		batchResultRec := T.Record(
			types.Field("node_ids", T.List(T.String())),
			types.Field("budget_remaining", T.Int()),
		)
		errRec := makeCognitionErrorType(T)
		resultType := T.App("Result", batchResultRec, errRec)
		return T.Func(T.String(), T.List(makeDOMPatchType(T))).Returns(resultType).Effects("DOM")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/dom", Name: "_dom_apply_batch_result", NumArgs: 2,
		IsPure: false, Effect: "DOM", Type: typ, Impl: impl,
		Metadata: &BuiltinMetadata{
			Description: "Result-returning variant of applyBatch — never panics; surfaces failures as Err({code, message})",
			Params: []ParamDoc{
				{Name: "region", Description: "Agent-scoped region identifier"},
				{Name: "patches", Description: "List of structured DOMPatch variants"},
			},
			Returns:   "Result[{node_ids, budget_remaining}, {code, message}]",
			Since:     "v0.21.0",
			Stability: StabilityExperimental,
			Tags:      []string{"dom", "cognitive-os", "result", "batch"},
			Category:  "cognitive-os",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _dom_apply_batch_result: %v", err))
	}
}
