package builtins

// BuiltinMeta holds metadata about a builtin function
// This is a lightweight struct that doesn't depend on eval types
type BuiltinMeta struct {
	Name    string
	NumArgs int
	IsPure  bool
}

// Registry holds all registered builtin function metadata
// This is a simple data structure with no dependencies on eval or runtime
var Registry = make(map[string]*BuiltinMeta)

func init() {
	registerArithmeticMeta()
	registerComparisonMeta()
	registerConversionMeta()
	registerStringMeta()
	registerBooleanMeta()
	registerStringPrimitiveMeta()
	registerIOMeta()
	registerJSONMeta()
	registerNetMeta()
}

// GetBuiltinNames returns all registered builtin names
func GetBuiltinNames() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	return names
}

// IsBuiltin checks if a name is a registered builtin
func IsBuiltin(name string) bool {
	_, ok := Registry[name]
	return ok
}

// registerArithmeticMeta registers metadata for arithmetic builtins
func registerArithmeticMeta() {
	// Integer operations
	Registry["add_Int"] = &BuiltinMeta{Name: "add_Int", NumArgs: 2, IsPure: true}
	Registry["sub_Int"] = &BuiltinMeta{Name: "sub_Int", NumArgs: 2, IsPure: true}
	Registry["mul_Int"] = &BuiltinMeta{Name: "mul_Int", NumArgs: 2, IsPure: true}
	Registry["div_Int"] = &BuiltinMeta{Name: "div_Int", NumArgs: 2, IsPure: true}
	Registry["mod_Int"] = &BuiltinMeta{Name: "mod_Int", NumArgs: 2, IsPure: true}
	Registry["neg_Int"] = &BuiltinMeta{Name: "neg_Int", NumArgs: 1, IsPure: true}

	// Float operations
	Registry["add_Float"] = &BuiltinMeta{Name: "add_Float", NumArgs: 2, IsPure: true}
	Registry["sub_Float"] = &BuiltinMeta{Name: "sub_Float", NumArgs: 2, IsPure: true}
	Registry["mul_Float"] = &BuiltinMeta{Name: "mul_Float", NumArgs: 2, IsPure: true}
	Registry["div_Float"] = &BuiltinMeta{Name: "div_Float", NumArgs: 2, IsPure: true}
	Registry["mod_Float"] = &BuiltinMeta{Name: "mod_Float", NumArgs: 2, IsPure: true}
	Registry["neg_Float"] = &BuiltinMeta{Name: "neg_Float", NumArgs: 1, IsPure: true}
}

// registerComparisonMeta registers metadata for comparison builtins
func registerComparisonMeta() {
	// Integer comparisons
	Registry["eq_Int"] = &BuiltinMeta{Name: "eq_Int", NumArgs: 2, IsPure: true}
	Registry["ne_Int"] = &BuiltinMeta{Name: "ne_Int", NumArgs: 2, IsPure: true}
	Registry["lt_Int"] = &BuiltinMeta{Name: "lt_Int", NumArgs: 2, IsPure: true}
	Registry["le_Int"] = &BuiltinMeta{Name: "le_Int", NumArgs: 2, IsPure: true}
	Registry["gt_Int"] = &BuiltinMeta{Name: "gt_Int", NumArgs: 2, IsPure: true}
	Registry["ge_Int"] = &BuiltinMeta{Name: "ge_Int", NumArgs: 2, IsPure: true}

	// Float comparisons
	Registry["eq_Float"] = &BuiltinMeta{Name: "eq_Float", NumArgs: 2, IsPure: true}
	Registry["ne_Float"] = &BuiltinMeta{Name: "ne_Float", NumArgs: 2, IsPure: true}
	Registry["lt_Float"] = &BuiltinMeta{Name: "lt_Float", NumArgs: 2, IsPure: true}
	Registry["le_Float"] = &BuiltinMeta{Name: "le_Float", NumArgs: 2, IsPure: true}
	Registry["gt_Float"] = &BuiltinMeta{Name: "gt_Float", NumArgs: 2, IsPure: true}
	Registry["ge_Float"] = &BuiltinMeta{Name: "ge_Float", NumArgs: 2, IsPure: true}
}

