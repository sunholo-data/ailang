package pipeline

import (
	"fmt"
	"os"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// cloneExpr recursively clones an expression with fresh node IDs
// and applies type substitution to all types in CoreTypeInfo
func (s *Specializer) cloneExpr(expr core.CoreExpr, typeSubst map[string]types.Type) (core.CoreExpr, error) {
	if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] cloneExpr: type=%T, NodeID=%d\n", expr, expr.ID())
	}
	switch e := expr.(type) {
	case *core.Var:
		cloned := &core.Var{
			CoreNode: core.CoreNode{
				NodeID:   s.freshNodeID(),
				CoreSpan: e.CoreSpan,
				OrigSpan: e.OrigSpan,
			},
			Name: e.Name,
		}
		// Apply type substitution
		if typ, ok := s.CoreTI.Get(e.ID()); ok {
			s.CoreTI.Set(cloned.ID(), substituteType(typ, typeSubst))
		}
		return cloned, nil

	case *core.Lit:
		cloned := &core.Lit{
			CoreNode: core.CoreNode{
				NodeID:   s.freshNodeID(),
				CoreSpan: e.CoreSpan,
				OrigSpan: e.OrigSpan,
			},
			Kind:  e.Kind,
			Value: e.Value,
		}
		if typ, ok := s.CoreTI.Get(e.ID()); ok {
			s.CoreTI.Set(cloned.ID(), substituteType(typ, typeSubst))
		}
		return cloned, nil

	case *core.Lambda:
		clonedBody, err := s.cloneExpr(e.Body, typeSubst)
		if err != nil {
			return nil, err
		}
		cloned := &core.Lambda{
			CoreNode: core.CoreNode{
				NodeID:   s.freshNodeID(),
				CoreSpan: e.CoreSpan,
				OrigSpan: e.OrigSpan,
			},
			Params: e.Params,
			Body:   clonedBody,
		}
		if typ, ok := s.CoreTI.Get(e.ID()); ok {
			s.CoreTI.Set(cloned.ID(), substituteType(typ, typeSubst))
		}
		return cloned, nil

	case *core.App:
		clonedFunc, err := s.cloneExpr(e.Func, typeSubst)
		if err != nil {
			return nil, err
		}
		clonedArgs := make([]core.CoreExpr, len(e.Args))
		for i, arg := range e.Args {
			clonedArgs[i], err = s.cloneExpr(arg, typeSubst)
			if err != nil {
				return nil, err
			}
		}
		cloned := &core.App{
			CoreNode: core.CoreNode{
				NodeID:   s.freshNodeID(),
				CoreSpan: e.CoreSpan,
				OrigSpan: e.OrigSpan,
			},
			Func: clonedFunc,
			Args: clonedArgs,
		}
		if typ, ok := s.CoreTI.Get(e.ID()); ok {
			s.CoreTI.Set(cloned.ID(), substituteType(typ, typeSubst))
		}
		return cloned, nil

	case *core.If:
		clonedCond, err := s.cloneExpr(e.Cond, typeSubst)
		if err != nil {
			return nil, err
		}
		clonedThen, err := s.cloneExpr(e.Then, typeSubst)
		if err != nil {
			return nil, err
		}
		clonedElse, err := s.cloneExpr(e.Else, typeSubst)
		if err != nil {
			return nil, err
		}
		cloned := &core.If{
			CoreNode: core.CoreNode{
				NodeID:   s.freshNodeID(),
				CoreSpan: e.CoreSpan,
				OrigSpan: e.OrigSpan,
			},
			Cond: clonedCond,
			Then: clonedThen,
			Else: clonedElse,
		}
		if typ, ok := s.CoreTI.Get(e.ID()); ok {
			s.CoreTI.Set(cloned.ID(), substituteType(typ, typeSubst))
		}
		return cloned, nil

	case *core.BinOp:
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] Cloning BinOp: %s (NodeID %d)\n", e.Op, e.ID())
		}
		clonedLeft, err := s.cloneExpr(e.Left, typeSubst)
		if err != nil {
			return nil, err
		}
		clonedRight, err := s.cloneExpr(e.Right, typeSubst)
		if err != nil {
			return nil, err
		}
		cloned := &core.BinOp{
			CoreNode: core.CoreNode{
				NodeID:   s.freshNodeID(),
				CoreSpan: e.CoreSpan,
				OrigSpan: e.OrigSpan,
			},
			Op:    e.Op,
			Left:  clonedLeft,
			Right: clonedRight,
		}
		if typ, ok := s.CoreTI.Get(e.ID()); ok {
			s.CoreTI.Set(cloned.ID(), substituteType(typ, typeSubst))
		}
		return cloned, nil

	case *core.Intrinsic:
		// Clone arguments recursively
		clonedArgs := make([]core.CoreExpr, len(e.Args))
		for i, arg := range e.Args {
			clonedArg, err := s.cloneExpr(arg, typeSubst)
			if err != nil {
				return nil, err
			}
			clonedArgs[i] = clonedArg
		}

		cloned := &core.Intrinsic{
			CoreNode: core.CoreNode{
				NodeID:   s.freshNodeID(),
				CoreSpan: e.CoreSpan,
				OrigSpan: e.OrigSpan,
			},
			Op:   e.Op,
			Args: clonedArgs,
		}
		if typ, ok := s.CoreTI.Get(e.ID()); ok {
			s.CoreTI.Set(cloned.ID(), substituteType(typ, typeSubst))
		}
		return cloned, nil

	case *core.DictApp:
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] Cloning DictApp: method=%s, NodeID=%d\n", e.Method, e.ID())
			if dictRef, ok := e.Dict.(*core.DictRef); ok {
				fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]   Original DictRef: class=%s, type=%s, NodeID=%d\n",
					dictRef.ClassName, dictRef.TypeName, dictRef.ID())
			}
		}
		// Clone dictionary reference
		clonedDict, err := s.cloneExpr(e.Dict, typeSubst)
		if err != nil {
			return nil, err
		}
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			if dictRef, ok := clonedDict.(*core.DictRef); ok {
				fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]   Cloned DictRef: class=%s, type=%s, NodeID=%d\n",
					dictRef.ClassName, dictRef.TypeName, dictRef.ID())
			}
		}

		// Clone arguments
		clonedArgs := make([]core.CoreExpr, len(e.Args))
		for i, arg := range e.Args {
			clonedArgs[i], err = s.cloneExpr(arg, typeSubst)
			if err != nil {
				return nil, err
			}
		}

		cloned := &core.DictApp{
			CoreNode: core.CoreNode{
				NodeID:   s.freshNodeID(),
				CoreSpan: e.CoreSpan,
				OrigSpan: e.OrigSpan,
			},
			Dict:   clonedDict,
			Method: e.Method,
			Args:   clonedArgs,
		}
		if typ, ok := s.CoreTI.Get(e.ID()); ok {
			s.CoreTI.Set(cloned.ID(), substituteType(typ, typeSubst))
		}
		return cloned, nil

	case *core.DictRef:
		// DictRef needs special handling - update the TypeName based on type substitution
		// The TypeName field determines which type class instance to use (e.g., "Int" vs "Float")

		// Get the original type from CoreTypeInfo
		var newTypeName string
		if typ, ok := s.CoreTI.Get(e.ID()); ok {
			// Apply substitution to the type
			substitutedType := substituteType(typ, typeSubst)
			// Normalize the substituted type to get the new TypeName
			newTypeName = types.NormalizeTypeName(substitutedType)
			if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] DictRef: oldTypeName=%s, newTypeName=%s, class=%s\n",
					e.TypeName, newTypeName, e.ClassName)
			}
		} else {
			// Fallback: keep original TypeName
			newTypeName = e.TypeName
			if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] DictRef: no type info, keeping oldTypeName=%s\n", e.TypeName)
			}
		}

		cloned := &core.DictRef{
			CoreNode: core.CoreNode{
				NodeID:   s.freshNodeID(),
				CoreSpan: e.CoreSpan,
				OrigSpan: e.OrigSpan,
			},
			ClassName: e.ClassName,
			TypeName:  newTypeName, // Updated TypeName based on substitution
		}
		if typ, ok := s.CoreTI.Get(e.ID()); ok {
			s.CoreTI.Set(cloned.ID(), substituteType(typ, typeSubst))
		}
		return cloned, nil

	case *core.Let:
		// Clone Let value and body
		clonedValue, err := s.cloneExpr(e.Value, typeSubst)
		if err != nil {
			return nil, err
		}
		clonedBody, err := s.cloneExpr(e.Body, typeSubst)
		if err != nil {
			return nil, err
		}

		cloned := &core.Let{
			CoreNode: core.CoreNode{
				NodeID:   s.freshNodeID(),
				CoreSpan: e.CoreSpan,
				OrigSpan: e.OrigSpan,
			},
			Name:  e.Name,
			Value: clonedValue,
			Body:  clonedBody,
		}
		if typ, ok := s.CoreTI.Get(e.ID()); ok {
			s.CoreTI.Set(cloned.ID(), substituteType(typ, typeSubst))
		}
		return cloned, nil

	// For other expression types, return as-is for now (v0.4.0 simplification)
	default:
		// In DEBUG_STRICT mode, panic to force developer to add case
		if os.Getenv("DEBUG_STRICT") != "" {
			panic(fmt.Sprintf("cloneExpr: unhandled node type %T (NodeID %d). "+
				"Add a case for this type or explicitly mark as unsupported. "+
				"This error only appears when DEBUG_STRICT=1 is set.",
				expr, expr.ID()))
		}

		// In verbose mode, log warning
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] cloneExpr: default case for type %T\n", expr)
		}
		return expr, nil
	}
}
