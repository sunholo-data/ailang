package builtins

import (
	"fmt"
	"testing"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// Helper: create an EffContext with FnCaller and FnCallerN wired to Go functions.
func newTestEffCtx() *effects.EffContext {
	ctx := effects.NewEffContext(nil)
	// FnCaller: single-arg — call Go function directly
	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		goFn, ok := fn.(*eval.BuiltinFunction)
		if !ok {
			return nil, fmt.Errorf("test FnCaller: expected BuiltinFunction, got %T", fn)
		}
		return goFn.Fn([]eval.Value{arg})
	}
	// FnCallerN: multi-arg — call Go function with multiple args
	ctx.FnCallerN = func(fn eval.Value, args []eval.Value) (eval.Value, error) {
		goFn, ok := fn.(*eval.BuiltinFunction)
		if !ok {
			return nil, fmt.Errorf("test FnCallerN: expected BuiltinFunction, got %T", fn)
		}
		return goFn.Fn(args)
	}
	return ctx
}

// Helper: make a Go function into an eval.Value (BuiltinFunction)
func goFn(f func([]eval.Value) (eval.Value, error)) eval.Value {
	return &eval.BuiltinFunction{Name: "test_fn", Fn: f}
}

// Helper: make an int list
func intList(vals ...int) *eval.ListValue {
	elems := make([]eval.Value, len(vals))
	for i, v := range vals {
		elems[i] = &eval.IntValue{Value: v}
	}
	return &eval.ListValue{Elements: elems}
}

// Helper: extract ints from list value
func toInts(t *testing.T, v eval.Value) []int {
	t.Helper()
	list, ok := v.(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", v)
	}
	result := make([]int, len(list.Elements))
	for i, e := range list.Elements {
		iv, ok := e.(*eval.IntValue)
		if !ok {
			t.Fatalf("element %d: expected IntValue, got %T", i, e)
		}
		result[i] = iv.Value
	}
	return result
}

// ============================================================================
// _list_map tests
// ============================================================================

func TestListMapIdentity(t *testing.T) {
	ctx := newTestEffCtx()
	identity := goFn(func(args []eval.Value) (eval.Value, error) {
		return args[0], nil
	})
	input := intList(1, 2, 3, 4, 5)

	result, err := listMapImpl(ctx, []eval.Value{identity, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	expected := []int{1, 2, 3, 4, 5}
	if len(got) != len(expected) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(expected))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("index %d: got %d, want %d", i, got[i], expected[i])
		}
	}
}

func TestListMapTransform(t *testing.T) {
	ctx := newTestEffCtx()
	double := goFn(func(args []eval.Value) (eval.Value, error) {
		v := args[0].(*eval.IntValue)
		return &eval.IntValue{Value: v.Value * 2}, nil
	})
	input := intList(1, 2, 3)

	result, err := listMapImpl(ctx, []eval.Value{double, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	expected := []int{2, 4, 6}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("index %d: got %d, want %d", i, got[i], expected[i])
		}
	}
}

