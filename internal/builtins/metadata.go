package builtins

// metadata.go defines enhanced metadata types for builtin functions (M-DX1.11)
//
// These types extend the basic BuiltinSpec with documentation, versioning,
// and semantic metadata to improve discoverability and tooling support.

// ParamDoc documents a single function parameter
type ParamDoc struct {
	Name        string // Parameter name (e.g., "url", "method")
	Description string // What the parameter does and any constraints
}

// Example provides a usage example for a builtin
type Example struct {
	Code        string // AILANG code demonstrating the builtin
	Description string // What the example demonstrates
}

// Stability indicates the maturity level of a builtin
type Stability int

const (
	// StabilityUnspecified means stability hasn't been declared yet
	StabilityUnspecified Stability = iota

	// StabilityExperimental means the API may change in any release
	// Use with caution in production code
	StabilityExperimental

	// StabilityStable means the API is backwards compatible
	// Breaking changes will only occur in major version bumps
	StabilityStable

	// StabilityDeprecated means the builtin is being phased out
	// Users should migrate to the recommended alternative
	StabilityDeprecated
)

// String returns the string representation of a Stability level
func (s Stability) String() string {
	switch s {
	case StabilityExperimental:
		return "experimental"
	case StabilityStable:
		return "stable"
	case StabilityDeprecated:
		return "deprecated"
	case StabilityUnspecified:
		return "unspecified"
	default:
		return "unknown"
	}
}

// ParseStability converts a string to a Stability constant
func ParseStability(s string) Stability {
	switch s {
	case "experimental":
		return StabilityExperimental
	case "stable":
		return StabilityStable
	case "deprecated":
		return StabilityDeprecated
	default:
		return StabilityUnspecified
	}
}

// BuiltinMetadata contains optional enhanced metadata for a builtin
// All fields are optional to maintain backward compatibility
type BuiltinMetadata struct {
	// === Documentation ===

	// Description is a one-line summary of what the builtin does
	// Example: "Make an HTTP request with custom headers and body"
	Description string

	// LongDesc is a detailed multi-line description (optional)
	// Can include usage notes, caveats, performance characteristics
	LongDesc string

	// Params documents each parameter (optional but recommended)
	Params []ParamDoc

	// Returns documents the return value (optional but recommended)
	// Example: "Result[HttpResponse, NetError] where HttpResponse contains status, headers, and body"
	Returns string

	// Examples provides usage examples (optional)
	// Highly recommended for complex builtins
	Examples []Example

	// SeeAlso lists related builtins (optional)
	// Example: []string{"_net_httpGet", "_net_httpPost"}
	SeeAlso []string

	// === Versioning ===

	// Since indicates when the builtin was added (optional)
	// Example: "v0.2.0"
	Since string

	// Deprecated contains a deprecation message if this builtin is being phased out (optional)
	// Should explain what to use instead
	// Example: "Use _net_httpRequest instead, which supports custom headers"
	Deprecated string

	// Stability indicates API stability guarantees (optional, defaults to Stable for old builtins)
	Stability Stability

	// === Semantic ===

	// Tags are searchable keywords (optional)
	// Example: []string{"http", "network", "request", "api"}
	Tags []string

	// Category is a high-level grouping (optional)
	// Examples: "network", "string", "math", "io"
	Category string
}

// IsDeprecated returns true if this builtin has been deprecated
func (m *BuiltinMetadata) IsDeprecated() bool {
	return m.Deprecated != "" || m.Stability == StabilityDeprecated
}

// HasDocumentation returns true if this builtin has meaningful documentation
func (m *BuiltinMetadata) HasDocumentation() bool {
	return m.Description != ""
}

// HasExamples returns true if this builtin has usage examples
func (m *BuiltinMetadata) HasExamples() bool {
	return len(m.Examples) > 0
}

// GetStabilityString returns a human-readable stability level
func (m *BuiltinMetadata) GetStabilityString() string {
	if m.Stability == StabilityUnspecified {
		// Default to Stable if not specified (for backward compatibility)
		return "stable"
	}
	return m.Stability.String()
}
