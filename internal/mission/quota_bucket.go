package mission

import (
	"errors"
	"strings"
)

// quotaBuckets names the provider pools; harnesses do not own quota.
var quotaBuckets = [...]string{"anthropic", "codex", "ollama", "openrouter"}

// ErrOpencodeQuotaBucket explains why a harness cannot be selected as a pool.
var ErrOpencodeQuotaBucket = errors.New("opencode is a harness, not a quota bucket; record the backend bucket instead")

func isQuotaBucket(bucket string) bool {
	for _, known := range quotaBuckets {
		if bucket == known {
			return true
		}
	}
	return false
}

// QuotaBucketFor maps opencode-qualified spend to its backend provider pool.
// Unrelated values are returned unchanged, including their original case.
func QuotaBucketFor(raw string) (string, error) {
	lower := strings.ToLower(raw)
	if lower == "opencode" {
		return "", ErrOpencodeQuotaBucket
	}
	if strings.HasPrefix(lower, "opencode:") || strings.HasPrefix(lower, "opencode/") {
		backend, _, _ := strings.Cut(lower[len("opencode:"):], "/")
		if isQuotaBucket(backend) {
			return backend, nil
		}
		return "", ErrOpencodeQuotaBucket
	}
	return raw, nil
}
