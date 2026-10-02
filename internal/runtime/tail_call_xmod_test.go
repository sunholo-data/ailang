package runtime

import "testing"

// M-EVAL-TAIL-CALLS (#1486): a tail chain that crosses modules, A.main → B.step →
// C.loop(×n). Each tail frame wraps the current resolver chain as a nested call
// does, so C's private helper `bump` and B's `start` resolve exactly as before,
// and the chain stays bounded (resolverCovers) over 10^5 iterations.
func TestTailCall_CrossModuleChain(t *testing.T) {
	rt := newTestRuntime(t)
	inst, err := rt.LoadAndEvaluate("tests/runtime_integration/tail_mod_a")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	main := getExportedFunc(t, inst, "main")
	if got := callIntFunc(t, rt, main, 100000); got != 100000 {
		t.Fatalf("main(100000) = %d, want 100000", got)
	}
}
