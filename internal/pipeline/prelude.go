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

	// Inject println type: string -> () ! {IO}
	// println is the common case (with newline), print (no newline) requires import
	T := types.NewBuilder()
	printlnType := T.Func(T.String()).Returns(T.Unit()).Effects("IO")

	// Wrap in a scheme (no type variables)
	printlnScheme := &types.Scheme{
		TypeVars: []string{},
		RowVars:  []string{},
		Type:     printlnType,
	}

	env = env.ExtendScheme("println", printlnScheme)
	return env
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
