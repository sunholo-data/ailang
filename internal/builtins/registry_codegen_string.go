package builtins

// ============================================================================
// std/string builtins — Go codegen specs
// ============================================================================

func registerStringCodegenSpecs() {
	setSpec("_str_trim", &GoCodegenSpec{
		Inline: `strings.TrimSpace({{arg0}}.(string))`,
		Helper: &GoHelperSpec{
			FuncName:  "Trim",
			Signature: "func Trim(s interface{}) interface{}",
			Body:      `return strings.TrimSpace(s.(string))`,
		},
		Imports:    []string{"strings"},
		StdlibName: "trim",
	})
	setSpec("_str_upper", &GoCodegenSpec{
		Inline:     `strings.ToUpper({{arg0}}.(string))`,
		Imports:    []string{"strings"},
		StdlibName: "toUpper",
	})
	setSpec("_str_lower", &GoCodegenSpec{
		Inline:     `strings.ToLower({{arg0}}.(string))`,
		Imports:    []string{"strings"},
		StdlibName: "toLower",
	})
	setSpec("_str_len", &GoCodegenSpec{
		Inline:     `int64(utf8.RuneCountInString({{arg0}}.(string)))`,
		Imports:    []string{"unicode/utf8"},
		StdlibName: "length",
	})
	setSpec("_str_compare", &GoCodegenSpec{
		Inline:     `int64(strings.Compare({{arg0}}.(string), {{arg1}}.(string)))`,
		Imports:    []string{"strings"},
		StdlibName: "compare",
	})
	setSpec("_str_find", &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "FindRune",
			Signature: "func FindRune(s interface{}, sub interface{}) interface{}",
			Body: `str := s.(string)
	byteIdx := strings.Index(str, sub.(string))
	if byteIdx == -1 { return int64(-1) }
	return int64(utf8.RuneCountInString(str[:byteIdx]))`,
		},
		Imports:    []string{"strings", "unicode/utf8"},
		StdlibName: "find",
	})
	setSpec("_str_eq", &GoCodegenSpec{
		Inline: `{{arg0}}.(string) == {{arg1}}.(string)`,
	})
	setSpec("_str_slice", &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Substring",
			Signature: "func Substring(s interface{}, start interface{}, end interface{}) interface{}",
			Body: `runes := []rune(s.(string))
	length := len(runes)
	st := int(toInt64(start))
	en := int(toInt64(end))
	if st < 0 { st = 0 }
	if en > length { en = length }
	if st > en { return "" }
	return string(runes[st:en])`,
		},
		StdlibName: "substring",
	})

	// String builtins not in registry.go but referenced by stdlib
	registerIfMissing("_str_split", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Split",
			Signature: "func Split(s interface{}, delimiter interface{}) interface{}",
			Body: `parts := strings.Split(s.(string), delimiter.(string))
	result := make([]interface{}, len(parts))
	for i, p := range parts {
		result[i] = p
	}
	return result`,
		},
		Imports:    []string{"strings"},
		StdlibName: "split",
	})
	registerIfMissing("_str_startsWith", 2, true, &GoCodegenSpec{
		Inline:     `strings.HasPrefix({{arg0}}.(string), {{arg1}}.(string))`,
		Imports:    []string{"strings"},
		StdlibName: "startsWith",
	})
	registerIfMissing("_str_endsWith", 2, true, &GoCodegenSpec{
		Inline:     `strings.HasSuffix({{arg0}}.(string), {{arg1}}.(string))`,
		Imports:    []string{"strings"},
		StdlibName: "endsWith",
	})
	registerIfMissing("_str_chars", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Chars",
			Signature: "func Chars(s interface{}) interface{}",
			Body: `str := s.(string)
	result := make([]interface{}, 0, len(str))
	for _, r := range str {
		result = append(result, string(r))
	}
	return result`,
		},
		StdlibName: "chars",
	})
	registerIfMissing("_str_words", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Words",
			Signature: "func Words(s interface{}) interface{}",
			Body: `parts := strings.Fields(s.(string))
	result := make([]interface{}, len(parts))
	for i, p := range parts {
		result[i] = p
	}
	return result`,
		},
		Imports:    []string{"strings"},
		StdlibName: "words",
	})
	registerIfMissing("_str_join", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Join",
			Signature: "func Join(delimiter interface{}, xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	strs := make([]string, len(list))
	for i, v := range list {
		strs[i] = v.(string)
	}
	return strings.Join(strs, delimiter.(string))`,
		},
		Imports:    []string{"strings"},
		StdlibName: "join",
	})
	registerIfMissing("_str_splitAny", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "SplitAny",
			Signature: "func SplitAny(s interface{}, delimiters interface{}) interface{}",
			Body: `str := s.(string)
	delims := toSlice(delimiters)
	delimSet := make(map[rune]bool)
	for _, d := range delims {
		for _, r := range d.(string) {
			delimSet[r] = true
		}
	}
	parts := strings.FieldsFunc(str, func(r rune) bool { return delimSet[r] })
	result := make([]interface{}, len(parts))
	for i, p := range parts {
		result[i] = p
	}
	return result`,
		},
		Imports:    []string{"strings"},
		StdlibName: "splitAny",
	})
	registerIfMissing("_str_contains", 2, true, &GoCodegenSpec{
		Inline:     `strings.Contains({{arg0}}.(string), {{arg1}}.(string))`,
		Imports:    []string{"strings"},
		StdlibName: "contains",
	})
	registerIfMissing("_str_replace", 3, true, &GoCodegenSpec{
		Inline:     `strings.ReplaceAll({{arg0}}.(string), {{arg1}}.(string), {{arg2}}.(string))`,
		Imports:    []string{"strings"},
		StdlibName: "replace",
	})
	registerIfMissing("_str_repeat", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Repeat",
			Signature: "func Repeat(s interface{}, n interface{}) interface{}",
			Body:      `return strings.Repeat(s.(string), int(toInt64(n)))`,
		},
		Imports:    []string{"strings"},
		StdlibName: "repeat",
	})
	// charAt indexes by RUNE, matching both interpreter tiers. The former body
	// indexed by BYTE (`str[i]`, bounds-checked against `len(str)`), so a
	// compiled program silently disagreed with the same source run under the
	// interpreter on any non-ASCII string -- charAt("h\u00e9llo", 1) returned
	// "\u00c3" compiled vs "\u00e9" interpreted -- and returned a character past
	// the end where the interpreter raises an out-of-bounds error (ailang#688).
	registerIfMissing("_str_charAt", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "CharAt",
			Signature: "func CharAt(s interface{}, idx interface{}) interface{}",
			// Uses only `for range` (which iterates runes natively) and fmt.
			// A Helper body CANNOT introduce a new import: runtime.go's import
			// block is a closed allowlist in codegen.go (fmt, reflect, and
			// conditionally sort/strconv/strings/math), and GoCodegenSpec.Imports
			// is explicitly skipped for Helper specs in
			// codegen_registry.go. A body that reaches for
			// anything else emits Go that does not compile.
			Body: `str := s.(string)
	i := int(toInt64(idx))
	n := 0
	if i >= 0 {
		for _, r := range str {
			if n == i { return string(r) }
			n++
		}
	} else {
		for range str { n++ }
	}
	panic(fmt.Sprintf("charAt: index %d out of bounds for string of length %d", i, n))`,
		},
		StdlibName: "charAt",
	})
	// charCode had NO codegen spec at all, so any compiled program using it
	// failed with "undefined: CharCode" (ailang#688). Same import constraint as
	// CharAt above: `for range` plus fmt only.
	registerIfMissing("_str_charCode", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "CharCode",
			Signature: "func CharCode(c interface{}) interface{}",
			Body: `str := c.(string)
	n := 0
	var first rune
	for _, r := range str {
		if n == 0 { first = r }
		n++
	}
	if n != 1 {
		panic(fmt.Sprintf("charCode: expected single-character string, got %d characters", n))
	}
	return int64(first)`,
		},
		StdlibName: "charCode",
	})
	registerIfMissing("_str_foldChars", 3, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "FoldChars",
			Signature: "func FoldChars(f interface{}, acc interface{}, s interface{}) interface{}",
			Body: `str := s.(string)
	result := acc
	for _, r := range str {
		result = CallFunc(f, result, string(r))
	}
	return result`,
		},
		StdlibName: "foldChars",
	})
	registerIfMissing("_stringToInt", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "StringToInt",
			Signature: "func StringToInt(s interface{}) interface{}",
			Body: `str := strings.TrimSpace(s.(string))
	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil { return NewOptionNone() }
	return NewOptionSome(n)`,
		},
		Imports:    []string{"strings", "strconv"},
		StdlibName: "stringToInt",
	})
	registerIfMissing("_stringToFloat", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "StringToFloat",
			Signature: "func StringToFloat(s interface{}) interface{}",
			Body: `str := strings.TrimSpace(s.(string))
	// Reject underscores: Go's ParseFloat silently accepts them as digit separators
	if strings.Contains(str, "_") { return NewOptionNone() }
	f, err := strconv.ParseFloat(str, 64)
	if err != nil { return NewOptionNone() }
	return NewOptionSome(f)`,
		},
		Imports:    []string{"strings", "strconv"},
		StdlibName: "stringToFloat",
	})
	registerIfMissing("_string_intToStr", 1, true, &GoCodegenSpec{
		Inline:     `fmt.Sprintf("%d", toInt64({{arg0}}))`,
		Imports:    []string{"fmt"},
		StdlibName: "intToStr",
	})
	registerIfMissing("_string_floatToStr", 1, true, &GoCodegenSpec{
		Inline:     `fmt.Sprintf("%g", {{arg0}}.(float64))`,
		Imports:    []string{"fmt"},
		StdlibName: "floatToStr",
	})
}
