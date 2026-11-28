package loader

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
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

// PreludeModules lists the stdlib modules whose types + data constructors are
// implicitly available (no explicit `import`) — see the boundary below.
//
// M-PRELUDE-OPTION-RESULT: Option/Some/None and Result/Ok/Err are near-universal
// in AI-authored code (they are in the standard prelude of Haskell, Rust, OCaml,
// F#, Swift, ...). Models routinely write idiomatically-correct AILANG that uses
// them but omit `import std/option` / `import std/result`. Rather than re-teach
// every model, these two modules are implicit dependencies — reusing the ENTIRE
// existing constructor-import machinery.
//
// THE BOUNDARY (#2371, maintainer-approved):
//   - Exactly six names: Option, Some, None (std/option) and Result, Ok, Err
//     (std/result) — preludeModuleSymbols. No helper function (map, getOrElse,
//     … still need `import std/option (map)`), and no other module.
//   - Every module outside std/: entry modules always; a library module when
//     its code names one of a module's three (see injectPreludeImports). The
//     effectful prelude (`println`, pipeline.InjectPrelude) stays ENTRY-ONLY.
//   - Lowest precedence: a module's own declaration of a prelude type or
//     constructor drops that prelude module, an import from std/option or
//     std/result replaces the implicit one (so `import std/option (Option,
//     Some, None)` stays valid, without warning), and an explicit import of
//     another type's Some/Ok wins for that name.
//   - std/ modules are untouched.
//
// Until #2371 library modules got none of the six ("library code stays
// self-documenting"), so a library's Some fell back to matching by name
// (#323) — the source of #2303, #2272 and #2353 — and the same source meant
// different things depending on whether the file had a `main`.
//
// This is the single source of truth shared by the compile pipeline and the
// runtime: both consume the loader's `LoadedModule.Imports` (string paths) and
// `LoadedModule.File.Imports` (AST decls), so injecting here — at the one loader
// call-site — keeps compile + runtime in lockstep (guard the call-site, not a
// helper the call-site might forget to use).
func PreludeModules() []string {
	return []string{"std/option", "std/result"}
}

// PreludeSymbols returns the type + constructor symbols brought into scope by
// the implicit prelude import of modPath (e.g. "std/option" -> Option, Some,
// None), or nil if modPath is not an implicit-prelude module. Exported accessor
// over preludeModuleSymbols (the single source of truth) so `ailang docs prelude`
// can render the implicit-prelude surface WITHOUT copying the symbol list — a
// symbol added here appears in the docs page automatically (M-DX-AI-DISCOVERY M4).
func PreludeSymbols(modPath string) []string {
	syms := preludeModuleSymbols[modPath]
	if syms == nil {
		return nil
	}
	return append([]string(nil), syms...)
}

