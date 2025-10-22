// Package pipeline provides monomorphization for polymorphic functions
package pipeline

import (
	"crypto/sha256"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// SpecializationLimits defines resource limits for monomorphization
type SpecializationLimits struct {
	MaxPerFunction int // Maximum specializations per function (default: 16)
	MaxPerModule   int // Maximum specializations per module (default: 512)
}

// DefaultSpecializationLimits returns conservative default limits
func DefaultSpecializationLimits() SpecializationLimits {
	return SpecializationLimits{
		MaxPerFunction: 16,
		MaxPerModule:   512,
	}
}

// SpecializationKey uniquely identifies a specialized function instance
type SpecializationKey struct {
	DefSym           string // Original function symbol
	TypesFingerprint string // Canonical fingerprint of argument types
}

// String returns a human-readable representation
func (k SpecializationKey) String() string {
	return fmt.Sprintf("%s[%s]", k.DefSym, k.TypesFingerprint)
}

// Specializer performs call-site monomorphization of polymorphic functions
type Specializer struct {
	CoreTI      *types.CoreTypeInfo                 // Type information for Core nodes
	Cache       map[SpecializationKey]core.CoreExpr // Memoization cache
	PerFunction map[string]int                      // Count specializations per function
	TotalCount  int                                 // Total specialization count
	Limits      SpecializationLimits                // Resource limits
	Skipped     []SkipReason                        // Functions skipped (for diagnostics)
	nextNodeID  uint64                              // Counter for fresh node IDs
	CacheHits   int                                 // Number of cache hits
	CacheMisses int                                 // Number of cache misses
}

// SkipReason describes why a function was not specialized
type SkipReason struct {
	DefSym   string // Function symbol
	Reason   string // Human-readable reason
	Location string // Source location (if available)
}

// NewSpecializer creates a new monomorphization pass
func NewSpecializer(coreTI *types.CoreTypeInfo) *Specializer {
	return &Specializer{
		CoreTI:      coreTI,
		Cache:       make(map[SpecializationKey]core.CoreExpr),
		PerFunction: make(map[string]int),
		TotalCount:  0,
		Limits:      DefaultSpecializationLimits(),
		Skipped:     make([]SkipReason, 0),
		nextNodeID:  1000000, // Start high to avoid conflicts with existing IDs
	}
}

// freshNodeID generates a fresh node ID for cloned expressions
func (s *Specializer) freshNodeID() uint64 {
	id := s.nextNodeID
	s.nextNodeID++
	return id
}

// canonicalTypeFingerprint produces a stable, normalized string representation of types
//
// Requirements:
// - Fully qualified type constructor names
// - Normalized ordering for row/effect sets (deterministic iteration)
// - Type variables removed (only monotypes supported)
// - Hash suffix to prevent collisions
func canonicalTypeFingerprint(typs []types.Type) string {
	if len(typs) == 0 {
		return "unit"
	}

	// Build normalized string representation
	var parts []string
	for _, t := range typs {
		parts = append(parts, typeToCanonicalString(t))
	}

	// Sort for determinism (in case types come in different orders)
	sort.Strings(parts)

	// Join with stable separator
	canonical := strings.Join(parts, ":")

	// Add hash suffix for collision resistance
	hash := sha256.Sum256([]byte(canonical))
	shortHash := fmt.Sprintf("%x", hash[:2]) // First 2 bytes = 4 hex chars

	return canonical + "$" + shortHash
}

// typeToCanonicalString converts a type to its canonical string representation
func typeToCanonicalString(t types.Type) string {
	switch typ := t.(type) {
	case *types.TCon:
		return typ.Name
	case *types.TVar:
		// Type variables should not appear in monomorphized code
		// This indicates the type is still polymorphic
		return fmt.Sprintf("α_%s", typ.Name)
	case *types.TApp:
		// Application: Constructor(arg1, arg2, ...)
		args := make([]string, len(typ.Args))
		for i, arg := range typ.Args {
			args[i] = typeToCanonicalString(arg)
		}
		// Constructor is a Type, need to extract its name
		conName := "Unknown"
		if con, ok := typ.Constructor.(*types.TCon); ok {
			conName = con.Name
		}
		return fmt.Sprintf("%s(%s)", conName, strings.Join(args, ","))
	case *types.TFunc2:
		// Function: param1 -> param2 -> ... -> ret ! effects
		params := make([]string, len(typ.Params))
		for i, p := range typ.Params {
			params[i] = typeToCanonicalString(p)
		}
		ret := typeToCanonicalString(typ.Return)

		// Include effects if present
		if typ.EffectRow != nil && len(typ.EffectRow.Labels) > 0 {
			// Sort effect labels for determinism
			var effects []string
			for eff := range typ.EffectRow.Labels {
				effects = append(effects, eff)
			}
			sort.Strings(effects)
			return fmt.Sprintf("(%s -> %s ! {%s})", strings.Join(params, " -> "), ret, strings.Join(effects, ","))
		}

		return fmt.Sprintf("(%s -> %s)", strings.Join(params, " -> "), ret)
	case *types.TRecord:
		// Record: {field1: type1, field2: type2, ...}
		if len(typ.Fields) == 0 {
			return "{}"
		}

		// Sort fields for determinism
		var fields []string
		for field, ftyp := range typ.Fields {
			fields = append(fields, fmt.Sprintf("%s:%s", field, typeToCanonicalString(ftyp)))
		}
		sort.Strings(fields)

		return fmt.Sprintf("{%s}", strings.Join(fields, ","))
	default:
		// Fallback for unknown types
		return fmt.Sprintf("%T", t)
	}
}

// typeHeads extracts the head type constructors from a list of types
//
// Example:
//
//	[Int, List(Float), Result(String, Error)] → [Int, List, Result]
//
// This is used for generating readable specialized function names.
func typeHeads(typs []types.Type) []string {
	heads := make([]string, len(typs))
	for i, t := range typs {
		heads[i] = typeHead(t)
	}
	return heads
}

// typeHead extracts the outermost type constructor
func typeHead(t types.Type) string {
	switch typ := t.(type) {
	case *types.TCon:
		return typ.Name
	case *types.TApp:
		// Extract constructor name
		if con, ok := typ.Constructor.(*types.TCon); ok {
			return con.Name
		}
		return "App"
	case *types.TFunc2:
		return "Func"
	case *types.TRecord:
		return "Record"
	case *types.TVar:
		return fmt.Sprintf("α_%s", typ.Name)
	default:
		return "Unknown"
	}
}

// generateSpecializedName creates a deterministic, readable name for a specialized function
//
// Format: _originalName$Type1$Type2$...$hash
//
// Example:
//
//	max specialized for (Int, Int) → _max$Int$Int$2f
//	sort specialized for (List(Int)) → _sort$List$8a
func generateSpecializedName(defSym string, argTypes []types.Type, fingerprint string) string {
	heads := typeHeads(argTypes)

	// Extract hash suffix from fingerprint
	parts := strings.Split(fingerprint, "$")
	hashSuffix := ""
	if len(parts) > 0 {
		hashSuffix = parts[len(parts)-1]
	}

	// Build name: _sym$Type1$Type2$hash
	name := "_" + defSym
	for _, head := range heads {
		name += "$" + head
	}
	if hashSuffix != "" {
		name += "$" + hashSuffix
	}

	return name
}

// isPolymorphic checks if a type contains type variables
func isPolymorphic(t types.Type) bool {
	if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]   isPolymorphic(%s) type=%T\n", t, t)
	}
	switch typ := t.(type) {
	case *types.TVar:
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     -> TVar, returning true\n")
		}
		return true
	case *types.TVar2:
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     -> TVar2, returning true\n")
		}
		return true
	case *types.TApp:
		for i, arg := range typ.Args {
			if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     TApp arg[%d]: %s\n", i, arg)
			}
			if isPolymorphic(arg) {
				return true
			}
		}
		return false
	case *types.TFunc2:
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     TFunc2 with %d params\n", len(typ.Params))
		}
		for i, p := range typ.Params {
			if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     Checking param[%d]: %s (type=%T)\n", i, p, p)
			}
			if isPolymorphic(p) {
				if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     -> param[%d] is polymorphic, returning true\n", i)
				}
				return true
			}
		}
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     Checking return: %s\n", typ.Return)
		}
		if isPolymorphic(typ.Return) {
			return true
		}
		// Check effect row
		if typ.EffectRow != nil {
			for _, eff := range typ.EffectRow.Labels {
				if isPolymorphic(eff) {
					return true
				}
			}
		}
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE]     -> TFunc2 not polymorphic, returning false\n")
		}
		return false
	case *types.TRecord:
		for _, ftyp := range typ.Fields {
			if isPolymorphic(ftyp) {
				return true
			}
		}
		// Also check row variable
		if typ.Row != nil && isPolymorphic(typ.Row) {
			return true
		}
		return false
	default:
		return false
	}
}

