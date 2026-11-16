package vm

import (
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

// Acceptance: a 200-char preview does not materialise the value's rendering,
// on either engine — the walk counts past the budget, it does not build the
// text. The property is that allocation does NOT SCALE with the value: a
// preview of range(0, 1000000) allocates about what a preview of
// range(0, 10000) does, although its full rendering is 100x longer.
//
// It used to assert an absolute "< 4096 bytes per preview". The deterministic
// cost is ~4040 bytes in 7 mallocs on every size (the 64-frame minimum render
// stack is 3 KB of it), so that bound left ~50 bytes of headroom, and
// testing.Benchmark charges every allocation the PROCESS makes during its
// window to the op, divided by a b.N that is only tens for a 1M-element walk.
// test-windows read 4288 on CI (2026-11). A ratio of the same measurement on
// two sizes is immune to both: the constant cancels, and a stray allocation
// would have to be half a preview to matter. Measured mutations: building the
// full rendering and cutting it reads ~200x; one 8-byte allocation per
// element past the budget reads ~98x.
//
// A ratio cannot see a cost that does not grow: rendering a FIXED 16k-char
// prefix before cutting (review mutant S2) costs ~70 KB per preview at both
// sizes, ratio 1.00. So each preview must also stay under an absolute
// ceiling, previewAllocCeiling — 4x the ~4 KB constant, far clear of the
// 232 B of noise that made the old bound flake.
const previewAllocCeiling = 16 << 10

func TestShowPreviewDoesNotMaterialise(t *testing.T) {
	const budget = 200
	small, big := sList(showInts(0, 10_000)...), sList(showInts(0, 1_000_000)...)
	smallVM := []bytecode.Value{toVMForShow(t, small), bytecode.NewInt(budget)}
	bigVM := []bytecode.Value{toVMForShow(t, big), bytecode.NewInt(budget)}
	vmPreview := func(args []bytecode.Value) func() string {
		return func() string {
			v, err := builtinShowPreview(args)
			if err != nil {
				t.Fatal(err)
			}
			return v.AsString()
		}
	}
	for _, e := range []struct {
		name       string
		small, big func() string
	}{
		{"evaluator", func() string { return eval.ShowPreview(small, budget) }, func() string { return eval.ShowPreview(big, budget) }},
		{"vm", vmPreview(smallVM), vmPreview(bigVM)},
	} {
		smallBytes, smallOut := previewAllocBytes(e.small)
		bigBytes, bigOut := previewAllocBytes(e.big)
		for _, out := range []string{smallOut, bigOut} {
			if !strings.HasPrefix(out, "[0, 1, 2, ") || !strings.HasSuffix(out, " chars)") {
				t.Fatalf("%s: unexpected preview %q", e.name, out)
			}
		}
		// The two previews must actually differ in what they leave out, or
		// the sizes were not really 100x apart.
		if smallOut == bigOut {
			t.Fatalf("%s: previews of a 10k and a 1M list are identical: %q", e.name, bigOut)
		}
		ratio := float64(bigBytes) / float64(smallBytes)
		t.Logf("%s: preview of 10k elements allocates %d B, of 1M elements %d B (ratio %.2f)", e.name, smallBytes, bigBytes, ratio)
		if ratio > 1.5 {
			t.Errorf("%s: a preview of a 100x larger value allocated %.1fx as much (%d B vs %d B); want <= 1.5x — showPreview must not build the rendering it cuts",
				e.name, ratio, bigBytes, smallBytes)
		}
		for _, m := range []struct {
			size  string
			bytes int64
		}{{"10k", smallBytes}, {"1M", bigBytes}} {
			if m.bytes >= previewAllocCeiling {
				t.Errorf("%s: a %d-char preview of %s elements allocated %d B; want < %d — the preview must cost about its budget, not a rendering",
					e.name, budget, m.size, m.bytes, previewAllocCeiling)
			}
		}
	}
}

// previewAllocBytes is the bytes one call of render allocates: the minimum,
// over three samples, of TotalAlloc growth across a few calls. GOMAXPROCS(1)
// (as testing.AllocsPerRun does) and the minimum keep another goroutine's
// allocation out of the reading; the ratio above keeps the platform out.
func previewAllocBytes(render func() string) (perCall int64, out string) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))
	const calls = 4
	out = render() // warm up: first-use allocations are not the walk's
	best := int64(-1)
	for sample := 0; sample < 3; sample++ {
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		for i := 0; i < calls; i++ {
			out = render()
		}
		runtime.ReadMemStats(&after)
		if n := int64(after.TotalAlloc-before.TotalAlloc) / calls; best < 0 || n < best {
			best = n
		}
	}
	return best, out
}
