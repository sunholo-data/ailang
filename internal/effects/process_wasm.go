//go:build js

package effects

// ResolveAllowlist is a no-op on WASM (os/exec not available).
func (pc *ProcessContext) ResolveAllowlist(allowlistStr string) error {
	return nil
}
