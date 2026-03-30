package builtins

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

var errTestCallbackStr = errors.New("test callback error")

// ============================================================================
// _str_foldChars tests
// ============================================================================

func TestStrFoldCharsConcat(t *testing.T) {
	ctx := newTestEffCtx()
	// Concatenate all characters: foldChars(\acc c -> acc ++ c, "", "abc") => "abc"
	concatFn := goFn(func(args []eval.Value) (eval.Value, error) {
		acc := args[0].(*eval.StringValue).Value
		ch := args[1].(*eval.StringValue).Value
		return &eval.StringValue{Value: acc + ch}, nil
	})

	result, err := strFoldCharsImpl(ctx, []eval.Value{
		concatFn,
		&eval.StringValue{Value: ""},
		&eval.StringValue{Value: "abc"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := result.(*eval.StringValue).Value
	if got != "abc" {
		t.Errorf("got %q, want %q", got, "abc")
	}
}

func TestStrFoldCharsCount(t *testing.T) {
	ctx := newTestEffCtx()
	// Count characters: foldChars(\acc _ -> acc + 1, 0, "hello") => 5
	countFn := goFn(func(args []eval.Value) (eval.Value, error) {
		acc := args[0].(*eval.IntValue).Value
		return &eval.IntValue{Value: acc + 1}, nil
	})

	result, err := strFoldCharsImpl(ctx, []eval.Value{
		countFn,
		&eval.IntValue{Value: 0},
		&eval.StringValue{Value: "hello"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := result.(*eval.IntValue).Value
	if got != 5 {
		t.Errorf("got %d, want 5", got)
	}
}

func TestStrFoldCharsEmpty(t *testing.T) {
	ctx := newTestEffCtx()
	called := false
	fn := goFn(func(args []eval.Value) (eval.Value, error) {
		called = true
		return args[0], nil
	})

	result, err := strFoldCharsImpl(ctx, []eval.Value{
		fn,
		&eval.StringValue{Value: "initial"},
		&eval.StringValue{Value: ""},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("callback should not be called for empty string")
	}
	got := result.(*eval.StringValue).Value
	if got != "initial" {
		t.Errorf("got %q, want %q", got, "initial")
	}
}

func TestStrFoldCharsUnicode(t *testing.T) {
	ctx := newTestEffCtx()
	// Collect characters from "a🎉b" — should be 3 runes, not 6 bytes
	var chars []string
	collectFn := goFn(func(args []eval.Value) (eval.Value, error) {
		ch := args[1].(*eval.StringValue).Value
		chars = append(chars, ch)
		return args[0], nil
	})

	_, err := strFoldCharsImpl(ctx, []eval.Value{
		collectFn,
		&eval.IntValue{Value: 0},
		&eval.StringValue{Value: "a🎉b"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chars) != 3 {
		t.Fatalf("got %d chars, want 3: %v", len(chars), chars)
	}
	if chars[0] != "a" || chars[1] != "🎉" || chars[2] != "b" {
		t.Errorf("got chars %v, want [a 🎉 b]", chars)
	}
}

func TestStrFoldCharsCallbackError(t *testing.T) {
	ctx := newTestEffCtx()
	errFn := goFn(func(args []eval.Value) (eval.Value, error) {
		ch := args[1].(*eval.StringValue).Value
		if ch == "c" {
			return nil, errTestCallbackStr
		}
		return args[0], nil
	})

	_, err := strFoldCharsImpl(ctx, []eval.Value{
		errFn,
		&eval.IntValue{Value: 0},
		&eval.StringValue{Value: "abcde"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "callback error") {
		t.Errorf("error should mention callback: %v", err)
	}
}

func TestStrFoldCharsNilFnCallerN(t *testing.T) {
	ctx := newTestEffCtx()
	ctx.FnCallerN = nil // Remove FnCallerN

	fn := goFn(func(args []eval.Value) (eval.Value, error) {
		return args[0], nil
	})

	_, err := strFoldCharsImpl(ctx, []eval.Value{
		fn,
		&eval.IntValue{Value: 0},
		&eval.StringValue{Value: "abc"},
	})
	if err == nil {
		t.Fatal("expected error for nil FnCallerN")
	}
	if !strings.Contains(err.Error(), "FnCallerN not set") {
		t.Errorf("error should mention FnCallerN: %v", err)
	}
}

func TestStrFoldCharsNonString(t *testing.T) {
	ctx := newTestEffCtx()
	fn := goFn(func(args []eval.Value) (eval.Value, error) {
		return args[0], nil
	})

	_, err := strFoldCharsImpl(ctx, []eval.Value{
		fn,
		&eval.IntValue{Value: 0},
		&eval.IntValue{Value: 42}, // not a string
	})
	if err == nil {
		t.Fatal("expected error for non-string argument")
	}
}

func TestStrFoldCharsStress10K(t *testing.T) {
	ctx := newTestEffCtx()
	// 10K character string — must not blow stack
	bigStr := strings.Repeat("x", 10000)
	countFn := goFn(func(args []eval.Value) (eval.Value, error) {
		acc := args[0].(*eval.IntValue).Value
		return &eval.IntValue{Value: acc + 1}, nil
	})

	result, err := strFoldCharsImpl(ctx, []eval.Value{
		countFn,
		&eval.IntValue{Value: 0},
		&eval.StringValue{Value: bigStr},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := result.(*eval.IntValue).Value
	if got != 10000 {
		t.Errorf("got %d, want 10000", got)
	}
}

// ============================================================================
// _str_charAt tests
// ============================================================================

func TestStrCharAtBasic(t *testing.T) {
	tests := []struct {
		str  string
		idx  int
		want string
	}{
		{"hello", 0, "h"},
		{"hello", 4, "o"},
		{"hello", 2, "l"},
	}
	for _, tt := range tests {
		result, err := strCharAtImpl(nil, []eval.Value{
			&eval.StringValue{Value: tt.str},
			&eval.IntValue{Value: tt.idx},
		})
		if err != nil {
			t.Fatalf("charAt(%q, %d): unexpected error: %v", tt.str, tt.idx, err)
		}
		got := result.(*eval.StringValue).Value
		if got != tt.want {
			t.Errorf("charAt(%q, %d) = %q, want %q", tt.str, tt.idx, got, tt.want)
		}
	}
}

func TestStrCharAtUnicode(t *testing.T) {
	// "a🎉b" has 3 runes: a (1 byte), 🎉 (4 bytes), b (1 byte)
	result, err := strCharAtImpl(nil, []eval.Value{
		&eval.StringValue{Value: "a🎉b"},
		&eval.IntValue{Value: 1},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := result.(*eval.StringValue).Value
	if got != "🎉" {
		t.Errorf("got %q, want %q", got, "🎉")
	}
}

func TestStrCharAtOutOfBounds(t *testing.T) {
	tests := []struct {
		str string
		idx int
	}{
		{"hello", 5},
		{"hello", -1},
		{"hello", 100},
	}
	for _, tt := range tests {
		_, err := strCharAtImpl(nil, []eval.Value{
			&eval.StringValue{Value: tt.str},
			&eval.IntValue{Value: tt.idx},
		})
		if err == nil {
			t.Errorf("charAt(%q, %d): expected error for out-of-bounds", tt.str, tt.idx)
		}
	}
}

func TestStrCharAtEmptyString(t *testing.T) {
	_, err := strCharAtImpl(nil, []eval.Value{
		&eval.StringValue{Value: ""},
		&eval.IntValue{Value: 0},
	})
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestStrCharAtNonString(t *testing.T) {
	_, err := strCharAtImpl(nil, []eval.Value{
		&eval.IntValue{Value: 42},
		&eval.IntValue{Value: 0},
	})
	if err == nil {
		t.Fatal("expected error for non-string first argument")
	}
}

func TestStrCharAtNonInt(t *testing.T) {
	_, err := strCharAtImpl(nil, []eval.Value{
		&eval.StringValue{Value: "hello"},
		&eval.StringValue{Value: "0"},
	})
	if err == nil {
		t.Fatal("expected error for non-int second argument")
	}
}

// ============================================================================
// _str_charCode tests
// ============================================================================

func TestStrCharCode(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"a", 97},
		{"A", 65},
		{"0", 48},
		{" ", 32},
		{"~", 126},
		{"🎉", 127881},
	}
	for _, tt := range tests {
		result, err := strCharCodeImpl(nil, []eval.Value{
			&eval.StringValue{Value: tt.input},
		})
		if err != nil {
			t.Fatalf("charCode(%q): unexpected error: %v", tt.input, err)
		}
		got := result.(*eval.IntValue).Value
		if got != tt.want {
			t.Errorf("charCode(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestStrCharCodeMultiChar(t *testing.T) {
	_, err := strCharCodeImpl(nil, []eval.Value{
		&eval.StringValue{Value: "ab"},
	})
	if err == nil {
		t.Fatal("expected error for multi-character string")
	}
}

func TestStrCharCodeEmpty(t *testing.T) {
	_, err := strCharCodeImpl(nil, []eval.Value{
		&eval.StringValue{Value: ""},
	})
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkStrFoldChars10K(b *testing.B) {
	ctx := newTestEffCtx()
	bigStr := &eval.StringValue{Value: strings.Repeat("x", 10000)}
	countFn := goFn(func(args []eval.Value) (eval.Value, error) {
		acc := args[0].(*eval.IntValue).Value
		return &eval.IntValue{Value: acc + 1}, nil
	})
	args := []eval.Value{countFn, &eval.IntValue{Value: 0}, bigStr}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = strFoldCharsImpl(ctx, args)
	}
}
