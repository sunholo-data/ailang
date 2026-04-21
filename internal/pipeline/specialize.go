// Package pipeline provides monomorphization for polymorphic functions
//
// # Architecture
//
// The monomorphization pass is split into several files:
//   - specialize.go: Specializer struct, Specialize entry point, statistics (THIS FILE)
//   - specialize_types.go: Type manipulation (canonicalTypeFingerprint, substituteType, etc.)
//   - specialize_expr.go: Expression specialization (specializeExpr)
//   - specialize_lambda.go: Lambda specialization (specializeLambda)
//   - specialize_clone.go: Expression cloning with fresh node IDs (cloneExpr)
//   - specialize_helpers.go: Helper functions (isRecursive, patternBoundVars, copyEnv, etc.)
//
// # Usage
//
//	specializer := NewSpecializer(&typeChecker.CoreTI)
//	specializedProg, err := specializer.Specialize(coreProg)
//	stats := specializer.GetStats()
//
// # See Also
//
//   - internal/core: Core AST
//   - internal/types: Type system
//   - design_docs/planned/v0_4_0/m-poly-a.md: Monomorphization design
package pipeline

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
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
