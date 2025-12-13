// Package types provides type report functionality for debugging.
//
// TypeReport is the canonical primitive for type debugging. It consolidates
// information from multiple data structures (CoreTI, substitution, constraints)
// into a single queryable API.
//
// Usage:
//
//	report := tc.TypeReport(nodeID)
//	fmt.Printf("Raw: %s, Resolved: %s\n", report.Raw, report.Resolved)
//
// See: design_docs/planned/v0_5_10/m-dx11-type-inference-debugging.md
package types

// TypeReport consolidates type information for a Core node.
// It is a thin façade over existing structures, always derived from live data.
type TypeReport struct {
	NodeID      uint64          // The Core node ID
	Raw         Type            // What's in CoreTI (may have TVars)
	Resolved    Type            // After applying full substitution
	Constraints []ConstraintRef // Constraints mentioning this type's variables
	Origins     []TypeOrigin    // M-DX11: Provenance chain (where type came from)
	Found       bool            // Whether the node was found in CoreTI
}

// ConstraintRef holds a reference to a constraint with source information.
type ConstraintRef struct {
	ClassName string // "Num", "Eq", "Ord", etc.
	Type      Type   // The type involved in the constraint
	NodeID    uint64 // Node where constraint was introduced
	Resolved  bool   // Whether the constraint has been resolved
	Method    string // Method name if resolved (e.g., "add", "eq")
}

// SourceSpan represents a source code location for provenance tracking.
// Defined here to avoid circular dependencies with ast package.
type SourceSpan struct {
	File   string // Source file path
	Line   int    // Line number (1-based)
	Column int    // Column number (1-based)
}

// String returns a human-readable location string.
func (s SourceSpan) String() string {
	if s.File == "" && s.Line == 0 {
		return "<unknown>"
	}
	if s.File == "" {
		return itoa(uint64(s.Line)) + ":" + itoa(uint64(s.Column))
	}
	return s.File + ":" + itoa(uint64(s.Line)) + ":" + itoa(uint64(s.Column))
}

// TypeOrigin tracks where a type came from for provenance debugging.
// Used by VerboseDebugSink to answer "why does this have type X?" questions.
type TypeOrigin struct {
	Kind   OriginKind // How this type was determined
	NodeID uint64     // Originating node
	Span   SourceSpan // Source location
	Note   string     // Human-readable description
}

// OriginKind classifies the source of a type.
type OriginKind int

const (
	// OriginUnknown means provenance tracking is not enabled
	OriginUnknown OriginKind = iota
	// OriginAnnotation means the type came from an explicit type annotation
	OriginAnnotation
	// OriginLiteral means the type was inferred from a literal (3.14 → float)
	OriginLiteral
	// OriginInferred means the type was inferred through unification
	OriginInferred
	// OriginDefaulted means the type was defaulted (Num → Int/Float)
	OriginDefaulted
	// OriginFromUse means the type was inferred from a call site
	OriginFromUse
	// OriginFromPattern means the type was inferred from pattern matching
	OriginFromPattern
)

// String returns a human-readable name for the origin kind.
func (k OriginKind) String() string {
	switch k {
	case OriginAnnotation:
		return "annotation"
	case OriginLiteral:
		return "literal"
	case OriginInferred:
		return "inferred"
	case OriginDefaulted:
		return "defaulted"
	case OriginFromUse:
		return "from_use"
	case OriginFromPattern:
		return "from_pattern"
	default:
		return "unknown"
	}
}

