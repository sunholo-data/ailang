// Package manifest provides types and validation for AILANG example manifests.
// The manifest system ensures documentation stays in sync with reality.
package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/schema"
)

// SchemaVersion is the current manifest schema version
const SchemaVersion = "ailang.manifest/v1"

// Status represents the status of an example
type Status string

const (
	StatusWorking      Status = "working"
	StatusBroken       Status = "broken"
	StatusExperimental Status = "experimental"
)

// Mode represents how an example should be executed
type Mode string

const (
	ModeFile Mode = "file"
	ModeREPL Mode = "repl"
)

// Environment captures execution environment settings
type Environment struct {
	Seed     int    `json:"seed"`
	Locale   string `json:"locale"`
	Timezone string `json:"timezone"`
}

// Expected captures expected output for validation
type Expected struct {
	Stdout       string `json:"stdout,omitempty"`
	Stderr       string `json:"stderr,omitempty"`
	ExitCode     int    `json:"exit_code"`
	ErrorPattern string `json:"error_pattern,omitempty"`
}

// BrokenInfo provides details about why an example is broken
type BrokenInfo struct {
	Reason       string   `json:"reason"`
	ErrorCode    string   `json:"error_code"`
	Requires     []string `json:"requires"`
	TrackedIssue string   `json:"tracked_issue,omitempty"`
}

// Example represents a single example file in the manifest
type Example struct {
	Path             string       `json:"path"`
	Status           Status       `json:"status"`
	Mode             Mode         `json:"mode"`
	Tags             []string     `json:"tags,omitempty"`
	Description      string       `json:"description,omitempty"`
	Expected         *Expected    `json:"expected,omitempty"`
	Environment      *Environment `json:"environment,omitempty"`
	Broken           *BrokenInfo  `json:"broken,omitempty"`
	RequiresFeatures []string     `json:"requires_features,omitempty"`
	SkipReason       string       `json:"skip_reason,omitempty"`
}

// Statistics provides aggregate information about examples
type Statistics struct {
	Total        int     `json:"total"`
	Working      int     `json:"working"`
	Broken       int     `json:"broken"`
	Experimental int     `json:"experimental"`
	Coverage     float64 `json:"coverage"`
}

// Manifest represents the complete example manifest
type Manifest struct {
	Schema        string     `json:"schema"`
	SchemaVersion string     `json:"schema_version"`
	SchemaDigest  string     `json:"schema_digest"`
	GeneratedAt   time.Time  `json:"generated_at"`
	Generator     string     `json:"generator"`
	Examples      []Example  `json:"examples"`
	Statistics    Statistics `json:"statistics"`
}

// New creates a new manifest with defaults
func New() *Manifest {
	return &Manifest{
		Schema:        SchemaVersion,
		SchemaVersion: "1.0.0",
		GeneratedAt:   time.Now().UTC(),
		Generator:     "ailang verify-examples",
		Examples:      []Example{},
		Statistics:    Statistics{},
	}
}

// Load reads and validates a manifest from a file
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("manifest validation failed: %w", err)
	}

	return &m, nil
}

// Save writes the manifest to a file with deterministic JSON
func (m *Manifest) Save(path string) error {
	// Update statistics before saving
	m.UpdateStatistics()

	// Calculate schema digest
	m.UpdateSchemaDigest()

	// Sort examples for deterministic output
	sort.Slice(m.Examples, func(i, j int) bool {
		return m.Examples[i].Path < m.Examples[j].Path
	})

	// Marshal with deterministic keys
	data, err := schema.MarshalDeterministic(m)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Pretty print with indentation
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return err
	}

	return os.WriteFile(path, append(buf.Bytes(), '\n'), 0644)
}

// Validate checks the manifest for consistency
func (m *Manifest) Validate() error {
	// Check schema version
	if !schema.Accepts(m.Schema, SchemaVersion) {
		return fmt.Errorf("unsupported schema version: %s (expected %s)", m.Schema, SchemaVersion)
	}

	// Verify schema digest if present
	if m.SchemaDigest != "" {
		expected := m.calculateSchemaDigest()
		if m.SchemaDigest != expected {
			return fmt.Errorf("schema digest mismatch: got %s, expected %s", m.SchemaDigest, expected)
		}
	}

	// Check for duplicate paths
	seen := make(map[string]bool)
	for _, ex := range m.Examples {
		if seen[ex.Path] {
			return fmt.Errorf("duplicate example path: %s", ex.Path)
		}
		seen[ex.Path] = true

		// Validate example
		if err := m.validateExample(ex); err != nil {
			return fmt.Errorf("invalid example %s: %w", ex.Path, err)
		}
	}

	// Verify statistics match
	stats := m.calculateStatistics()
	if m.Statistics != stats {
		return fmt.Errorf("statistics mismatch: recorded %+v, calculated %+v", m.Statistics, stats)
	}

	return nil
}

