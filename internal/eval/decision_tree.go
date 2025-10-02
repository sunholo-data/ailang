package eval

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/dtree"
)

// evalDecisionTree evaluates a match using a pre-compiled decision tree
func (e *CoreEvaluator) evalDecisionTree(scrutineeVal Value, tree dtree.DecisionTree, arms []core.MatchArm) (Value, error) {
	return e.walkTree(scrutineeVal, tree, arms, make(map[string]Value))
}

// walkTree walks the decision tree with the scrutinee value
func (e *CoreEvaluator) walkTree(scrutinee Value, tree dtree.DecisionTree, arms []core.MatchArm, bindings map[string]Value) (Value, error) {
	switch node := tree.(type) {
	case *dtree.LeafNode:
		// Reached a leaf - check guard and execute body
		arm := arms[node.ArmIndex]

		// Match the pattern to collect bindings
		patBindings, matched := matchPattern(arm.Pattern, scrutinee)
		if !matched {
			return nil, fmt.Errorf("internal error: leaf pattern didn't match (arm %d)", node.ArmIndex)
		}

		// Merge bindings
		for k, v := range patBindings {
			bindings[k] = v
		}

		// Check guard if present
		if node.Guard != nil {
			newEnv := e.env.NewChildEnvironment()
			for name, val := range bindings {
				newEnv.Set(name, val)
			}

			oldEnv := e.env
			e.env = newEnv
			guardVal, err := e.evalCore(node.Guard)
			e.env = oldEnv

			if err != nil {
				return nil, fmt.Errorf("guard evaluation failed: %w", err)
			}

			boolVal, ok := guardVal.(*BoolValue)
			if !ok {
				return nil, fmt.Errorf("guard must evaluate to Bool, got %T", guardVal)
			}

			if !boolVal.Value {
				return nil, fmt.Errorf("internal error: guard failed in decision tree")
			}
		}

		// Execute body with bindings
		newEnv := e.env.NewChildEnvironment()
		for name, val := range bindings {
			newEnv.Set(name, val)
		}

		oldEnv := e.env
		e.env = newEnv
		result, err := e.evalCore(node.Body)
		e.env = oldEnv

		return result, err

	case *dtree.SwitchNode:
		// Get the discriminator value at the specified path
		discValue, err := e.getValueAtPath(scrutinee, node.Path)
		if err != nil {
			return nil, err
		}

		// Try to find a matching case
		var key interface{}
		switch v := discValue.(type) {
		case *BoolValue:
			key = v.Value
		case *IntValue:
			key = v.Value
		case *FloatValue:
			key = v.Value
		case *StringValue:
			key = v.Value
		case *TaggedValue:
			key = v.CtorName
		default:
			// For other types, use default
			if node.Default != nil {
				return e.walkTree(scrutinee, node.Default, arms, bindings)
			}
			return nil, fmt.Errorf("no matching case in switch node")
		}

		// Look up the case
		if subtree, ok := node.Cases[key]; ok {
			return e.walkTree(scrutinee, subtree, arms, bindings)
		}

		// Fall back to default
		if node.Default != nil {
			return e.walkTree(scrutinee, node.Default, arms, bindings)
		}

		return nil, fmt.Errorf("no matching case in switch node for key: %v", key)

	case *dtree.FailNode:
		// Reached a fail node - non-exhaustive match
		return nil, fmt.Errorf("no pattern matched in match expression")

	default:
		return nil, fmt.Errorf("unknown decision tree node type: %T", node)
	}
}

// getValueAtPath extracts a value at the specified path
// Path is a sequence of indices into nested structures
// For now, we only support path [0] which is the scrutinee itself
func (e *CoreEvaluator) getValueAtPath(value Value, path []int) (Value, error) {
	if len(path) == 0 || (len(path) == 1 && path[0] == 0) {
		return value, nil
	}

	// For nested paths, we'd need to descend into tuples/records/constructors
	// For now, simple implementation
	current := value
	for i, index := range path {
		if i == 0 && index == 0 {
			continue // Skip root
		}

		// Try to descend into the structure
		switch v := current.(type) {
		case *TupleValue:
			if index >= len(v.Elements) {
				return nil, fmt.Errorf("tuple index out of bounds: %d >= %d", index, len(v.Elements))
			}
			current = v.Elements[index]

		case *TaggedValue:
			if index >= len(v.Fields) {
				return nil, fmt.Errorf("constructor arg index out of bounds: %d >= %d", index, len(v.Fields))
			}
			current = v.Fields[index]

		default:
			return nil, fmt.Errorf("cannot index into value of type %T at path index %d", v, i)
		}
	}

	return current, nil
}
