package effects

import "sync/atomic"

// M-NET-SCOPE-PUBLIC (#1522, S0 of m-mcp-oauth-package): the declared Net
// scope reaches the destination policy the way the declared Rand mode reaches
// the Rand builtins (rand_mode.go) — the evaluator pushes it at the entry of a
// frame whose type declares !{Net[scope=public]} and pops it on exit, and
// netPolicy reads it at request time.
//
// "public" is the only scope, so the state is a depth counter, not a stack:
// the policy is public while ANY public frame of this execution is active.
// Bare Net pushes nothing, so a callee can never widen a public caller.
//
// Lifecycle mirrors randModeState: held behind a pointer so WithBudget scopes
// (the same execution) share it and Clone (a new request) resets it.
//
// Concurrency: tasks sharing one EffContext share the counter, so a
// concurrent bare-Net task on the SAME context is also public while a public
// frame runs. That is over-restrictive, never under-restrictive.
type netScopeState struct {
	publicDepth atomic.Int32
}

// NetScopeValuePublic is the scope value that forces the public-only policy.
const NetScopeValuePublic = "public"

func (ctx *EffContext) netScopeStateFor() *netScopeState {
	if ctx.netScope == nil {
		ctx.netScope = &netScopeState{}
	}
	return ctx.netScope
}

// PushNetScope enters a frame that declared Net[scope=scope]. Only "public"
// changes anything; "" (bare Net) is a no-op, keeping push/pop balanced
// without the caller tracking whether a push happened.
func (ctx *EffContext) PushNetScope(scope string) {
	if scope != NetScopeValuePublic {
		return
	}
	ctx.netScopeStateFor().publicDepth.Add(1)
}

// PopNetScope leaves a frame entered with PushNetScope(scope).
func (ctx *EffContext) PopNetScope(scope string) {
	if scope != NetScopeValuePublic || ctx.netScope == nil {
		return
	}
	for {
		d := ctx.netScope.publicDepth.Load()
		if d <= 0 || ctx.netScope.publicDepth.CompareAndSwap(d, d-1) {
			return
		}
	}
}

// NetScopePublic reports whether a Net[scope=public] frame is active, in
// which case netPolicy forces allowLocalhost and allowMetadata off.
func (ctx *EffContext) NetScopePublic() bool {
	return ctx != nil && ctx.netScope != nil && ctx.netScope.publicDepth.Load() > 0
}
