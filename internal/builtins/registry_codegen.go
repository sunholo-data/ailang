package builtins

// registerCodegenSpecs adds Go codegen specifications to all builtins.
// M-CODEGEN-SUSTAINABILITY: This is the single source of truth for how
// each builtin is emitted in generated Go code. Replaces mapPureMathBuiltin,
// mapPureListBuiltin, mapStdlibBuiltin, and codegen_runtime_stdlib.go.
//
// Two spec types:
//   - Inline: single Go expression with {{arg0}}, {{arg1}} placeholders
//   - Helper: runtime function emitted in runtime.go (for complex builtins)
func registerCodegenSpecs() {
	registerStringCodegenSpecs()
	registerMathCodegenSpecs()
	registerListCodegenSpecs()
	registerConversionCodegenSpecs()
	registerIOCodegenSpecs()
	registerJSONCodegenSpecs()
	registerEffectCodegenSpecs()
}

// ============================================================================
// std/string builtins
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
	registerIfMissing("_str_charAt", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "CharAt",
			Signature: "func CharAt(s interface{}, idx interface{}) interface{}",
			Body: `str := s.(string)
	i := int(toInt64(idx))
	if i < 0 || i >= len(str) { return "" }
	return string(str[i])`,
		},
		StdlibName: "charAt",
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

// ============================================================================
// std/math builtins
// ============================================================================

func registerMathCodegenSpecs() {
	mathFuncs := map[string]struct{ goExpr, stdlibName string }{
		"_math_sin":       {"math.Sin({{arg0}}.(float64))", "sin"},
		"_math_cos":       {"math.Cos({{arg0}}.(float64))", "cos"},
		"_math_tan":       {"math.Tan({{arg0}}.(float64))", "tan"},
		"_math_asin":      {"math.Asin({{arg0}}.(float64))", "asin"},
		"_math_acos":      {"math.Acos({{arg0}}.(float64))", "acos"},
		"_math_atan":      {"math.Atan({{arg0}}.(float64))", "atan"},
		"_math_atan2":     {"math.Atan2({{arg0}}.(float64), {{arg1}}.(float64))", "atan2"},
		"_math_exp":       {"math.Exp({{arg0}}.(float64))", "exp"},
		"_math_log":       {"math.Log({{arg0}}.(float64))", "log"},
		"_math_log10":     {"math.Log10({{arg0}}.(float64))", "log10"},
		"_math_pow":       {"math.Pow({{arg0}}.(float64), {{arg1}}.(float64))", "pow"},
		"_math_sqrt":      {"math.Sqrt({{arg0}}.(float64))", "sqrt"},
		"_math_ceil":      {"math.Ceil({{arg0}}.(float64))", "ceil"},
		"_math_floor":     {"math.Floor({{arg0}}.(float64))", "floor"},
		"_math_round":     {"math.Round({{arg0}}.(float64))", "round"},
		"_math_abs_Float": {"math.Abs({{arg0}}.(float64))", "absFloat"},
		"_math_abs_Int":   {"int64(math.Abs(float64(toInt64({{arg0}}))))", "absInt"},
	}
	for name, spec := range mathFuncs {
		numArgs := 1
		if name == "_math_atan2" || name == "_math_pow" {
			numArgs = 2
		}
		registerIfMissing(name, numArgs, true, &GoCodegenSpec{
			Inline:     spec.goExpr,
			Imports:    []string{"math"},
			StdlibName: spec.stdlibName,
		})
	}
	// Math constants
	registerIfMissing("_math_PI", 0, true, &GoCodegenSpec{
		Inline:     `math.Pi`,
		Imports:    []string{"math"},
		StdlibName: "PI",
	})
	registerIfMissing("_math_E", 0, true, &GoCodegenSpec{
		Inline:     `math.E`,
		Imports:    []string{"math"},
		StdlibName: "E",
	})
	// Conversion builtins used by math
	registerIfMissing("_int_to_float", 1, true, &GoCodegenSpec{
		Inline:     `float64(toInt64({{arg0}}))`,
		StdlibName: "intToFloat",
	})
	registerIfMissing("_float_to_int", 1, true, &GoCodegenSpec{
		Inline:     `int64({{arg0}}.(float64))`,
		StdlibName: "floatToInt",
	})
}

// ============================================================================
// std/list builtins
// ============================================================================

