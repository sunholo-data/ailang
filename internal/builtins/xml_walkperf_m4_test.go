package builtins

// M-STDLIB-XML-WALK-PERF M4 tests + benchmarks.
// Split out of xml_walkperf_test.go to keep both files under the 800 LOC guidance.
//
// Covers _xml_flatMapChildren and _xml_mapChildren — the Go-iterative
// list-output primitives shipped after sunholo/ailang-parse's f639ad74
// regression report.

import (
	"fmt"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// ============================================================================
// _xml_flatMapChildren (M4)
// ============================================================================

// tagListHandler maps each child to a single-element list of its tag string.
// Mirrors the real "produce 1+ output blocks per child" pattern of HTML walkers.
func tagListHandler() eval.Value {
	return &eval.BuiltinFunction{
		Name: "tag_list",
		Fn: func(args []eval.Value) (eval.Value, error) {
			child := args[0].(*eval.TaggedValue)
			tag := child.Fields[0].(*eval.StringValue).Value
			return &eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: tag}}}, nil
		},
	}
}

// twoPerChildHandler returns a 2-element list per child — exercises the
// flatten part of flatMap (not just the map part).
func twoPerChildHandler() eval.Value {
	return &eval.BuiltinFunction{
		Name: "two_per_child",
		Fn: func(args []eval.Value) (eval.Value, error) {
			child := args[0].(*eval.TaggedValue)
			tag := child.Fields[0].(*eval.StringValue).Value
			return &eval.ListValue{Elements: []eval.Value{
				&eval.StringValue{Value: tag + "-a"},
				&eval.StringValue{Value: tag + "-b"},
			}}, nil
		},
	}
}

// emptyListHandler returns [] for every child — caller should get [] back
// without crashing or allocating per-child.
func emptyListHandler() eval.Value {
	return &eval.BuiltinFunction{
		Name: "empty_list",
		Fn: func(args []eval.Value) (eval.Value, error) {
			return &eval.ListValue{Elements: []eval.Value{}}, nil
		},
	}
}

// fnCallerCtx provides the single-arg FnCaller plumbing flatMapChildren/mapChildren need.
func fnCallerCtx() *effects.EffContext {
	ctx := effects.NewEffContext(nil)
	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		goFn, ok := fn.(*eval.BuiltinFunction)
		if !ok {
			return nil, fmt.Errorf("test FnCaller: expected BuiltinFunction, got %T", fn)
		}
		return goFn.Fn([]eval.Value{arg})
	}
	return ctx
}

