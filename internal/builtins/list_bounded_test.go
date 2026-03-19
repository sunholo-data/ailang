package builtins

import (
	"fmt"
	"testing"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// ============================================================================
// _list_takeMap tests
// ============================================================================

func TestTakeMapBasic(t *testing.T) {
	ctx := newTestEffCtx()
	double := goFn(func(args []eval.Value) (eval.Value, error) {
		v := args[0].(*eval.IntValue)
		return &eval.IntValue{Value: v.Value * 2}, nil
	})
	input := intList(1, 2, 3, 4, 5)

	result, err := takeMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 2}, double, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	expected := []int{2, 4}
	if len(got) != len(expected) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(expected))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("index %d: got %d, want %d", i, got[i], expected[i])
		}
	}
}

func TestTakeMapNGreaterThanLength(t *testing.T) {
	ctx := newTestEffCtx()
	double := goFn(func(args []eval.Value) (eval.Value, error) {
		v := args[0].(*eval.IntValue)
		return &eval.IntValue{Value: v.Value * 2}, nil
	})
	input := intList(1, 2, 3)

	result, err := takeMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 100}, double, input})
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

func TestTakeMapZero(t *testing.T) {
	ctx := newTestEffCtx()
	double := goFn(func(args []eval.Value) (eval.Value, error) {
		t.Fatal("callback should not be called when n=0")
		return nil, nil
	})
	input := intList(1, 2, 3)

	result, err := takeMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 0}, double, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	if len(got) != 0 {
		t.Fatalf("expected empty list, got %d elements", len(got))
	}
}

func TestTakeMapEmptyInput(t *testing.T) {
	ctx := newTestEffCtx()
	double := goFn(func(args []eval.Value) (eval.Value, error) {
		t.Fatal("callback should not be called on empty list")
		return nil, nil
	})
	input := intList()

	result, err := takeMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 5}, double, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	if len(got) != 0 {
		t.Fatalf("expected empty list, got %d elements", len(got))
	}
}

func TestTakeMapEarlyExit(t *testing.T) {
	ctx := newTestEffCtx()
	callCount := 0
	counter := goFn(func(args []eval.Value) (eval.Value, error) {
		callCount++
		return args[0], nil
	})
	input := intList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

	_, err := takeMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 3}, counter, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 3 {
		t.Errorf("expected f to be called 3 times, got %d", callCount)
	}
}

func TestTakeMapCallbackError(t *testing.T) {
	ctx := newTestEffCtx()
	failing := goFn(func(args []eval.Value) (eval.Value, error) {
		return nil, fmt.Errorf("boom")
	})
	input := intList(1, 2, 3)

	_, err := takeMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 2}, failing, input})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTakeMapNilFnCaller(t *testing.T) {
	ctx := effects.NewEffContext(nil)
	identity := goFn(func(args []eval.Value) (eval.Value, error) { return args[0], nil })
	input := intList(1)

	_, err := takeMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 1}, identity, input})
	if err == nil {
		t.Fatal("expected error when FnCaller is nil")
	}
}

func TestTakeMapNonIntN(t *testing.T) {
	ctx := newTestEffCtx()
	identity := goFn(func(args []eval.Value) (eval.Value, error) { return args[0], nil })
	input := intList(1)

	_, err := takeMapImpl(ctx, []eval.Value{&eval.StringValue{Value: "bad"}, identity, input})
	if err == nil {
		t.Fatal("expected error for non-Int n")
	}
}

func TestTakeMapNonListInput(t *testing.T) {
	ctx := newTestEffCtx()
	identity := goFn(func(args []eval.Value) (eval.Value, error) { return args[0], nil })

	_, err := takeMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 1}, identity, &eval.IntValue{Value: 42}})
	if err == nil {
		t.Fatal("expected error for non-list arg")
	}
}

// ============================================================================
// _list_takeFlatMap tests
// ============================================================================