// allConcrete checks if all types in a list are concrete (no type variables)
func allConcrete(typs []types.Type) bool {
	for _, t := range typs {
		if isPolymorphic(t) {
			return false
		}
	}
	return true
}

// Statistics returns diagnostic information about specialization
type SpecializationStats struct {
	TotalSpecializations int
	PerFunction          map[string]int
	SkippedFunctions     []SkipReason
	CacheHits            int
	CacheMisses          int
}

// GetStats returns specialization statistics for debugging
func (s *Specializer) GetStats() SpecializationStats {
	return SpecializationStats{
		TotalSpecializations: s.TotalCount,
		PerFunction:          s.PerFunction,
		SkippedFunctions:     s.Skipped,
		CacheHits:            s.CacheHits,
		CacheMisses:          s.CacheMisses,
	}
}

// isRecursive checks if an expression contains a self-reference to the given name
func isRecursive(expr core.CoreExpr, name string) bool {
	switch e := expr.(type) {
	case *core.Var:
		return e.Name == name
	case *core.Lambda:
		// Check if name is shadowed by a parameter
		for _, param := range e.Params {
			if param == name {
				return false // Name is shadowed, not recursive
			}
		}
		return isRecursive(e.Body, name)
	case *core.Let:
		// Check value
		if isRecursive(e.Value, name) {
			return true
		}
		// Check body (name might be shadowed)
		if e.Name == name {
			return false // Shadowed
		}
		return isRecursive(e.Body, name)
	case *core.LetRec:
		// Check all bindings
		for _, binding := range e.Bindings {
			if isRecursive(binding.Value, name) {
				return true
			}
		}
		return isRecursive(e.Body, name)
	case *core.App:
		if isRecursive(e.Func, name) {
			return true
		}
		for _, arg := range e.Args {
			if isRecursive(arg, name) {
				return true
			}
		}
		return false
	case *core.If:
		return isRecursive(e.Cond, name) ||
			isRecursive(e.Then, name) ||
			isRecursive(e.Else, name)
	case *core.Match:
		if isRecursive(e.Scrutinee, name) {
			return true
		}
		for _, arm := range e.Arms {
			// Check if name is bound in pattern (shadowing)
			boundVars := patternBoundVars(arm.Pattern)
			shadowed := false
			for _, v := range boundVars {
				if v == name {
					shadowed = true
					break
				}
			}
			if !shadowed && isRecursive(arm.Body, name) {
				return true
			}
		}
		return false
	case *core.BinOp:
		return isRecursive(e.Left, name) || isRecursive(e.Right, name)
	case *core.UnOp:
		return isRecursive(e.Operand, name)
	case *core.RecordAccess:
		return isRecursive(e.Record, name)
	case *core.RecordUpdate:
		if isRecursive(e.Base, name) {
			return true
		}
		for _, val := range e.Updates {
			if isRecursive(val, name) {
				return true
			}
		}
		return false
	case *core.Lit, *core.VarGlobal, *core.Intrinsic:
		return false
	default:
		// For unknown expression types, conservatively assume non-recursive
		return false
	}
}

