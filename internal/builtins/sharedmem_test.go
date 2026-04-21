package builtins

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// createSharedMemContext creates a test context with SharedMem enabled
func createSharedMemContext() *effects.EffContext {
	cache := effects.NewInMemorySharedCache()
	return &effects.EffContext{
		SharedMem: effects.NewSharedMemContext(cache),
	}
}

func TestSharedMemGet_NotFound(t *testing.T) {
	ctx := createSharedMemContext()
	args := []eval.Value{&eval.StringValue{Value: "nonexistent"}}

	result, err := sharedMemGetImpl(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tagged, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}

	if tagged.CtorName != "None" {
		t.Errorf("expected None, got %s", tagged.CtorName)
	}
}

func TestSharedMemPutAndGet(t *testing.T) {
	ctx := createSharedMemContext()

	// Put a value
	putArgs := []eval.Value{
		&eval.StringValue{Value: "mykey"},
		&eval.BytesValue{Value: []byte("myvalue")},
	}

	result, err := sharedMemPutImpl(ctx, putArgs)
	if err != nil {
		t.Fatalf("put error: %v", err)
	}

	if _, ok := result.(*eval.UnitValue); !ok {
		t.Errorf("put should return UnitValue, got %T", result)
	}

	// Get it back
	getArgs := []eval.Value{&eval.StringValue{Value: "mykey"}}
	result, err = sharedMemGetImpl(ctx, getArgs)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}

	tagged, ok := result.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", result)
	}

	if tagged.CtorName != "Some" {
		t.Fatalf("expected Some, got %s", tagged.CtorName)
	}

	if len(tagged.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(tagged.Fields))
	}

	bytesVal, ok := tagged.Fields[0].(*eval.BytesValue)
	if !ok {
		t.Fatalf("expected BytesValue, got %T", tagged.Fields[0])
	}

	if string(bytesVal.Value) != "myvalue" {
		t.Errorf("expected 'myvalue', got '%s'", bytesVal.Value)
	}
}

func TestSharedMemCAS_CreateIfAbsent(t *testing.T) {
	ctx := createSharedMemContext()

	// CAS with None (create-if-absent)
	casArgs := []eval.Value{
		&eval.StringValue{Value: "newkey"},
		&eval.TaggedValue{
			ModulePath: "std/option",
			TypeName:   "Option",
			CtorName:   "None",
			Fields:     []eval.Value{},
		},
		&eval.BytesValue{Value: []byte("initial")},
	}

	result, err := sharedMemCASImpl(ctx, casArgs)
	if err != nil {
		t.Fatalf("cas error: %v", err)
	}

	boolVal, ok := result.(*eval.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T", result)
	}

	if !boolVal.Value {
		t.Error("CAS with None should succeed for new key")
	}

	// Verify the value was set
	getArgs := []eval.Value{&eval.StringValue{Value: "newkey"}}
	result, err = sharedMemGetImpl(ctx, getArgs)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}

	tagged := result.(*eval.TaggedValue)
	if tagged.CtorName != "Some" {
		t.Error("key should exist after CAS")
	}
}

func TestSharedMemCAS_UpdateExisting(t *testing.T) {
	ctx := createSharedMemContext()

	// First, put a value
	putArgs := []eval.Value{
		&eval.StringValue{Value: "key"},
		&eval.BytesValue{Value: []byte("original")},
	}
	_, _ = sharedMemPutImpl(ctx, putArgs)

	// CAS with correct old value
	casArgs := []eval.Value{
		&eval.StringValue{Value: "key"},
		&eval.TaggedValue{
			ModulePath: "std/option",
			TypeName:   "Option",
			CtorName:   "Some",
			Fields:     []eval.Value{&eval.BytesValue{Value: []byte("original")}},
		},
		&eval.BytesValue{Value: []byte("updated")},
	}

	result, err := sharedMemCASImpl(ctx, casArgs)
	if err != nil {
		t.Fatalf("cas error: %v", err)
	}

	if !result.(*eval.BoolValue).Value {
		t.Error("CAS with matching old value should succeed")
	}

	// Verify update
	getArgs := []eval.Value{&eval.StringValue{Value: "key"}}
	result, _ = sharedMemGetImpl(ctx, getArgs)
	tagged := result.(*eval.TaggedValue)
	bytesVal := tagged.Fields[0].(*eval.BytesValue)
	if string(bytesVal.Value) != "updated" {
		t.Errorf("expected 'updated', got '%s'", bytesVal.Value)
	}
}

