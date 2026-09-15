package eval_harness

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/prompt"
)

// PromptVersion represents metadata about a prompt version. Same JSON shape as
// prompt.VersionMetadata; kept as its own type because FrozenMarker (in
// prompt_frozen.go) and the freeze tooling in cmd/ailang name it.
type PromptVersion struct {
	File        string        `json:"file"`
	Hash        string        `json:"hash"`
	Description string        `json:"description"`
	Created     string        `json:"created"`
	Tags        []string      `json:"tags"`
	Notes       string        `json:"notes"`
	Frozen      *FrozenMarker `json:"frozen,omitempty"`
}

// PromptRegistry contains all registered prompt versions
type PromptRegistry struct {
	SchemaVersion string                   `json:"schema_version"`
	Versions      map[string]PromptVersion `json:"versions"`
	Active        string                   `json:"active"`
	Notes         []string                 `json:"notes"`
}

// PromptLoader loads prompt versions from an explicit registry file WITH hash
// verification: a frozen version must match its recorded sha256 exactly, a
// mutable one must match unless its hash is the PLACEHOLDER escape hatch
// (D-41c). It is prompt.Loader with WithVerify — the eval harness must never
// measure edited bytes under a banked version's name.
type PromptLoader struct {
	loader *prompt.Loader
}

// NewPromptLoader creates a loader from versions.json. Prompt `file` entries
// resolve against the registry's project root (the directory above prompts/).
func NewPromptLoader(registryPath string) (*PromptLoader, error) {
	rootDir := filepath.Dir(registryPath)
	if filepath.Base(rootDir) == "prompts" {
		rootDir = filepath.Dir(rootDir) // Go up one level to project root
	}
	l := prompt.NewLoader(prompt.Syntax,
		prompt.WithManifestFile(registryPath),
		prompt.WithRoot(rootDir),
		prompt.WithVerify(),
	)
	// Fail at construction, as before, rather than on the first load.
	if _, err := l.Manifest(); err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}
	return &PromptLoader{loader: l}, nil
}

// LoadPrompt loads a prompt by version ID with hash verification.
func (l *PromptLoader) LoadPrompt(versionID string) (string, error) {
	return l.loader.LoadPrompt(versionID)
}

// GetActivePrompt loads the active prompt version. A registry whose active
// field is "latest" resolves to its most recent production version.
func (l *PromptLoader) GetActivePrompt() (string, error) {
	return l.loader.LoadPrompt("")
}

// GetActiveVersionID returns the active version ID (resolving "latest" if
// needed); "" when the registry names none.
func (l *PromptLoader) GetActiveVersionID() string {
	v, _ := l.loader.GetActiveVersion()
	return v
}

// GetVersion returns metadata for a specific version
func (l *PromptLoader) GetVersion(versionID string) (*PromptVersion, error) {
	manifest, err := l.loader.Manifest()
	if err != nil {
		return nil, err
	}
	meta, ok := manifest.Versions[versionID]
	if !ok {
		return nil, fmt.Errorf("prompt version %q not found", versionID)
	}
	v := promptVersionFrom(meta)
	return &v, nil
}

// ListVersions returns all available prompt versions
func (l *PromptLoader) ListVersions() map[string]PromptVersion {
	manifest, err := l.loader.Manifest()
	if err != nil {
		return nil
	}
	out := make(map[string]PromptVersion, len(manifest.Versions))
	for id, meta := range manifest.Versions {
		out[id] = promptVersionFrom(meta)
	}
	return out
}

func promptVersionFrom(m prompt.VersionMetadata) PromptVersion {
	return PromptVersion{
		File:        m.File,
		Hash:        m.Hash,
		Description: m.Description,
		Created:     m.Created,
		Tags:        m.Tags,
		Notes:       m.Notes,
		Frozen:      (*FrozenMarker)(m.Frozen),
	}
}

// ComputePromptHash is a helper to compute hash for a prompt file (for updating registry)
func ComputePromptHash(filePath string) (string, error) {
	content, err := os.ReadFile(filePath) // #nosec G304 -- operator-supplied prompt path
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return prompt.SHA256Hex(content), nil
}
