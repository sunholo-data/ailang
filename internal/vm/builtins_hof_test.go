package vm

import (
	"fmt"
	"testing"

	"github.com/sunholo/ailang/internal/bytecode"
)

// mockClosureCaller is a test double for ClosureCaller. It wraps a Go function
// so we can test HOF builtins without needing a full VM.
type mockClosureCaller struct {
	fn func(args []bytecode.Value) (bytecode.Value, error)
}

func (m *mockClosureCaller) CallClosure(closure bytecode.Value, args []bytecode.Value) (bytecode.Value, error) {
	return m.fn(args)
}

// helper: make a mock caller from a simple Go function.
func mockCaller(fn func(args []bytecode.Value) (bytecode.Value, error)) ClosureCaller {
	return &mockClosureCaller{fn: fn}
}

// dummyClosure returns a Value with TagClosure for passing as the function arg.
// The actual closure content doesn't matter since mockClosureCaller ignores it.
var dummyClosure = bytecode.NewClosure(&bytecode.FuncPrototype{Name: "test_fn"}, nil)

// ============================================================================
// __list_map tests
// ============================================================================

func TestHOFListMap_Identity(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return args[0], nil // identity
	})
	list := bytecode.NewList([]bytecode.Value{
		bytecode.NewInt(1), bytecode.NewInt(2), bytecode.NewInt(3),
	})
	result, err := hofBuiltinListMap(caller, []bytecode.Value{dummyClosure, list})
	if err != nil {
		t.Fatal(err)
	}
	elems := result.AsList()
	if len(elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elems))
	}
	for i, v := range elems {
		if v.Int != int64(i+1) {
			t.Errorf("elem %d: expected %d, got %d", i, i+1, v.Int)
		}
	}
}

func TestHOFListMap_Double(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return bytecode.NewInt(args[0].Int * 2), nil
	})
	list := bytecode.NewList([]bytecode.Value{
		bytecode.NewInt(10), bytecode.NewInt(20),
	})
	result, err := hofBuiltinListMap(caller, []bytecode.Value{dummyClosure, list})
	if err != nil {
		t.Fatal(err)
	}
	elems := result.AsList()
	if elems[0].Int != 20 || elems[1].Int != 40 {
		t.Errorf("expected [20, 40], got [%d, %d]", elems[0].Int, elems[1].Int)
	}
}

func TestHOFListMap_EmptyList(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		t.Fatal("should not be called for empty list")
		return bytecode.Value{}, nil
	})
	list := bytecode.NewList(nil)
	result, err := hofBuiltinListMap(caller, []bytecode.Value{dummyClosure, list})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.AsList()) != 0 {
		t.Errorf("expected empty list, got %d elements", len(result.AsList()))
	}
}

func TestHOFListMap_CallbackError(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return bytecode.Value{}, fmt.Errorf("boom")
	})
	list := bytecode.NewList([]bytecode.Value{bytecode.NewInt(1)})
	_, err := hofBuiltinListMap(caller, []bytecode.Value{dummyClosure, list})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ============================================================================
// __list_filter tests
// ============================================================================

func TestHOFListFilter_KeepEvens(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return bytecode.NewBool(args[0].Int%2 == 0), nil
	})
	list := bytecode.NewList([]bytecode.Value{
		bytecode.NewInt(1), bytecode.NewInt(2), bytecode.NewInt(3), bytecode.NewInt(4),
	})
	result, err := hofBuiltinListFilter(caller, []bytecode.Value{dummyClosure, list})
	if err != nil {
		t.Fatal(err)
	}
	elems := result.AsList()
	if len(elems) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(elems))
	}
	if elems[0].Int != 2 || elems[1].Int != 4 {
		t.Errorf("expected [2, 4], got [%d, %d]", elems[0].Int, elems[1].Int)
	}
}

func TestHOFListFilter_KeepNone(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return bytecode.NewBool(false), nil
	})
	list := bytecode.NewList([]bytecode.Value{bytecode.NewInt(1), bytecode.NewInt(2)})
	result, err := hofBuiltinListFilter(caller, []bytecode.Value{dummyClosure, list})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.AsList()) != 0 {
		t.Errorf("expected empty list")
	}
}