func TestTakeFlatMapBasic(t *testing.T) {
	ctx := newTestEffCtx()
	// f(x) = [x, x] — each element doubles
	duplicate := goFn(func(args []eval.Value) (eval.Value, error) {
		v := args[0]
		return &eval.ListValue{Elements: []eval.Value{v, v}}, nil
	})
	input := intList(1, 2, 3, 4, 5)

	result, err := takeFlatMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 3}, duplicate, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	// f(1) = [1,1], f(2) = [2,2] → [1,1,2] (first 3)
	expected := []int{1, 1, 2}
	if len(got) != len(expected) {
		t.Fatalf("length mismatch: got %v, want %v", got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("index %d: got %d, want %d", i, got[i], expected[i])
		}
	}
}

func TestTakeFlatMapNGreaterThanTotal(t *testing.T) {
	ctx := newTestEffCtx()
	duplicate := goFn(func(args []eval.Value) (eval.Value, error) {
		v := args[0]
		return &eval.ListValue{Elements: []eval.Value{v, v}}, nil
	})
	input := intList(1, 2)

	result, err := takeFlatMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 100}, duplicate, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	expected := []int{1, 1, 2, 2} // all 4 results (less than 100)
	if len(got) != len(expected) {
		t.Fatalf("length mismatch: got %v, want %v", got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("index %d: got %d, want %d", i, got[i], expected[i])
		}
	}
}

func TestTakeFlatMapZero(t *testing.T) {
	ctx := newTestEffCtx()
	duplicate := goFn(func(args []eval.Value) (eval.Value, error) {
		t.Fatal("callback should not be called when n=0")
		return nil, nil
	})
	input := intList(1, 2, 3)

	result, err := takeFlatMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 0}, duplicate, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	if len(got) != 0 {
		t.Fatalf("expected empty list, got %d elements", len(got))
	}
}

func TestTakeFlatMapEmptyInput(t *testing.T) {
	ctx := newTestEffCtx()
	duplicate := goFn(func(args []eval.Value) (eval.Value, error) {
		t.Fatal("callback should not be called on empty list")
		return nil, nil
	})
	input := intList()

	result, err := takeFlatMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 5}, duplicate, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	if len(got) != 0 {
		t.Fatalf("expected empty list, got %d elements", len(got))
	}
}

func TestTakeFlatMapEarlyExit(t *testing.T) {
	ctx := newTestEffCtx()
	callCount := 0
	// Each call returns 3 elements
	expand := goFn(func(args []eval.Value) (eval.Value, error) {
		callCount++
		v := args[0].(*eval.IntValue)
		return &eval.ListValue{Elements: []eval.Value{
			&eval.IntValue{Value: v.Value * 10},
			&eval.IntValue{Value: v.Value*10 + 1},
			&eval.IntValue{Value: v.Value*10 + 2},
		}}, nil
	})
	input := intList(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

	result, err := takeFlatMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 5}, expand, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	// f(1)=[10,11,12], f(2)=[20,21,22] → take 5 → [10,11,12,20,21]
	expected := []int{10, 11, 12, 20, 21}
	if len(got) != len(expected) {
		t.Fatalf("length mismatch: got %v, want %v", got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("index %d: got %d, want %d", i, got[i], expected[i])
		}
	}

	// Should only call f twice (3 from first + 2 from second = 5 total)
	if callCount != 2 {
		t.Errorf("expected f to be called 2 times, got %d", callCount)
	}
}

func TestTakeFlatMapEmptyInnerLists(t *testing.T) {
	ctx := newTestEffCtx()
	callCount := 0
	// Alternates: empty, [x], empty, [x], ...
	alternate := goFn(func(args []eval.Value) (eval.Value, error) {
		callCount++
		v := args[0].(*eval.IntValue)
		if v.Value%2 == 0 {
			return &eval.ListValue{Elements: []eval.Value{}}, nil
		}
		return &eval.ListValue{Elements: []eval.Value{v}}, nil
	})
	input := intList(0, 1, 2, 3, 4, 5, 6, 7)

	result, err := takeFlatMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 3}, alternate, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	// evens produce [], odds produce [x]: 0→[], 1→[1], 2→[], 3→[3], 4→[], 5→[5]
	expected := []int{1, 3, 5}
	if len(got) != len(expected) {
		t.Fatalf("length mismatch: got %v, want %v", got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("index %d: got %d, want %d", i, got[i], expected[i])
		}
	}

	// Should call f 6 times (0,1,2,3,4,5) to get 3 results
	if callCount != 6 {
		t.Errorf("expected f to be called 6 times, got %d", callCount)
	}
}