// patternBoundVars extracts all variables bound by a pattern
func patternBoundVars(pat core.CorePattern) []string {
	switch p := pat.(type) {
	case *core.VarPattern:
		return []string{p.Name}
	case *core.ConstructorPattern:
		var vars []string
		for _, arg := range p.Args {
			vars = append(vars, patternBoundVars(arg)...)
		}
		return vars
	case *core.ListPattern:
		var vars []string
		for _, elem := range p.Elements {
			vars = append(vars, patternBoundVars(elem)...)
		}
		if p.Tail != nil {
			vars = append(vars, patternBoundVars(*p.Tail)...)
		}
		return vars
	case *core.RecordPattern:
		var vars []string
		for _, field := range p.Fields {
			vars = append(vars, patternBoundVars(field)...)
		}
		return vars
	case *core.TuplePattern:
		var vars []string
		for _, elem := range p.Elements {
			vars = append(vars, patternBoundVars(elem)...)
		}
		return vars
	case *core.LitPattern, *core.WildcardPattern:
		return nil
	default:
		return nil
	}
}

// isMutuallyRecursive checks if any binding in a LetRec group is mutually recursive
func isMutuallyRecursive(bindings []core.RecBinding) bool {
	// Build set of all binding names
	names := make(map[string]bool)
	for _, binding := range bindings {
		names[binding.Name] = true
	}

	// Check if any binding references another binding in the group
	for _, binding := range bindings {
		for otherName := range names {
			if otherName != binding.Name && isRecursive(binding.Value, otherName) {
				return true
			}
		}
	}

	return false
}

