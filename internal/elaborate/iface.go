package elaborate

import (
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/types"
)

// BuildInterface extracts a module interface from an elaborated Core program
func BuildInterface(prog *core.Program, modulePath string, typeEnv *types.TypeEnv) *iface.Iface {
	moduleIface := iface.NewIface(modulePath)
	
	// Extract exports from top-level declarations
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *core.Let:
			// Single non-recursive binding
			if isExported(d.Name) {
				// Get the type from the type environment
				typ := getTypeScheme(d.Name, typeEnv)
				purity := isPure(d.Value)
				moduleIface.AddExport(d.Name, typ, purity)
			}
			
		case *core.LetRec:
			// Recursive bindings
			for _, binding := range d.Bindings {
				if isExported(binding.Name) {
					typ := getTypeScheme(binding.Name, typeEnv)
					purity := isPure(binding.Value)
					moduleIface.AddExport(binding.Name, typ, purity)
				}
			}
		}
	}
	
	return moduleIface
}

// isExported checks if a name should be exported (public names don't start with _)
func isExported(name string) bool {
	return len(name) > 0 && name[0] != '_'
}

// getTypeScheme retrieves the generalized type scheme for a name
func getTypeScheme(name string, typeEnv *types.TypeEnv) *types.Scheme {
	// For now, return a placeholder
	// TODO: Properly extract and generalize the type from the environment
	return &types.Scheme{
		TypeVars:    []string{},
		Constraints: []types.Constraint{},
		Type:        &types.TCon{Name: "placeholder"},
	}
}

// isPure checks if an expression is pure (no side effects)
func isPure(expr core.CoreExpr) bool {
	// For now, assume all functions are pure
	// TODO: Implement proper purity analysis
	return true
}