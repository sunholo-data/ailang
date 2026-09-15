// Package agentprompt is a thin alias of prompt.Agent, kept so existing
// callers compile; new code should use prompt.Agent directly. The loader that
// used to live here was a byte-for-byte copy of internal/prompt/loader.go
// differing only in the "agent" literal (M-V1-SIMPLIFY-S3 M4).
package agentprompt

import "github.com/sunholo-data/ailang/internal/prompt"

// VersionMetadata represents metadata for an agent prompt version.
type VersionMetadata = prompt.VersionMetadata

var (
	// SetEmbeddedFS sets the embedded filesystem shared by every prompt kind.
	SetEmbeddedFS = prompt.SetEmbeddedFS
	// LoadPrompt loads an agent prompt by version ("" or "latest" = active).
	LoadPrompt = prompt.Agent.LoadPrompt
	// GetActiveVersion returns the active version from agent versions.json.
	GetActiveVersion = prompt.Agent.GetActiveVersion
	// ListVersions returns all available agent prompt versions.
	ListVersions = prompt.Agent.ListVersions
	// GetVersionMetadata returns metadata for a specific agent prompt version.
	GetVersionMetadata = prompt.Agent.GetVersionMetadata
)
