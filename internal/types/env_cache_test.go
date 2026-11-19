package types

import "testing"

// Cache behaviour of environment free-variable queries (env_cache.go). Not
// parallel: these read the package-global work counter.

func TestEnvCache_SecondQueryVisitsNothing(t *testing.T) {
	defer SetEnvVarCrossCheck(SetEnvVarCrossCheck(false)) // counts work; the cross-check adds its own
	_, _, top := threeFrameEnv()
	first := top.FreeTypeVars()
	ResetEnvScanWork()
	second := top.FreeTypeVars()
	_ = top.FreeRowVars()
	_ = top.FreeEffectRowVars()
	if got := EnvScanWork(); got != 0 {
		t.Fatalf("repeat queries re-walked %d bindings", got)
	}
	if !sameVarSet(first, second) || !second["α1"] || !second["α2"] || second["β"] {
		t.Fatalf("cached = %v, first = %v", second, first)
	}
}

func TestEnvCache_ExtendReusesAncestors(t *testing.T) {
	defer SetEnvVarCrossCheck(SetEnvVarCrossCheck(false)) // counts work; the cross-check adds its own
	_, _, top := threeFrameEnv()
	_ = top.FreeTypeVars()
	child := top.Extend("e", &TVar2{Name: "α3", Kind: Star})
	ResetEnvScanWork()
	free := child.FreeTypeVars()
	if got := EnvScanWork(); got != 1 { // e only; one walk fills all three sets
		t.Fatalf("a new frame over a cached chain walked %d bindings, want 1", got)
	}
	if !free["α1"] || !free["α2"] || !free["α3"] {
		t.Fatalf("free = %v", free)
	}
}

// BindScheme/BindType are the in-place writers: they must invalidate the frame
// and every descendant's chain.
func TestEnvCache_BindInvalidatesFrameAndDescendants(t *testing.T) {
	root, mid, top := threeFrameEnv()
	if top.FreeTypeVars()["α9"] || mid.FreeTypeVars()["α9"] {
		t.Fatal("α9 free before it was bound")
	}
	root.BindType("z", &TVar2{Name: "α9", Kind: Star})
	if !root.FreeTypeVars()["α9"] {
		t.Fatal("frame cache not invalidated by BindType")
	}
	if !top.FreeTypeVars()["α9"] || !mid.FreeTypeVars()["α9"] {
		t.Fatal("descendant chains did not see an ancestor's BindType")
	}
	mid.BindScheme("y", &Scheme{Type: &TVar2{Name: "α8", Kind: Star}})
	if !top.FreeTypeVars()["α8"] || root.FreeTypeVars()["α8"] {
		t.Fatal("BindScheme on mid: top must see α8, root must not")
	}
}

func TestEnvCache_ReturnedSetsAreCopies(t *testing.T) {
	_, _, top := threeFrameEnv()
	got := top.FreeTypeVars()
	got["intruder"] = true
	delete(got, "α1")
	again := top.FreeTypeVars()
	if again["intruder"] || !again["α1"] {
		t.Fatalf("mutating a returned set changed the cache: %v", again)
	}
}

func TestEnvCache_WindowedQueriesUseOwnFrames(t *testing.T) {
	_, mid, top := threeFrameEnv()
	above := top.FreeTypeVarsAbove(mid)
	if len(above) != 0 { // top binds only the quantified β
		t.Fatalf("FreeTypeVarsAbove(mid) = %v, want empty", above)
	}
	child := top.Extend("e", &TVar2{Name: "α3", Kind: Star})
	if got := child.FreeTypeVarsAbove(mid); len(got) != 1 || !got["α3"] {
		t.Fatalf("FreeTypeVarsAbove(mid) from child = %v, want {α3}", got)
	}
	if got := child.FreeTypeVarsAbove(NewTypeEnv()); len(got) != 0 {
		t.Fatalf("non-ancestor base must give an empty set, got %v", got)
	}
}

func TestEnvCache_CrossCheckCatchesStaleCache(t *testing.T) {
	_, _, top := threeFrameEnv()
	_ = top.FreeTypeVars()
	top.chain.types["stale"] = true // simulate a missed invalidation
	prev := SetEnvVarCrossCheck(true)
	defer SetEnvVarCrossCheck(prev)
	defer func() {
		if recover() == nil {
			t.Fatal("cross-check did not catch a stale cached set")
		}
	}()
	_ = top.FreeTypeVars()
}
