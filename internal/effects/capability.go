package effects

// Capability represents a granted runtime capability
//
// Capabilities are tokens that grant permission to execute effects.
// Each capability has a name (e.g., "IO", "FS", "Net") and optional
// metadata for future extensions (budgets, tracing, sandboxing rules).
//
// Example:
//
//	ioCap := NewCapability("IO")
//	fsCap := NewCapability("FS")
//	fsCap.Meta["sandbox"] = "/tmp"
type Capability struct {
	Name string // Effect name: "IO", "FS", "Net", etc.

	// Meta holds optional metadata for future use
	// Examples:
	//   - "budget": rate limits or quotas
	//   - "trace": tracing context
	//   - "sandbox": filesystem root restriction
	Meta map[string]any
}

// NewCapability creates a basic capability with the given name
//
// The capability is created with an empty metadata map that can be
// populated for advanced use cases.
//
// Parameters:
//   - name: The effect name (e.g., "IO", "FS")
//
// Returns:
//   - A new Capability with empty metadata
//
// Example:
//
//	cap := NewCapability("IO")
//	cap.Meta["max_writes"] = 100  // Optional: add budget limit
func NewCapability(name string) Capability {
	return Capability{
		Name: name,
		Meta: make(map[string]any),
	}
}
