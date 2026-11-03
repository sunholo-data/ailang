package types

import (
	"fmt"
	"sort"
	"sync/atomic"
)

// Free-variable sets of a type environment, memoised per frame
// (M-TYPECHECK-ENV-SCAN-COST).
//
// inferApp, generalize and every declaration asked "which variables does the
// enclosing environment own?" by walking every frame to the root — the
// builtin/prelude frame alone is ~930 closed schemes — once per application,
// three times over. A frame's bindings never change after Extend/ExtendScheme
// except through BindScheme/BindType, so each frame computes its own sets once
// and a cumulative chain set (own ∪ parent's chain) once.
//
// Invariants:
//   - envVarSets are immutable once published; callers that need to write get
//     a copy (FreeTypeVars & co.).
//   - own sets are computed by the single-frame form of the original walks, so
//     their definition cannot drift from the uncached one.
//   - BindScheme/BindType clear the frame's caches and bump envEpoch; a chain
//     built under an older epoch revalidates against its parent's stamp.

// envVarSets holds the free type, row and effect-row variables of one frame
// (own) or of a frame and all its ancestors (chain).
type envVarSets struct {
	types      map[string]bool
	rows       map[string]bool
	effectRows map[string]bool
	stamp      uint64 // unique per built set; children record the parent's
}

func (s *envVarSets) empty() bool {
	return len(s.types) == 0 && len(s.rows) == 0 && len(s.effectRows) == 0
}

var (
	// envEpoch is bumped by every in-place binding write.
	envEpoch atomic.Uint64
	// envStamp issues envVarSets stamps.
	envStamp atomic.Uint64
	// envCrossCheck recomputes every cached query uncached and panics on a
	// mismatch. Tests turn it on (SetEnvVarCrossCheck); production never does.
	envCrossCheck atomic.Bool
)

// SetEnvVarCrossCheck turns the uncached cross-check of environment
// free-variable queries on or off and returns the previous setting. Test-only:
// it re-walks the whole environment on every query.
func SetEnvVarCrossCheck(on bool) bool { return envCrossCheck.Swap(on) }

// invalidateVarCaches is called by the in-place binding writers.
func (env *TypeEnv) invalidateVarCaches() {
	env.own, env.chain = nil, nil
	envEpoch.Add(1)
}

// ownSets returns the free variables of this frame's bindings only.
func (env *TypeEnv) ownSets() *envVarSets {
	if env.own == nil {
		frame := &TypeEnv{bindings: env.bindings}
		s := &envVarSets{
			types:      make(map[string]bool),
			rows:       make(map[string]bool),
			effectRows: make(map[string]bool),
			stamp:      envStamp.Add(1),
		}
		frame.collectFreeTypeVars(s.types)
		frame.collectFreeRowVars(s.rows)
		frame.collectFreeEffectRowVars(s.effectRows)
		env.own = s
	}
	return env.own
}

// chainSets returns the free variables of this frame and all its ancestors.
func (env *TypeEnv) chainSets() *envVarSets {
	epoch := envEpoch.Load()
	if env.chain != nil && env.chainEpoch == epoch {
		return env.chain
	}
	own := env.ownSets()
	if env.parent == nil {
		env.chain, env.chainEpoch = own, epoch
		return own
	}
	parent := env.parent.chainSets()
	if env.chain != nil && env.chainParent == parent.stamp {
		env.chainEpoch = epoch
		return env.chain
	}
	chain := parent // a frame of closed schemes adds nothing: share the parent's sets
	if !own.empty() {
		chain = &envVarSets{
			types:      unionVarSets(parent.types, own.types),
			rows:       unionVarSets(parent.rows, own.rows),
			effectRows: unionVarSets(parent.effectRows, own.effectRows),
			stamp:      envStamp.Add(1),
		}
	}
	env.chain, env.chainParent, env.chainEpoch = chain, parent.stamp, epoch
	return chain
}

// ownedVars returns the cached free-variable sets of the whole environment.
// The result is shared and must not be modified.
func (env *TypeEnv) ownedVars() *envVarSets {
	sets := env.chainSets()
	if envCrossCheck.Load() {
		env.crossCheckVarSets(sets)
	}
	return sets
}

// framesAbove returns the frames from env (inclusive) up to base (exclusive),
// and false when base is not an ancestor of env.
func (env *TypeEnv) framesAbove(base *TypeEnv) ([]*TypeEnv, bool) {
	var frames []*TypeEnv
	for e := env; e != base; e = e.parent {
		if e == nil {
			return nil, false
		}
		frames = append(frames, e)
	}
	return frames, true
}

func unionVarSets(a, b map[string]bool) map[string]bool {
	m := make(map[string]bool, len(a)+len(b))
	for v := range a {
		m[v] = true
	}
	for v := range b {
		m[v] = true
	}
	return m
}

func copyVarSet(s map[string]bool) map[string]bool {
	return unionVarSets(s, nil)
}

// crossCheckVarSets compares cached sets with the original whole-chain walks.
func (env *TypeEnv) crossCheckVarSets(sets *envVarSets) {
	want := map[string]map[string]bool{"type": {}, "row": {}, "effect-row": {}}
	env.collectFreeTypeVars(want["type"])
	env.collectFreeRowVars(want["row"])
	env.collectFreeEffectRowVars(want["effect-row"])
	got := map[string]map[string]bool{"type": sets.types, "row": sets.rows, "effect-row": sets.effectRows}
	for kind, w := range want {
		if !sameVarSet(w, got[kind]) {
			panic(fmt.Sprintf("env free %s vars: cached %v, uncached %v (frame depth %d)",
				kind, sortedVars(got[kind]), sortedVars(w), env.depth()))
		}
	}
}

func sameVarSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for v := range a {
		if !b[v] {
			return false
		}
	}
	return true
}

func sortedVars(s map[string]bool) []string {
	out := make([]string, 0, len(s))
	for v := range s {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func (env *TypeEnv) depth() int {
	n := 0
	for e := env; e != nil; e = e.parent {
		n++
	}
	return n
}
