package pkg

// RegistryIndex is the top-level index.json from the registry.
type RegistryIndex struct {
	Schema    string       `json:"schema"`
	UpdatedAt string       `json:"updated_at"`
	Packages  []IndexEntry `json:"packages"`
}

// IndexEntry is a package listing in the registry index.
type IndexEntry struct {
	Name              string   `json:"name"`
	Latest            string   `json:"latest"`
	Versions          []string `json:"versions"`
	AISummary         string   `json:"ai_summary"`
	Tags              []string `json:"tags"`
	Effects           []string `json:"effects"`
	Stability         string   `json:"stability"`
	Exports           []string `json:"exports"`
	ContractsVerified int      `json:"contracts_verified"`
	HasAgentDoc       bool     `json:"has_agent_doc"`
}

// PackageMetadata is the per-version metadata.json from the registry.
type PackageMetadata struct {
	Schema      string           `json:"schema"`
	Name        string           `json:"name"`
	Version     string           `json:"version"`
	PublishedAt string           `json:"published_at"`
	PublishedBy string           `json:"published_by"`
	ContentHash string           `json:"content_hash"`
	InterfHash  string           `json:"interface_hash"`
	TarballHash string           `json:"tarball_hash"`
	TarballSize int64            `json:"tarball_size_bytes"`
	Validation  ValidationResult `json:"validation"`
	Manifest    MetadataManifest `json:"manifest"`
}

// ValidationResult records what the validator checked.
type ValidationResult struct {
	Compiles          bool   `json:"compiles"`
	EffectsValid      bool   `json:"effects_valid"`
	ContractsVerified int    `json:"contracts_verified"`
	ContractsTotal    int    `json:"contracts_total"`
	ContractsSkipped  int    `json:"contracts_skipped"`
	AILANGVersion     string `json:"ailang_version"`
}

// MetadataManifest is the manifest section of metadata.json.
type MetadataManifest struct {
	Edition     string   `json:"edition"`
	EffectsMax  []string `json:"effects_max"`
	Exports     []string `json:"exports"`
	Stability   string   `json:"stability"`
	AISummary   string   `json:"ai_summary"`
	HasAgentDoc bool     `json:"has_agent_doc"`
}
