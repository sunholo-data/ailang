package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
)

// PolicyDigest is the sha256 of the program-policy file a run was gated by,
// or "" when no policy was passed or the file cannot be read. A digest rather
// than a path: the path is per-deployment, the digest says WHICH policy, and
// two rows with equal digests ran under identical authority.
func PolicyDigest(path string) string {
	if path == "" {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
