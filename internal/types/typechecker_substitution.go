package types

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/typedast"
)

// ApplySubstEverywhere applies substitution coherently to all relevant data structures
func (tc *CoreTypeChecker) ApplySubstEverywhere(
	sub Substitution,
	monotype Type,
	constraints []ClassConstraint,
	typedNode typedast.TypedNode,
	envEntry interface{},
	bindingName string,
) (Type, []ClassConstraint, typedast.TypedNode, interface{}) {

	// Apply to monotype
	newMonotype := ApplySubstitution(sub, monotype)

	// Apply to constraints
	newConstraints := tc.applySubstitutionToConstraints(sub, constraints)

	// Apply to TypedAST
	newTypedNode := tc.applySubstitutionToTyped(sub, typedNode)

	// Apply to environment entry
	var newEnvEntry interface{}
	if scheme, ok := envEntry.(*Scheme); ok {
		// Apply substitution to the underlying type in the scheme
		newScheme := &Scheme{
			TypeVars:    scheme.TypeVars,
			RowVars:     scheme.RowVars,
			Constraints: scheme.Constraints,
			Type:        ApplySubstitution(sub, scheme.Type),
		}
		newEnvEntry = newScheme
	} else if typ, ok := envEntry.(Type); ok {
		newEnvEntry = ApplySubstitution(sub, typ)
	} else {
		newEnvEntry = envEntry
	}

	// Apply to resolved constraints
	tc.applySubstitutionToResolvedConstraints(sub)

	return newMonotype, newConstraints, newTypedNode, newEnvEntry
}

// applySubstitutionToResolvedConstraints updates the resolved constraints map
func (tc *CoreTypeChecker) applySubstitutionToResolvedConstraints(sub Substitution) {
	for nodeID, rc := range tc.resolvedConstraints {
		rc.Type = ApplySubstitution(sub, rc.Type)
		tc.resolvedConstraints[nodeID] = rc
	}
}

// applySubstitutionToConstraints applies a substitution to class constraints
func (tc *CoreTypeChecker) applySubstitutionToConstraints(sub Substitution, constraints []ClassConstraint) []ClassConstraint {
	result := make([]ClassConstraint, len(constraints))
	for i, c := range constraints {
		result[i] = ClassConstraint{
			Class:  c.Class,
			Type:   c.Type.Substitute(sub),
			Path:   c.Path,
			NodeID: c.NodeID,
		}
	}
	return result
}

// composeSubstitutions composes two substitutions: (S2 ∘ S1)(t) = S2(S1(t))
func composeSubstitutions(s1, s2 Substitution) Substitution {
	result := make(Substitution)

	// Apply s2 to the codomain of s1
	for v, t := range s1 {
		result[v] = ApplySubstitution(s2, t)
	}

	// Add bindings from s2 that aren't in s1
	for v, t := range s2 {
		if _, exists := result[v]; !exists {
			result[v] = t
		}
	}

	return result
}

// applySubstitutionToTyped applies substitution to typed nodes
func (tc *CoreTypeChecker) applySubstitutionToTyped(sub Substitution, node typedast.TypedNode) typedast.TypedNode {
	// Apply substitution to the type in the node
	if typ, ok := node.GetType().(Type); ok {
		substitutedType := ApplySubstitution(sub, typ)

		// We need to update the type in the node
		// Since TypedNode is an interface, we need to handle each concrete type
		switch n := node.(type) {
		case *typedast.TypedLit:
			n.Type = substitutedType
			return n
		case *typedast.TypedVar:
			n.Type = substitutedType
			return n
		case *typedast.TypedLambda:
			n.Type = substitutedType
			// Recursively apply to body
			n.Body = tc.applySubstitutionToTyped(sub, n.Body)
			return n
		case *typedast.TypedLet:
			n.Type = substitutedType
			// Recursively apply to value and body
			n.Value = tc.applySubstitutionToTyped(sub, n.Value)
			n.Body = tc.applySubstitutionToTyped(sub, n.Body)
			return n
		case *typedast.TypedBinOp:
			n.Type = substitutedType
			n.Left = tc.applySubstitutionToTyped(sub, n.Left)
			n.Right = tc.applySubstitutionToTyped(sub, n.Right)
			return n
		case *typedast.TypedApp:
			n.Type = substitutedType
			n.Func = tc.applySubstitutionToTyped(sub, n.Func)
			for i, arg := range n.Args {
				n.Args[i] = tc.applySubstitutionToTyped(sub, arg)
			}
			return n
		// Add more cases as needed
		default:
			// For other types, just return as is (temporary)
			return node
		}
	}
	return node
}

