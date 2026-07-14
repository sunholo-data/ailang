package pipeline

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

// InjectPrelude adds prelude type bindings to a type environment.
//
// The prelude is a curated set of convenience functions designed to reduce
// syntactic friction for entry modules and REPL use. It provides commonly-used
// functions without requiring explicit imports.
//
// **AI-First Design Philosophy:**
// "Minimize syntactic entropy" - teach the compiler to carry context so the AI doesn't have to.
// The prelude removes boilerplate (e.g., `import std/io (println)`) while preserving AILANG's
// core principle: effects remain explicit in type signatures (! {IO}).
//
// **Prelude contents:**
//   - println : string -> () ! {IO}   -- Print with newline (most common use case)
//
// **Note:** `print` (no newline) requires explicit import: `import std/io (print)`
//
// **Shadowing:** User definitions shadow prelude (no warning, intentional)
//
// **Entry Module Detection:** Caller is responsible for determining if this is an entry module.
// Use IsEntryModuleFromAST(file) to check before type checking.
//
// **Note:** This only injects TYPE bindings. Value bindings are injected separately
// in the evaluator via InjectPreludeValues().
func InjectPrelude(env *types.TypeEnv) *types.TypeEnv {
	// Debug logging (controlled by DEBUG_PRELUDE env var)
	if os.Getenv("DEBUG_PRELUDE") != "" {
		fmt.Fprintf(os.Stderr, "prelude: injecting type for [println]\n")
	}

	// The injected surface is enumerated by PreludeSurface() — ONE source of
	// truth shared by the real injection here and `ailang docs prelude`'s
	// renderer / reverse drift test. Adding a binding to preludeSurface() adds
	// it both to the type env AND to the docs page; removing it drops both.
	for _, b := range preludeSurface() {
		env = env.ExtendScheme(b.Name, b.Scheme)
	}
	return env
}

// PreludeBinding is one type binding injected into entry-module / REPL type
// environments by InjectPrelude, exposed so `ailang docs prelude` can render
// the real injected surface (name + scheme) without a hand-copied table
// (M-DX-AI-DISCOVERY M4).
type PreludeBinding struct {
	Name   string
	Scheme *types.Scheme
}

// preludeSurface is the single, authoritative list of type bindings InjectPrelude
// adds. Both InjectPrelude (the real injection) and PreludeSurface (docs +
// drift test) iterate this, so the two can never diverge.
func preludeSurface() []PreludeBinding {
	T := types.NewBuilder()

	// println : string -> () ! {IO}
	// println is the common case (with newline); print (no newline) requires import.
	printlnScheme := &types.Scheme{
		TypeVars: []string{},
		RowVars:  []string{},
		Type:     T.Func(T.String()).Returns(T.Unit()).Effects("IO"),
	}

	return []PreludeBinding{
		{Name: "println", Scheme: printlnScheme},
	}
}

// PreludeSurface returns the bindings InjectPrelude injects, in a stable order.
// Used by `ailang docs prelude` to render the injected surface from the live
// mechanism (no copied table) and by the reverse drift test.
func PreludeSurface() []PreludeBinding {
	return preludeSurface()
}

// TODO: InjectPreludeValues - inject runtime value bindings for print
// For now, print resolution happens via the global resolver looking up builtins
// Future enhancement: inject print as an actual binding in the eval environment

// IsEntryModule checks if the given public environment contains an exported main function.
//
// An entry module is defined as:
//   - Module exports a function named "main"
//   - The main function has arity 0 (no parameters)
//
// Entry modules get the prelude injected automatically. Library modules do not.
func IsEntryModule(publicEnv *iface.Iface) bool {
	if publicEnv == nil {
		return false
	}

	// Check if there's an exported symbol named "main"
	mainItem, exists := publicEnv.Exports["main"]
	if !exists {
		return false
	}

	// Check if it's a function type
	if mainItem.Type == nil {
		return false
	}

	// Extract the actual type from the scheme
	mainType := mainItem.Type.Type

	// Check if it's a function with 0 parameters or 1 unit parameter
	// Fixed v0.4.2: Accept both zero-param and unit-param (for S-CALL0 compatibility)
	if fn, ok := mainType.(*types.TFunc2); ok {
		// Zero params: func main() -> () (old style, may still exist)
		if len(fn.Params) == 0 {
			return true
		}
		// Unit param: func main(_: ()) -> () (new style with S-CALL0)
		if len(fn.Params) == 1 {
			// Check if the parameter is unit type
			if tcon, ok := fn.Params[0].(*types.TCon); ok {
				if tcon.Name == "()" {
					return true
				}
			}
		}
	}

	return false
}

// IsEntryModuleFromAST checks if the given AST file contains an exported main function.
//
// This is used for early detection before type checking, allowing prelude injection
// to happen before type checking rather than after.
//
// An entry module is defined as:
//   - File contains a function declaration named "main"
//   - The main function is exported (IsExport = true)
//   - The main function has 0 parameters
//
// Entry modules get the prelude injected automatically. Library modules do not.
func IsEntryModuleFromAST(file *ast.File) bool {
	if file == nil {
		return false
	}

	// Scan top-level declarations for exported main function
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Name == "main" && funcDecl.IsExport {
				// Fixed v0.4.2: Accept both zero-param (old) and unit-param (new with S-CALL0)
				// Zero params: export func main() -> () (old, but still may exist)
				// Unit param: export func main(_: ()) -> () (new, implicit from parser)
				if len(funcDecl.Params) == 0 {
					return true
				}
				if len(funcDecl.Params) == 1 {
					// Check if single param is unit type
					if simpleType, ok := funcDecl.Params[0].Type.(*ast.SimpleType); ok {
						if simpleType.Name == "()" {
							return true
						}
					}
				}
			}
		}
	}

	return false
}
