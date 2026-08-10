package testing

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
)

// newSpliceRunner returns a bare Runner whose valueToLiteral method can be
// exercised directly.
func newSpliceRunner() *Runner {
	return NewRunner("value_splice_test.ail")
}

// TestValueToLiteral_UnitValue verifies that a UnitValue splices to a UnitLit
// literal.
func TestValueToLiteral_UnitValue(t *testing.T) {
	r := newSpliceRunner()
	expr := r.valueToLiteral(&eval.UnitValue{})

	lit, ok := expr.(*ast.Literal)
	if !ok {
		t.Fatalf("expected *ast.Literal, got %T", expr)
	}
	if lit.Kind != ast.UnitLit {
		t.Errorf("expected UnitLit, got %v", lit.Kind)
	}
}

// TestValueToLiteral_RecordValue verifies that a RecordValue splices to an
// ast.Record with a field per entry and sorted field order.
func TestValueToLiteral_RecordValue(t *testing.T) {
	r := newSpliceRunner()
	expr := r.valueToLiteral(&eval.RecordValue{
		Fields: map[string]eval.Value{
			"x": &eval.IntValue{Value: 1},
			"y": &eval.IntValue{Value: 2},
		},
	})

	rec, ok := expr.(*ast.Record)
	if !ok {
		t.Fatalf("expected *ast.Record, got %T", expr)
	}
	if len(rec.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(rec.Fields))
	}
	if rec.Fields[0].Name != "x" || rec.Fields[1].Name != "y" {
		t.Errorf("expected sorted order [x y], got [%s %s]", rec.Fields[0].Name, rec.Fields[1].Name)
	}
}

// TestValueToLiteral_TupleValue verifies that a TupleValue splices to an
// ast.Tuple keeping element order.
func TestValueToLiteral_TupleValue(t *testing.T) {
	r := newSpliceRunner()
	expr := r.valueToLiteral(&eval.TupleValue{
		Elements: []eval.Value{
			&eval.IntValue{Value: 7},
			&eval.BoolValue{Value: true},
		},
	})

	tup, ok := expr.(*ast.Tuple)
	if !ok {
		t.Fatalf("expected *ast.Tuple, got %T", expr)
	}
	if len(tup.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(tup.Elements))
	}
	first, ok := tup.Elements[0].(*ast.Literal)
	if !ok || first.Kind != ast.IntLit {
		t.Errorf("expected first element to be IntLit, got %T", tup.Elements[0])
	}
}

// TestValueToLiteral_TaggedValue_Nullary verifies that a nullary TaggedValue
// splices to a bare *ast.Identifier (a FuncCall over a nullary constructor
// would die at runtime with "cannot apply non-function value").
func TestValueToLiteral_TaggedValue_Nullary(t *testing.T) {
	r := newSpliceRunner()
	expr := r.valueToLiteral(&eval.TaggedValue{
		ModulePath: "test",
		TypeName:   "Season",
		CtorName:   "SPRING",
		Fields:     []eval.Value{},
	})

	ident, ok := expr.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected *ast.Identifier for nullary constructor, got %T", expr)
	}
	if ident.Name != "SPRING" {
		t.Errorf("expected identifier name SPRING, got %s", ident.Name)
	}
}