func registerListCodegenSpecs() {
	// ConcatList is emitted by codegen_runtime_collections.go (infrastructure).
	// Concat is a thin alias for VarGlobal resolution; ConcatList handles the actual work.
	setSpec("concat_List", &GoCodegenSpec{
		Inline: `ConcatList({{arg0}}, {{arg1}})`,
		Helper: &GoHelperSpec{
			FuncName:  "Concat",
			Signature: "func Concat(a, b interface{}) interface{}",
			Body:      `return ConcatList(a, b)`,
		},
		StdlibName: "concat",
	})

	registerIfMissing("_list_map", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Map",
			Signature: "func Map(f, xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	result := make([]interface{}, len(list))
	for i, x := range list {
		result[i] = CallFunc(f, x)
	}
	return result`,
		},
		StdlibName: "map",
	})
	registerIfMissing("_list_filter", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Filter",
			Signature: "func Filter(p, xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	var result []interface{}
	for _, x := range list {
		if CallFunc(p, x).(bool) {
			result = append(result, x)
		}
	}
	if result == nil { result = []interface{}{} }
	return result`,
		},
		StdlibName: "filter",
	})
	registerIfMissing("_list_foldl", 3, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Foldl",
			Signature: "func Foldl(f, acc, xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	result := acc
	for _, x := range list {
		result = CallFunc(f, result, x)
	}
	return result`,
		},
		StdlibName: "foldl",
	})
	registerIfMissing("_list_foldr", 3, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Foldr",
			Signature: "func Foldr(f, acc, xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	result := acc
	for i := len(list) - 1; i >= 0; i-- {
		result = CallFunc(f, list[i], result)
	}
	return result`,
		},
		StdlibName: "foldr",
	})
	registerIfMissing("_list_dedup", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Dedup",
			Signature: "func Dedup(xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	seen := make(map[interface{}]bool)
	var result []interface{}
	for _, x := range list {
		if !seen[x] {
			seen[x] = true
			result = append(result, x)
		}
	}
	if result == nil { result = []interface{}{} }
	return result`,
		},
		StdlibName: "dedup",
	})

	// M-CODEGEN-LETBIND-FIX: Set operations (intersect, union) used by DocParse
	registerIfMissing("_list_intersect", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Intersect",
			Signature: "func Intersect(xs, ys interface{}) interface{}",
			Body: `listA := toSlice(xs)
	listB := toSlice(ys)
	set := make(map[interface{}]bool)
	for _, x := range listB { set[x] = true }
	var result []interface{}
	seen := make(map[interface{}]bool)
	for _, x := range listA {
		if set[x] && !seen[x] {
			seen[x] = true
			result = append(result, x)
		}
	}
	if result == nil { result = []interface{}{} }
	return result`,
		},
		StdlibName: "intersect",
	})
	registerIfMissing("_list_union", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Union",
			Signature: "func Union(xs, ys interface{}) interface{}",
			Body: `listA := toSlice(xs)
	listB := toSlice(ys)
	seen := make(map[interface{}]bool)
	var result []interface{}
	for _, x := range listA {
		if !seen[x] {
			seen[x] = true
			result = append(result, x)
		}
	}
	for _, x := range listB {
		if !seen[x] {
			seen[x] = true
			result = append(result, x)
		}
	}
	if result == nil { result = []interface{}{} }
	return result`,
		},
		StdlibName: "union",
	})

	// Additional list helpers used by stdlib but not as builtins
	for _, spec := range []struct {
		name, stdlib string
		numArgs      int
		helper       *GoHelperSpec
	}{
		{"_list_reverse", "reverse", 1, &GoHelperSpec{
			FuncName: "Reverse", Signature: "func Reverse(xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	n := len(list)
	result := make([]interface{}, n)
	for i, v := range list { result[n-1-i] = v }
	return result`,
		}},
		{"_list_take", "take", 2, &GoHelperSpec{
			FuncName: "Take", Signature: "func Take(n, xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	count := int(toInt64(n))
	if count > len(list) { count = len(list) }
	if count < 0 { count = 0 }
	result := make([]interface{}, count)
	copy(result, list[:count])
	return result`,
		}},
		{"_list_drop", "drop", 2, &GoHelperSpec{
			FuncName: "Drop", Signature: "func Drop(n, xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	count := int(toInt64(n))
	if count > len(list) { count = len(list) }
	if count < 0 { count = 0 }
	result := make([]interface{}, len(list)-count)
	copy(result, list[count:])
	return result`,
		}},
		{"_list_any", "any", 2, &GoHelperSpec{
			FuncName: "Any", Signature: "func Any(p, xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	for _, x := range list {
		if CallFunc(p, x).(bool) { return true }
	}
	return false`,
		}},
		{"_list_sortBy", "sortBy", 2, &GoHelperSpec{
			FuncName: "SortBy", Signature: "func SortBy(cmp, xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	result := make([]interface{}, len(list))
	copy(result, list)
	sort.Slice(result, func(i, j int) bool {
		return toInt64(CallFunc(cmp, result[i], result[j])) < 0
	})
	return result`,
		}},
		{"_list_flatMap", "flatMap", 2, &GoHelperSpec{
			FuncName: "FlatMap", Signature: "func FlatMap(f, xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	var result []interface{}
	for _, x := range list {
		inner := toSlice(CallFunc(f, x))
		result = append(result, inner...)
	}
	if result == nil { result = []interface{}{} }
	return result`,
		}},
		{"_list_zip", "zip", 2, &GoHelperSpec{
			FuncName: "Zip", Signature: "func Zip(xs, ys interface{}) interface{}",
			Body: `listX := toSlice(xs)
	listY := toSlice(ys)
	n := len(listX)
	if len(listY) < n { n = len(listY) }
	result := make([]interface{}, n)
	for i := 0; i < n; i++ {
		result[i] = []interface{}{listX[i], listY[i]}
	}
	return result`,
		}},
	} {
		registerIfMissing(spec.name, spec.numArgs, true, &GoCodegenSpec{
			Helper:     spec.helper,
			StdlibName: spec.stdlib,
		})
	}

	// Additional stdlib names that map to existing helpers
	registerIfMissing("_list_nth", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "Nth", Signature: "func Nth(xs, idx interface{}) interface{}",
			Body: `list := toSlice(xs)
	i := int(toInt64(idx))
	if i < 0 || i >= len(list) { return NewOptionNone() }
	return NewOptionSome(list[i])`,
		},
		StdlibName: "nth",
	})
	registerIfMissing("_list_last", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "Last", Signature: "func Last(xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	if len(list) == 0 { return NewOptionNone() }
	return NewOptionSome(list[len(list)-1])`,
		},
		StdlibName: "last",
	})
	registerIfMissing("_list_findIndex", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "FindIndex", Signature: "func FindIndex(p, xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	for i, x := range list {
		if CallFunc(p, x).(bool) {
			return NewOptionSome(int64(i))
		}
	}
	return NewOptionNone()`,
		},
		StdlibName: "findIndex",
	})
	registerIfMissing("_list_mapE", 2, false, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "MapE", Signature: "func MapE(f, xs interface{}) interface{}",
			Body: `return Map(f, xs)`,
		},
		StdlibName: "mapE",
	})
	registerIfMissing("_list_forEachE", 2, false, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "ForEachE", Signature: "func ForEachE(f, xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	for _, x := range list {
		CallFunc(f, x)
	}
	return struct{}{}`,
		},
		StdlibName: "forEachE",
	})
}

