package builtins

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

func init() {
	registerShow()
}

func registerShow() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "show",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeShowType,
		Impl:    showImpl,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register show: %v", err))
	}
}

func makeShowType() types.Type {
	// show : ∀α. α -> string
	// For now, we use a type variable directly which will be generalized
	// by the type system. This is similar to how v0.3.9 worked.
	alpha := &types.TVar2{Name: "α", Kind: types.Star}
	return &types.TFunc2{
		Params:    []types.Type{alpha},
		Return:    types.TString,
		EffectRow: types.EmptyEffectRow(),
	}
}

func showImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	val := args[0]
	return &eval.StringValue{Value: showValue(val, 0)}, nil
}

// Constants for show function
const (
	maxDepth      = 3
	maxWidth      = 80
	elisionPrefix = 20
	elisionSuffix = 20
)

// showValue converts a value to its canonical string representation
// with proper quoting, escaping, and deterministic output.
// This implementation is based on v0.3.9's showValue function.
func showValue(v eval.Value, depth int) string {
	if depth > maxDepth {
		return "..."
	}

	switch val := v.(type) {
	case *eval.IntValue:
		return strconv.Itoa(val.Value)

	case *eval.FloatValue:
		// Handle special cases
		if math.IsNaN(val.Value) {
			return "NaN"
		}
		if math.IsInf(val.Value, 1) {
			return "Inf"
		}
		if math.IsInf(val.Value, -1) {
			return "-Inf"
		}
		// Use %g format for general float representation
		// This gives "5" for 5.0, "3.14" for 3.14, etc.
		return strconv.FormatFloat(val.Value, 'g', -1, 64)

	case *eval.BoolValue:
		if val.Value {
			return "true"
		}
		return "false"

	case *eval.StringValue:
		// Return string without quotes (identity for strings)
		return val.Value

	case *eval.ListValue:
		if len(val.Elements) == 0 {
			return "[]"
		}
		var parts []string
		for _, elem := range val.Elements {
			parts = append(parts, showValue(elem, depth+1))
		}
		result := "[" + strings.Join(parts, ", ") + "]"
		return truncateIfNeeded(result)

	case *eval.RecordValue:
		if len(val.Fields) == 0 {
			return "{}"
		}
		// Sort keys for deterministic output
		keys := make([]string, 0, len(val.Fields))
		for k := range val.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var parts []string
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s: %s", k, showValue(val.Fields[k], depth+1)))
		}
		result := "{" + strings.Join(parts, ", ") + "}"
		return truncateIfNeeded(result)

	case *eval.TaggedValue:
		// ADT constructors: Some(42) → "Some(42)"
		if len(val.Fields) == 0 {
			return val.CtorName
		}
		var argStrs []string
		for _, arg := range val.Fields {
			argStrs = append(argStrs, showValue(arg, depth+1))
		}
		return val.CtorName + "(" + strings.Join(argStrs, ", ") + ")"

	case *eval.FunctionValue:
		return "<function>"

	case *eval.ErrorValue:
		return fmt.Sprintf("Error: %s", val.Message)

	default:
		// Fallback for unknown types
		return fmt.Sprintf("<%T>", val)
	}
}

// truncateIfNeeded elides the middle of long strings to keep under maxWidth
func truncateIfNeeded(s string) string {
	if len(s) <= maxWidth {
		return s
	}

	// Calculate elision: keep prefix and suffix, replace middle with "..."
	if elisionPrefix+elisionSuffix+3 >= len(s) {
		return s // Too short to bother eliding
	}

	prefix := s[:elisionPrefix]
	suffix := s[len(s)-elisionSuffix:]
	return prefix + "..." + suffix
}
