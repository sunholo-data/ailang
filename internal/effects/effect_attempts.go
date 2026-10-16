package effects

import "sync/atomic"

// effectAttempts counts capability checks. Every effect operation checks its
// capability before acting, so a count that moved means a side effect may have
// happened — including a denied one. The bytecode runner compares it across a
// VM run: evaluator-only callees reached through the bridge perform IO that the
// VM's own EffectCalls never sees, and once one has, falling back to the
// evaluator would replay the entry (duplicated output, consumed stdin; #1576).
type effectAttempts struct{ n atomic.Uint64 }

func (a *effectAttempts) record() {
	if a != nil {
		a.n.Add(1)
	}
}

// EffectAttempts returns the number of capability checks made so far in this
// execution. Only differences are meaningful; contexts built without
// NewEffContext report 0.
func (ctx *EffContext) EffectAttempts() uint64 {
	if ctx == nil || ctx.attempts == nil {
		return 0
	}
	return ctx.attempts.n.Load()
}