// TypeReport returns consolidated type information for a Core node.
// This is the canonical primitive for type debugging, consolidating info
// from CoreTI, substitution, and resolved constraints.
//
// The report includes:
//   - Raw: The type as stored in CoreTI (may contain type variables)
//   - Resolved: The type after applying the full substitution closure
//   - Constraints: Any class constraints mentioning this node's type variables
//   - Origins: Provenance chain showing where the type came from (if debug enabled)
//   - Found: Whether the node exists in CoreTI
func (tc *CoreTypeChecker) TypeReport(nodeID uint64) TypeReport {
	report := TypeReport{
		NodeID: nodeID,
		Found:  false,
	}

	// 1. Get raw type from CoreTI
	raw, ok := tc.CoreTI.Get(nodeID)
	if !ok {
		return report
	}
	report.Found = true
	report.Raw = raw

	// 2. Apply full substitution to get resolved type
	// ApplySubstitution already follows chains (M-FIX-FLOAT-OP)
	report.Resolved = ApplySubstitution(tc.getSubstitution(), raw)

	// 3. Find constraints mentioning this node
	report.Constraints = tc.findConstraintsFor(nodeID, raw)

	// 4. Get provenance if VerboseDebugSink is active (M-DX11)
	report.Origins = tc.getProvenanceFor(raw)

	return report
}

// getProvenanceFor returns type provenance if VerboseDebugSink is active.
func (tc *CoreTypeChecker) getProvenanceFor(t Type) []TypeOrigin {
	// Check if debug sink is VerboseDebugSink
	verbose, ok := tc.DebugSink.(*VerboseDebugSink)
	if !ok || verbose == nil {
		return nil
	}

	// Get the type variable name to look up provenance
	if t == nil {
		return nil
	}
	tvName := t.String()
	return verbose.GetProvenance(tvName)
}

// getSubstitution returns the current substitution map.
// Returns an empty map if not available.
func (tc *CoreTypeChecker) getSubstitution() Substitution {
	// The substitution is stored in the Unifier during inference
	// For post-inference queries, we may not have direct access
	// Return empty - the resolved type will equal raw if no sub available
	// TODO: Store final substitution in CoreTypeChecker for post-inference queries
	return make(Substitution)
}

// findConstraintsFor returns constraints related to a node's type.
func (tc *CoreTypeChecker) findConstraintsFor(nodeID uint64, t Type) []ConstraintRef {
	var refs []ConstraintRef

	// Check resolved constraints for this node
	if rc, ok := tc.resolvedConstraints[nodeID]; ok {
		refs = append(refs, ConstraintRef{
			ClassName: rc.ClassName,
			Type:      rc.Type,
			NodeID:    rc.NodeID,
			Resolved:  true,
			Method:    rc.Method,
		})
	}

	return refs
}

// String returns a human-readable representation of the report.
func (r TypeReport) String() string {
	if !r.Found {
		return "TypeReport{not found}"
	}

	rawStr := "nil"
	if r.Raw != nil {
		rawStr = r.Raw.String()
	}

	resolvedStr := "nil"
	if r.Resolved != nil {
		resolvedStr = r.Resolved.String()
	}

	result := "TypeReport{" +
		"NodeID:" + itoa(r.NodeID) +
		", Raw:" + rawStr +
		", Resolved:" + resolvedStr +
		", Constraints:" + itoa(uint64(len(r.Constraints))) +
		", Origins:" + itoa(uint64(len(r.Origins))) +
		"}"

	return result
}

// FormatDetailed returns a multi-line detailed representation of the report.
// This format matches the design doc output format for --debug-types.
func (r TypeReport) FormatDetailed() string {
	if !r.Found {
		return "NodeID " + itoa(r.NodeID) + ": not found"
	}

	rawStr := "nil"
	if r.Raw != nil {
		rawStr = r.Raw.String()
	}

	resolvedStr := "nil"
	if r.Resolved != nil {
		resolvedStr = r.Resolved.String()
	}

	result := "NodeID " + itoa(r.NodeID) + ": " + resolvedStr + "\n" +
		"  Raw: " + rawStr + "\n" +
		"  Resolved: " + resolvedStr

	if len(r.Origins) > 0 {
		result += "\n  Origins:"
		for _, origin := range r.Origins {
			result += "\n    - " + origin.Kind.String()
			if origin.Note != "" {
				result += ": " + origin.Note
			}
			if origin.Span.Line != 0 || origin.Span.File != "" {
				result += " at " + origin.Span.String()
			}
		}
	}

	return result
}

// itoa converts uint64 to string without importing strconv
func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
