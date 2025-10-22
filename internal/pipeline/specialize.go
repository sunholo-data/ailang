// Package pipeline provides monomorphization for polymorphic functions
package pipeline

import (
	"crypto/sha256"
	"fmt"
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
	}
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
	switch typ := t.(type) {
	case *types.TVar:
		return true
	case *types.TApp:
		for _, arg := range typ.Args {
			if isPolymorphic(arg) {
				return true
			}
		}
		return false
	case *types.TFunc2:
		for _, p := range typ.Params {
			if isPolymorphic(p) {
				return true
			}
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
}

// GetStats returns specialization statistics for debugging
func (s *Specializer) GetStats() SpecializationStats {
	return SpecializationStats{
		TotalSpecializations: s.TotalCount,
		PerFunction:          s.PerFunction,
		SkippedFunctions:     s.Skipped,
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
		specialized, err := s.specializeExpr(decl, make(map[string]types.Type))
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
func (s *Specializer) specializeExpr(expr core.CoreExpr, env map[string]types.Type) (core.CoreExpr, error) {
	switch e := expr.(type) {
	case *core.Let:
		// Specialize the value
		newValue, err := s.specializeExpr(e.Value, env)
		if err != nil {
			return nil, err
		}

		// Add binding to environment
		newEnv := copyEnv(env)
		if typ, ok := s.CoreTI.Get(e.Value.ID()); ok {
			newEnv[e.Name] = typ
		}

		// Specialize the body
		newBody, err := s.specializeExpr(e.Body, newEnv)
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
			newBody, err := s.specializeExpr(e.Body, env)
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
				specialized, err := s.specializeExpr(binding.Value, newEnv)
				if err != nil {
					return nil, err
				}
				newBindings[i] = core.RecBinding{
					Name:  binding.Name,
					Value: specialized,
				}
			}

			// Add to environment for subsequent bindings
			if typ, ok := s.CoreTI.Get(binding.Value.ID()); ok {
				newEnv[binding.Name] = typ
			}
		}

		newBody, err := s.specializeExpr(e.Body, newEnv)
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
		// Note: We don't know parameter types here without more context
		// Lambda specialization happens when applied

		newBody, err := s.specializeExpr(e.Body, newEnv)
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

		// First, specialize the function and arguments
		newFunc, err := s.specializeExpr(e.Func, env)
		if err != nil {
			return nil, err
		}

		newArgs := make([]core.CoreExpr, len(e.Args))
		for i, arg := range e.Args {
			newArgs[i], err = s.specializeExpr(arg, env)
			if err != nil {
				return nil, err
			}
		}

		// TODO (Day 2.3): Check if function is polymorphic and arguments are concrete
		// If so, specialize the function for these concrete types
		// For now, just return the specialized app

		return &core.App{
			CoreNode: e.CoreNode,
			Func:     newFunc,
			Args:     newArgs,
		}, nil

	case *core.If:
		newCond, err := s.specializeExpr(e.Cond, env)
		if err != nil {
			return nil, err
		}
		newThen, err := s.specializeExpr(e.Then, env)
		if err != nil {
			return nil, err
		}
		newElse, err := s.specializeExpr(e.Else, env)
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
		newScrutinee, err := s.specializeExpr(e.Scrutinee, env)
		if err != nil {
			return nil, err
		}

		newArms := make([]core.MatchArm, len(e.Arms))
		for i, arm := range e.Arms {
			newBody, err := s.specializeExpr(arm.Body, env)
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
		newLeft, err := s.specializeExpr(e.Left, env)
		if err != nil {
			return nil, err
		}
		newRight, err := s.specializeExpr(e.Right, env)
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
		newOperand, err := s.specializeExpr(e.Operand, env)
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
