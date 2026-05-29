package builtins

// ============================================================================
// XML + other effect stubs — Go codegen specs
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
		{"_net_httpRequestBytes", "httpRequestBytes", "HttpRequestBytes", 1},
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
