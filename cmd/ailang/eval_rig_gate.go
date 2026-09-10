package main

import (
	"context"
	"sync"

	"github.com/sunholo-data/ailang/internal/riglock"
)

// evalRigGate keeps read ownership through the complete trial (including tools).
// A priority handoff takes write ownership, draining every in-flight trial.
type evalRigGate struct {
	mu      sync.RWMutex
	err     error
	pending func() bool
	yield   func(context.Context) error
}

func newEvalRigGate() *evalRigGate {
	return &evalRigGate{
		pending: func() bool { return riglock.HeldByAncestor() && riglock.PriorityPending() },
		yield:   riglock.Yield,
	}
}

func (g *evalRigGate) enter(ctx context.Context) (func(), error) {
	g.mu.RLock()
	if g.pending() {
		g.mu.RUnlock()
		g.mu.Lock()
		if g.err == nil {
			g.err = g.yield(ctx)
		}
		g.mu.Unlock()
		g.mu.RLock()
	}
	if g.err != nil {
		g.mu.RUnlock()
		return func() {}, g.err
	}
	return g.mu.RUnlock, nil
}
