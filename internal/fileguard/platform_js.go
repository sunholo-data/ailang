//go:build js

package fileguard

// supported: os.Root is documented as vulnerable to TOCTOU in symlink
// validation on GOOS=js and cannot ensure operations stay inside the root.
// Restricted execution refuses rather than pretending (D6).
const supported = false