// injectPreludeImports mutates a module's AST + import list so that
// PreludeModules() are implicitly imported.
//
// It is:
//   - Every module but the standard library's: no-op unless
//     preludeEligible(file, modID).
//   - Pay-for-use outside entry modules: a library module gets a prelude
//     module only when its code names one of that module's names (namesIn: an
//     identifier token, so not a comment or a string's text, but code inside
//     `${…}`). A name a module never writes cannot be observed in scope, so
//     the scope is the entry module's for every name it uses, while a module
//     that uses none keeps its dependencies, cache key and program image
//     exactly as before (#2371).
//   - Deduped: a prelude module the user ALREADY imports (explicitly, in any
//     form — whole-module or selective) is skipped, so there is no double-load
//     and no duplicate constructor registration / ambiguity.
//   - Precedence-safe: implicit imports are PREPENDED (registered first =
//     lowest precedence) so a user's own local `type Option` / `type Result`
//     or an explicit import shadows them cleanly downstream.
//
// Returns the (possibly extended) import-path slice for LoadedModule.Imports.
func injectPreludeImports(file *ast.File, modID, source string, importPaths []string) []string {
	if !preludeEligible(file, modID) {
		return importPaths
	}
	entry := isEntryModuleFile(file)
	// DERIVE-JSON: `deriving (Json)` synthesizes decode<T> bodies that name
	// Option/Some/None and Result/Ok/Err even when the source text never does,
	// so such a library module needs both prelude modules (shadowing still wins).
	derivesJson := fileHasDerivingJson(file)
	var used map[string]bool
	if !entry && !derivesJson {
		used = namesIn(source)
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
	for _, modPath := range PreludeModules() {
		if existing[modPath] {
			continue // user already imports it — leave their import untouched
		}
		if preludeModuleShadowed(modPath, localTypes, localCtors) {
			// The local definition wins. A deriving (Json) module still needs
			// the prelude ADTs for its synthesized codecs, so it gets them under
			// hygienic `$` names no source can write (TST-RC1-07).
			if derivesJson {
				synthetic = append(synthetic, hygienicPreludeImport(modPath))
				syntheticPaths = append(syntheticPaths, modPath)
			}
			continue // user defines a colliding type/constructor locally — shadow
		}
		if !entry && !derivesJson && !usesAny(used, preludeModuleSymbols[modPath]) {
			continue // a library module whose code never names Option/Some/None (resp. Result/Ok/Err)
		}
		// Selective import naming the type + constructors, exactly as a user
		// would write `import std/option (Option, Some, None)`. This routes
		// through resolveSelectiveImports so the type, the $adt factories, and
		// the constructor infos (for elaborator match-pattern registration) all
		// land in scope.
		synthetic = append(synthetic, &ast.ImportDecl{
			Path:     modPath,
			Symbols:  preludeModuleSymbols[modPath],
			Implicit: true,
		})
		syntheticPaths = append(syntheticPaths, modPath)
	}

	if len(synthetic) == 0 {
		return importPaths
	}

	if os.Getenv("DEBUG_PRELUDE") != "" {
		fmt.Fprintf(os.Stderr, "prelude: %s — injecting implicit imports %v\n", modID, syntheticPaths)
	}

	// PREPEND so implicit imports are lowest precedence (registered first;
	// later user imports / local decls override on same-name collisions).
	file.Imports = append(synthetic, file.Imports...)
	return append(syntheticPaths, importPaths...)
}

// namesIn returns the identifiers source's code uses, from its tokens: a name
// in a comment or in a string's text does not count, a name inside a `${…}`
// interpolation does (the lexer emits interpolated code as tokens).
func namesIn(source string) map[string]bool {
	names := make(map[string]bool)
	l := lexer.New(source, "")
	for tok := l.NextToken(); tok.Type != lexer.EOF; tok = l.NextToken() {
		if tok.Type == lexer.IDENT {
			names[tok.Literal] = true
		}
	}
	return names
}

// hygienicPreludeImport imports modPath's prelude symbols under `$`-suffixed
// aliases (std/option's Some as Some$option, …) — the names the derived Json
// codecs reference (elaborate.djSome etc.) — so a deriving (Json) module that
// shadows Option/Result locally keeps its own names and still type-checks.
func hygienicPreludeImport(modPath string) *ast.ImportDecl {
	suffix := "$" + strings.TrimPrefix(modPath, "std/")
	aliases := make(map[string]string, len(preludeModuleSymbols[modPath]))
	for _, sym := range preludeModuleSymbols[modPath] {
		aliases[sym] = sym + suffix
	}
	return &ast.ImportDecl{Path: modPath, Symbols: preludeModuleSymbols[modPath], SymbolAliases: aliases, Implicit: true}
}

// fileHasDerivingJson reports whether any type declared in file derives Json.
func fileHasDerivingJson(file *ast.File) bool {
	for _, list := range [][]ast.Node{file.Decls, file.Statements} {
		for _, node := range list {
			if td, ok := node.(*ast.TypeDecl); ok && slices.Contains(td.Deriving, ast.DeriveJson) {
				return true
			}
		}
	}
	return false
}

func usesAny(used map[string]bool, names []string) bool {
	for _, n := range names {
		if used[n] {
			return true
		}
	}
	return false
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

// preludeEligible reports whether file (module modID) receives the implicit
// Option/Result prelude (#2371): an entry module always does, as before; any
// other module does unless it is
//   - a standard-library module (std/option and std/result cannot import
//     themselves, and std modules import what they use — unchanged), or
//   - a file with a top-level expression: such a file is type-checked only
//     while it has no imports (#2372), so an injected import would stop its
//     expression being checked. Before #2371 it had no prelude either.
func preludeEligible(file *ast.File, modID string) bool {
	if file == nil {
		return false
	}
	if isEntryModuleFile(file) {
		return true
	}
	if strings.HasPrefix(modID, "std/") || (file.Module != nil && strings.HasPrefix(file.Module.Path, "std/")) {
		return false
	}
	for _, st := range file.Statements {
		if _, isExpr := st.(ast.Expr); isExpr {
			return false
		}
	}
	return true
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