// validateExample validates a single example entry
func (m *Manifest) validateExample(ex Example) error {
	// Check required fields
	if ex.Path == "" {
		return fmt.Errorf("missing path")
	}
	if ex.Status == "" {
		return fmt.Errorf("missing status")
	}
	if ex.Mode == "" {
		return fmt.Errorf("missing mode")
	}

	// Validate status
	switch ex.Status {
	case StatusWorking:
		if ex.Expected == nil {
			return fmt.Errorf("working example missing expected output")
		}
		if ex.Broken != nil {
			return fmt.Errorf("working example should not have broken info")
		}
	case StatusBroken:
		if ex.Broken == nil {
			return fmt.Errorf("broken example missing broken info")
		}
		if ex.Broken.ErrorCode == "" {
			return fmt.Errorf("broken example missing error code")
		}
	case StatusExperimental:
		// Experimental can have various states
	default:
		return fmt.Errorf("invalid status: %s", ex.Status)
	}

	// Validate mode
	switch ex.Mode {
	case ModeFile, ModeREPL:
		// Valid
	default:
		return fmt.Errorf("invalid mode: %s", ex.Mode)
	}

	// Check file extension
	if !strings.HasSuffix(ex.Path, ".ail") {
		return fmt.Errorf("example must have .ail extension")
	}

	return nil
}

// UpdateStatistics recalculates the statistics
func (m *Manifest) UpdateStatistics() {
	m.Statistics = m.calculateStatistics()
}

// calculateStatistics computes statistics from examples
func (m *Manifest) calculateStatistics() Statistics {
	stats := Statistics{Total: len(m.Examples)}

	for _, ex := range m.Examples {
		switch ex.Status {
		case StatusWorking:
			stats.Working++
		case StatusBroken:
			stats.Broken++
		case StatusExperimental:
			stats.Experimental++
		}
	}

	if stats.Total > 0 {
		stats.Coverage = float64(stats.Working) / float64(stats.Total)
	}

	return stats
}

// UpdateSchemaDigest recalculates the schema digest
func (m *Manifest) UpdateSchemaDigest() {
	m.SchemaDigest = m.calculateSchemaDigest()
}

// calculateSchemaDigest computes a SHA256 digest of the schema
func (m *Manifest) calculateSchemaDigest() string {
	// Create a canonical representation of the schema
	schemaData := fmt.Sprintf("%s:%s", m.Schema, m.SchemaVersion)
	hash := sha256.Sum256([]byte(schemaData))
	return "sha256:" + hex.EncodeToString(hash[:])[:16] // First 16 chars of hex
}

// FindExample locates an example by path
func (m *Manifest) FindExample(path string) (*Example, bool) {
	for i := range m.Examples {
		if m.Examples[i].Path == path {
			return &m.Examples[i], true
		}
	}
	return nil, false
}

// GetWorkingExamples returns all working examples
func (m *Manifest) GetWorkingExamples() []Example {
	var working []Example
	for _, ex := range m.Examples {
		if ex.Status == StatusWorking {
			working = append(working, ex)
		}
	}
	return working
}

// GetBrokenExamples returns all broken examples
func (m *Manifest) GetBrokenExamples() []Example {
	var broken []Example
	for _, ex := range m.Examples {
		if ex.Status == StatusBroken {
			broken = append(broken, ex)
		}
	}
	return broken
}

// GenerateREADMESection generates the status table for README
func (m *Manifest) GenerateREADMESection() string {
	var buf strings.Builder

	buf.WriteString("## Example Status\n\n")
	buf.WriteString("_Generated from manifest.json - do not edit manually_\n\n")

	// Summary
	buf.WriteString(fmt.Sprintf("**Coverage: %.1f%%** (%d/%d working)\n\n",
		m.Statistics.Coverage*100,
		m.Statistics.Working,
		m.Statistics.Total))

	// Working examples
	if working := m.GetWorkingExamples(); len(working) > 0 {
		buf.WriteString("### ✅ Working Examples\n\n")
		buf.WriteString("| File | Description | Mode |\n")
		buf.WriteString("|------|-------------|------|\n")
		for _, ex := range working {
			desc := ex.Description
			if desc == "" {
				desc = filepath.Base(ex.Path)
			}
			buf.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", ex.Path, desc, ex.Mode))
		}
		buf.WriteString("\n")
	}

	// Broken examples
	if broken := m.GetBrokenExamples(); len(broken) > 0 {
		buf.WriteString("### ❌ Broken Examples\n\n")
		buf.WriteString("| File | Reason | Required Features | Issue |\n")
		buf.WriteString("|------|--------|-------------------|-------|\n")
		for _, ex := range broken {
			requires := strings.Join(ex.Broken.Requires, ", ")
			issue := ex.Broken.TrackedIssue
			if issue != "" {
				// Make it a link if it's a URL
				if strings.HasPrefix(issue, "http") {
					parts := strings.Split(issue, "/")
					issueNum := parts[len(parts)-1]
					issue = fmt.Sprintf("[#%s](%s)", issueNum, issue)
				}
			}
			buf.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n",
				ex.Path, ex.Broken.Reason, requires, issue))
		}
		buf.WriteString("\n")
	}

	// Experimental examples
	var experimental []Example
	for _, ex := range m.Examples {
		if ex.Status == StatusExperimental {
			experimental = append(experimental, ex)
		}
	}
	if len(experimental) > 0 {
		buf.WriteString("### 🧪 Experimental Examples\n\n")
		buf.WriteString("| File | Required Features | Note |\n")
		buf.WriteString("|------|-------------------|------|\n")
		for _, ex := range experimental {
			features := strings.Join(ex.RequiresFeatures, ", ")
			buf.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n",
				ex.Path, features, ex.SkipReason))
		}
		buf.WriteString("\n")
	}

	buf.WriteString(fmt.Sprintf("_Last updated: %s_\n", m.GeneratedAt.Format("2006-01-02 15:04:05 UTC")))

	return buf.String()
}