// ============================================================================
// Conversion builtins
// ============================================================================

func registerConversionCodegenSpecs() {
	setSpec("intToFloat", &GoCodegenSpec{
		Inline:     `float64(toInt64({{arg0}}))`,
		StdlibName: "intToFloat",
	})
	setSpec("floatToInt", &GoCodegenSpec{
		Inline:     `int64({{arg0}}.(float64))`,
		StdlibName: "floatToInt",
	})
}

// ============================================================================
// std/io builtins
// ============================================================================

func registerIOCodegenSpecs() {
	setSpec("_io_println", &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "Println", Signature: "func Println(v interface{}) interface{}",
			Body: `fmt.Println(Show(v))
	return struct{}{}`,
		},
		Imports:    []string{"fmt"},
		StdlibName: "println",
	})
	setSpec("_io_print", &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "Print", Signature: "func Print(v interface{}) interface{}",
			Body: `fmt.Print(Show(v))
	return struct{}{}`,
		},
		Imports:    []string{"fmt"},
		StdlibName: "print",
	})

	// Effect stubs — these panic with clear messages
	for _, spec := range []struct {
		name, stdlib, funcName, msg string
		numArgs                     int
	}{
		{"_fs_readFile", "readFile", "ReadFile", "FS", 1},
		{"_fs_writeFile", "writeFile", "WriteFile", "FS", 2},
		{"_fs_exists", "fileExists", "FileExists", "FS", 1},
		{"_fs_readFileBytes", "readFileBytes", "ReadFileBytes", "FS", 1},
		{"_fs_mkdir", "mkdir", "Mkdir", "FS", 1},
		{"_fs_mkdirAll", "mkdirAll", "MkdirAll", "FS", 1},
		{"_fs_isDir", "isDir", "IsDir", "FS", 1},
		{"_fs_isFile", "isFile", "IsFile", "FS", 1},
		{"_fs_removeFile", "removeFile", "RemoveFile", "FS", 1},
		{"_zip_readEntry", "readEntry", "ReadEntry", "zip", 0},
		{"_zip_readEntryBytes", "readEntryBytes", "ReadEntryBytes", "zip", 0},
		{"_zip_listEntries", "listEntries", "ListEntries", "zip", 0},
		{"_zip_createArchive", "createArchive", "CreateArchive", "zip", 0},
		{"_env_getEnvOr", "getEnvOr", "GetEnvOr", "Env", 2},
		{"_env_getArgs", "getArgs", "GetArgs", "Env", 0},
		{"_env_getEnv", "getEnv", "GetEnv", "Env", 1},
		{"_ai_call", "call", "Call", "AI", 0},
		{"_ai_callJson", "callJson", "CallJson", "AI", 0},
		{"_ai_callJsonSimple", "callJsonSimple", "CallJsonSimple", "AI", 0},
	} {
		funcName := spec.funcName
		msg := spec.msg
		registerIfMissing(spec.name, spec.numArgs, false, &GoCodegenSpec{
			Helper: &GoHelperSpec{
				FuncName:  funcName,
				Signature: "func " + funcName + "(args ...interface{}) interface{}",
				Body:      `panic("` + funcName + `: ` + msg + ` effect not available in compiled mode - provide a handler")`,
			},
			StdlibName: spec.stdlib,
		})
	}
}