// Specialize performs monomorphization on a Core program
// Returns the specialized program and any errors encountered
func (s *Specializer) Specialize(prog *core.Program) (*core.Program, error) {
	// Specialize each top-level declaration
	newDecls := make([]core.CoreExpr, 0, len(prog.Decls))

	for _, decl := range prog.Decls {
		specialized, err := s.specializeExpr(decl, make(map[string]types.Type), make(map[string]core.CoreExpr))
		if err != nil {
			return nil, err
		}
		newDecls = append(newDecls, specialized)
	}

	// Create new program with specialized declarations
	result := &core.Program{
		Decls: newDecls,
		Meta:  prog.Meta,
		Flags: prog.Flags,
	}

	return result, nil
}

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

	// Atomic expressions - no specialization needed
	case *core.Var, *core.VarGlobal, *core.Lit, *core.Intrinsic:
		return expr, nil

	// TODO: Handle other expression types (Record, List, Tuple, etc.)
	default:
		// For now, return expression as-is
		return expr, nil
	}
}

// copyEnv creates a shallow copy of an environment map
func copyEnv(env map[string]types.Type) map[string]types.Type {
	newEnv := make(map[string]types.Type, len(env))
	for k, v := range env {
		newEnv[k] = v
	}
	return newEnv
}

// copyBindings creates a shallow copy of a bindings map
func copyBindings(bindings map[string]core.CoreExpr) map[string]core.CoreExpr {
	newBindings := make(map[string]core.CoreExpr, len(bindings))
	for k, v := range bindings {
		newBindings[k] = v
	}
	return newBindings
}

