package loader

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/ast"
)

// preludeModuleSymbols maps each implicit-prelude module to the type +
// constructor symbols brought into scope. A whole-module `import std/option`
// (no symbol list) does NOT auto-import constructors — the constructor
// auto-import (M-CTOR-AUTO) only runs on the SELECTIVE-import path — so the
// synthetic imports must name their symbols explicitly, exactly as a user would
// write `import std/option (Option, Some, None)`.
var preludeModuleSymbols = map[string][]string{
	"std/option": {"Option", "Some", "None"},
	"std/result": {"Result", "Ok", "Err"},
}

// EntryPreludeModules lists the stdlib modules whose types + data constructors
// are made implicitly available (no explicit `import`) in ENTRY modules.
//
// M-PRELUDE-OPTION-RESULT: Option/Some/None and Result/Ok/Err are near-universal
// in AI-authored code (they are in the standard prelude of Haskell, Rust, OCaml,
// F#, Swift, ...). Models routinely write idiomatically-correct AILANG that uses
// them but omit `import std/option` / `import std/result`. Rather than re-teach
// every model, we inject these two modules as implicit dependencies of entry
// modules — reusing the ENTIRE existing constructor-import machinery.
//
// This is the single source of truth shared by the compile pipeline and the
// runtime: both consume the loader's `LoadedModule.Imports` (string paths) and
// `LoadedModule.File.Imports` (AST decls), so injecting here — at the one loader
// call-site — keeps compile + runtime in lockstep (guard the call-site, not a
// helper the call-site might forget to use).
//
// Entry-only: library modules are NOT touched, so library code stays
// self-documenting and must import Option/Result explicitly.
func EntryPreludeModules() []string {
	return []string{"std/option", "std/result"}
}

// EntryPreludeSymbols returns the type + constructor symbols brought into scope
// by the implicit prelude import of modPath (e.g. "std/option" -> Option, Some,
// None), or nil if modPath is not an implicit-prelude module. Exported accessor
// over preludeModuleSymbols (the single source of truth) so `ailang docs prelude`
// can render the implicit-prelude surface WITHOUT copying the symbol list — a
// symbol added here appears in the docs page automatically (M-DX-AI-DISCOVERY M4).
func EntryPreludeSymbols(modPath string) []string {
	syms := preludeModuleSymbols[modPath]
	if syms == nil {
		return nil
	}
	return append([]string(nil), syms...)
}

// injectEntryPreludeImports mutates an entry module's AST + import list so that
// EntryPreludeModules() are implicitly imported.
//
// It is:
//   - Entry-only: no-op unless isEntryModuleFile(file) is true.
//   - Deduped: a prelude module the user ALREADY imports (explicitly, in any
//     form — whole-module or selective) is skipped, so there is no double-load
//     and no duplicate constructor registration / ambiguity.
//   - Precedence-safe: implicit imports are PREPENDED (registered first =
//     lowest precedence) so a user's own local `type Option` / `type Result`
//     or an explicit import shadows them cleanly downstream.
//
// Returns the (possibly extended) import-path slice for LoadedModule.Imports.
func injectEntryPreludeImports(file *ast.File, importPaths []string) []string {
	if file == nil || !isEntryModuleFile(file) {
		return importPaths
	}

	// Collect paths the user already imports (dedupe key = canonical module path).
	existing := make(map[string]bool, len(file.Imports))
	for _, imp := range file.Imports {
		existing[imp.Path] = true
	}

	// Collect local type names + constructor names declared in this file, so a
	// user's own `type Option` / `type Result` (or a locally-defined ctor whose
	// name collides, e.g. `Some`/`Ok`) SHADOWS the prelude: the module carrying
	// that colliding symbol is not injected at all, avoiding an ambiguous-
	// constructor error at compile/runtime. Prelude is lowest precedence.
	localTypes, localCtors := collectLocalTypesAndCtors(file)

	var synthetic []*ast.ImportDecl
	var syntheticPaths []string
	for _, modPath := range EntryPreludeModules() {
		if existing[modPath] {
			continue // user already imports it — leave their import untouched
		}
		if preludeModuleShadowed(modPath, localTypes, localCtors) {
			continue // user defines a colliding type/constructor locally — shadow
		}
		// Selective import naming the type + constructors, exactly as a user
		// would write `import std/option (Option, Some, None)`. This routes
		// through resolveSelectiveImports so the type, the $adt factories, and
		// the constructor infos (for elaborator match-pattern registration) all
		// land in scope.
		synthetic = append(synthetic, &ast.ImportDecl{
			Path:    modPath,
			Symbols: preludeModuleSymbols[modPath],
		})
		syntheticPaths = append(syntheticPaths, modPath)
	}

	if len(synthetic) == 0 {
		return importPaths
	}

	if os.Getenv("DEBUG_PRELUDE") != "" {
		fmt.Fprintf(os.Stderr, "prelude: entry module — injecting implicit imports %v\n", syntheticPaths)
	}

	// PREPEND so implicit imports are lowest precedence (registered first;
	// later user imports / local decls override on same-name collisions).
	file.Imports = append(synthetic, file.Imports...)
	return append(syntheticPaths, importPaths...)
}

// collectLocalTypesAndCtors returns the set of type names and constructor names
// declared locally in the file. Used to shadow the implicit prelude: any prelude
// module that would collide with a local definition is not injected.
func collectLocalTypesAndCtors(file *ast.File) (map[string]bool, map[string]bool) {
	types := make(map[string]bool)
	ctors := make(map[string]bool)
	for _, decl := range file.Decls {
		td, ok := decl.(*ast.TypeDecl)
		if !ok {
			continue
		}
		types[td.Name] = true
		if adt, ok := td.Definition.(*ast.AlgebraicType); ok {
			for _, c := range adt.Constructors {
				ctors[c.Name] = true
			}
		}
	}
	return types, ctors
}

// preludeModuleShadowed reports whether a prelude module's type or any of its
// constructors is locally (re)defined in the entry file. If so, injecting the
// prelude import would clash with the user's definition, so the module is
// skipped (the local definition wins — prelude is lowest precedence).
func preludeModuleShadowed(modPath string, localTypes, localCtors map[string]bool) bool {
	for _, sym := range preludeModuleSymbols[modPath] {
		// sym[0] is the type name; the rest are constructor names.
		if localTypes[sym] || localCtors[sym] {
			return true
		}
	}
	return false
}

// isEntryModuleFile reports whether the AST file is an entry module: it declares
// an exported `main` with zero params or a single unit param. This mirrors
// pipeline.IsEntryModuleFromAST, duplicated here because the loader package
// cannot import the pipeline package (pipeline imports loader).
func isEntryModuleFile(file *ast.File) bool {
	if file == nil {
		return false
	}
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if funcDecl.Name != "main" || !funcDecl.IsExport {
			continue
		}
		if len(funcDecl.Params) == 0 {
			return true
		}
		if len(funcDecl.Params) == 1 {
			if simpleType, ok := funcDecl.Params[0].Type.(*ast.SimpleType); ok && simpleType.Name == "()" {
				return true
			}
		}
	}
	return false
}
