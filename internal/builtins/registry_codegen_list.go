package builtins

// ============================================================================
// std/list builtins — Go codegen specs
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
