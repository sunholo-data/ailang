package types

// Module-mode configuration: the module pipelines (internal/pipeline and the
// REPL's module registry) call InferWithConstraints once per module-level
// declaration and tell the checker so through these setters.

// SetModuleFuncNames records the module-level function names of the current
// module. Consumed ONLY by inferVar's undefined-variable diagnostic to detect
// the #327 shape (a local function that resolves as "undefined variable" in a
// record-update field). Interim; retired when the real resolution fix ships.
func (tc *CoreTypeChecker) SetModuleFuncNames(names map[string]bool) {
	tc.moduleFuncNames = names
}

// SetModuleDeclarationMode marks every InferWithConstraints call on this
// checker as checking ONE module-level declaration, whose own binding is then
// exempt from the integer-literal restriction. Set by the module pipelines;
// expression / REPL inference leaves it off, so every let is restricted.
func (tc *CoreTypeChecker) SetModuleDeclarationMode(on bool) {
	tc.moduleDeclMode = on
}
