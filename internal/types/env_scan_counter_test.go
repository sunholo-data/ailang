package types

import "testing"

// The env-scan counter is the deterministic work measure behind the linearity
// test (M-TYPECHECK-ENV-SCAN-COST): one tick per binding a free-variable walk
// visits. Not parallel: the counter is package-global.
func TestEnvScanWork_CountsEachBindingVisited(t *testing.T) {
	root := NewTypeEnv()
	root.BindType("a", &TVar2{Name: "α1", Kind: Star})
	root.BindType("b", TInt)
	mid := root.Extend("c", &TVar2{Name: "α2", Kind: Star})
	top := mid.ExtendScheme("d", &Scheme{TypeVars: []string{"β"}, Type: &TVar2{Name: "β", Kind: Star}})

	ResetEnvScanWork()
	free := top.FreeTypeVars()
	if got := EnvScanWork(); got != 4 {
		t.Fatalf("FreeTypeVars over 4 bindings in 3 frames visited %d", got)
	}
	if !free["α1"] || !free["α2"] || free["β"] {
		t.Fatalf("free vars = %v, want α1 and α2 but not the quantified β", free)
	}

	ResetEnvScanWork()
	_ = top.FreeTypeVarsAbove(mid)
	if got := EnvScanWork(); got != 1 {
		t.Fatalf("FreeTypeVarsAbove(mid) visited %d bindings, want only the top frame's 1", got)
	}
}
