package vm

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// M-BYTECODE-HOF-BUILTINS: Higher-order builtin implementations.
// Each function receives a ClosureCaller (the VM) and can invoke closure
// arguments via caller.CallClosure.

// hofBuiltinListMap implements __list_map: (a -> b, [a]) -> [b]
func hofBuiltinListMap(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__list_map: expected 2 args, got %d", len(args))
	}
	fn := args[0]
	if args[1].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__list_map: arg 1 must be list, got %s", args[1].Tag)
	}
	elems := args[1].AsList()
	result := make([]bytecode.Value, len(elems))
	for i, e := range elems {
		val, err := caller.CallClosure(fn, []bytecode.Value{e})
		if err != nil {
			return bytecode.Value{}, fmt.Errorf("__list_map: callback error at index %d: %w", i, err)
		}
		result[i] = val
	}
	return bytecode.NewList(result), nil
}

// hofBuiltinListFilter implements __list_filter: (a -> bool, [a]) -> [a]
func hofBuiltinListFilter(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__list_filter: expected 2 args, got %d", len(args))
	}
	fn := args[0]
	if args[1].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__list_filter: arg 1 must be list, got %s", args[1].Tag)
	}
	elems := args[1].AsList()
	result := make([]bytecode.Value, 0, len(elems))
	for i, e := range elems {
		val, err := caller.CallClosure(fn, []bytecode.Value{e})
		if err != nil {
			return bytecode.Value{}, fmt.Errorf("__list_filter: callback error at index %d: %w", i, err)
		}
		if val.Tag != bytecode.TagBool {
			return bytecode.Value{}, fmt.Errorf("__list_filter: predicate must return bool, got %s at index %d", val.Tag, i)
		}
		if val.Bool {
			result = append(result, e)
		}
	}
	return bytecode.NewList(result), nil
}

// hofBuiltinListFoldl implements __list_foldl: ((b, a) -> b, b, [a]) -> b
func hofBuiltinListFoldl(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 3 {
		return bytecode.Value{}, fmt.Errorf("__list_foldl: expected 3 args, got %d", len(args))
	}
	fn := args[0]
	acc := args[1]
	if args[2].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__list_foldl: arg 2 must be list, got %s", args[2].Tag)
	}
	var err error
	for i, e := range args[2].AsList() {
		acc, err = caller.CallClosure(fn, []bytecode.Value{acc, e})
		if err != nil {
			return bytecode.Value{}, fmt.Errorf("__list_foldl: callback error at index %d: %w", i, err)
		}
	}
	return acc, nil
}

// hofBuiltinStrFoldChars implements __str_foldChars: ((a, string) -> a, a, string) -> a
func hofBuiltinStrFoldChars(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 3 {
		return bytecode.Value{}, fmt.Errorf("__str_foldChars: expected 3 args, got %d", len(args))
	}
	fn := args[0]
	acc := args[1]
	if args[2].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_foldChars: arg 2 must be string, got %s", args[2].Tag)
	}
	s := args[2].AsString()
	var err error
	for _, r := range s {
		charVal := bytecode.NewString(string(r))
		acc, err = caller.CallClosure(fn, []bytecode.Value{acc, charVal})
		if err != nil {
			return bytecode.Value{}, fmt.Errorf("__str_foldChars: %w", err)
		}
	}
	return acc, nil
}

// hofBuiltinStrFoldSlices implements __str_foldSlices: (string, string, a, (a, string) -> a) -> a
func hofBuiltinStrFoldSlices(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 4 {
		return bytecode.Value{}, fmt.Errorf("__str_foldSlices: expected 4 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_foldSlices: arg 0 must be string, got %s", args[0].Tag)
	}
	if args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_foldSlices: arg 1 must be string, got %s", args[1].Tag)
	}
	s := args[0].AsString()
	delim := args[1].AsString()
	acc := args[2]
	fn := args[3]

	var err error
	if delim == "" {
		// Empty delimiter: fold over each character
		for _, r := range s {
			segVal := bytecode.NewString(string(r))
			acc, err = caller.CallClosure(fn, []bytecode.Value{acc, segVal})
			if err != nil {
				return bytecode.Value{}, fmt.Errorf("__str_foldSlices: %w", err)
			}
		}
		return acc, nil
	}

	for {
		idx := strings.Index(s, delim)
		var segment string
		if idx == -1 {
			segment = s
		} else {
			segment = s[:idx]
		}
		segVal := bytecode.NewString(segment)
		acc, err = caller.CallClosure(fn, []bytecode.Value{acc, segVal})
		if err != nil {
			return bytecode.Value{}, fmt.Errorf("__str_foldSlices: %w", err)
		}
		if idx == -1 {
			break
		}
		s = s[idx+len(delim):]
	}
	return acc, nil
}

// hofBuiltinStrMapSlicesJoin implements __str_mapSlicesJoin: (string, string, (string) -> string) -> string
func hofBuiltinStrMapSlicesJoin(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 3 {
		return bytecode.Value{}, fmt.Errorf("__str_mapSlicesJoin: expected 3 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_mapSlicesJoin: arg 0 must be string, got %s", args[0].Tag)
	}
	if args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_mapSlicesJoin: arg 1 must be string, got %s", args[1].Tag)
	}
	s := args[0].AsString()
	delim := args[1].AsString()
	fn := args[2]

	var builder strings.Builder
	builder.Grow(len(s))

	if delim == "" {
		// Empty delimiter: transform each character
		for _, r := range s {
			segVal := bytecode.NewString(string(r))
			result, err := caller.CallClosure(fn, []bytecode.Value{segVal})
			if err != nil {
				return bytecode.Value{}, fmt.Errorf("__str_mapSlicesJoin: %w", err)
			}
			if result.Tag != bytecode.TagString {
				return bytecode.Value{}, fmt.Errorf("__str_mapSlicesJoin: callback must return string, got %s", result.Tag)
			}
			builder.WriteString(result.AsString())
		}
		return bytecode.NewString(builder.String()), nil
	}

	for {
		idx := strings.Index(s, delim)
		var segment string
		if idx == -1 {
			segment = s
		} else {
			segment = s[:idx]
		}
		segVal := bytecode.NewString(segment)
		result, err := caller.CallClosure(fn, []bytecode.Value{segVal})
		if err != nil {
			return bytecode.Value{}, fmt.Errorf("__str_mapSlicesJoin: %w", err)
		}
		if result.Tag != bytecode.TagString {
			return bytecode.Value{}, fmt.Errorf("__str_mapSlicesJoin: callback must return string, got %s", result.Tag)
		}
		builder.WriteString(result.AsString())
		if idx == -1 {
			break
		}
		s = s[idx+len(delim):]
	}
	return bytecode.NewString(builder.String()), nil
}
