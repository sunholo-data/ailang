package effects

import (
	"os"
	"strconv"
)

// EffContext holds runtime capability grants and environment configuration
//
// The effect context is the central runtime structure for effect execution.
// It tracks which capabilities have been granted and provides environment
// configuration for deterministic effect execution.
//
// Thread-safety: EffContext is typically created once per evaluation and
// should not be mutated concurrently.
type EffContext struct {
	Caps map[string]Capability // Effect name → Capability grant
	Env  EffEnv                 // Environment configuration
}

// EffEnv provides deterministic effect execution configuration
//
// The environment holds configuration from OS environment variables
// that control effect behavior:
//   - AILANG_SEED: Seed for reproducible randomness
//   - TZ: Timezone for deterministic time operations
//   - LANG: Locale for deterministic string operations
//   - AILANG_FS_SANDBOX: Root directory for sandboxed FS operations
type EffEnv struct {
	Seed    int64  // AILANG_SEED for reproducible randomness
	TZ      string // TZ for deterministic time operations
	Locale  string // LANG for deterministic string operations
	Sandbox string // Root directory for FS operations (empty = no sandbox)
}

// NewEffContext creates a new effect context
//
// The context is initialized with no capabilities granted (deny-by-default)
// and environment loaded from OS environment variables.
//
// Returns:
//   - A new EffContext ready to use
//
// Example:
//
//	ctx := NewEffContext()
//	ctx.Grant(NewCapability("IO"))
//	ctx.Grant(NewCapability("FS"))
func NewEffContext() *EffContext {
	return &EffContext{
		Caps: make(map[string]Capability),
		Env:  loadEffEnv(),
	}
}

// Grant adds a capability to the context
//
// Once granted, the capability allows execution of the corresponding
// effect operations. Granting is idempotent - granting the same
// capability twice has no additional effect.
//
// Parameters:
//   - cap: The capability to grant
//
// Example:
//
//	ctx.Grant(NewCapability("IO"))
func (ctx *EffContext) Grant(cap Capability) {
	ctx.Caps[cap.Name] = cap
}

// HasCap checks if a capability is granted
//
// Parameters:
//   - name: The capability name to check (e.g., "IO", "FS")
//
// Returns:
//   - true if the capability is granted, false otherwise
//
// Example:
//
//	if ctx.HasCap("IO") {
//	    // IO operations allowed
//	}
func (ctx *EffContext) HasCap(name string) bool {
	_, ok := ctx.Caps[name]
	return ok
}

// RequireCap checks for a capability and returns an error if missing
//
// This is the primary capability check used by effect operations.
// It provides a consistent error type (CapabilityError) when a
// capability is not granted.
//
// Parameters:
//   - name: The required capability name
//
// Returns:
//   - nil if the capability is granted
//   - CapabilityError if the capability is missing
//
// Example:
//
//	if err := ctx.RequireCap("FS"); err != nil {
//	    return nil, err
//	}
//	// FS operations allowed here
func (ctx *EffContext) RequireCap(name string) error {
	if !ctx.HasCap(name) {
		return NewCapabilityError(name)
	}
	return nil
}

// loadEffEnv loads effect environment from OS environment variables
//
// Environment variables:
//   - AILANG_SEED: Integer seed for deterministic randomness (default: 0)
//   - TZ: Timezone string (default: "UTC")
//   - LANG: Locale string (default: "C")
//   - AILANG_FS_SANDBOX: Filesystem sandbox root (default: "" = no sandbox)
//
// Returns:
//   - Populated EffEnv with values from environment
func loadEffEnv() EffEnv {
	seed := int64(0)
	if seedStr := os.Getenv("AILANG_SEED"); seedStr != "" {
		if s, err := strconv.ParseInt(seedStr, 10, 64); err == nil {
			seed = s
		}
	}

	return EffEnv{
		Seed:    seed,
		TZ:      getEnv("TZ", "UTC"),
		Locale:  getEnv("LANG", "C"),
		Sandbox: os.Getenv("AILANG_FS_SANDBOX"),
	}
}

// getEnv gets an environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
