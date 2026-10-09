package effects

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func TestIOReadLineOpt_LinesAndEOF(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		lines       []string
	}{
		{"empty", "", nil},
		{"blank", "\n", []string{""}},
		{"lines", "a\n\nb\n", []string{"a", "", "b"}},
		{"partial", "a\nb", []string{"a", "b"}},
		{"crlf", "a\r\n\r\nb\r", []string{"a", "", "b"}},
		{"bare carriage return", "\r", []string{""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := NewEffContext(nil)
			ctx.Grant(NewCapability("IO"))
			ctx.IOReader = strings.NewReader(tc.input)
			for i := 0; i < len(tc.lines)+3; i++ {
				v, err := Call(ctx, "IO", "readLineOpt", nil)
				if err != nil {
					t.Fatal(err)
				}
				opt, ok := v.(*eval.TaggedValue)
				if !ok {
					t.Fatalf("expected Option, got %T", v)
				}
				if opt.ModulePath != "std/option" || opt.TypeName != "Option" {
					t.Fatalf("wrong Option identity: %+v", opt)
				}
				if i >= len(tc.lines) {
					if opt.CtorName != "None" || len(opt.Fields) != 0 {
						t.Fatalf("expected None at EOF, got %+v", opt)
					}
				} else {
					if opt.CtorName != "Some" || len(opt.Fields) != 1 {
						t.Fatalf("expected Some, got %+v", opt)
					}
					if got := opt.Fields[0].(*eval.StringValue).Value; got != tc.lines[i] {
						t.Fatalf("line %d: got %q, want %q", i, got, tc.lines[i])
					}
				}
			}
		})
	}
}

type failingLineReader struct {
	err  error
	data string
}

func (r failingLineReader) Read(p []byte) (int, error) { return copy(p, r.data), r.err }

func TestIOReadLineOpt_ErrorAndValidation(t *testing.T) {
	ctx := NewEffContext(nil)
	if _, err := Call(ctx, "IO", "readLineOpt", nil); err == nil {
		t.Fatal("missing IO capability accepted")
	}
	ctx.Grant(NewCapability("IO"))
	if _, err := Call(ctx, "IO", "readLineOpt", []eval.Value{&eval.UnitValue{}}); err == nil {
		t.Fatal("effect operation accepted arguments")
	}
	want := errors.New("broken stdin")
	for _, data := range []string{"", "partial"} {
		ctx := NewEffContext(nil)
		ctx.Grant(NewCapability("IO"))
		ctx.IOReader = failingLineReader{want, data}
		if v, err := Call(ctx, "IO", "readLineOpt", nil); !errors.Is(err, want) || v != nil {
			t.Fatalf("expected reader error, got %v, %v", v, err)
		}
	}
}

func TestIOReadLineOpt_SharedReader(t *testing.T) {
	for _, ops := range [][]string{{"readLine", "readLineOpt", "readLine"}, {"readLineOpt", "readLine", "readLineOpt"}} {
		ctx := NewEffContext(nil)
		ctx.Grant(NewCapability("IO"))
		ctx.IOReader = strings.NewReader("a\nb\nc")
		for i, op := range ops {
			v, err := Call(ctx, "IO", op, nil)
			if err != nil {
				t.Fatal(err)
			}
			if opt, ok := v.(*eval.TaggedValue); ok {
				v = opt.Fields[0]
			}
			if got := v.(*eval.StringValue).Value; got != []string{"a", "b", "c"}[i] {
				t.Fatalf("%s lost buffered data: %q", op, got)
			}
		}
		v, err := Call(ctx, "IO", "readLineOpt", nil)
		if err != nil || v.(*eval.TaggedValue).CtorName != "None" {
			t.Fatalf("expected EOF: %v, %v", v, err)
		}
	}
}

var _ io.Reader = failingLineReader{}
