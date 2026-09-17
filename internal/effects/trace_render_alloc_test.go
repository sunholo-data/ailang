package effects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
)

// M-V1-MEMORY-FOOTPRINT M1, effect site. An effect whose result is a large
// tagged value (Ok(<8 MB file>)) must not be materialised for the trace: the
// old site called result.String() — "Ok(" + 8 MB + ")" — and then cut it to
// 1 KB, one full extra copy per call at EVERY tier.
//
// Measured as allocated bytes per call, traced minus untraced, which is
// deterministic. The RSS-ratio form of this check (cmd/ailang memprobe) read
// 0.96x on the rig and 1.37x on the ubuntu runner: peak RSS under GOGC=500
// depends on where GC cycles land, so it measured the kernel, not the site.
func TestEffectTraceRenderDoesNotMaterialiseLargeResults(t *testing.T) {
	const size = 8 << 20
	dir := t.TempDir()
	path := filepath.Join(dir, "big.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("q", size)), 0o644); err != nil {
		t.Fatal(err)
	}
	args := []eval.Value{&eval.StringValue{Value: path}}

	call := func(ctx *EffContext) {
		v, err := Call(ctx, "FS", "readFileResult", args)
		if err != nil {
			t.Fatal(err)
		}
		if tv, ok := v.(*eval.TaggedValue); !ok || tv.CtorName != "Ok" {
			t.Fatalf("unexpected result %T", v)
		}
	}
	bytesPerOp := func(ctx *EffContext) int64 {
		return testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				call(ctx)
			}
		}).AllocedBytesPerOp()
	}

	plain := NewEffContext(nil)
	plain.Grant(NewCapability("FS"))
	untraced := bytesPerOp(plain)

	traced := NewEffContext(nil)
	traced.Grant(NewCapability("FS"))
	traced.Trace = trace.NewCollectorWithTier(trace.TierStandard)
	withTrace := bytesPerOp(traced)

	delta := withTrace - untraced
	t.Logf("readFileResult of %d MB: untraced %d MB/op, standard-tier traced %d MB/op, delta %d KB", size>>20, untraced>>20, withTrace>>20, delta>>10)
	if delta > 256<<10 {
		t.Fatalf("tracing an 8 MB effect result allocated %d KB extra per call; want < 256 KB (the result must be rendered bounded, not materialised)", delta>>10)
	}
	// The event was recorded, and bounded.
	evts := traced.Trace.Events()
	var last *trace.EffectEvent
	for i := len(evts) - 1; i >= 0; i-- {
		if evts[i].Effect != nil && evts[i].Effect.OpName == "readFileResult" {
			last = evts[i].Effect
			break
		}
	}
	if last == nil {
		t.Fatal("no readFileResult effect event recorded at standard tier")
	}
	if len(last.Result) > 2048 || !strings.HasPrefix(last.Result, "Ok(qqq") || !strings.Contains(last.Result, "bytes elided") {
		t.Fatalf("recorded result not the bounded rendering: len %d, %.60q", len(last.Result), last.Result)
	}
}