func TestTakeFlatMapCallbackError(t *testing.T) {
	ctx := newTestEffCtx()
	failing := goFn(func(args []eval.Value) (eval.Value, error) {
		return nil, fmt.Errorf("flatmap boom")
	})
	input := intList(1, 2, 3)

	_, err := takeFlatMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 2}, failing, input})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTakeFlatMapNonListReturn(t *testing.T) {
	ctx := newTestEffCtx()
	returnsInt := goFn(func(args []eval.Value) (eval.Value, error) {
		return &eval.IntValue{Value: 42}, nil
	})
	input := intList(1)

	_, err := takeFlatMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 5}, returnsInt, input})
	if err == nil {
		t.Fatal("expected error when callback returns non-list")
	}
}

func TestTakeFlatMapNilFnCaller(t *testing.T) {
	ctx := effects.NewEffContext(nil)
	duplicate := goFn(func(args []eval.Value) (eval.Value, error) {
		return &eval.ListValue{Elements: []eval.Value{args[0]}}, nil
	})
	input := intList(1)

	_, err := takeFlatMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 1}, duplicate, input})
	if err == nil {
		t.Fatal("expected error when FnCaller is nil")
	}
}

func TestTakeFlatMapOne(t *testing.T) {
	ctx := newTestEffCtx()
	duplicate := goFn(func(args []eval.Value) (eval.Value, error) {
		v := args[0]
		return &eval.ListValue{Elements: []eval.Value{v, v, v}}, nil
	})
	input := intList(7, 8, 9)

	result, err := takeFlatMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 1}, duplicate, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := toInts(t, result)
	if len(got) != 1 || got[0] != 7 {
		t.Errorf("expected [7], got %v", got)
	}
}

// ============================================================================
// Stress test: bounded vs unbounded comparison
// ============================================================================

func TestTakeFlatMapLargeInput(t *testing.T) {
	ctx := newTestEffCtx()
	callCount := 0
	expand := goFn(func(args []eval.Value) (eval.Value, error) {
		callCount++
		v := args[0].(*eval.IntValue)
		return &eval.ListValue{Elements: []eval.Value{
			&eval.IntValue{Value: v.Value},
		}}, nil
	})

	// 10K input elements, but only take 10
	n := 10000
	elems := make([]eval.Value, n)
	for i := 0; i < n; i++ {
		elems[i] = &eval.IntValue{Value: i}
	}
	input := &eval.ListValue{Elements: elems}

	result, err := takeFlatMapImpl(ctx, []eval.Value{&eval.IntValue{Value: 10}, expand, input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list := result.(*eval.ListValue)
	if len(list.Elements) != 10 {
		t.Fatalf("expected 10 elements, got %d", len(list.Elements))
	}

	// Key assertion: f should only be called 10 times, NOT 10000
	if callCount != 10 {
		t.Errorf("expected f to be called 10 times (early exit), got %d", callCount)
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkTakeMap(b *testing.B) {
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
	args := []eval.Value{&eval.IntValue{Value: 100}, double, input}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = takeMapImpl(ctx, args)
	}
}

func BenchmarkTakeFlatMap(b *testing.B) {
	ctx := newTestEffCtx()
	expand := goFn(func(args []eval.Value) (eval.Value, error) {
		v := args[0].(*eval.IntValue)
		return &eval.ListValue{Elements: []eval.Value{
			&eval.IntValue{Value: v.Value},
			&eval.IntValue{Value: v.Value + 1},
		}}, nil
	})

	n := 50000
	elems := make([]eval.Value, n)
	for i := 0; i < n; i++ {
		elems[i] = &eval.IntValue{Value: i}
	}
	input := &eval.ListValue{Elements: elems}
	args := []eval.Value{&eval.IntValue{Value: 100}, expand, input}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = takeFlatMapImpl(ctx, args)
	}
}
