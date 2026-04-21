package builtins

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/effects/testctx"
	"github.com/sunholo-data/ailang/internal/eval"
)

func TestArrayMake(t *testing.T) {
	mockCtx := testctx.NewMockEffContext()
	ctx := mockCtx.EffContext

	tests := []struct {
		name    string
		size    int
		defVal  int
		wantLen int
	}{
		{"empty", 0, 0, 0},
		{"single", 1, 42, 1},
		{"multiple", 5, 99, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := arrayMakeImpl(ctx, []eval.Value{
				&eval.IntValue{Value: tt.size},
				&eval.IntValue{Value: tt.defVal},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			arr, ok := result.(*eval.ArrayValue)
			if !ok {
				t.Fatalf("expected ArrayValue, got %T", result)
			}
			if len(arr.Elements) != tt.wantLen {
				t.Errorf("want len %d, got %d", tt.wantLen, len(arr.Elements))
			}
			for i, elem := range arr.Elements {
				v, ok := elem.(*eval.IntValue)
				if !ok {
					t.Errorf("element %d: expected IntValue, got %T", i, elem)
					continue
				}
				if v.Value != tt.defVal {
					t.Errorf("element %d: want %d, got %d", i, tt.defVal, v.Value)
				}
			}
		})
	}
}

func TestArrayMakeNegativeSize(t *testing.T) {
	ctx := testctx.NewMockEffContext().EffContext
	_, err := arrayMakeImpl(ctx, []eval.Value{
		&eval.IntValue{Value: -1},
		&eval.IntValue{Value: 0},
	})
	if err == nil {
		t.Fatal("expected error for negative size")
	}
}

func TestArrayGet(t *testing.T) {
	ctx := testctx.NewMockEffContext().EffContext
	arr := &eval.ArrayValue{
		Elements: []eval.Value{
			&eval.IntValue{Value: 10},
			&eval.IntValue{Value: 20},
			&eval.IntValue{Value: 30},
		},
	}

	tests := []struct {
		name    string
		idx     int
		want    int
		wantErr bool
	}{
		{"first", 0, 10, false},
		{"middle", 1, 20, false},
		{"last", 2, 30, false},
		{"out_of_bounds", 5, 0, true},
		{"negative", -1, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := arrayGetImpl(ctx, []eval.Value{
				arr,
				&eval.IntValue{Value: tt.idx},
			})
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			v, ok := result.(*eval.IntValue)
			if !ok {
				t.Fatalf("expected IntValue, got %T", result)
			}
			if v.Value != tt.want {
				t.Errorf("want %d, got %d", tt.want, v.Value)
			}
		})
	}
}

func TestArraySet(t *testing.T) {
	ctx := testctx.NewMockEffContext().EffContext
	original := &eval.ArrayValue{
		Elements: []eval.Value{
			&eval.IntValue{Value: 1},
			&eval.IntValue{Value: 2},
			&eval.IntValue{Value: 3},
		},
	}

	// Set middle element
	result, err := arraySetImpl(ctx, []eval.Value{
		original,
		&eval.IntValue{Value: 1},
		&eval.IntValue{Value: 99},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newArr, ok := result.(*eval.ArrayValue)
	if !ok {
		t.Fatalf("expected ArrayValue, got %T", result)
	}

	// Check new array has updated value
	if v, ok := newArr.Elements[1].(*eval.IntValue); !ok || v.Value != 99 {
		t.Error("new array should have value 99 at index 1")
	}

	// Check original is unchanged (copy-on-write)
	if v, ok := original.Elements[1].(*eval.IntValue); !ok || v.Value != 2 {
		t.Error("original array should be unchanged")
	}
}

func TestArrayLength(t *testing.T) {
	ctx := testctx.NewMockEffContext().EffContext

	tests := []struct {
		name    string
		len     int
		wantLen int
	}{
		{"empty", 0, 0},
		{"single", 1, 1},
		{"multiple", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arr := &eval.ArrayValue{Elements: make([]eval.Value, tt.len)}
			result, err := arrayLengthImpl(ctx, []eval.Value{arr})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			v, ok := result.(*eval.IntValue)
			if !ok {
				t.Fatalf("expected IntValue, got %T", result)
			}
			if v.Value != tt.wantLen {
				t.Errorf("want %d, got %d", tt.wantLen, v.Value)
			}
		})
	}
}

func TestArrayFromList(t *testing.T) {
	ctx := testctx.NewMockEffContext().EffContext
	list := &eval.ListValue{
		Elements: []eval.Value{
			&eval.IntValue{Value: 1},
			&eval.IntValue{Value: 2},
			&eval.IntValue{Value: 3},
		},
	}

	result, err := arrayFromListImpl(ctx, []eval.Value{list})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	arr, ok := result.(*eval.ArrayValue)
	if !ok {
		t.Fatalf("expected ArrayValue, got %T", result)
	}

	if len(arr.Elements) != 3 {
		t.Errorf("want len 3, got %d", len(arr.Elements))
	}

	// Verify values
	for i, want := range []int{1, 2, 3} {
		v, ok := arr.Elements[i].(*eval.IntValue)
		if !ok {
			t.Errorf("element %d: expected IntValue, got %T", i, arr.Elements[i])
			continue
		}
		if v.Value != want {
			t.Errorf("element %d: want %d, got %d", i, want, v.Value)
		}
	}
}

func TestArrayToList(t *testing.T) {
	ctx := testctx.NewMockEffContext().EffContext
	arr := &eval.ArrayValue{
		Elements: []eval.Value{
			&eval.StringValue{Value: "a"},
			&eval.StringValue{Value: "b"},
			&eval.StringValue{Value: "c"},
		},
	}

	result, err := arrayToListImpl(ctx, []eval.Value{arr})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list, ok := result.(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", result)
	}

	if len(list.Elements) != 3 {
		t.Errorf("want len 3, got %d", len(list.Elements))
	}

	// Verify values
	for i, want := range []string{"a", "b", "c"} {
		v, ok := list.Elements[i].(*eval.StringValue)
		if !ok {
			t.Errorf("element %d: expected StringValue, got %T", i, list.Elements[i])
			continue
		}
		if v.Value != want {
			t.Errorf("element %d: want %s, got %s", i, want, v.Value)
		}
	}
}

func TestArrayUnsafeGet(t *testing.T) {
	ctx := testctx.NewMockEffContext().EffContext
	arr := &eval.ArrayValue{
		Elements: []eval.Value{
			&eval.IntValue{Value: 42},
		},
	}

	// Valid access
	result, err := arrayUnsafeGetImpl(ctx, []eval.Value{
		arr,
		&eval.IntValue{Value: 0},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}
	if v.Value != 42 {
		t.Errorf("want 42, got %d", v.Value)
	}

	// Out of bounds should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for out of bounds access")
		}
	}()
	_, _ = arrayUnsafeGetImpl(ctx, []eval.Value{
		arr,
		&eval.IntValue{Value: 99},
	})
}