func TestXmlFlatMapChildren_OnePerChild(t *testing.T) {
	root := makeElementWP("root", nil,
		makeElementWP("a", nil),
		makeElementWP("b", nil),
		makeElementWP("c", nil),
	)
	ctx := fnCallerCtx()
	result, err := xmlFlatMapChildrenImpl(ctx, []eval.Value{root, tagListHandler()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lst := result.(*eval.ListValue)
	if len(lst.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(lst.Elements))
	}
	for i, want := range []string{"a", "b", "c"} {
		got := lst.Elements[i].(*eval.StringValue).Value
		if got != want {
			t.Errorf("[%d]: expected %q, got %q", i, want, got)
		}
	}
}

func TestXmlFlatMapChildren_TwoPerChildFlattens(t *testing.T) {
	root := makeElementWP("root", nil,
		makeElementWP("a", nil),
		makeElementWP("b", nil),
	)
	ctx := fnCallerCtx()
	result, err := xmlFlatMapChildrenImpl(ctx, []eval.Value{root, twoPerChildHandler()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lst := result.(*eval.ListValue)
	if len(lst.Elements) != 4 {
		t.Fatalf("expected 4 elements (2 per child × 2 children), got %d", len(lst.Elements))
	}
	want := []string{"a-a", "a-b", "b-a", "b-b"}
	for i, w := range want {
		got := lst.Elements[i].(*eval.StringValue).Value
		if got != w {
			t.Errorf("[%d]: expected %q, got %q", i, w, got)
		}
	}
}

func TestXmlFlatMapChildren_EmptyChildren(t *testing.T) {
	root := makeElementWP("root", nil) // no children
	ctx := fnCallerCtx()
	result, err := xmlFlatMapChildrenImpl(ctx, []eval.Value{root, tagListHandler()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.(*eval.ListValue).Elements) != 0 {
		t.Errorf("expected [] for empty children, got %v", result)
	}
}

func TestXmlFlatMapChildren_AllEmptyResults(t *testing.T) {
	root := makeElementWP("root", nil,
		makeElementWP("a", nil), makeElementWP("b", nil),
	)
	ctx := fnCallerCtx()
	result, err := xmlFlatMapChildrenImpl(ctx, []eval.Value{root, emptyListHandler()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.(*eval.ListValue).Elements) != 0 {
		t.Errorf("expected [] when every callback returns [], got %v", result)
	}
}

func TestXmlFlatMapChildren_NonElementReturnsEmpty(t *testing.T) {
	result, err := xmlFlatMapChildrenImpl(fnCallerCtx(), []eval.Value{makeXmlText("hi"), tagListHandler()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.(*eval.ListValue).Elements) != 0 {
		t.Errorf("expected [] on Text node, got %v", result)
	}
}

func TestXmlFlatMapChildren_HandlerErrorPropagates(t *testing.T) {
	root := makeElementWP("root", nil, makeElementWP("a", nil))
	failing := &eval.BuiltinFunction{
		Name: "fail",
		Fn:   func(args []eval.Value) (eval.Value, error) { return nil, fmt.Errorf("boom") },
	}
	_, err := xmlFlatMapChildrenImpl(fnCallerCtx(), []eval.Value{root, failing})
	if err == nil {
		t.Fatal("expected handler error to propagate")
	}
}

func TestXmlFlatMapChildren_RejectsNonListCallback(t *testing.T) {
	root := makeElementWP("root", nil, makeElementWP("a", nil))
	bogus := &eval.BuiltinFunction{
		Name: "bogus",
		Fn:   func(args []eval.Value) (eval.Value, error) { return &eval.StringValue{Value: "not a list"}, nil },
	}
	_, err := xmlFlatMapChildrenImpl(fnCallerCtx(), []eval.Value{root, bogus})
	if err == nil {
		t.Fatal("expected error when callback returns non-list")
	}
}

func TestXmlFlatMapChildren_NilFnCaller(t *testing.T) {
	root := makeElementWP("root", nil, makeElementWP("a", nil))
	ctx := effects.NewEffContext(nil) // FnCaller not wired
	_, err := xmlFlatMapChildrenImpl(ctx, []eval.Value{root, tagListHandler()})
	if err == nil {
		t.Fatal("expected error when FnCaller not set")
	}
}

// ============================================================================
// _xml_mapChildren (M4)
// ============================================================================

func TestXmlMapChildren_Basic(t *testing.T) {
	root := makeElementWP("root", nil,
		makeElementWP("a", nil), makeElementWP("b", nil), makeElementWP("c", nil),
	)
	mapToTag := &eval.BuiltinFunction{
		Name: "to_tag",
		Fn: func(args []eval.Value) (eval.Value, error) {
			child := args[0].(*eval.TaggedValue)
			return &eval.StringValue{Value: child.Fields[0].(*eval.StringValue).Value}, nil
		},
	}
	result, err := xmlMapChildrenImpl(fnCallerCtx(), []eval.Value{root, mapToTag})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lst := result.(*eval.ListValue)
	if len(lst.Elements) != 3 {
		t.Fatalf("expected 3, got %d", len(lst.Elements))
	}
	for i, want := range []string{"a", "b", "c"} {
		if got := lst.Elements[i].(*eval.StringValue).Value; got != want {
			t.Errorf("[%d]: expected %q, got %q", i, want, got)
		}
	}
}

func TestXmlMapChildren_EmptyChildren(t *testing.T) {
	root := makeElementWP("root", nil)
	mapToTag := &eval.BuiltinFunction{
		Name: "to_tag",
		Fn:   func(args []eval.Value) (eval.Value, error) { return &eval.StringValue{Value: "x"}, nil },
	}
	result, err := xmlMapChildrenImpl(fnCallerCtx(), []eval.Value{root, mapToTag})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.(*eval.ListValue).Elements) != 0 {
		t.Errorf("expected []")
	}
}

func TestXmlMapChildren_NonElementReturnsEmpty(t *testing.T) {
	result, err := xmlMapChildrenImpl(fnCallerCtx(), []eval.Value{
		makeXmlText("hi"),
		&eval.BuiltinFunction{Name: "_", Fn: func(args []eval.Value) (eval.Value, error) { return &eval.IntValue{Value: 0}, nil }},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.(*eval.ListValue).Elements) != 0 {
		t.Errorf("expected []")
	}
}

func TestXmlMapChildren_PreservesDocumentOrder(t *testing.T) {
	root := makeElementWP("root", nil,
		makeElementWP("first", nil),
		makeElementWP("second", nil),
		makeElementWP("third", nil),
	)
	idx := 0
	tracker := &eval.BuiltinFunction{
		Name: "tracker",
		Fn: func(args []eval.Value) (eval.Value, error) {
			child := args[0].(*eval.TaggedValue)
			v := child.Fields[0].(*eval.StringValue).Value
			result := fmt.Sprintf("%d:%s", idx, v)
			idx++
			return &eval.StringValue{Value: result}, nil
		},
	}
	result, _ := xmlMapChildrenImpl(fnCallerCtx(), []eval.Value{root, tracker})
	lst := result.(*eval.ListValue)
	for i, want := range []string{"0:first", "1:second", "2:third"} {
		if got := lst.Elements[i].(*eval.StringValue).Value; got != want {
			t.Errorf("[%d]: expected %q, got %q", i, want, got)
		}
	}
}

// ============================================================================
// M4 benchmarks: list-output workload (the regression shape)
// ============================================================================

// BenchmarkXmlFlatMap_ViaFoldChildren simulates the regression sunholo/ailang-parse
// reported: producing a flat [Block] list via foldChildren forces an AILANG-side
// prepend+reverse pattern. Here we approximate that in Go by appending to a slice
// then reversing — measures the cost foldChildren imposes when the output shape
// is a list.
func BenchmarkXmlFlatMap_ViaFoldChildren(b *testing.B) {
	root := makeWideTree(1900)
	ctx := xmlFoldTestCtx()
	prependHandler := &eval.BuiltinFunction{
		Name: "prepend_tag",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.ListValue)
			child := args[1].(*eval.TaggedValue)
			tag := child.Fields[0].(*eval.StringValue).Value
			next := make([]eval.Value, 0, len(acc.Elements)+1)
			next = append(next, &eval.StringValue{Value: tag})
			next = append(next, acc.Elements...)
			return &eval.ListValue{Elements: next}, nil
		},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := xmlFoldChildrenImpl(ctx, []eval.Value{
			root,
			&eval.ListValue{Elements: nil},
			prependHandler,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkXmlFlatMap_ViaFlatMapChildren does the same logical work via the
// M4 flatMapChildren primitive — one Go-native slice with amortised append.
func BenchmarkXmlFlatMap_ViaFlatMapChildren(b *testing.B) {
	root := makeWideTree(1900)
	ctx := fnCallerCtx()
	handler := tagListHandler()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := xmlFlatMapChildrenImpl(ctx, []eval.Value{root, handler})
		if err != nil {
			b.Fatal(err)
		}
	}
}
