// Package sid provides Stable ID calculation for AST nodes
package sid

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// SID represents a Stable Identifier for an AST node
type SID string

// NewSID calculates a stable ID for an AST node
// Formula: hash(canonical_path | start_offset | end_offset | node_kind | child_path)
func NewSID(path string, start, end int, kind string, childPath []int) SID {
	// Canonicalize the path
	canonPath := canonicalizePath(path)

	// Build the hash input
	var parts []string
	parts = append(parts, canonPath)
	parts = append(parts, fmt.Sprintf("%d", start))
	parts = append(parts, fmt.Sprintf("%d", end))
	parts = append(parts, kind)

	// Add child path
	for _, idx := range childPath {
		parts = append(parts, fmt.Sprintf("%d", idx))
	}

	// Hash the combined string
	input := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(input))

	// Return first 16 hex chars for brevity
	return SID(hex.EncodeToString(hash[:])[:16])
}

// canonicalizePath normalizes a file path for stable SID calculation
func canonicalizePath(path string) string {
	// Clean the path
	path = filepath.Clean(path)

	// Resolve symlinks if possible
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}

	// Make path absolute if not already
	if !filepath.IsAbs(path) {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}

	// On case-insensitive filesystems (Windows, macOS), normalize to lowercase
	// This is for SID stability only - actual resolution uses real FS semantics
	if isCaseInsensitive() {
		path = strings.ToLower(path)
	}

	// Use forward slashes consistently
	path = filepath.ToSlash(path)

	return path
}

// isCaseInsensitive checks if we're on a case-insensitive filesystem
func isCaseInsensitive() bool {
	return runtime.GOOS == "windows" || runtime.GOOS == "darwin"
}

// SIDMap maintains the mapping from surface SIDs to core SIDs
type SIDMap struct {
	SurfaceToCore map[SID][]SID
	CoreToSurface map[SID]SID
}

// NewSIDMap creates a new SID mapping
func NewSIDMap() *SIDMap {
	return &SIDMap{
		SurfaceToCore: make(map[SID][]SID),
		CoreToSurface: make(map[SID]SID),
	}
}

// AddMapping records a surface→core SID mapping
func (m *SIDMap) AddMapping(surfaceSID SID, coreSID SID) {
	m.SurfaceToCore[surfaceSID] = append(m.SurfaceToCore[surfaceSID], coreSID)
	m.CoreToSurface[coreSID] = surfaceSID
}

// GetCoreSIDs returns all core SIDs derived from a surface SID
func (m *SIDMap) GetCoreSIDs(surfaceSID SID) []SID {
	return m.SurfaceToCore[surfaceSID]
}

// GetSurfaceSID returns the surface SID that generated a core SID
func (m *SIDMap) GetSurfaceSID(coreSID SID) (SID, bool) {
	sid, ok := m.CoreToSurface[coreSID]
	return sid, ok
}

// TraceSlice represents the transformation path from surface to core
type TraceSlice struct {
	SurfaceSID  SID
	SurfaceKind string
	CoreSIDs    []SID
	CoreKinds   []string
	Steps       []TransformStep
}

// TransformStep represents one step in the transformation
type TransformStep struct {
	Description string
	FromSID     SID
	ToSID       SID
}

// GetTraceSlice returns the transformation trace for a surface SID
func (m *SIDMap) GetTraceSlice(surfaceSID SID) *TraceSlice {
	coreSIDs := m.GetCoreSIDs(surfaceSID)

	trace := &TraceSlice{
		SurfaceSID: surfaceSID,
		CoreSIDs:   coreSIDs,
	}

	// Build transformation steps
	for i, coreSID := range coreSIDs {
		if i == 0 {
			trace.Steps = append(trace.Steps, TransformStep{
				Description: "Initial elaboration",
				FromSID:     surfaceSID,
				ToSID:       coreSID,
			})
		} else {
			trace.Steps = append(trace.Steps, TransformStep{
				Description: "Further transformation",
				FromSID:     coreSIDs[i-1],
				ToSID:       coreSID,
			})
		}
	}

	return trace
}
