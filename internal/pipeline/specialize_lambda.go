package pipeline

import (
	"fmt"
	"os"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

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

	// Build type substitution map from TYPE VARIABLES to argument types
	// The lambda's type in CoreTI has TVars for parameters (e.g., α1 -> α2 -> α3)
	// We need to map those TVars to the concrete argTypes
	typeSubst := make(map[string]types.Type)

	// Extract TVars from lambda's function type
	if lambdaType, ok := s.CoreTI.Get(lambda.ID()); ok {
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] lambda type from CoreTI: %v (type: %T)\n", lambdaType, lambdaType)
		}

		// Collect parameter TVars from the function type
		paramTVars := extractParamTVars(lambdaType, len(lambda.Params))

		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] extracted paramTVars: %v\n", paramTVars)
		}

		// Map each TVar to its concrete type
		for i, tvar := range paramTVars {
			if i < len(argTypes) && tvar != "" {
				typeSubst[tvar] = argTypes[i]
			}
		}

		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] typeSubst built: %v\n", typeSubst)
		}
	} else {
		// Fallback: if lambda type not in CoreTI, can't specialize properly
		if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] WARNING: lambda type not in CoreTI, skipping\n")
		}
		return nil, nil
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
	if os.Getenv("DEBUG_MONO_VERBOSE") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_MONO_VERBOSE] lambda.Body type: %T, NodeID: %d\n", lambda.Body, lambda.Body.ID())
	}
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