// TestValueToLiteral_TaggedValue_Nary verifies that an n-ary TaggedValue
// splices to an *ast.FuncCall over the constructor identifier.
func TestValueToLiteral_TaggedValue_Nary(t *testing.T) {
	r := newSpliceRunner()
	expr := r.valueToLiteral(&eval.TaggedValue{
		ModulePath: "test",
		TypeName:   "Block",
		CtorName:   "Para",
		Fields: []eval.Value{
			&eval.StringValue{Value: "hello"},
			&eval.IntValue{Value: 3},
		},
	})

	call, ok := expr.(*ast.FuncCall)
	if !ok {
		t.Fatalf("expected *ast.FuncCall for n-ary constructor, got %T", expr)
	}
	fn, ok := call.Func.(*ast.Identifier)
	if !ok || fn.Name != "Para" {
		t.Errorf("expected Func to be identifier Para, got %T %v", call.Func, call.Func)
	}
	if len(call.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(call.Args))
	}

	// Assert the argument VALUES, not just the count. Checking name+arity only
	// would pass identically if the arm spliced constants instead of recursing
	// through valueToLiteral — measured: replacing both args with a literal 999
	// leaves the whole internal/testing suite rc=0 without these two blocks.
	arg0, ok := call.Args[0].(*ast.Literal)
	if !ok || arg0.Kind != ast.StringLit || arg0.Value != "hello" {
		t.Errorf("arg 0: expected string literal %q, got %T %+v", "hello", call.Args[0], call.Args[0])
	}
	arg1, ok := call.Args[1].(*ast.Literal)
	if !ok || arg1.Kind != ast.IntLit || arg1.Value != 3 {
		t.Errorf("arg 1: expected int literal 3, got %T %+v", call.Args[1], call.Args[1])
	}
}

// TestValueToLiteral_Nested verifies that composite values recurse through
// valueToLiteral: a record containing a tuple containing a list.
func TestValueToLiteral_Nested(t *testing.T) {
	r := newSpliceRunner()
	expr := r.valueToLiteral(&eval.RecordValue{
		Fields: map[string]eval.Value{
			"outer": &eval.TupleValue{
				Elements: []eval.Value{
					&eval.ListValue{
						Elements: []eval.Value{
							&eval.IntValue{Value: 1},
							&eval.IntValue{Value: 2},
						},
					},
				},
			},
		},
	})

	rec, ok := expr.(*ast.Record)
	if !ok {
		t.Fatalf("expected *ast.Record, got %T", expr)
	}
	if len(rec.Fields) != 1 || rec.Fields[0].Name != "outer" {
		t.Fatalf("unexpected record fields: %+v", rec.Fields)
	}
	tup, ok := rec.Fields[0].Value.(*ast.Tuple)
	if !ok {
		t.Fatalf("expected outer field value to be *ast.Tuple, got %T", rec.Fields[0].Value)
	}
	if len(tup.Elements) != 1 {
		t.Fatalf("expected 1 tuple element, got %d", len(tup.Elements))
	}
	lst, ok := tup.Elements[0].(*ast.List)
	if !ok {
		t.Fatalf("expected tuple element to be *ast.List, got %T", tup.Elements[0])
	}
	if len(lst.Elements) != 2 {
		t.Fatalf("expected 2 list elements, got %d", len(lst.Elements))
	}
	if lit, ok := lst.Elements[0].(*ast.Literal); !ok || lit.Value != 1 {
		t.Errorf("expected first list element IntLit 1, got %T %v", lst.Elements[0], lst.Elements[0])
	}
}

// TestValueToLiteral_RecordFieldOrderSorted verifies that RecordValue fields
// are emitted in sorted order regardless of Go map iteration order. It runs
// many iterations so that a broken unsorted implementation (ranging directly
// over the map) cannot pass by a map-order fluke.
func TestValueToLiteral_RecordFieldOrderSorted(t *testing.T) {
	r := newSpliceRunner()
	// Chosen so the sorted order differs from typical random map orders.
	names := []string{"zebra", "apple", "mango", "banana"}
	want := []string{"apple", "banana", "mango", "zebra"}

	for i := 0; i < 1000; i++ {
		fields := make(map[string]eval.Value, len(names))
		for _, n := range names {
			fields[n] = &eval.IntValue{Value: 1}
		}
		expr := r.valueToLiteral(&eval.RecordValue{Fields: fields})
		rec, ok := expr.(*ast.Record)
		if !ok {
			t.Fatalf("expected *ast.Record, got %T", expr)
		}
		if len(rec.Fields) != len(want) {
			t.Fatalf("iteration %d: expected %d fields, got %d", i, len(want), len(rec.Fields))
		}
		for j, f := range rec.Fields {
			if f.Name != want[j] {
				t.Fatalf("iteration %d: field %d = %q, want %q (order not sorted)", i, j, f.Name, want[j])
			}
		}
	}
}
