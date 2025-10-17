package pipeline

import (
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/types"
)

// Converter functions for pipeline data structures

// convertParserErrors converts parser errors to structured AILANG errors
func convertParserErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	// For now, return the first error
	// TODO: Return all errors with proper structure
	return errs[0]
}

// convertConstructors converts elaborator constructors to pipeline ConstructorInfo
func convertConstructors(elabCtors map[string]*elaborate.ConstructorInfo) map[string]*ConstructorInfo {
	ctors := make(map[string]*ConstructorInfo)
	for name, elabCtor := range elabCtors {
		ctors[name] = &ConstructorInfo{
			TypeName:   elabCtor.TypeName,
			CtorName:   elabCtor.CtorName,
			FieldTypes: nil, // We don't have AST types here, will infer from Core
			Arity:      elabCtor.Arity,
		}
	}
	return ctors
}

// convertToIfaceConstructors converts pipeline constructors to iface constructors
func convertToIfaceConstructors(pipeCtors map[string]*ConstructorInfo) map[string]*iface.ConstructorInfo {
	if pipeCtors == nil {
		return nil
	}
	ifaceCtors := make(map[string]*iface.ConstructorInfo)
	for name, pipeCtor := range pipeCtors {
		ifaceCtors[name] = &iface.ConstructorInfo{
			TypeName: pipeCtor.TypeName,
			CtorName: pipeCtor.CtorName,
			Arity:    pipeCtor.Arity,
		}
	}
	return ifaceCtors
}

// extractTypeVarsFromType extracts type variable names from a type
// For example: Option[a] -> ["a"], Result[t, e] -> ["t", "e"]
func extractTypeVarsFromType(typ types.Type) []string {
	var vars []string
	seen := make(map[string]bool)

	var extract func(types.Type)
	extract = func(t types.Type) {
		if t == nil {
			return
		}
		switch typ := t.(type) {
		case *types.TVar2:
			if !seen[typ.Name] {
				vars = append(vars, typ.Name)
				seen[typ.Name] = true
			}
		case *types.TApp:
			extract(typ.Constructor)
			for _, arg := range typ.Args {
				extract(arg)
			}
		case *types.TFunc2:
			for _, param := range typ.Params {
				extract(param)
			}
			extract(typ.Return)
		case *types.TList:
			extract(typ.Element)
		case *types.TTuple:
			for _, elem := range typ.Elements {
				extract(elem)
			}
			// TCon and other base types don't have type variables
		}
	}

	extract(typ)
	return vars
}