func TestListMapEmpty(t *testing.T) {
	ctx := newTestEffCtx()
	identity := goFn(func(args []eval.Value) (eval.Value, error) {
		return args[0], nil
	})
	input := intList()

	result, err := listMapImpl(ctx, []eval.Value{identity, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	if len(got) != 0 {
		t.Fatalf("expected empty list, got %d elements", len(got))
	}
}

func TestListMapCallbackError(t *testing.T) {
	ctx := newTestEffCtx()
	failing := goFn(func(args []eval.Value) (eval.Value, error) {
		return nil, fmt.Errorf("boom")
	})
	input := intList(1, 2, 3)

	_, err := listMapImpl(ctx, []eval.Value{failing, input})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "_list_map: callback error at index 0: boom" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestListMapNonList(t *testing.T) {
	ctx := newTestEffCtx()
	identity := goFn(func(args []eval.Value) (eval.Value, error) { return args[0], nil })

	_, err := listMapImpl(ctx, []eval.Value{identity, &eval.IntValue{Value: 42}})
	if err == nil {
		t.Fatal("expected error for non-list arg")
	}
}

func TestListMapNilFnCaller(t *testing.T) {
	ctx := effects.NewEffContext(nil) // FnCaller not set
	identity := goFn(func(args []eval.Value) (eval.Value, error) { return args[0], nil })
	input := intList(1)

	_, err := listMapImpl(ctx, []eval.Value{identity, input})
	if err == nil {
		t.Fatal("expected error when FnCaller is nil")
	}
}

// ============================================================================
// _list_filter tests
// ============================================================================

func TestListFilterKeepEvens(t *testing.T) {
	ctx := newTestEffCtx()
	isEven := goFn(func(args []eval.Value) (eval.Value, error) {
		v := args[0].(*eval.IntValue)
		return &eval.BoolValue{Value: v.Value%2 == 0}, nil
	})
	input := intList(1, 2, 3, 4, 5, 6)

	result, err := listFilterImpl(ctx, []eval.Value{isEven, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	expected := []int{2, 4, 6}
	if len(got) != len(expected) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(expected))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("index %d: got %d, want %d", i, got[i], expected[i])
		}
	}
}

func TestListFilterKeepAll(t *testing.T) {
	ctx := newTestEffCtx()
	alwaysTrue := goFn(func(args []eval.Value) (eval.Value, error) {
		return &eval.BoolValue{Value: true}, nil
	})
	input := intList(1, 2, 3)

	result, err := listFilterImpl(ctx, []eval.Value{alwaysTrue, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	if len(got) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(got))
	}
}

func TestListFilterKeepNone(t *testing.T) {
	ctx := newTestEffCtx()
	alwaysFalse := goFn(func(args []eval.Value) (eval.Value, error) {
		return &eval.BoolValue{Value: false}, nil
	})
	input := intList(1, 2, 3)

	result, err := listFilterImpl(ctx, []eval.Value{alwaysFalse, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	if len(got) != 0 {
		t.Fatalf("expected empty list, got %d elements", len(got))
	}
}

func TestListFilterEmpty(t *testing.T) {
	ctx := newTestEffCtx()
	alwaysTrue := goFn(func(args []eval.Value) (eval.Value, error) {
		return &eval.BoolValue{Value: true}, nil
	})
	input := intList()

	result, err := listFilterImpl(ctx, []eval.Value{alwaysTrue, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	if len(got) != 0 {
		t.Fatalf("expected empty list, got %d elements", len(got))
	}
}

func TestListFilterCallbackError(t *testing.T) {
	ctx := newTestEffCtx()
	failing := goFn(func(args []eval.Value) (eval.Value, error) {
		return nil, fmt.Errorf("filter boom")
	})
	input := intList(1)

	_, err := listFilterImpl(ctx, []eval.Value{failing, input})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListFilterNonBoolReturn(t *testing.T) {
	ctx := newTestEffCtx()
	returnsInt := goFn(func(args []eval.Value) (eval.Value, error) {
		return &eval.IntValue{Value: 42}, nil
	})
	input := intList(1)

	_, err := listFilterImpl(ctx, []eval.Value{returnsInt, input})
	if err == nil {
		t.Fatal("expected error for non-bool predicate return")
	}
}

// ============================================================================
// _list_foldl tests
// ============================================================================

func TestListFoldlSum(t *testing.T) {
	ctx := newTestEffCtx()
	add := goFn(func(args []eval.Value) (eval.Value, error) {
		acc := args[0].(*eval.IntValue)
		elem := args[1].(*eval.IntValue)
		return &eval.IntValue{Value: acc.Value + elem.Value}, nil
	})
	input := intList(1, 2, 3, 4, 5)

	result, err := listFoldlImpl(ctx, []eval.Value{add, &eval.IntValue{Value: 0}, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	iv, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}
	if iv.Value != 15 {
		t.Errorf("expected 15, got %d", iv.Value)
	}
}

func TestListFoldlStringConcat(t *testing.T) {
	ctx := newTestEffCtx()
	concat := goFn(func(args []eval.Value) (eval.Value, error) {
		acc := args[0].(*eval.StringValue)
		elem := args[1].(*eval.StringValue)
		return &eval.StringValue{Value: acc.Value + elem.Value}, nil
	})
	input := &eval.ListValue{Elements: []eval.Value{
		&eval.StringValue{Value: "a"},
		&eval.StringValue{Value: "b"},
		&eval.StringValue{Value: "c"},
	}}

	result, err := listFoldlImpl(ctx, []eval.Value{concat, &eval.StringValue{Value: ""}, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sv, ok := result.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", result)
	}
	if sv.Value != "abc" {
		t.Errorf("expected 'abc', got '%s'", sv.Value)
	}
}

func TestListFoldlEmpty(t *testing.T) {
	ctx := newTestEffCtx()
	add := goFn(func(args []eval.Value) (eval.Value, error) {
		t.Fatal("callback should not be called on empty list")
		return nil, nil
	})
	input := intList()

	result, err := listFoldlImpl(ctx, []eval.Value{add, &eval.IntValue{Value: 42}, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	iv := result.(*eval.IntValue)
	if iv.Value != 42 {
		t.Errorf("expected initial acc 42, got %d", iv.Value)
	}
}

func TestListFoldlCallbackError(t *testing.T) {
	ctx := newTestEffCtx()
	failing := goFn(func(args []eval.Value) (eval.Value, error) {
		return nil, fmt.Errorf("fold boom")
	})
	input := intList(1)

	_, err := listFoldlImpl(ctx, []eval.Value{failing, &eval.IntValue{Value: 0}, input})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListFoldlNilFnCallerN(t *testing.T) {
	ctx := effects.NewEffContext(nil) // FnCallerN not set
	add := goFn(func(args []eval.Value) (eval.Value, error) { return args[0], nil })
	input := intList(1)

	_, err := listFoldlImpl(ctx, []eval.Value{add, &eval.IntValue{Value: 0}, input})
	if err == nil {
		t.Fatal("expected error when FnCallerN is nil")
	}
}

// ============================================================================
// Stress tests: 50K elements
// ============================================================================

func TestListMap50KElements(t *testing.T) {
	ctx := newTestEffCtx()
	double := goFn(func(args []eval.Value) (eval.Value, error) {
		v := args[0].(*eval.IntValue)
		return &eval.IntValue{Value: v.Value * 2}, nil
	})

	n := 50000
	elems := make([]eval.Value, n)
	for i := 0; i < n; i++ {
		elems[i] = &eval.IntValue{Value: i}
	}
	input := &eval.ListValue{Elements: elems}

	result, err := listMapImpl(ctx, []eval.Value{double, input})
	if err != nil {
		t.Fatalf("unexpected error on 50K map: %v", err)
	}

	list := result.(*eval.ListValue)
	if len(list.Elements) != n {
		t.Fatalf("expected %d elements, got %d", n, len(list.Elements))
	}
	// Spot check first, middle, last
	check := func(idx, expected int) {
		v := list.Elements[idx].(*eval.IntValue)
		if v.Value != expected {
			t.Errorf("index %d: got %d, want %d", idx, v.Value, expected)
		}
	}
	check(0, 0)
	check(n/2, n/2*2)
	check(n-1, (n-1)*2)
}

func TestListFilter50KElements(t *testing.T) {
	ctx := newTestEffCtx()
	isEven := goFn(func(args []eval.Value) (eval.Value, error) {
		v := args[0].(*eval.IntValue)
		return &eval.BoolValue{Value: v.Value%2 == 0}, nil
	})

	n := 50000
	elems := make([]eval.Value, n)
	for i := 0; i < n; i++ {
		elems[i] = &eval.IntValue{Value: i}
	}
	input := &eval.ListValue{Elements: elems}

	result, err := listFilterImpl(ctx, []eval.Value{isEven, input})
	if err != nil {
		t.Fatalf("unexpected error on 50K filter: %v", err)
	}

	list := result.(*eval.ListValue)
	if len(list.Elements) != n/2 {
		t.Fatalf("expected %d elements, got %d", n/2, len(list.Elements))
	}
}

func TestListFoldl50KElements(t *testing.T) {
	ctx := newTestEffCtx()
	add := goFn(func(args []eval.Value) (eval.Value, error) {
		acc := args[0].(*eval.IntValue)
		elem := args[1].(*eval.IntValue)
		return &eval.IntValue{Value: acc.Value + elem.Value}, nil
	})

	n := 50000
	elems := make([]eval.Value, n)
	for i := 0; i < n; i++ {
		elems[i] = &eval.IntValue{Value: i}
	}
	input := &eval.ListValue{Elements: elems}

	result, err := listFoldlImpl(ctx, []eval.Value{add, &eval.IntValue{Value: 0}, input})
	if err != nil {
		t.Fatalf("unexpected error on 50K foldl: %v", err)
	}

	iv := result.(*eval.IntValue)
	expected := n * (n - 1) / 2 // sum 0..49999
	if iv.Value != expected {
		t.Errorf("expected %d, got %d", expected, iv.Value)
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkListMap50K(b *testing.B) {
	ctx := newTestEffCtx()
	double := goFn(func(args []eval.Value) (eval.Value, error) {
		v := args[0].(*eval.IntValue)
		return &eval.IntValue{Value: v.Value * 2}, nil
	})

	n := 50000
	elems := make([]eval.Value, n)
	for i := 0; i < n; i++ {
		elems[i] = &eval.IntValue{Value: i}
	}
	input := &eval.ListValue{Elements: elems}
	args := []eval.Value{double, input}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = listMapImpl(ctx, args)
	}
}

func BenchmarkListFilter50K(b *testing.B) {
	ctx := newTestEffCtx()
	isEven := goFn(func(args []eval.Value) (eval.Value, error) {
		v := args[0].(*eval.IntValue)
		return &eval.BoolValue{Value: v.Value%2 == 0}, nil
	})

	n := 50000
	elems := make([]eval.Value, n)
	for i := 0; i < n; i++ {
		elems[i] = &eval.IntValue{Value: i}
	}
	input := &eval.ListValue{Elements: elems}
	args := []eval.Value{isEven, input}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = listFilterImpl(ctx, args)
	}
}

func BenchmarkListFoldl50K(b *testing.B) {
	ctx := newTestEffCtx()
	add := goFn(func(args []eval.Value) (eval.Value, error) {
		acc := args[0].(*eval.IntValue)
		elem := args[1].(*eval.IntValue)
		return &eval.IntValue{Value: acc.Value + elem.Value}, nil
	})

	n := 50000
	elems := make([]eval.Value, n)
	for i := 0; i < n; i++ {
		elems[i] = &eval.IntValue{Value: i}
	}
	input := &eval.ListValue{Elements: elems}
	args := []eval.Value{add, &eval.IntValue{Value: 0}, input}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = listFoldlImpl(ctx, args)
	}
}
