package pipeline

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/link"
	"github.com/sunholo/ailang/internal/types"
)

// importedCtorInfo tracks full constructor info for elaborator registration.
// This is needed so the elaborator can recognize nullary constructors
// (like None) in pattern matching and not treat them as variable patterns.
type importedCtorInfo struct {
	TypeName       string
	Arity          int
	TypeParamCount int
}

// moduleImports holds the resolved import data for a module being compiled.
type moduleImports struct {
	ExternalTypes         map[string]*types.Scheme
	GlobalRefs            map[string]core.GlobalRef
	ImportedTypeAliases   map[string]types.Type
	ImportedCtorTypes     map[string]string
	ImportedADTTypeParams map[string]int
	ImportedCtorInfos     map[string]*importedCtorInfo
}

// resolveModuleImports builds the external type environment and global references
// by collecting exports, types, and constructors from already-compiled dependencies.
func resolveModuleImports(
	fileImports []*ast.ImportDecl,
	modID string,
	modLinker *link.ModuleLinker,
	cfg Config,
) *moduleImports {
	imports := &moduleImports{
		ExternalTypes:         make(map[string]*types.Scheme),
		GlobalRefs:            make(map[string]core.GlobalRef),
		ImportedTypeAliases:   make(map[string]types.Type),
		ImportedCtorTypes:     make(map[string]string),
		ImportedADTTypeParams: make(map[string]int),
		ImportedCtorInfos:     make(map[string]*importedCtorInfo),
	}

	// Always include $builtin module exports (available to all modules)
	if builtinIface := modLinker.GetIface("$builtin"); builtinIface != nil {
		for name, item := range builtinIface.Exports {
			// Add with qualified key (for explicit $builtin.name references)
			key := fmt.Sprintf("%s.%s", item.Ref.Module, item.Ref.Name)
			imports.ExternalTypes[key] = item.Type

			// CRITICAL FIX: Also add with simple name so stdlib can reference _io_print directly
			// This preserves the effect row from the spec registry
			imports.ExternalTypes[name] = item.Type

			imports.GlobalRefs[name] = item.Ref
		}
	}

	// Get imports for this module
	if len(fileImports) == 0 {
		return imports
	}

	for _, imp := range fileImports {
		// Get the interface of the imported module
		depIface := modLinker.GetIface(imp.Path)
		if depIface == nil {
			if cfg.TraceDefaulting {
				fmt.Printf("WARNING: No interface for module %s (importing from %s)\n", imp.Path, modID)
			}
			continue
		}
		if len(imp.Symbols) > 0 {
			resolveSelectiveImports(imp, depIface, imports, cfg)
		}

		// Handle module alias: import std/list as List
		// Add all exports with qualified names (List.map, List.filter, etc.)
		if imp.ModuleAlias != "" {
			for name, item := range depIface.Exports {
				qualifiedName := fmt.Sprintf("%s.%s", imp.ModuleAlias, name)
				key := fmt.Sprintf("%s.%s", item.Ref.Module, item.Ref.Name)
				imports.ExternalTypes[key] = item.Type
				imports.GlobalRefs[qualifiedName] = item.Ref
				if cfg.TraceDefaulting {
					fmt.Printf("  Module alias %s.%s -> %s\n", imp.ModuleAlias, name, key)
				}
			}
		}
	}

	return imports
}