// ============================================================================
// std/json builtins
// ============================================================================

func registerJSONCodegenSpecs() {
	setSpec("_json_decode", &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "Decode", Signature: "func Decode(s interface{}) interface{}",
			Body: `return NewResultErr("JSON decode not yet available in compiled Go mode")`,
		},
		StdlibName:  "decode",
		RequiresADT: "Json",
	})
	setSpec("_json_encode", &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName: "Encode", Signature: "func Encode(obj interface{}) interface{}",
			Body: `return "{}"`,
		},
		StdlibName:  "encode",
		RequiresADT: "Json",
	})

	// JSON constructor and accessor helpers
	for _, spec := range []struct {
		name, stdlib string
		numArgs      int
		helper       *GoHelperSpec
		requiresADT  string
	}{
		{"_json_js", "js", 1, &GoHelperSpec{
			FuncName: "Js", Signature: "func Js(s interface{}) interface{}",
			Body: `return NewJsonJString(s.(string))`,
		}, "Json"},
		{"_json_jn", "jn", 0, &GoHelperSpec{
			FuncName: "Jn", Signature: "func Jn() interface{}",
			Body: `return NewJsonJNull()`,
		}, "Json"},
		{"_json_jb", "jb", 1, &GoHelperSpec{
			FuncName: "Jb", Signature: "func Jb(b interface{}) interface{}",
			Body: `return NewJsonJBool(b.(bool))`,
		}, "Json"},
		{"_json_jnum", "jnum", 1, &GoHelperSpec{
			FuncName: "Jnum", Signature: "func Jnum(x interface{}) interface{}",
			Body: `return NewJsonJNumber(x.(float64))`,
		}, "Json"},
		{"_json_ja", "ja", 1, &GoHelperSpec{
			FuncName: "Ja", Signature: "func Ja(xs interface{}) interface{}",
			Body: `return NewJsonJArray(ConvertToJsonSlice(xs))`,
		}, "Json"},
		{"_json_jo", "jo", 1, &GoHelperSpec{
			FuncName: "Jo", Signature: "func Jo(kvs interface{}) interface{}",
			Body: `return NewJsonJObject(ConvertToRecordSlice(kvs))`,
		}, "Json"},
		{"_json_kv", "kv", 2, &GoHelperSpec{
			FuncName: "Kv", Signature: `func Kv(k, v interface{}) interface{}`,
			Body: `return map[string]interface{}{"key": k, "value": v}`,
		}, "Json"},
		{"_json_get", "get", 2, &GoHelperSpec{
			FuncName: "JsonGet", Signature: "func JsonGet(obj, key interface{}) interface{}",
			Body: `json := obj.(*Json)
	if json.Kind != JsonKindJObject { return NewOptionNone() }
	k := key.(string)
	kvs := toSlice(json.JObject.Value0)
	for _, kv := range kvs {
		rec := kv.(map[string]interface{})
		if rec["Key"] == k || rec["key"] == k {
			val := rec["Value"]
			if val == nil { val = rec["value"] }
			return NewOptionSome(val)
		}
	}
	return NewOptionNone()`,
		}, "Json"},
		{"_json_has", "has", 2, &GoHelperSpec{
			FuncName: "JsonHas", Signature: "func JsonHas(obj, key interface{}) interface{}",
			Body: `return IsSome(JsonGet(obj, key))`,
		}, "Json"},
		{"_json_getString", "getString", 2, &GoHelperSpec{
			FuncName: "GetString", Signature: "func GetString(obj, key interface{}) interface{}",
			Body: `opt := JsonGet(obj, key)
	if IsNone(opt).(bool) { return NewOptionNone() }
	return AsString(OptionGetOrElse(opt, nil))`,
		}, "Json"},
		{"_json_getInt", "getInt", 2, &GoHelperSpec{
			FuncName: "GetInt", Signature: "func GetInt(obj, key interface{}) interface{}",
			Body: `opt := JsonGet(obj, key)
	if IsNone(opt).(bool) { return NewOptionNone() }
	val := OptionGetOrElse(opt, nil)
	json := val.(*Json)
	if json.Kind == JsonKindJNumber {
		return NewOptionSome(int64(json.JNumber.Value0))
	}
	return NewOptionNone()`,
		}, "Json"},
		{"_json_getBool", "getBool", 2, &GoHelperSpec{
			FuncName: "GetBool", Signature: "func GetBool(obj, key interface{}) interface{}",
			Body: `opt := JsonGet(obj, key)
	if IsNone(opt).(bool) { return NewOptionNone() }
	val := OptionGetOrElse(opt, nil)
	json := val.(*Json)
	if json.Kind == JsonKindJBool {
		return NewOptionSome(json.JBool.Value0)
	}
	return NewOptionNone()`,
		}, "Json"},
		{"_json_getArray", "getArray", 2, &GoHelperSpec{
			FuncName: "GetArray", Signature: "func GetArray(obj, key interface{}) interface{}",
			Body: `opt := JsonGet(obj, key)
	if IsNone(opt).(bool) { return NewOptionNone() }
	return AsArray(OptionGetOrElse(opt, nil))`,
		}, "Json"},
		{"_json_asString", "asString", 1, &GoHelperSpec{
			FuncName: "AsString", Signature: "func AsString(j interface{}) interface{}",
			Body: `json := j.(*Json)
	if json.Kind == JsonKindJString { return NewOptionSome(json.JString.Value0) }
	return NewOptionNone()`,
		}, "Json"},
		{"_json_asNumber", "asNumber", 1, &GoHelperSpec{
			FuncName: "AsNumber", Signature: "func AsNumber(j interface{}) interface{}",
			Body: `json := j.(*Json)
	if json.Kind == JsonKindJNumber { return NewOptionSome(json.JNumber.Value0) }
	return NewOptionNone()`,
		}, "Json"},
		{"_json_asBool", "asBool", 1, &GoHelperSpec{
			FuncName: "AsBool", Signature: "func AsBool(j interface{}) interface{}",
			Body: `json := j.(*Json)
	if json.Kind == JsonKindJBool { return NewOptionSome(json.JBool.Value0) }
	return NewOptionNone()`,
		}, "Json"},
		{"_json_asArray", "asArray", 1, &GoHelperSpec{
			FuncName: "AsArray", Signature: "func AsArray(j interface{}) interface{}",
			Body: `json := j.(*Json)
	if json.Kind == JsonKindJArray { return NewOptionSome(json.JArray.Value0) }
	return NewOptionNone()`,
		}, "Json"},
		{"_json_asObject", "asObject", 1, &GoHelperSpec{
			FuncName: "AsObject", Signature: "func AsObject(j interface{}) interface{}",
			Body: `json := j.(*Json)
	if json.Kind == JsonKindJObject { return NewOptionSome(json.JObject.Value0) }
	return NewOptionNone()`,
		}, "Json"},
		{"_json_keys", "keys", 1, &GoHelperSpec{
			FuncName: "JsonKeys", Signature: "func JsonKeys(obj interface{}) interface{}",
			Body: `json := obj.(*Json)
	if json.Kind != JsonKindJObject { return []interface{}{} }
	kvs := toSlice(json.JObject.Value0)
	result := make([]interface{}, len(kvs))
	for i, kv := range kvs {
		rec := kv.(map[string]interface{})
		k := rec["Key"]
		if k == nil { k = rec["key"] }
		result[i] = k
	}
	return result`,
		}, "Json"},
		{"_json_getOr", "getOr", 3, &GoHelperSpec{
			FuncName: "JsonGetOr", Signature: "func JsonGetOr(obj, key, defaultVal interface{}) interface{}",
			Body: `return OptionGetOrElse(JsonGet(obj, key), defaultVal)`,
		}, "Json"},
		{"_json_repair", "repair", 1, &GoHelperSpec{
			FuncName: "JsonRepair", Signature: "func JsonRepair(s interface{}) interface{}",
			Body: `return NewResultOk(s)`,
		}, "Json"},
	} {
		registerIfMissing(spec.name, spec.numArgs, true, &GoCodegenSpec{
			Helper:      spec.helper,
			StdlibName:  spec.stdlib,
			RequiresADT: spec.requiresADT,
		})
	}

	// Option/Result helpers
	for _, spec := range []struct {
		name, stdlib string
		numArgs      int
		helper       *GoHelperSpec
		requiresADT  string
	}{
		{"_option_getOrElse", "getOrElse", 2, &GoHelperSpec{
			FuncName: "OptionGetOrElse", Signature: "func OptionGetOrElse(opt, defaultVal interface{}) interface{}",
			Body: `o := opt.(*Option)
	if o.Kind == OptionKindSome { return o.Some.Value0 }
	return defaultVal`,
		}, "Option"},
		{"_option_isNone", "isNone", 1, &GoHelperSpec{
			FuncName: "IsNone", Signature: "func IsNone(opt interface{}) interface{}",
			Body: `return opt.(*Option).Kind == OptionKindNone`,
		}, "Option"},
		{"_option_isSome", "isSome", 1, &GoHelperSpec{
			FuncName: "IsSome", Signature: "func IsSome(opt interface{}) interface{}",
			Body: `return opt.(*Option).Kind == OptionKindSome`,
		}, "Option"},
		{"_result_isOk", "isOk", 1, &GoHelperSpec{
			FuncName: "IsOk", Signature: "func IsOk(r interface{}) interface{}",
			Body: `return r.(*Result).Kind == ResultKindOk`,
		}, "Result"},
		{"_result_isErr", "isErr", 1, &GoHelperSpec{
			FuncName: "IsErr", Signature: "func IsErr(r interface{}) interface{}",
			Body: `return r.(*Result).Kind == ResultKindErr`,
		}, "Result"},
	} {
		registerIfMissing(spec.name, spec.numArgs, true, &GoCodegenSpec{
			Helper:      spec.helper,
			StdlibName:  spec.stdlib,
			RequiresADT: spec.requiresADT,
		})
	}
}

