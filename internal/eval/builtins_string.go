package eval

import (
	"strings"
	"unicode/utf8"
)

// registerStringBuiltins registers string comparison operations
func registerStringBuiltins() {
	Builtins["concat_String"] = &BuiltinFunc{
		Name:    "concat_String",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *StringValue) (*StringValue, error) {
			return &StringValue{Value: a.Value + b.Value}, nil
		},
	}

	Builtins["eq_String"] = &BuiltinFunc{
		Name:    "eq_String",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *StringValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value == b.Value}, nil
		},
	}

	Builtins["ne_String"] = &BuiltinFunc{
		Name:    "ne_String",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *StringValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value != b.Value}, nil
		},
	}

	Builtins["lt_String"] = &BuiltinFunc{
		Name:    "lt_String",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *StringValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value < b.Value}, nil
		},
	}

	Builtins["le_String"] = &BuiltinFunc{
		Name:    "le_String",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *StringValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value <= b.Value}, nil
		},
	}

	Builtins["gt_String"] = &BuiltinFunc{
		Name:    "gt_String",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *StringValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value > b.Value}, nil
		},
	}

	Builtins["ge_String"] = &BuiltinFunc{
		Name:    "ge_String",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a, b *StringValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value >= b.Value}, nil
		},
	}
}

// registerStringPrimitiveBuiltins registers low-level string operations
// These are the critical primitives that can't be efficiently implemented in AILANG
func registerStringPrimitiveBuiltins() {
	// _str_len: UTF-8 aware string length (returns number of runes, not bytes)
	Builtins["_str_len"] = &BuiltinFunc{
		Name:    "_str_len",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(s *StringValue) (*IntValue, error) {
			count := utf8.RuneCountInString(s.Value)
			return &IntValue{Value: count}, nil
		},
	}

	// _str_slice: substring with UTF-8 rune indices (not byte indices)
	// Indices are inclusive start, exclusive end
	Builtins["_str_slice"] = &BuiltinFunc{
		Name:    "_str_slice",
		NumArgs: 3,
		IsPure:  true,
		Impl: func(s *StringValue, start *IntValue, end *IntValue) (*StringValue, error) {
			runes := []rune(s.Value)
			length := len(runes)

			// Clamp indices to valid range
			st := start.Value
			if st < 0 {
				st = 0
			}
			if st > length {
				st = length
			}

			en := end.Value
			if en < st {
				en = st
			}
			if en > length {
				en = length
			}

			return &StringValue{Value: string(runes[st:en])}, nil
		},
	}

	// _str_compare: lexicographic comparison (-1, 0, 1)
	Builtins["_str_compare"] = &BuiltinFunc{
		Name:    "_str_compare",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a *StringValue, b *StringValue) (*IntValue, error) {
			if a.Value < b.Value {
				return &IntValue{Value: -1}, nil
			} else if a.Value > b.Value {
				return &IntValue{Value: 1}, nil
			} else {
				return &IntValue{Value: 0}, nil
			}
		},
	}

	// _str_eq: check if two strings are equal (returns bool)
	Builtins["_str_eq"] = &BuiltinFunc{
		Name:    "_str_eq",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(a *StringValue, b *StringValue) (*BoolValue, error) {
			return &BoolValue{Value: a.Value == b.Value}, nil
		},
	}

	// _str_find: find first occurrence of substring (returns -1 if not found)
	Builtins["_str_find"] = &BuiltinFunc{
		Name:    "_str_find",
		NumArgs: 2,
		IsPure:  true,
		Impl: func(s *StringValue, sub *StringValue) (*IntValue, error) {
			// Find byte index first
			byteIdx := strings.Index(s.Value, sub.Value)
			if byteIdx == -1 {
				return &IntValue{Value: -1}, nil
			}
			// Convert byte index to rune index
			runeIdx := utf8.RuneCountInString(s.Value[:byteIdx])
			return &IntValue{Value: runeIdx}, nil
		},
	}

	// _str_upper: Unicode-aware uppercase conversion
	Builtins["_str_upper"] = &BuiltinFunc{
		Name:    "_str_upper",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(s *StringValue) (*StringValue, error) {
			return &StringValue{Value: strings.ToUpper(s.Value)}, nil
		},
	}

	// _str_lower: Unicode-aware lowercase conversion
	Builtins["_str_lower"] = &BuiltinFunc{
		Name:    "_str_lower",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(s *StringValue) (*StringValue, error) {
			return &StringValue{Value: strings.ToLower(s.Value)}, nil
		},
	}

	// _str_trim: Unicode-aware whitespace trimming
	Builtins["_str_trim"] = &BuiltinFunc{
		Name:    "_str_trim",
		NumArgs: 1,
		IsPure:  true,
		Impl: func(s *StringValue) (*StringValue, error) {
			return &StringValue{Value: strings.TrimSpace(s.Value)}, nil
		},
	}
}
