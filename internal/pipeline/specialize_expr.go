package pipeline

import (
	"fmt"
	"os"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// specializeExpr recursively specializes an expression
// env maps variable names to their types (for tracking polymorphic bindings)
// bindings maps variable names to their Core expressions (for Var→Lam resolution)
func (s *Specializer) specializeExpr(expr core.CoreExpr, env map[string]types.Type, bindings map[string]core.CoreExpr) (core.CoreExpr, error) {
	switch e := expr.(type) {
	case *core.Let:
		// Specialize the value
		newValue, err := s.specializeExpr(e.Value, env, bindings)
		if err != nil {
			return nil, err
		}

		// Add binding to environment and bindings map
		newEnv := copyEnv(env)
		if typ, ok := s.CoreTI.Get(e.Value.ID()); ok {
			newEnv[e.Name] = typ
		}

		newBindings := copyBindings(bindings)
		newBindings[e.Name] = newValue // Map variable to its value for resolution

		// Specialize the body
		newBody, err := s.specializeExpr(e.Body, newEnv, newBindings)
		if err != nil {
			return nil, err
		}

		return &core.Let{
			CoreNode: e.CoreNode,
			Name:     e.Name,
			Value:    newValue,
			Body:     newBody,
		}, nil

	case *core.LetRec:
		// Skip recursive bindings (v0.4.0 policy)
		if isMutuallyRecursive(e.Bindings) {
			s.Skipped = append(s.Skipped, SkipReason{
				DefSym:   "(letrec group)",
				Reason:   "Mutually recursive bindings not specialized in v0.4.0",
				Location: e.OriginalSpan().String(),
			})

			// Still need to specialize the body, but skip the bindings
			newBody, err := s.specializeExpr(e.Body, env, bindings)
			if err != nil {
				return nil, err
			}

			return &core.LetRec{
				CoreNode: e.CoreNode,
				Bindings: e.Bindings, // Keep original bindings
				Body:     newBody,
			}, nil
		}

		// For non-recursive letrec, specialize each binding
		newBindings := make([]core.RecBinding, len(e.Bindings))
		newEnv := copyEnv(env)
		newBindingsMap := copyBindings(bindings)

		for i, binding := range e.Bindings {
			// Check if binding is self-recursive
			if isRecursive(binding.Value, binding.Name) {
				s.Skipped = append(s.Skipped, SkipReason{
					DefSym:   binding.Name,
					Reason:   "Recursive function not specialized in v0.4.0",
					Location: e.OriginalSpan().String(),
				})
				newBindings[i] = binding // Keep original
			} else {
				specialized, err := s.specializeExpr(binding.Value, newEnv, newBindingsMap)
				if err != nil {
					return nil, err
				}
				newBindings[i] = core.RecBinding{
					Name:  binding.Name,
					Value: specialized,
				}
			}

			// Add to environment and bindings map for subsequent bindings
			if typ, ok := s.CoreTI.Get(binding.Value.ID()); ok {
				newEnv[binding.Name] = typ
			}
			newBindingsMap[binding.Name] = binding.Value
		}

		newBody, err := s.specializeExpr(e.Body, newEnv, newBindingsMap)
		if err != nil {
			return nil, err
		}

		return &core.LetRec{
			CoreNode: e.CoreNode,
			Bindings: newBindings,
			Body:     newBody,
		}, nil

	case *core.Lambda:
		// Specialize lambda body
		newEnv := copyEnv(env)
		newBindings := copyBindings(bindings)
		// Note: We don't know parameter types here without more context
		// Lambda specialization happens when applied

		newBody, err := s.specializeExpr(e.Body, newEnv, newBindings)
		if err != nil {
			return nil, err
		}

		return &core.Lambda{
			CoreNode: e.CoreNode,
			Params:   e.Params,
			Body:     newBody,
		}, nil

	case *core.App:
		// This is the key case: function application
		// Check if this is a call to a polymorphic function with concrete arguments

		// First, specialize arguments (bottom-up)
		newArgs := make([]core.CoreExpr, len(e.Args))
		argTypes := make([]types.Type, len(e.Args))
		for i, arg := range e.Args {
			var err error
			newArgs[i], err = s.specializeExpr(arg, env, bindings)
			if err != nil {
				return nil, err
			}
			// Get argument type
			if typ, ok := s.CoreTI.Get(arg.ID()); ok {
				argTypes[i] = typ
			}
		}

		// Resolve the callee to find the underlying lambda
		// This handles both inline lambdas and Var-bound lambdas
		var lambda *core.Lambda
		var lambdaID uint64

		if lam, ok := e.Func.(*core.Lambda); ok {
			// Direct lambda application
			lambda = lam
			lambdaID = lam.ID()
		} else if v, ok := e.Func.(*core.Var); ok {
			// Var-bound lambda - resolve through bindings
			resolved := core.ResolveValue(v, bindings)
			if lam, ok := resolved.(*core.Lambda); ok {
				lambda = lam
				lambdaID = lam.ID()
			}
		}

		// If we found a lambda (either inline or via Var resolution), try to specialize it
		if lambda != nil {
			// Check if lambda has polymorphic type and all arguments are concrete
			if funcType, ok := s.CoreTI.Get(lambdaID); ok {
				// Debug logging
				if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] Found lambda, type=%s (type=%T), isPoly=%v, allConc=%v\n",
						funcType, funcType, isPolymorphic(funcType), allConcrete(argTypes))
					fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]   argTypes: %v (len=%d)\n", argTypes, len(argTypes))
					if fn, ok := funcType.(*types.TFunc2); ok {
						fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]   Params: %v (len=%d)\n", fn.Params, len(fn.Params))
						fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]   Return: %s (type=%T)\n", fn.Return, fn.Return)
					}
				}
				if isPolymorphic(funcType) && allConcrete(argTypes) {
					// Attempt to specialize this lambda
					if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
						fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] Calling specializeLambda with argTypes=%v\n", argTypes)
					}
					specialized, err := s.specializeLambda(lambda, argTypes, env)
					if err != nil {
						if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
							fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] specializeLambda returned error: %v\n", err)
						}
						return nil, err
					}
					if specialized != nil {
						if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
							fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] Got specialized lambda, returning specialized App\n")
						}
						// Return application with specialized lambda
						return &core.App{
							CoreNode: e.CoreNode,
							Func:     specialized,
							Args:     newArgs,
						}, nil
					} else {
						if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
							fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] specializeLambda returned nil (skipped)\n")
						}
					}
				}
			}
		}

		// Otherwise, just specialize the function normally
		newFunc, err := s.specializeExpr(e.Func, env, bindings)
		if err != nil {
			return nil, err
		}

		return &core.App{
			CoreNode: e.CoreNode,
			Func:     newFunc,
			Args:     newArgs,
		}, nil

	case *core.If:
		newCond, err := s.specializeExpr(e.Cond, env, bindings)
		if err != nil {
			return nil, err
		}
		newThen, err := s.specializeExpr(e.Then, env, bindings)
		if err != nil {
			return nil, err
		}
		newElse, err := s.specializeExpr(e.Else, env, bindings)
		if err != nil {
			return nil, err
		}

		return &core.If{
			CoreNode: e.CoreNode,
			Cond:     newCond,
			Then:     newThen,
			Else:     newElse,
		}, nil

	case *core.Match:
		newScrutinee, err := s.specializeExpr(e.Scrutinee, env, bindings)
		if err != nil {
			return nil, err
		}

		newArms := make([]core.MatchArm, len(e.Arms))
		for i, arm := range e.Arms {
			newBody, err := s.specializeExpr(arm.Body, env, bindings)
			if err != nil {
				return nil, err
			}
			newArms[i] = core.MatchArm{
				Pattern: arm.Pattern,
				Guard:   arm.Guard,
				Body:    newBody,
			}
		}

		return &core.Match{
			CoreNode:   e.CoreNode,
			Scrutinee:  newScrutinee,
			Arms:       newArms,
			Exhaustive: e.Exhaustive,
		}, nil

	case *core.BinOp:
		newLeft, err := s.specializeExpr(e.Left, env, bindings)
		if err != nil {
			return nil, err
		}
		newRight, err := s.specializeExpr(e.Right, env, bindings)
		if err != nil {
			return nil, err
		}

		return &core.BinOp{
			CoreNode: e.CoreNode,
			Op:       e.Op,
			Left:     newLeft,
			Right:    newRight,
		}, nil

	case *core.UnOp:
		newOperand, err := s.specializeExpr(e.Operand, env, bindings)
		if err != nil {
			return nil, err
		}

		return &core.UnOp{
			CoreNode: e.CoreNode,
			Op:       e.Op,
			Operand:  newOperand,
		}, nil

	case *core.DictApp:
		// Dictionary applications (elaborated operators)
		newDict, err := s.specializeExpr(e.Dict, env, bindings)
		if err != nil {
			return nil, err
		}

		newArgs := make([]core.CoreExpr, len(e.Args))
		for i, arg := range e.Args {
			newArgs[i], err = s.specializeExpr(arg, env, bindings)
			if err != nil {
				return nil, err
			}
		}

		return &core.DictApp{
			CoreNode: e.CoreNode,
			Dict:     newDict,
			Method:   e.Method,
			Args:     newArgs,
		}, nil

	// Atomic expressions - no specialization needed
	case *core.Var, *core.VarGlobal, *core.Lit, *core.Intrinsic:
		return expr, nil

	// TODO: Handle other expression types (Record, List, Tuple, etc.)
	default:
		// In DEBUG_STRICT mode, panic to force developer to add case
		if os.Getenv("DEBUG_STRICT") != "" {
			panic(fmt.Sprintf("specializeExpr: unhandled node type %T (NodeID %d). "+
				"Add a case for this type or explicitly mark as unsupported. "+
				"This error only appears when DEBUG_STRICT=1 is set.",
				expr, expr.ID()))
		}

		// For now, return expression as-is
		return expr, nil
	}
}