func TestSharedMemCAS_FailsOnMismatch(t *testing.T) {
	ctx := createSharedMemContext()

	// First, put a value
	putArgs := []eval.Value{
		&eval.StringValue{Value: "key"},
		&eval.BytesValue{Value: []byte("original")},
	}
	_, _ = sharedMemPutImpl(ctx, putArgs)

	// CAS with wrong old value
	casArgs := []eval.Value{
		&eval.StringValue{Value: "key"},
		&eval.TaggedValue{
			ModulePath: "std/option",
			TypeName:   "Option",
			CtorName:   "Some",
			Fields:     []eval.Value{&eval.BytesValue{Value: []byte("wrong")}},
		},
		&eval.BytesValue{Value: []byte("updated")},
	}

	result, err := sharedMemCASImpl(ctx, casArgs)
	if err != nil {
		t.Fatalf("cas error: %v", err)
	}

	if result.(*eval.BoolValue).Value {
		t.Error("CAS with wrong old value should fail")
	}

	// Verify original unchanged
	getArgs := []eval.Value{&eval.StringValue{Value: "key"}}
	result, _ = sharedMemGetImpl(ctx, getArgs)
	tagged := result.(*eval.TaggedValue)
	bytesVal := tagged.Fields[0].(*eval.BytesValue)
	if string(bytesVal.Value) != "original" {
		t.Errorf("expected 'original', got '%s'", bytesVal.Value)
	}
}

func TestSharedMemDelete(t *testing.T) {
	ctx := createSharedMemContext()

	// Put a value
	putArgs := []eval.Value{
		&eval.StringValue{Value: "key"},
		&eval.BytesValue{Value: []byte("value")},
	}
	_, _ = sharedMemPutImpl(ctx, putArgs)

	// Delete it
	deleteArgs := []eval.Value{&eval.StringValue{Value: "key"}}
	result, err := sharedMemDeleteImpl(ctx, deleteArgs)
	if err != nil {
		t.Fatalf("delete error: %v", err)
	}

	if _, ok := result.(*eval.UnitValue); !ok {
		t.Errorf("delete should return UnitValue, got %T", result)
	}

	// Verify deleted
	getArgs := []eval.Value{&eval.StringValue{Value: "key"}}
	result, _ = sharedMemGetImpl(ctx, getArgs)
	tagged := result.(*eval.TaggedValue)
	if tagged.CtorName != "None" {
		t.Error("key should be gone after delete")
	}
}

func TestSharedMemKeys(t *testing.T) {
	ctx := createSharedMemContext()

	// Add some keys
	for _, key := range []string{"a", "b", "c"} {
		putArgs := []eval.Value{
			&eval.StringValue{Value: key},
			&eval.BytesValue{Value: []byte("value")},
		}
		_, _ = sharedMemPutImpl(ctx, putArgs)
	}

	// Get keys (pass unit argument per M-DX10 unit-argument model)
	result, err := sharedMemKeysImpl(ctx, []eval.Value{&eval.UnitValue{}})
	if err != nil {
		t.Fatalf("keys error: %v", err)
	}

	listVal, ok := result.(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", result)
	}

	if len(listVal.Elements) != 3 {
		t.Errorf("expected 3 keys, got %d", len(listVal.Elements))
	}

	// Verify all keys present
	keySet := make(map[string]bool)
	for _, elem := range listVal.Elements {
		strVal, ok := elem.(*eval.StringValue)
		if !ok {
			t.Errorf("expected StringValue, got %T", elem)
			continue
		}
		keySet[strVal.Value] = true
	}

	for _, expected := range []string{"a", "b", "c"} {
		if !keySet[expected] {
			t.Errorf("missing key: %s", expected)
		}
	}
}

func TestSharedMemGet_NoContext(t *testing.T) {
	ctx := &effects.EffContext{} // No SharedMem
	args := []eval.Value{&eval.StringValue{Value: "key"}}

	_, err := sharedMemGetImpl(ctx, args)
	if err == nil {
		t.Error("expected error when SharedMem not enabled")
	}
}

func TestSharedMemStats(t *testing.T) {
	ctx := createSharedMemContext()

	// Do some operations
	putArgs := []eval.Value{
		&eval.StringValue{Value: "key"},
		&eval.BytesValue{Value: []byte("value")},
	}
	_, _ = sharedMemPutImpl(ctx, putArgs)

	getArgs := []eval.Value{&eval.StringValue{Value: "key"}}
	_, _ = sharedMemGetImpl(ctx, getArgs)
	_, _ = sharedMemGetImpl(ctx, getArgs)

	casArgs := []eval.Value{
		&eval.StringValue{Value: "key"},
		&eval.TaggedValue{
			ModulePath: "std/option",
			TypeName:   "Option",
			CtorName:   "Some",
			Fields:     []eval.Value{&eval.BytesValue{Value: []byte("value")}},
		},
		&eval.BytesValue{Value: []byte("new")},
	}
	_, _ = sharedMemCASImpl(ctx, casArgs)

	// Check stats
	gets, puts, cas, casSuccess := ctx.SharedMem.Stats()
	if gets != 2 {
		t.Errorf("expected 2 gets, got %d", gets)
	}
	if puts != 1 {
		t.Errorf("expected 1 put, got %d", puts)
	}
	if cas != 1 {
		t.Errorf("expected 1 CAS, got %d", cas)
	}
	if casSuccess != 1 {
		t.Errorf("expected 1 CAS success, got %d", casSuccess)
	}
}