// resolveSelectiveImports resolves selective symbol imports from a dependency interface.
func resolveSelectiveImports(
	imp *ast.ImportDecl,
	depIface *iface.Iface,
	imports *moduleImports,
	cfg Config,
) {
	for _, sym := range imp.Symbols {
		found := false

		// Determine the name to bind (use alias if present)
		bindName := sym
		if imp.SymbolAliases != nil {
			if alias, ok := imp.SymbolAliases[sym]; ok {
				bindName = alias
			}
		}

		// Try to import as a regular export (function/value)
		if item, ok := depIface.GetExport(sym); ok {
			key := fmt.Sprintf("%s.%s", item.Ref.Module, item.Ref.Name)
			imports.ExternalTypes[key] = item.Type
			imports.GlobalRefs[bindName] = item.Ref // Use alias name for binding
			if cfg.TraceDefaulting {
				fmt.Printf("  Import value %s as %s -> %s (%s)\n", sym, bindName, key, item.Type)
			}
			found = true
		}

		// Try to import as a type name
		if typ, ok := depIface.GetType(sym); ok {
			if cfg.TraceDefaulting {
				fmt.Printf("  Import type %s (arity %d)\n", typ.Name, typ.Arity)
			}
			// M-FIX-RECORD-UPDATE: Also import type alias if present
			// This enables cross-module record update syntax
			if alias, hasAlias := depIface.GetTypeAlias(sym); hasAlias {
				imports.ImportedTypeAliases[sym] = alias
				if cfg.TraceDefaulting {
					fmt.Printf("  Import type alias %s -> %s\n", sym, alias)
				}
			}
			found = true
		}

		// Try to import as a constructor
		// DEBUG: fmt.Printf("DEBUG: Checking if %s is a constructor in %s (has %d constructors)...\n", sym, imp.Path, len(depIface.Constructors))
		for range depIface.Constructors {
			// DEBUG: fmt.Printf("DEBUG:   Constructor %s in interface\n", k)
		}
		if ctor, ok := depIface.GetConstructor(sym); ok {
			resolveConstructorImport(sym, ctor, imports, cfg)
			found = true
		}
		// No else needed - if constructor not found, we continue searching

		if !found && cfg.TraceDefaulting {
			fmt.Printf("  Symbol %s not found in %s\n", sym, imp.Path)
		}
	}

	// M-TYPE-ALIAS: Import ALL type aliases from this module regardless of what symbols
	// were imported. Previously this only ran when importing a type name, which meant
	// importing functions/constructors from a module didn't bring in its type aliases.
	// This is needed for cross-package type alias unification: if Package C imports
	// function applyDelta(Usage, ...) from Package A, it needs the Usage alias for unification.
	for aliasName, aliasTarget := range depIface.TypeAliases {
		if _, exists := imports.ImportedTypeAliases[aliasName]; !exists {
			imports.ImportedTypeAliases[aliasName] = aliasTarget
			if cfg.TraceDefaulting {
				fmt.Printf("  Import type alias %s -> %s from %s\n", aliasName, aliasTarget, imp.Path)
			}
		}
	}

	// M-CTOR-AUTO: Auto-import ALL constructors from imported modules so the
	// elaborator can recognize nullary constructors (like None, Err) in pattern
	// matching even when the user only imports functions (e.g., getOrElse, isSome).
	// Without this, `None` in a match pattern is treated as a variable binding,
	// silently matching everything. Same pattern as type alias auto-import above.
	for _, ctor := range depIface.Constructors {
		if _, exists := imports.ImportedCtorInfos[ctor.CtorName]; !exists {
			resolveConstructorImport(ctor.CtorName, ctor, imports, cfg)
		}
	}
}

// resolveConstructorImport adds a single constructor from a dependency to the import set.
func resolveConstructorImport(
	sym string,
	ctor *iface.ConstructorScheme,
	imports *moduleImports,
	cfg Config,
) {
	factoryName := fmt.Sprintf("make_%s_%s", ctor.TypeName, ctor.CtorName)
	key := fmt.Sprintf("$adt.%s", factoryName)

	imports.GlobalRefs[sym] = core.GlobalRef{
		Module: "$adt",
		Name:   factoryName,
	}

	// CRITICAL FIX: Also add to externalTypes so type checker knows the signature
	// Build the factory type scheme from the constructor info
	var factoryType types.Type
	if ctor.Arity == 0 {
		// Nullary constructor: just the result type
		factoryType = ctor.ResultType
	} else {
		// Constructor with fields: FieldTypes -> ResultType
		factoryType = &types.TFunc2{
			Params:    ctor.FieldTypes,
			EffectRow: nil, // Pure constructor
			Return:    ctor.ResultType,
		}
	}

	// Extract type variables from result type for polymorphism
	var typeVars []string
	if ctor.ResultType != nil {
		// Extract type vars from result type (e.g., Option[a] -> ["a"])
		typeVars = extractTypeVarsFromType(ctor.ResultType)
	}

	imports.ExternalTypes[key] = &types.Scheme{
		TypeVars: typeVars,
		Type:     factoryType,
	}

	// DEBUG: fmt.Printf("DEBUG: Import constructor %s -> %s with type scheme (vars: %v)\n", sym, key, typeVars)
	if cfg.TraceDefaulting {
		fmt.Printf("  Import constructor %s -> %s\n", sym, key)
	}

	// M-TAPP-FIX: Track imported constructor for pattern matching type inference
	imports.ImportedCtorTypes[sym] = ctor.TypeName

	// M-TAPP-FIX: Derive type param count from ResultType
	// If ResultType is TApp, count the args; otherwise 0
	paramCount := 0
	if tapp, ok := ctor.ResultType.(*types.TApp); ok {
		paramCount = len(tapp.Args)
	}
	if _, exists := imports.ImportedADTTypeParams[ctor.TypeName]; !exists {
		imports.ImportedADTTypeParams[ctor.TypeName] = paramCount
	}

	// M-RT1-FIX: Track full constructor info for elaborator registration
	// This is needed so the elaborator can recognize nullary constructors
	// (like None) in pattern matching and not treat them as variable patterns
	imports.ImportedCtorInfos[sym] = &importedCtorInfo{
		TypeName:       ctor.TypeName,
		Arity:          ctor.Arity,
		TypeParamCount: paramCount,
	}
}
