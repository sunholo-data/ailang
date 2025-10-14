package repl

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/typedast"
	"github.com/sunholo/ailang/internal/types"
)

// formatCore formats Core AST for display
func formatCore(expr core.CoreExpr, indent string) string {
	switch e := expr.(type) {
	case *core.Var:
		return fmt.Sprintf("%sVar(%s)", indent, e.Name)
	case *core.Lit:
		return fmt.Sprintf("%sLit(%v)", indent, e.Value)
	case *core.Lambda:
		return fmt.Sprintf("%sLam(%v) ->\n%s", indent, e.Params, formatCore(e.Body, indent+"  "))
	case *core.App:
		args := ""
		for i, arg := range e.Args {
			if i > 0 {
				args += ",\n"
			}
			args += formatCore(arg, indent+"  ")
		}
		return fmt.Sprintf("%sApp(\n%s,\n%s)", indent,
			formatCore(e.Func, indent+"  "), args)
	case *core.Let:
		return fmt.Sprintf("%sLet(%s) =\n%s\n%sin\n%s", indent, e.Name,
			formatCore(e.Value, indent+"  "), indent,
			formatCore(e.Body, indent+"  "))
	case *core.DictApp:
		return fmt.Sprintf("%sDictApp(%s, %s, [...])", indent, e.Dict, e.Method)
	default:
		return fmt.Sprintf("%s%T", indent, e)
	}
}

// formatTyped formats TypedAST for display
func formatTyped(expr typedast.TypedNode, indent string) string {
	typ := expr.GetType()

	// Convert interface{} to string for display
	typeStr := fmt.Sprintf("%v", typ)

	switch e := expr.(type) {
	case *typedast.TypedVar:
		return fmt.Sprintf("%sVar(%s : %s)", indent, e.Name, typeStr)
	case *typedast.TypedLit:
		return fmt.Sprintf("%sLit(%v : %s)", indent, e.Value, typeStr)
	case *typedast.TypedLambda:
		paramStr := fmt.Sprintf("%v", e.Params)
		return fmt.Sprintf("%sLam(%s) ->\n%s", indent, paramStr,
			formatTyped(e.Body, indent+"  "))
	case *typedast.TypedApp:
		argsStr := ""
		for i, arg := range e.Args {
			if i > 0 {
				argsStr += "\n"
			}
			argsStr += formatTyped(arg, indent+"  ")
		}
		return fmt.Sprintf("%sApp : %s\n%s\n%s", indent, typeStr,
			formatTyped(e.Func, indent+"  "), argsStr)
	default:
		return fmt.Sprintf("%s%T : %s", indent, e, typeStr)
	}
}

