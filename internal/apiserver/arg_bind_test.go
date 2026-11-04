package apiserver

import (
	"encoding/json"
	"github.com/sunholo-data/ailang/internal/eval"
	"math"
	"strings"
	"testing"
)

func TestTypedArgumentNumbers(t *testing.T) {
	for _, v := range []any{21.0, int64(21), json.Number("21")} {
		out, err := bindCallArgs(ExportInfo{ParamNames: []string{"n"}, ParamSchemas: []ParamSchema{{Kind: "int"}}}, []any{v})
		if err != nil {
			t.Fatal(err)
		}
		if n, ok := out[0].(*eval.IntValue); !ok || n.Value != 21 {
			t.Fatalf("%T %v", out[0], out[0])
		}
	}
	for _, v := range []any{2.5, math.Inf(1), math.NaN(), math.Exp2(63), uint64(math.MaxUint64), "21", json.Number("9223372036854775808")} {
		if _, err := bindCallArgs(ExportInfo{ParamSchemas: []ParamSchema{{Kind: "int"}}}, []any{v}); err == nil {
			t.Fatalf("accepted %v", v)
		}
	}
	out, err := bindCallArgs(ExportInfo{ParamSchemas: []ParamSchema{{Kind: "float"}}}, []any{2.0})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out[0].(*eval.FloatValue); !ok {
		t.Fatalf("%T", out[0])
	}
}

func TestTypedArgumentIdentityAndOpenFields(t *testing.T) {
	marker := &eval.BytesValue{Value: []byte("file")}
	out, err := bindCallArgs(ExportInfo{ParamSchemas: []ParamSchema{{Kind: "record"}}}, []any{marker})
	if err != nil || out[0] != marker {
		t.Fatalf("identity lost: %v %v", out, err)
	}
	schema := ParamSchema{Kind: "record", Fields: []paramField{{"x", ParamSchema{Kind: "float"}}}}
	got, err := bindArg(schema, map[string]any{"x": 2.0, "extra": 3.0}, "r")
	if err != nil {
		t.Fatal(err)
	}
	fields := got.(*eval.RecordValue).Fields
	if _, ok := fields["x"].(*eval.FloatValue); !ok {
		t.Fatalf("float field: %T", fields["x"])
	}
	if _, ok := fields["extra"].(*eval.IntValue); !ok {
		t.Fatalf("unknown field: %T", fields["extra"])
	}
	if _, err := bindArg(schema, map[string]any{}, "r"); err != nil {
		t.Fatalf("missing fields must retain old behavior: %v", err)
	}
	unit, err := bindArg(ParamSchema{Kind: "int"}, nil, "n")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := unit.(*eval.UnitValue); !ok {
		t.Fatalf("nil: %T", unit)
	}
}

func TestTypedArgumentArray(t *testing.T) {
	s := ParamSchema{Kind: "array", Element: &ParamSchema{Kind: "float"}}
	got, err := bindArg(s, []any{2.0}, "xs")
	if err != nil {
		t.Fatal(err)
	}
	a, ok := got.(*eval.ArrayValue)
	if !ok {
		t.Fatalf("array bound as %T", got)
	}
	if _, ok := a.Elements()[0].(*eval.FloatValue); !ok {
		t.Fatalf("array element: %T", a.Elements()[0])
	}
}

func TestTypedArgumentErrorOrder(t *testing.T) {
	schema := ParamSchema{Kind: "record", Fields: []paramField{{"second", ParamSchema{Kind: "int"}}, {"first", ParamSchema{Kind: "int"}}}}
	_, err := bindArg(schema, map[string]any{"first": 2.5, "second": 3.5}, "r")
	if err == nil || !strings.HasPrefix(err.Error(), "r.second:") {
		t.Fatalf("error order: %v", err)
	}
}