// partitionConstraints separates ground (concrete) from non-ground (polymorphic) constraints
func (tc *CoreTypeChecker) partitionConstraints(constraints []ClassConstraint) (ground, nonGround []ClassConstraint) {
	for _, c := range constraints {
		if isGround(c.Type) {
			ground = append(ground, c)
		} else {
			nonGround = append(nonGround, c)
		}
	}
	return
}

// isGround checks if a type is ground (contains no type variables)
func isGround(t Type) bool {
	switch typ := t.(type) {
	case *TVar:
		return false
	case *TVar2:
		return false
	case *TApp:
		// Check constructor
		if !isGround(typ.Constructor) {
			return false
		}
		// Check all args
		for _, arg := range typ.Args {
			if !isGround(arg) {
				return false
			}
		}
		return true
	case *TFunc:
		for _, p := range typ.Params {
			if !isGround(p) {
				return false
			}
		}
		return isGround(typ.Return)
	case *TRecord:
		for _, fieldType := range typ.Fields {
			if !isGround(fieldType) {
				return false
			}
		}
		return true
	case *Row:
		// Check all label types
		for _, labelType := range typ.Labels {
			if !isGround(labelType) {
				return false
			}
		}
		// If there's a tail variable, it's not ground
		if typ.Tail != nil {
			return false
		}
		return true
	case *RowVar:
		// Row variables are not ground
		return false
	default:
		return true // TCon, TInt, TFloat, TString, TBool, TUnit
	}
}

// resolveGroundConstraints resolves ground class constraints using the instance environment
func (tc *CoreTypeChecker) resolveGroundConstraints(constraints []ClassConstraint, expr core.CoreExpr) error {
	for _, c := range constraints {
		// CRITICAL: Assert that constraint type is ground before resolution
		if !isGround(c.Type) {
			return fmt.Errorf("INTERNAL ERROR: attempting to resolve non-ground constraint %s[%s] - defaulting failed to make this ground", c.Class, c.Type)
		}

		// Look up instance in the environment
		_, err := tc.instanceEnv.Lookup(c.Class, c.Type)
		if err != nil {
			// No instance found - return error with hint
			if missingErr, ok := err.(*MissingInstanceError); ok {
				return fmt.Errorf("at %s: %v", c.Path[0], missingErr)
			}
			return err
		}

		// Instance found - record the resolved constraint if it has a NodeID
		if c.NodeID != 0 {
			// CRITICAL: Double-check that the type we're recording is ground
			if !isGround(c.Type) {
				return fmt.Errorf("INTERNAL ERROR: storing non-ground type %s in ResolvedConstraints for node %d", c.Type, c.NodeID)
			}

			// We need to determine the method based on the node
			// This will be done when we scan the Core AST
			// Create normalized type for dictionary lookup consistency
			normalizedType := &TCon{Name: NormalizeTypeName(c.Type)}
			// fmt.Printf("DEBUG RESOLVE: NodeID=%d, Class=%s, OrigType=%v, NormType=%s\n",
			// 	c.NodeID, c.Class, c.Type, normalizedType.Name)
			tc.resolvedConstraints[c.NodeID] = &ResolvedConstraint{
				NodeID:    c.NodeID,
				ClassName: c.Class,
				Type:      normalizedType, // Normalized type (float→Float, int→Int)
				Method:    "",             // Will be filled in during Core traversal
			}
		}
	}
	return nil
}