// formatValue formats evaluation result
func formatValue(val interface{}) string {
	switch v := val.(type) {
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case string:
		return v
	case eval.Value:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatType formats types for display
func formatType(t types.Type) string {
	switch typ := t.(type) {
	case *types.TVar:
		return typ.Name
	case *types.TVar2:
		// Handle TVar2 type variables (used during type checking)
		return typ.Name
	case *types.TCon:
		// Normalize type constructor names for display
		return types.NormalizeTypeName(typ)
	case *types.TApp:
		// Check if it's a function type (-> constructor)
		if con, ok := typ.Constructor.(*types.TCon); ok && con.Name == "->" {
			if len(typ.Args) == 2 {
				return fmt.Sprintf("%s → %s", formatType(typ.Args[0]), formatType(typ.Args[1]))
			}
		}
		// Generic application
		args := make([]string, len(typ.Args))
		for i, arg := range typ.Args {
			args[i] = formatType(arg)
		}
		return fmt.Sprintf("%s %s", formatType(typ.Constructor), strings.Join(args, " "))
	case *types.TList:
		return fmt.Sprintf("[%s]", formatType(typ.Element))
	case *types.TRecord:
		// Sort field names for deterministic output
		keys := make([]string, 0, len(typ.Fields))
		for k := range typ.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		fields := make([]string, len(keys))
		for i, k := range keys {
			fields[i] = fmt.Sprintf("%s: %s", k, formatType(typ.Fields[k]))
		}
		return fmt.Sprintf("{%s}", strings.Join(fields, ", "))
	default:
		return fmt.Sprintf("%v", t)
	}
}

// formatConstraint formats constraints for display
func formatConstraint(c types.Constraint) string {
	return fmt.Sprintf("%s %s", c.Class, formatType(c.Type))
}

// isAmbiguous checks if a constraint is ambiguous
func isAmbiguous(c types.Constraint) bool {
	// A constraint is ambiguous if its type variable doesn't appear in the result type
	if _, ok := c.Type.(*types.TVar); ok {
		// In a complete implementation, would check if var appears in the result type
		return true
	}
	return false
}

// constraintToKey converts a constraint to a dictionary key
func constraintToKey(c types.Constraint) string {
	typeStr := formatType(c.Type)
	// Normalize type string for key
	typeStr = strings.ReplaceAll(typeStr, " ", "")
	return fmt.Sprintf("%s[%s]", c.Class, typeStr)
}

// isInPrelude checks if an instance is in the prelude
func isInPrelude(key string) bool {
	preludeInstances := []string{
		"Show[Int]", "Show[Float]", "Show[String]", "Show[Bool]",
		"Read[Int]", "Read[Float]", "Read[String]",
		"Enum[Int]", "Bounded[Int]", "Bounded[Bool]",
	}

	for _, inst := range preludeInstances {
		if inst == key {
			return true
		}
	}
	return false
}

// getFinalTypeAfterDefaulting gets the final type after defaulting has been applied
func (r *REPL) getFinalTypeAfterDefaulting(typedNode typedast.TypedNode, qualType types.Type, resolved map[uint64]*types.ResolvedConstraint) types.Type {
	// Debug: print what we're getting (only if trace is enabled)
	// if r.config.TraceDefaulting {
	// 	// Getting final type after constraint resolution
	// }

	// Strategy: Prefer concrete types over type variables, in this order:
	// 1. Concrete TCon from typedNode.GetType() (if not a TVar)
	// 2. Concrete type from resolved constraints for this node ID
	// 3. Any concrete type from resolved constraints (from defaulting)
	// 4. Fallback to qualType

	// First check if the typed node already has a concrete type
	nodeType := typedNode.GetType()
	if t, ok := nodeType.(types.Type); ok {
		switch typ := t.(type) {
		case *types.TCon:
			// Already concrete - use it
			return typ
		}
	}

	// If we have resolved constraints, look for concrete types
	if len(resolved) > 0 {
		// Check if the root node has a defaulted type
		if rc, ok := resolved[typedNode.GetNodeID()]; ok && rc.Type != nil {
			if con, ok := rc.Type.(*types.TCon); ok {
				return con
			}
		}

		// Look for any resolved constraint with a concrete type from defaulting
		for _, rc := range resolved {
			if rc.Type != nil {
				if con, ok := rc.Type.(*types.TCon); ok {
					// Found a concrete type from defaulting
					return con
				}
			}
		}
	}

	// Fall back to the original qualified type
	return qualType
}

// prettyPrintFinalType formats the final type after defaulting
func (r *REPL) prettyPrintFinalType(typ types.Type, constraints []types.Constraint) string {
	// First normalize the type name
	normalizedType := r.normalizeTypeName(typ)

	// If there are no remaining constraints, just return the type
	remainingConstraints := r.filterResolvedConstraints(constraints, typ)
	if len(remainingConstraints) == 0 {
		return normalizedType
	}

	// Format with remaining constraints
	var parts []string
	for _, c := range remainingConstraints {
		parts = append(parts, formatConstraint(c))
	}
	parts = append(parts, normalizedType)
	return strings.Join(parts, " => ")
}

// normalizeTypeName converts internal type representations to user-friendly names
func (r *REPL) normalizeTypeName(typ types.Type) string {
	switch t := typ.(type) {
	case *types.TCon:
		// Normalize common type constructor names
		switch t.Name {
		case "int":
			return "Int"
		case "float":
			return "Float"
		case "bool":
			return "Bool"
		case "string":
			return "String"
		default:
			return t.Name
		}
	case *types.TVar:
		// Format type variables nicely
		return t.Name
	case *types.TVar2:
		// Check if it was defaulted to a concrete type
		if t.Name == "int" || t.Name == "Int" {
			return "Int"
		} else if t.Name == "float" || t.Name == "Float" {
			return "Float"
		}
		// Otherwise show as a type variable
		return t.Name
	default:
		return formatType(typ)
	}
}

// filterResolvedConstraints removes constraints that have been resolved via defaulting
func (r *REPL) filterResolvedConstraints(constraints []types.Constraint, finalType types.Type) []types.Constraint {
	var remaining []types.Constraint

	// If the final type is concrete, all constraints have been resolved
	switch finalType.(type) {
	case *types.TCon:
		// Concrete type - all constraints resolved
		return remaining
	}

	// Otherwise keep constraints on remaining type variables
	for _, c := range constraints {
		if _, ok := c.Type.(*types.TVar); ok {
			remaining = append(remaining, c)
		} else if _, ok := c.Type.(*types.TVar2); ok {
			remaining = append(remaining, c)
		}
	}

	return remaining
}
