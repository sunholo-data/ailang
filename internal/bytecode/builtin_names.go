package bytecode

// BuiltinNames is the canonical, source-ordered table of pure builtins
// reachable from the Phase 2C golden corpus. Indices in this table become
// the B field of OpBuiltinCall and are interpreted by the VM's builtin
// dispatch table (internal/vm/builtins.go).
//
// It lives here, not in the compiler or the VM, because it is the contract
// between them: the compiler emits these indices and the VM dispatches them.
//
// Adding a native builtin is a two-step process:
//  1. Append its name here.
//  2. Add a matching entry in vm.BuiltinTable.
//
// The VM checks at package init that the lengths agree (validateBuiltinTables).
var BuiltinNames = []string{
	"_show",
	"_len",
	"_list_get",
	"_list_tail",
	"_concat_String",
	"_record_get",
	"_not_Bool",
	"_intToFloat",
	"__list_length",
	"_concat_List",
	// M-BYTECODE-STDLIB-BUILTINS M1: string builtins
	"__str_len",
	"__str_compare",
	"__str_eq",
	"__str_find",
	"__str_slice",
	"__str_trim",
	"__str_upper",
	"__str_lower",
	"__str_split",
	"__str_chars",
	"__str_startsWith",
	"__str_endsWith",
	"__str_join",
	"__str_words",
	"__str_splitAny",
	"__str_replace",
	"__str_replaceMany",
	"__str_startsWithIC",
	"__str_charAt",
	"__str_charCode",
	"__str_decodeQP",
	"__escapeXml",
	"__string_intToStr",
	"__string_floatToStr",
	"__stringToInt",
	"__stringToFloat",
	// M-BYTECODE-STDLIB-BUILTINS M2: math + conversion builtins
	"__math_sin",
	"__math_cos",
	"__math_tan",
	"__math_asin",
	"__math_acos",
	"__math_atan",
	"__math_atan2",
	"__math_sqrt",
	"__math_pow",
	"__math_exp",
	"__math_log",
	"__math_log10",
	"__math_floor",
	"__math_ceil",
	"__math_round",
	"__math_abs_Float",
	"__math_abs_Int",
	"__math_PI",
	"__math_E",
	"_floatToInt",
	"_mod_Int",
	"__float_to_int",
	"__int_to_float",
	// M-BYTECODE-STDLIB-BUILTINS M3: list builtins
	"__list_nth",
	"__list_member",
	"__list_dedup",
	"__list_difference",
	"__list_intersect",
	"__list_union",
	// M-BYTECODE-PURE-EFFECTS M1: JSON builtins
	"__json_encode",
	"__json_decode",
	"__json_repair",
	// M-BYTECODE-XML-BUILTINS: XML builtins
	"__xmlElement",
	"__xmlText",
	"__xmlComment",
	"__xml_getText",
	"__xml_getTag",
	"__xml_serialize",
	"__xml_serializeWithDecl",
	"__xml_parse",
	"__xml_parseElements",
	"__xml_parseWithLimit",
	"__xml_findAll",
	"__xml_findFirst",
	"__xml_getAttr",
	"__xml_getChildren",
	"__xml_findAllTexts",
	"__xml_findAllAttrs",
	// ailang#1354/#1355: by-name record update for bases of unknown type
	"_record_set",
	// M-VM-PURE-BUILTIN-COVERAGE M3 (#1447): polymorphic list ops, native
	// because their elements may be ADTs or closures
	"__list_reverse",
	"__list_take",
	"__list_drop",
	"__list_zip",
	"__list_contains",
	"__list_head",
	"__list_extract",
	// polymorphic std/array ops (stapledons_godot 2026-10-01)
	"__array_empty",
	"__array_make",
	"__array_get",
	"__array_unsafe_get",
	"__array_length",
	"__array_set",
	"__array_from_list",
	"__array_to_list",
	"__array_append",
	"__array_update_many",
}

// HOFBuiltinNames lists builtins that take closure arguments. These are
// dispatched via OpBuiltinCallHOF, which passes the VM as a ClosureCaller
// so the builtin can invoke its closure arguments.
// Order MUST match vm.HOFBuiltinTable.
var HOFBuiltinNames = []string{
	"__list_map",
	"__list_filter",
	"__list_foldl",
	"__str_foldChars",
	"__str_foldSlices",
	"__str_mapSlicesJoin",
	"__xml_parseFold",
	"__list_sortBy",
	"__list_flatMap",
}