func TestHOFListFilter_EmptyList(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		t.Fatal("should not be called")
		return bytecode.Value{}, nil
	})
	list := bytecode.NewList(nil)
	result, err := hofBuiltinListFilter(caller, []bytecode.Value{dummyClosure, list})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.AsList()) != 0 {
		t.Errorf("expected empty list")
	}
}

// ============================================================================
// __list_foldl tests
// ============================================================================

func TestHOFListFoldl_Sum(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return bytecode.NewInt(args[0].Int + args[1].Int), nil
	})
	list := bytecode.NewList([]bytecode.Value{
		bytecode.NewInt(1), bytecode.NewInt(2), bytecode.NewInt(3),
	})
	result, err := hofBuiltinListFoldl(caller, []bytecode.Value{dummyClosure, bytecode.NewInt(0), list})
	if err != nil {
		t.Fatal(err)
	}
	if result.Int != 6 {
		t.Errorf("expected 6, got %d", result.Int)
	}
}

func TestHOFListFoldl_EmptyList(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		t.Fatal("should not be called")
		return bytecode.Value{}, nil
	})
	list := bytecode.NewList(nil)
	result, err := hofBuiltinListFoldl(caller, []bytecode.Value{dummyClosure, bytecode.NewInt(42), list})
	if err != nil {
		t.Fatal(err)
	}
	if result.Int != 42 {
		t.Errorf("expected initial acc 42, got %d", result.Int)
	}
}

func TestHOFListFoldl_StringConcat(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return bytecode.NewString(args[0].AsString() + args[1].AsString()), nil
	})
	list := bytecode.NewList([]bytecode.Value{
		bytecode.NewString("a"), bytecode.NewString("b"), bytecode.NewString("c"),
	})
	result, err := hofBuiltinListFoldl(caller, []bytecode.Value{dummyClosure, bytecode.NewString(""), list})
	if err != nil {
		t.Fatal(err)
	}
	if result.AsString() != "abc" {
		t.Errorf("expected \"abc\", got %q", result.AsString())
	}
}

// ============================================================================
// __str_foldChars tests
// ============================================================================