// specializeLambda attempts to specialize a polymorphic lambda for concrete argument types
// Returns nil if specialization is not possible or not beneficial
func (s *Specializer) specializeLambda(lambda *core.Lambda, argTypes []types.Type, env map[string]types.Type) (*core.Lambda, error) {
	if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] specializeLambda: START with lambda.Params=%v, argTypes=%v\n", lambda.Params, argTypes)
	}

	// Check module-wide cap
	if s.TotalCount >= s.Limits.MaxPerModule {
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] specializeLambda: SKIP - module limit reached\n")
		}
		s.Skipped = append(s.Skipped, SkipReason{
			DefSym:   "(lambda)",
			Reason:   fmt.Sprintf("Module specialization limit reached (%d/%d)", s.TotalCount, s.Limits.MaxPerModule),
			Location: lambda.OriginalSpan().String(),
		})
		return nil, nil
	}

	// Check per-function cap (using "(lambda)" as the function key for anonymous lambdas)
	funcKey := "(lambda)"
	if s.PerFunction[funcKey] >= s.Limits.MaxPerFunction {
		s.Skipped = append(s.Skipped, SkipReason{
			DefSym:   funcKey,
			Reason:   fmt.Sprintf("Per-function specialization limit reached (%d/%d)", s.PerFunction[funcKey], s.Limits.MaxPerFunction),
			Location: lambda.OriginalSpan().String(),
		})
		return nil, nil
	}

	// Build type substitution map from parameters to argument types
	// For now, simplified: assume 1:1 mapping between params and argTypes
	typeSubst := make(map[string]types.Type)
	for i, param := range lambda.Params {
		if i < len(argTypes) {
			typeSubst[param] = argTypes[i]
		}
	}

	// Generate cache key
	fingerprint := canonicalTypeFingerprint(argTypes)
	key := SpecializationKey{
		DefSym:           "(lambda)",
		TypesFingerprint: fingerprint,
	}

	// Check cache
	if cached, ok := s.Cache[key]; ok {
		s.CacheHits++
		if cachedLambda, ok := cached.(*core.Lambda); ok {
			return cachedLambda, nil
		}
	}
	s.CacheMisses++

	// Clone the lambda body with fresh node IDs and type substitution
	clonedBody, err := s.cloneExpr(lambda.Body, typeSubst)
	if err != nil {
		return nil, err
	}

	// Create specialized lambda with fresh node ID
	specialized := &core.Lambda{
		CoreNode: core.CoreNode{
			NodeID:   s.freshNodeID(),
			CoreSpan: lambda.CoreSpan,
			OrigSpan: lambda.OrigSpan,
		},
		Params: lambda.Params, // Keep same parameter names (simple approach)
		Body:   clonedBody,
	}

	// Populate CoreTypeInfo for the specialized lambda
	// Use the concrete function type (argTypes -> returnType)
	if lambdaType, ok := s.CoreTI.Get(lambda.ID()); ok {
		// Apply type substitution to the lambda's type
		specializedType := substituteType(lambdaType, typeSubst)
		s.CoreTI.Set(specialized.ID(), specializedType)
	}

	// Cache the specialized lambda
	s.Cache[key] = specialized

	// Increment counters
	s.TotalCount++
	s.PerFunction["(lambda)"]++

	if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] specializeLambda: SUCCESS - created specialized lambda (count=%d)\n", s.TotalCount)
	}

	return specialized, nil
}

// cloneExpr recursively clones an expression with fresh node IDs
// and applies type substitution to all types in CoreTypeInfo
func (s *Specializer) cloneExpr(expr core.CoreExpr, typeSubst map[string]types.Type) (core.CoreExpr, error) {
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
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] Cloning DictApp: method=%s\n", e.Method)
		}
		// Clone dictionary reference
		clonedDict, err := s.cloneExpr(e.Dict, typeSubst)
		if err != nil {
			return nil, err
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

	// For other expression types, return as-is for now (v0.4.0 simplification)
	default:
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] cloneExpr: default case for type %T\n", expr)
		}
		return expr, nil
	}
}

// substituteType applies a type substitution to a type
func substituteType(typ types.Type, subst map[string]types.Type) types.Type {
	switch t := typ.(type) {
	case *types.TVar:
		// If we have a substitution for this variable, use it
		if concrete, ok := subst[t.Name]; ok {
			return concrete
		}
		return typ
	case *types.TFunc2:
		// Substitute in parameters and return type
		newParams := make([]types.Type, len(t.Params))
		for i, p := range t.Params {
			newParams[i] = substituteType(p, subst)
		}
		newReturn := substituteType(t.Return, subst)
		return &types.TFunc2{
			Params:    newParams,
			Return:    newReturn,
			EffectRow: t.EffectRow, // Keep effects as-is for now
		}
	case *types.TApp:
		// Substitute in constructor and args
		newConstructor := substituteType(t.Constructor, subst)
		newArgs := make([]types.Type, len(t.Args))
		for i, arg := range t.Args {
			newArgs[i] = substituteType(arg, subst)
		}
		return &types.TApp{
			Constructor: newConstructor,
			Args:        newArgs,
		}
	default:
		// For concrete types (TCon, etc.), no substitution needed
		return typ
	}
}
