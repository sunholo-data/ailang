package builtins

// ============================================================================
// std/json builtins — Go codegen specs
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
