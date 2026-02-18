package effects

import "time"

// ProcessContext provides configuration for Process effect security
//
// The process context holds security settings for command execution:
//   - Timeout enforcement (default: 30s)
//   - Output size limits (default: 10MB)
//   - Command allowlist with path-pinned resolution
//   - Working directory from sandbox
type ProcessContext struct {
	Timeout      time.Duration     // Maximum execution time
	MaxOutput    int64             // Maximum stdout/stderr bytes before kill
	Allowlist    map[string]string // Allowed commands: name → resolved absolute path (nil = all)
	HasAllowlist bool              // True if allowlist was explicitly set
}

// NewProcessContext creates a new process context with secure defaults
func NewProcessContext() *ProcessContext {
	return &ProcessContext{
		Timeout:   30 * time.Second,
		MaxOutput: 10 * 1024 * 1024, // 10 MB
	}
}
