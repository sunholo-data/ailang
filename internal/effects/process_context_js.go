//go:build js

package effects

// managedState is a no-op stub for js/wasm where subprocess management is unavailable.
type managedState struct{}

// CloseAllManaged is a no-op on js/wasm.
func (pc *ProcessContext) CloseAllManaged() {}
