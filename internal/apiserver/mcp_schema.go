package apiserver

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// mcpToolNameRegex is the strict regex enforced by Claude Desktop and most
// current MCP clients: alphanumeric, underscore, hyphen, 1-64 chars.
// SEP-986 would relax this to allow dots/slashes, but it isn't ratified yet.
var mcpToolNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// mcpToolName generates an MCP-compliant tool name for an exported function.
//
// Resolution order:
//  1. Author override via @mcp_name("name") — must already be valid; caller
//     should validate via validateMCPName before reaching this point.
//  2. Bare function name — when preferBare is true (funcName is globally
//     unique among exposed exports).
//  3. Sanitized fallback — last meaningful module segment + "_" + funcName,
//     with dots/slashes/illegal chars replaced by underscores.
//
// All non-override outputs are guaranteed to match mcpToolNameRegex by
// construction (sanitize + truncateWithHash).
func mcpToolName(modPath, funcName, override string, preferBare bool) string {
	if override != "" {
		return override
	}
	if preferBare {
		bare := sanitizeMCPName(funcName)
		return truncateWithHash(bare, modPath, funcName)
	}
	prefix := lastMeaningfulSegment(modPath)
	combined := funcName
	if prefix != "" {
		combined = prefix + "_" + funcName
	}
	return truncateWithHash(sanitizeMCPName(combined), modPath, funcName)
}

// sanitizeMCPName replaces any character not in [a-zA-Z0-9_-] with '_'.
// Empty input becomes "_" so the result always satisfies the min-length rule.
func sanitizeMCPName(s string) string {
	if s == "" {
		return "_"
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "_"
	}
	return out
}

// truncateWithHash enforces the 64-character limit. If name fits, it's returned
// unchanged. Otherwise, name is truncated to 56 chars and a 7-char deterministic
// hash of (modPath + "." + funcName) is appended after a single underscore,
// preserving uniqueness across modules with shared prefixes.
func truncateWithHash(name, modPath, funcName string) string {
	if len(name) <= 64 {
		return name
	}
	sum := sha1.Sum([]byte(modPath + "." + funcName))
	hash := hex.EncodeToString(sum[:])[:7]
	return name[:56] + "_" + hash
}

// lastMeaningfulSegment returns the last path segment of modPath, ignoring
// empty segments and treating both '/' and '.' as separators. Used as a
// disambiguation prefix when bare function names collide.
func lastMeaningfulSegment(modPath string) string {
	cleaned := strings.ReplaceAll(modPath, "/", ".")
	cleaned = strings.Trim(cleaned, ".")
	if cleaned == "" {
		return ""
	}
	parts := strings.Split(cleaned, ".")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return ""
}

// validateMCPName returns an error if name does not match the MCP tool name
// regex. Used to vet author-supplied @mcp_name values at registration time.
func validateMCPName(name string) error {
	if !mcpToolNameRegex.MatchString(name) {
		return fmt.Errorf("MCP tool name %q is invalid: must match %s", name, mcpToolNameRegex.String())
	}
	return nil
}