func TestHOFStrFoldChars_CountChars(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return bytecode.NewInt(args[0].Int + 1), nil
	})
	result, err := hofBuiltinStrFoldChars(caller, []bytecode.Value{
		dummyClosure, bytecode.NewInt(0), bytecode.NewString("hello"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Int != 5 {
		t.Errorf("expected 5, got %d", result.Int)
	}
}

func TestHOFStrFoldChars_ConcatChars(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return bytecode.NewString(args[0].AsString() + args[1].AsString()), nil
	})
	result, err := hofBuiltinStrFoldChars(caller, []bytecode.Value{
		dummyClosure, bytecode.NewString(""), bytecode.NewString("abc"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AsString() != "abc" {
		t.Errorf("expected \"abc\", got %q", result.AsString())
	}
}

func TestHOFStrFoldChars_EmptyString(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		t.Fatal("should not be called")
		return bytecode.Value{}, nil
	})
	result, err := hofBuiltinStrFoldChars(caller, []bytecode.Value{
		dummyClosure, bytecode.NewInt(0), bytecode.NewString(""),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Int != 0 {
		t.Errorf("expected 0, got %d", result.Int)
	}
}

func TestHOFStrFoldChars_Unicode(t *testing.T) {
	// Count runes in a Unicode string
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return bytecode.NewInt(args[0].Int + 1), nil
	})
	result, err := hofBuiltinStrFoldChars(caller, []bytecode.Value{
		dummyClosure, bytecode.NewInt(0), bytecode.NewString("héllo"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Int != 5 {
		t.Errorf("expected 5 runes, got %d", result.Int)
	}
}

// ============================================================================
// __str_foldSlices tests
// ============================================================================

func TestHOFStrFoldSlices_CountSegments(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return bytecode.NewInt(args[0].Int + 1), nil
	})
	result, err := hofBuiltinStrFoldSlices(caller, []bytecode.Value{
		bytecode.NewString("a,b,c"), bytecode.NewString(","),
		bytecode.NewInt(0), dummyClosure,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Int != 3 {
		t.Errorf("expected 3, got %d", result.Int)
	}
}

func TestHOFStrFoldSlices_ConcatSegments(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		acc := args[0].AsString()
		seg := args[1].AsString()
		if acc == "" {
			return bytecode.NewString(seg), nil
		}
		return bytecode.NewString(acc + "|" + seg), nil
	})
	result, err := hofBuiltinStrFoldSlices(caller, []bytecode.Value{
		bytecode.NewString("hello world foo"), bytecode.NewString(" "),
		bytecode.NewString(""), dummyClosure,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AsString() != "hello|world|foo" {
		t.Errorf("expected \"hello|world|foo\", got %q", result.AsString())
	}
}

func TestHOFStrFoldSlices_EmptyDelimiter(t *testing.T) {
	// Empty delimiter folds over each character
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return bytecode.NewInt(args[0].Int + 1), nil
	})
	result, err := hofBuiltinStrFoldSlices(caller, []bytecode.Value{
		bytecode.NewString("abc"), bytecode.NewString(""),
		bytecode.NewInt(0), dummyClosure,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Int != 3 {
		t.Errorf("expected 3, got %d", result.Int)
	}
}

func TestHOFStrFoldSlices_NoDelimiterFound(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return args[1], nil // return the segment
	})
	result, err := hofBuiltinStrFoldSlices(caller, []bytecode.Value{
		bytecode.NewString("hello"), bytecode.NewString(","),
		bytecode.NewString(""), dummyClosure,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AsString() != "hello" {
		t.Errorf("expected \"hello\", got %q", result.AsString())
	}
}

// ============================================================================
// __str_mapSlicesJoin tests
// ============================================================================

func TestHOFStrMapSlicesJoin_UpperSegments(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		s := args[0].AsString()
		// Simple uppercase: just prepend "[" and append "]"
		return bytecode.NewString("[" + s + "]"), nil
	})
	result, err := hofBuiltinStrMapSlicesJoin(caller, []bytecode.Value{
		bytecode.NewString("a,b,c"), bytecode.NewString(","), dummyClosure,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AsString() != "[a][b][c]" {
		t.Errorf("expected \"[a][b][c]\", got %q", result.AsString())
	}
}

func TestHOFStrMapSlicesJoin_Identity(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return args[0], nil // identity
	})
	result, err := hofBuiltinStrMapSlicesJoin(caller, []bytecode.Value{
		bytecode.NewString("hello world"), bytecode.NewString(" "), dummyClosure,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Identity on segments, but delimiter is consumed: "hello" + "world"
	if result.AsString() != "helloworld" {
		t.Errorf("expected \"helloworld\", got %q", result.AsString())
	}
}

func TestHOFStrMapSlicesJoin_EmptyDelimiter(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		s := args[0].AsString()
		return bytecode.NewString(s + s), nil // double each char
	})
	result, err := hofBuiltinStrMapSlicesJoin(caller, []bytecode.Value{
		bytecode.NewString("abc"), bytecode.NewString(""), dummyClosure,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AsString() != "aabbcc" {
		t.Errorf("expected \"aabbcc\", got %q", result.AsString())
	}
}

func TestHOFStrMapSlicesJoin_NoDelimiterFound(t *testing.T) {
	caller := mockCaller(func(args []bytecode.Value) (bytecode.Value, error) {
		return bytecode.NewString("X"), nil // replace with X
	})
	result, err := hofBuiltinStrMapSlicesJoin(caller, []bytecode.Value{
		bytecode.NewString("hello"), bytecode.NewString(","), dummyClosure,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AsString() != "X" {
		t.Errorf("expected \"X\", got %q", result.AsString())
	}
}
