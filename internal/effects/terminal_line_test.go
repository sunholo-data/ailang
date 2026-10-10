package effects

import (
	"github.com/sunholo-data/ailang/internal/eval"
	"io"
	"strings"
	"testing"
)

func TestReadLineOptExactEOFAndBudgetScope(t *testing.T) {
	ctx := &EffContext{IOReader: strings.NewReader("\nlast")}
	for i, want := range []string{"Some()", "Some(last)", "None", "None"} {
		scope := ctx.WithBudget(nil)
		got, err := ioReadLineOpt(scope, nil)
		if err != nil || got.String() != want {
			t.Fatalf("read %d: %v %v want %s", i, got, err, want)
		}
	}
}
func TestTerminalRejectsForgedSessionAndTimeout(t *testing.T) {
	ctx := &EffContext{}
	handle := &eval.TaggedValue{CtorName: "TerminalSession", Fields: []eval.Value{&eval.IntValue{Value: 999}}}
	got, err := TerminalReadEvent(ctx, []eval.Value{handle, &eval.IntValue{Value: 0}})
	if err != nil || !strings.Contains(got.String(), "InvalidSession") {
		t.Fatalf("forged: %v %v", got, err)
	}
	got, err = TerminalReadEvent(ctx, []eval.Value{handle, &eval.IntValue{Value: 60001}})
	if err != nil || !strings.Contains(got.String(), "InvalidTimeout") {
		t.Fatalf("timeout: %v %v", got, err)
	}
}

type lateLineReader struct{ reads int }

func (r *lateLineReader) Read(data []byte) (int, error) {
	r.reads++
	if r.reads == 1 {
		return 0, io.EOF
	}
	return copy(data, "late\n"), nil
}
func TestReadLineOptStickyEOF(t *testing.T) {
	reader := &lateLineReader{}
	ctx := &EffContext{IOReader: reader}
	for range 2 {
		value, err := ioReadLineOpt(ctx, nil)
		if err != nil || value.String() != "None" {
			t.Fatalf("EOF not sticky: %v %v", value, err)
		}
	}
	if reader.reads != 1 {
		t.Fatalf("read underlying source after EOF: %d", reader.reads)
	}
}
