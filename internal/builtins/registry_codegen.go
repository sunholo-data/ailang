package builtins

// registerCodegenSpecs adds Go codegen specifications to all builtins.
// M-CODEGEN-SUSTAINABILITY: This is the single source of truth for how
// each builtin is emitted in generated Go code. Replaces mapPureMathBuiltin,
// mapPureListBuiltin, mapStdlibBuiltin, and codegen_runtime_stdlib.go.
//
// Two spec types:
//   - Inline: single Go expression with {{arg0}}, {{arg1}} placeholders
//   - Helper: runtime function emitted in runtime.go (for complex builtins)
//
// Implementations are split across focused files for AI-maintainability:
//
//   - registry_codegen_string.go  : std/string builtins
//   - registry_codegen_math.go    : std/math builtins + conversion
//   - registry_codegen_list.go    : std/list builtins
//   - registry_codegen_io.go      : std/io + effect stubs (FS, Env, AI)
//   - registry_codegen_json.go    : std/json + Option/Result helpers
//   - registry_codegen_effects.go : XML stubs, debug, effectful list, math helpers, process/IO
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
