// Package devtoolsprompt is a thin alias of prompt.DevTools, kept so existing
// callers compile; new code should use prompt.DevTools directly. The loader
// that used to live here was a byte-for-byte copy of internal/prompt/loader.go
// differing only in the "devtools" literal (M-V1-SIMPLIFY-S3 M4).
package devtoolsprompt

import "github.com/sunholo-data/ailang/internal/prompt"

// VersionMetadata represents metadata for a devtools prompt version.
type VersionMetadata = prompt.VersionMetadata

var (
	// SetEmbeddedFS sets the embedded filesystem shared by every prompt kind.
	SetEmbeddedFS = prompt.SetEmbeddedFS
	// LoadPrompt loads a devtools prompt by version ("" or "latest" = active).
	LoadPrompt = prompt.DevTools.LoadPrompt
	// GetActiveVersion returns the active version from devtools versions.json.
	GetActiveVersion = prompt.DevTools.GetActiveVersion
	// ListVersions returns all available devtools prompt versions.
	ListVersions = prompt.DevTools.ListVersions
	// GetVersionMetadata returns metadata for a specific devtools prompt version.
	GetVersionMetadata = prompt.DevTools.GetVersionMetadata
)