// registerConversionMeta registers metadata for numeric conversion builtins
func registerConversionMeta() {
	Registry["intToFloat"] = &BuiltinMeta{Name: "intToFloat", NumArgs: 1, IsPure: true}
	Registry["floatToInt"] = &BuiltinMeta{Name: "floatToInt", NumArgs: 1, IsPure: true}
}

// registerStringMeta registers metadata for string operation builtins
func registerStringMeta() {
	Registry["concat_String"] = &BuiltinMeta{Name: "concat_String", NumArgs: 2, IsPure: true}
	Registry["eq_String"] = &BuiltinMeta{Name: "eq_String", NumArgs: 2, IsPure: true}
	Registry["ne_String"] = &BuiltinMeta{Name: "ne_String", NumArgs: 2, IsPure: true}
	Registry["lt_String"] = &BuiltinMeta{Name: "lt_String", NumArgs: 2, IsPure: true}
	Registry["le_String"] = &BuiltinMeta{Name: "le_String", NumArgs: 2, IsPure: true}
	Registry["gt_String"] = &BuiltinMeta{Name: "gt_String", NumArgs: 2, IsPure: true}
	Registry["ge_String"] = &BuiltinMeta{Name: "ge_String", NumArgs: 2, IsPure: true}
}

// registerBooleanMeta registers metadata for boolean operation builtins
func registerBooleanMeta() {
	Registry["and_Bool"] = &BuiltinMeta{Name: "and_Bool", NumArgs: 2, IsPure: true}
	Registry["or_Bool"] = &BuiltinMeta{Name: "or_Bool", NumArgs: 2, IsPure: true}
	Registry["not_Bool"] = &BuiltinMeta{Name: "not_Bool", NumArgs: 1, IsPure: true}
	Registry["eq_Bool"] = &BuiltinMeta{Name: "eq_Bool", NumArgs: 2, IsPure: true}
	Registry["ne_Bool"] = &BuiltinMeta{Name: "ne_Bool", NumArgs: 2, IsPure: true}
}

// registerStringPrimitiveMeta registers metadata for low-level string operation builtins
func registerStringPrimitiveMeta() {
	Registry["_str_len"] = &BuiltinMeta{Name: "_str_len", NumArgs: 1, IsPure: true}
	Registry["_str_slice"] = &BuiltinMeta{Name: "_str_slice", NumArgs: 3, IsPure: true}
	Registry["_str_compare"] = &BuiltinMeta{Name: "_str_compare", NumArgs: 2, IsPure: true}
	Registry["_str_eq"] = &BuiltinMeta{Name: "_str_eq", NumArgs: 2, IsPure: true}
	Registry["_str_find"] = &BuiltinMeta{Name: "_str_find", NumArgs: 2, IsPure: true}
	Registry["_str_upper"] = &BuiltinMeta{Name: "_str_upper", NumArgs: 1, IsPure: true}
	Registry["_str_lower"] = &BuiltinMeta{Name: "_str_lower", NumArgs: 1, IsPure: true}
	Registry["_str_trim"] = &BuiltinMeta{Name: "_str_trim", NumArgs: 1, IsPure: true}
}

// registerIOMeta registers metadata for I/O operation builtins
func registerIOMeta() {
	Registry["_io_print"] = &BuiltinMeta{Name: "_io_print", NumArgs: 1, IsPure: false}
	Registry["_io_println"] = &BuiltinMeta{Name: "_io_println", NumArgs: 1, IsPure: false}
	Registry["_io_readLine"] = &BuiltinMeta{Name: "_io_readLine", NumArgs: 0, IsPure: false}
}

// registerJSONMeta registers metadata for JSON encoding builtins
func registerJSONMeta() {
	Registry["_json_encode"] = &BuiltinMeta{Name: "_json_encode", NumArgs: 1, IsPure: true}
	Registry["_json_decode"] = &BuiltinMeta{Name: "_json_decode", NumArgs: 1, IsPure: true}
}

// registerNetMeta registers metadata for Net effect builtins
func registerNetMeta() {
	Registry["_net_httpGet"] = &BuiltinMeta{Name: "_net_httpGet", NumArgs: 1, IsPure: false}
	Registry["_net_httpPost"] = &BuiltinMeta{Name: "_net_httpPost", NumArgs: 2, IsPure: false}
	Registry["_net_httpRequest"] = &BuiltinMeta{Name: "_net_httpRequest", NumArgs: 4, IsPure: false}
}
