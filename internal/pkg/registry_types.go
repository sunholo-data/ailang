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
	Dependencies      []string `json:"dependencies,omitempty"`   // Package names this depends on (M-PKG-AUTONOMOUS-UPDATES)
	LastUpdated       string   `json:"last_updated,omitempty"`   // When latest version was published
	UpdatedBy         string   `json:"updated_by,omitempty"`     // "human", "agent", or agent ID
	LatestSummary     string   `json:"latest_summary,omitempty"` // From history.json summary
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
	Provenance  *ProvenanceInfo  `json:"provenance,omitempty"` // M-PKG-AUTONOMOUS-UPDATES
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

// FindDependents returns the names of all packages in the index that
// list the given package as a dependency.
func (idx *RegistryIndex) FindDependents(pkgName string) []string {
	var dependents []string
	for _, entry := range idx.Packages {
		if entry.Name == pkgName {
			continue
		}
		for _, dep := range entry.Dependencies {
			if dep == pkgName {
				dependents = append(dependents, entry.Name)
				break
			}
		}
	}
	return dependents
}

// MetadataManifest is the manifest section of metadata.json.
type MetadataManifest struct {
	Edition     string   `json:"edition"`
	AILANG      string   `json:"ailang,omitempty"` // Minimum AILANG version (e.g., ">=0.9.5")
	EffectsMax  []string `json:"effects_max"`
	Exports     []string `json:"exports"`
	Stability   string   `json:"stability"`
	AISummary   string   `json:"ai_summary"`
	HasAgentDoc bool     `json:"has_agent_doc"`
}

// ProvenanceInfo records who/what triggered a package version and the approval chain.
// Stored in PackageMetadata for audit trail. All fields optional for backward compat.
type ProvenanceInfo struct {
	TriggerMessageID string   `json:"trigger_message_id,omitempty"` // Message that started this update
	CorrelationIDs   []string `json:"correlation_ids,omitempty"`    // Full message chain
	AgentTraceID     string   `json:"agent_trace_id,omitempty"`     // OTEL trace ID
	ChainID          string   `json:"chain_id,omitempty"`           // Observatory chain ID
	ApprovedBy       string   `json:"approved_by,omitempty"`        // Human approver GitHub handle
	ApprovedAt       string   `json:"approved_at,omitempty"`        // ISO 8601 timestamp
	AutoApproved     bool     `json:"auto_approved"`                // true for class A
	ChangeClass      string   `json:"change_class,omitempty"`       // "A", "B", or "C"
	PreviousVersion  string   `json:"previous_version,omitempty"`   // Version before this update
}

// VersionHistory records the full message and action trail for a published version.
// Stored as history.json in the registry, surfaced on the website for package discovery.
type VersionHistory struct {
	Schema    string         `json:"schema"`     // "ailang.version-history/v1"
	Package   string         `json:"package"`    // "sunholo/auth"
	Version   string         `json:"version"`    // "0.2.0"
	Previous  string         `json:"previous"`   // "0.1.0"
	CreatedAt string         `json:"created_at"` // ISO 8601
	Messages  []HistoryEntry `json:"messages"`   // Ordered message trail
	Summary   string         `json:"summary"`    // AI-generated 1-line summary
}

// HistoryEntry is a single event in the version's message/action trail.
type HistoryEntry struct {
	Timestamp string `json:"timestamp"`
	Kind      string `json:"kind"`   // Message kind or action type
	From      string `json:"from"`   // Who sent/did it
	Title     string `json:"title"`  // Brief description
	Detail    string `json:"detail"` // Full content
	Status    string `json:"status"` // "received", "acknowledged", "completed", "failed"
}

// VersionHistorySchema is the canonical schema identifier for history.json.
const VersionHistorySchema = "ailang.version-history/v1"