// ============================================================================
// XML + other effect stubs
// ============================================================================

func registerEffectCodegenSpecs() {
	xmlFuncs := []struct {
		name, stdlib, funcName string
		numArgs                int
	}{
		{"_xml_parse", "parse", "XmlParse", 1},
		{"_xml_findAll", "findAll", "XmlFindAll", 2},
		{"_xml_findFirst", "findFirst", "XmlFindFirst", 2},
		{"_xml_getText", "getText", "XmlGetText", 1},
		{"_xml_getAttr", "getAttr", "GetAttr", 2},
		{"_xml_getChildren", "getChildren", "XmlGetChildren", 1},
		{"_xml_getTag", "getTag", "XmlGetTag", 1},
		{"_xml_findAllTexts", "findAllTexts", "FindAllTexts", 2},
		{"_xml_findAllAttrs", "findAllAttrs", "FindAllAttrs", 3},
		{"_xml_serialize", "serialize", "XmlSerialize", 1},
	}
	for _, spec := range xmlFuncs {
		funcName := spec.funcName
		registerIfMissing(spec.name, spec.numArgs, true, &GoCodegenSpec{
			Helper: &GoHelperSpec{
				FuncName:  funcName,
				Signature: "func " + funcName + "(args ...interface{}) interface{}",
				Body:      `panic("` + funcName + `: XML operations not yet available in compiled mode")`,
			},
			StdlibName: spec.stdlib,
		})
	}

	// XML streaming — panic stub for compiled mode
	registerIfMissing("_xml_parseElements", 3, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "ParseElements",
			Signature: "func ParseElements(args ...interface{}) interface{}",
			Body:      `panic("ParseElements: XML streaming not available in compiled mode - provide an XML handler")`,
		},
		StdlibName: "parseElements",
	})

	// JSON helpers for DocParse
	registerIfMissing("_json_filterStrings", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "FilterStrings",
			Signature: "func FilterStrings(xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	var result []interface{}
	for _, x := range list {
		if json, ok := x.(*Json); ok && json.Kind == JsonKindJString {
			result = append(result, json.JString.Value0)
		}
	}
	if result == nil { result = []interface{}{} }
	return result`,
		},
		StdlibName:  "filterStrings",
		RequiresADT: "Json",
	})
	registerIfMissing("_json_getObject", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "GetObject",
			Signature: "func GetObject(obj, key interface{}) interface{}",
			Body: `opt := JsonGet(obj, key)
	if IsNone(opt).(bool) { return NewOptionNone() }
	return AsObject(OptionGetOrElse(opt, nil))`,
		},
		StdlibName:  "getObject",
		RequiresADT: "Json",
	})

	// NotBool helper — registered as not_Bool to match Core IR VarGlobal name
	registerIfMissing("not_Bool", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "NotBool",
			Signature: "func NotBool(v interface{}) interface{}",
			Body:      `return !v.(bool)`,
		},
	})

	// Debug effect helpers
	registerIfMissing("_debug_check", 2, false, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Check",
			Signature: "func Check(label interface{}, value interface{}) interface{}",
			Body:      `return value`,
		},
		StdlibName: "check",
	})

	// Effectful list combinators — panic stubs for compiled mode
	effectfulListFuncs := []struct {
		name, stdlib, funcName string
		numArgs                int
	}{
		{"_list_filterE", "filterE", "FilterE", 2},
		{"_list_foldlE", "foldlE", "FoldlE", 3},
		{"_list_flatMapE", "flatMapE", "FlatMapE", 2},
	}
	for _, spec := range effectfulListFuncs {
		funcName := spec.funcName
		registerIfMissing(spec.name, spec.numArgs, false, &GoCodegenSpec{
			Helper: &GoHelperSpec{
				FuncName:  funcName,
				Signature: "func " + funcName + "(args ...interface{}) interface{}",
				Body:      `panic("` + funcName + `: effectful list operation not available in compiled mode - provide a handler")`,
			},
			StdlibName: spec.stdlib,
		})
	}

	// JSON helpers missing from registry
	jsonHelpers := []struct {
		name, stdlib, funcName string
		numArgs                int
	}{
		{"_json_getNumber", "getNumber", "GetNumber", 2},
		{"_json_allStrings", "allStrings", "AllStrings", 1},
		{"_json_allNumbers", "allNumbers", "AllNumbers", 1},
		{"_json_filterNumbers", "filterNumbers", "FilterNumbers", 1},
		{"_json_getStringArray", "getStringArray", "GetStringArray", 2},
		{"_json_getStringArrayOrEmpty", "getStringArrayOrEmpty", "GetStringArrayOrEmpty", 2},
		{"_json_getNumberArrayOrEmpty", "getNumberArrayOrEmpty", "GetNumberArrayOrEmpty", 2},
	}
	for _, spec := range jsonHelpers {
		funcName := spec.funcName
		registerIfMissing(spec.name, spec.numArgs, true, &GoCodegenSpec{
			Helper: &GoHelperSpec{
				FuncName:  funcName,
				Signature: "func " + funcName + "(args ...interface{}) interface{}",
				Body:      `panic("` + funcName + `: JSON helper not yet available in compiled mode")`,
			},
			StdlibName:  spec.stdlib,
			RequiresADT: "Json",
		})
	}

	// Conversion helpers
	registerIfMissing("_str_toString", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "ToString",
			Signature: "func ToString(v interface{}) interface{}",
			Body:      `return Show(v)`,
		},
		StdlibName: "toString",
	})
	registerIfMissing("_str_fromString", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "FromString",
			Signature: "func FromString(s interface{}) interface{}",
			Body:      `return s`,
		},
		StdlibName: "fromString",
	})

	// Math helpers
	mathHelpers := []struct {
		name, stdlib, funcName, body string
	}{
		{"_math_maximumInt", "maximumInt", "MaximumInt", `a := toInt64(args[0]); b := toInt64(args[1]); if a > b { return a }; return b`},
		{"_math_minimumInt", "minimumInt", "MinimumInt", `a := toInt64(args[0]); b := toInt64(args[1]); if a < b { return a }; return b`},
		{"_math_maximumFloat", "maximumFloat", "MaximumFloat", `a := args[0].(float64); b := args[1].(float64); if a > b { return a }; return b`},
		{"_math_minimumFloat", "minimumFloat", "MinimumFloat", `a := args[0].(float64); b := args[1].(float64); if a < b { return a }; return b`},
		{"_math_absInt", "absInt", "AbsInt", `v := toInt64(args[0]); if v < 0 { return -v }; return v`},
		{"_math_maximumString", "maximumString", "MaximumString", `a := args[0].(string); b := args[1].(string); if a > b { return a }; return b`},
		{"_math_minimumString", "minimumString", "MinimumString", `a := args[0].(string); b := args[1].(string); if a < b { return a }; return b`},
	}
	for _, spec := range mathHelpers {
		registerIfMissing(spec.name, 2, true, &GoCodegenSpec{
			Helper: &GoHelperSpec{
				FuncName:  spec.funcName,
				Signature: "func " + spec.funcName + "(args ...interface{}) interface{}",
				Body:      spec.body,
			},
			StdlibName: spec.stdlib,
		})
	}

	// Process/IO effect stubs
	ioEffectFuncs := []struct {
		name, stdlib, funcName string
		numArgs                int
	}{
		{"_process_spawn", "spawnProcess", "SpawnProcess", 1},
		{"_process_exec", "exec", "Exec", 1},
		{"_process_asyncExec", "asyncExecProcess", "AsyncExecProcess", 1},
		{"_process_writeStdin", "writeProcessStdin", "WriteProcessStdin", 2},
		{"_process_closeStdin", "closeProcessStdin", "CloseProcessStdin", 1},
		{"_process_asyncReadStdinLines", "asyncReadStdinLines", "AsyncReadStdinLines", 1},
		{"_io_listDir", "listDir", "ListDir", 1},
		{"_clock_now", "now", "Now", 0},
		{"_net_httpGet", "httpGet", "HttpGet", 1},
		{"_net_httpRequest", "httpRequest", "HttpRequest", 1},
	}
	for _, spec := range ioEffectFuncs {
		funcName := spec.funcName
		registerIfMissing(spec.name, spec.numArgs, false, &GoCodegenSpec{
			Helper: &GoHelperSpec{
				FuncName:  funcName,
				Signature: "func " + funcName + "(args ...interface{}) interface{}",
				Body:      `panic("` + funcName + `: effect not available in compiled mode - provide a handler")`,
			},
			StdlibName: spec.stdlib,
		})
	}

	// List utilities
	registerIfMissing("_list_head", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Head",
			Signature: "func Head(xs interface{}) interface{}",
			Body:      `return ListHead(xs)`,
		},
		StdlibName: "head",
	})
	registerIfMissing("_list_tail", 1, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Tail",
			Signature: "func Tail(xs interface{}) interface{}",
			Body:      `return ListTail(xs)`,
		},
		StdlibName: "tail",
	})
	registerIfMissing("_list_member", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Member",
			Signature: "func Member(x, xs interface{}) interface{}",
			Body: `list := toSlice(xs)
	for _, v := range list {
		if v == x { return true }
	}
	return false`,
		},
		StdlibName: "member",
	})
	registerIfMissing("_list_difference", 2, true, &GoCodegenSpec{
		Helper: &GoHelperSpec{
			FuncName:  "Difference",
			Signature: "func Difference(xs, ys interface{}) interface{}",
			Body: `listA := toSlice(xs)
	listB := toSlice(ys)
	set := make(map[interface{}]bool)
	for _, b := range listB { set[b] = true }
	var result []interface{}
	for _, a := range listA {
		if !set[a] { result = append(result, a) }
	}
	if result == nil { result = []interface{}{} }
	return result`,
		},
		StdlibName: "difference",
	})
}

// ============================================================================
// Helper functions
// ============================================================================

// setSpec sets the GoCodegenSpec on an existing registry entry.
func setSpec(name string, spec *GoCodegenSpec) {
	if meta, ok := Registry[name]; ok {
		meta.GoCodegen = spec
	}
}

// registerIfMissing registers a builtin with codegen spec if not already in registry.
// Used for builtins that exist in the interpreter but aren't in the lightweight registry.
func registerIfMissing(name string, numArgs int, isPure bool, spec *GoCodegenSpec) {
	if _, ok := Registry[name]; !ok {
		Registry[name] = &BuiltinMeta{Name: name, NumArgs: numArgs, IsPure: isPure}
	}
	Registry[name].GoCodegen = spec
}
