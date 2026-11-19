package effects

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
)

// M-TRACE-LABEL-AWARE: inside a label scope a trace site never renders the
// value at all — it is not materialised as a trace string and then replaced
// by the collector, it is never built.
func TestRenderTraceValueSkipsRenderingInsideLabelScope(t *testing.T) {
	const canary = "CANARY_FAKE_NOT_A_REAL_SECRET"
	big := &eval.StringValue{Value: canary + strings.Repeat("q", 4<<20)}
	ctx := NewEffContext(nil)
	ctx.Trace = trace.NewCollectorWithTier(trace.TierDeep)

	if got := ctx.RenderTraceValue(big); !strings.HasPrefix(got, canary) {
		t.Fatalf("control: outside a scope the value renders (bounded); got %.40q", got)
	}

	closeScope := ctx.PushTraceLabelScope("secret")
	if closeScope == nil {
		t.Fatal("PushTraceLabelScope with an enabled collector returned nil")
	}
	defer closeScope()
	if got := ctx.RenderTraceValue(big); got != trace.LabelRedactedMarker("secret") {
		t.Fatalf("inside a scope RenderTraceValue = %.60q, want the marker", got)
	}
	perOp := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = ctx.RenderTraceValue(big)
		}
	}).AllocedBytesPerOp()
	if perOp > 256 {
		t.Fatalf("rendering a 4 MB value inside a scope allocated %d bytes/op; it must not render it", perOp)
	}
}

func TestPushTraceLabelScopeWithoutCollector(t *testing.T) {
	ctx := NewEffContext(nil)
	if closeScope := ctx.PushTraceLabelScope("secret"); closeScope != nil {
		t.Fatal("no collector: there is nothing to redact, so no scope")
	}
}

// The OTEL effect-span wrapper enriches spans from raw args (fs.path,
// net.url, process.command). Inside a scope it must see markers instead.
func TestEffectSpanWrapperSeesNoLabelledArgs(t *testing.T) {
	const canary = "CANARY_FAKE_NOT_A_REAL_SECRET"
	dir := t.TempDir()
	path := filepath.Join(dir, canary+".txt")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("FS"))
	ctx.Trace = trace.NewCollectorWithTier(trace.TierStandard)
	ctx.GoCtx = context.Background()
	var seen []string
	ctx.SpanWrapper = func(_ context.Context, _, _ string, args []eval.Value, fn func() (eval.Value, error)) (eval.Value, error) {
		for _, a := range args {
			seen = append(seen, a.String())
		}
		return fn()
	}
	args := []eval.Value{&eval.StringValue{Value: path}}

	if _, err := Call(ctx, "FS", "readFileResult", args); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || !strings.Contains(seen[0], canary) {
		t.Fatalf("control: outside a scope the wrapper sees the path; saw %q", seen)
	}

	seen = nil
	closeScope := ctx.PushTraceLabelScope("secret")
	v, err := Call(ctx, "FS", "readFileResult", args)
	closeScope()
	if err != nil {
		t.Fatal(err)
	}
	if tv, ok := v.(*eval.TaggedValue); !ok || tv.CtorName != "Ok" {
		t.Fatalf("the effect itself must still see the real path; got %v", v)
	}
	if len(seen) != 1 || strings.Contains(seen[0], canary) {
		t.Fatalf("inside a scope the span wrapper saw %q; want one redacted arg", seen)
	}
}
