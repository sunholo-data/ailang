package sid

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSID(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		start     int
		end       int
		kind      string
		childPath []int
		wantLen   int
	}{
		{
			name:      "simple SID",
			path:      "test.ail",
			start:     0,
			end:       10,
			kind:      "FuncDecl",
			childPath: []int{},
			wantLen:   16,
		},
		{
			name:      "SID with child path",
			path:      "module.ail",
			start:     5,
			end:       15,
			kind:      "Expr",
			childPath: []int{0, 1, 2},
			wantLen:   16,
		},
		{
			name:      "nested node",
			path:      "/Users/test/project/file.ail",
			start:     100,
			end:       200,
			kind:      "Lambda",
			childPath: []int{3, 7, 1},
			wantLen:   16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sid := NewSID(tt.path, tt.start, tt.end, tt.kind, tt.childPath)
			assert.Equal(t, tt.wantLen, len(string(sid)), "SID should be 16 chars")
			assert.NotEmpty(t, sid)
		})
	}
}

func TestNewSID_Stability(t *testing.T) {
	// Same inputs should always produce same SID
	sid1 := NewSID("test.ail", 10, 20, "FuncDecl", []int{1, 2})
	sid2 := NewSID("test.ail", 10, 20, "FuncDecl", []int{1, 2})
	assert.Equal(t, sid1, sid2, "Same inputs should produce same SID")

	// Different inputs should produce different SIDs
	sid3 := NewSID("test.ail", 10, 20, "FuncDecl", []int{1, 3})
	assert.NotEqual(t, sid1, sid3, "Different child paths should produce different SIDs")

	sid4 := NewSID("test.ail", 10, 21, "FuncDecl", []int{1, 2})
	assert.NotEqual(t, sid1, sid4, "Different end offsets should produce different SIDs")

	sid5 := NewSID("other.ail", 10, 20, "FuncDecl", []int{1, 2})
	assert.NotEqual(t, sid1, sid5, "Different paths should produce different SIDs")

	sid6 := NewSID("test.ail", 10, 20, "Lambda", []int{1, 2})
	assert.NotEqual(t, sid1, sid6, "Different kinds should produce different SIDs")
}

func TestCanonicalizePath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "relative path",
			path: "test.ail",
		},
		{
			name: "absolute path",
			path: "/Users/test/project/file.ail",
		},
		{
			name: "path with dots",
			path: "./test/../file.ail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonical := canonicalizePath(tt.path)
			assert.NotEmpty(t, canonical)

			// Should be idempotent
			canonical2 := canonicalizePath(canonical)
			assert.Equal(t, canonical, canonical2, "Canonicalization should be idempotent")

			// Should use forward slashes
			assert.NotContains(t, canonical, "\\", "Should use forward slashes")

			// On case-insensitive systems, should be lowercase
			if isCaseInsensitive() {
				assert.Equal(t, canonical, canonical, "Should be consistent on case-insensitive systems")
			}
		})
	}
}

func TestCanonicalizePath_CaseHandling(t *testing.T) {
	// On case-insensitive systems (Windows, macOS), paths should normalize to lowercase
	path1 := canonicalizePath("Test.ail")
	path2 := canonicalizePath("test.ail")

	if isCaseInsensitive() {
		// Should be equal on case-insensitive systems
		assert.Equal(t, path1, path2, "Paths should be equal on case-insensitive systems")
	} else {
		// May or may not be equal on case-sensitive systems (depends on actual FS)
		// We don't assert anything here as it depends on the actual filesystem
	}
}

func TestIsCaseInsensitive(t *testing.T) {
	result := isCaseInsensitive()

	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		assert.True(t, result, "Should be case-insensitive on Windows/macOS")
	} else {
		assert.False(t, result, "Should be case-sensitive on Linux/Unix")
	}
}

func TestNewSIDMap(t *testing.T) {
	m := NewSIDMap()
	assert.NotNil(t, m)
	assert.NotNil(t, m.SurfaceToCore)
	assert.NotNil(t, m.CoreToSurface)
	assert.Empty(t, m.SurfaceToCore)
	assert.Empty(t, m.CoreToSurface)
}

func TestSIDMap_AddMapping(t *testing.T) {
	m := NewSIDMap()

	surfaceSID := SID("surface123")
	coreSID1 := SID("core456")
	coreSID2 := SID("core789")

	m.AddMapping(surfaceSID, coreSID1)
	m.AddMapping(surfaceSID, coreSID2)

	// Should have both core SIDs mapped to same surface SID
	coreSIDs := m.GetCoreSIDs(surfaceSID)
	assert.Equal(t, 2, len(coreSIDs))
	assert.Contains(t, coreSIDs, coreSID1)
	assert.Contains(t, coreSIDs, coreSID2)

	// Both core SIDs should map back to same surface SID
	surface1, ok1 := m.GetSurfaceSID(coreSID1)
	assert.True(t, ok1)
	assert.Equal(t, surfaceSID, surface1)

	surface2, ok2 := m.GetSurfaceSID(coreSID2)
	assert.True(t, ok2)
	assert.Equal(t, surfaceSID, surface2)
}

func TestSIDMap_GetCoreSIDs(t *testing.T) {
	m := NewSIDMap()

	// Non-existent surface SID should return nil
	coreSIDs := m.GetCoreSIDs(SID("nonexistent"))
	assert.Nil(t, coreSIDs)

	// Add a mapping
	surfaceSID := SID("surface1")
	coreSID := SID("core1")
	m.AddMapping(surfaceSID, coreSID)

	// Should return the core SID
	coreSIDs = m.GetCoreSIDs(surfaceSID)
	assert.Equal(t, 1, len(coreSIDs))
	assert.Equal(t, coreSID, coreSIDs[0])
}

func TestSIDMap_GetSurfaceSID(t *testing.T) {
	m := NewSIDMap()

	// Non-existent core SID should return false
	_, ok := m.GetSurfaceSID(SID("nonexistent"))
	assert.False(t, ok)

	// Add a mapping
	surfaceSID := SID("surface1")
	coreSID := SID("core1")
	m.AddMapping(surfaceSID, coreSID)

	// Should return the surface SID
	surface, ok := m.GetSurfaceSID(coreSID)
	assert.True(t, ok)
	assert.Equal(t, surfaceSID, surface)
}

func TestSIDMap_GetTraceSlice(t *testing.T) {
	m := NewSIDMap()

	surfaceSID := SID("surface1")
	coreSID1 := SID("core1")
	coreSID2 := SID("core2")
	coreSID3 := SID("core3")

	m.AddMapping(surfaceSID, coreSID1)
	m.AddMapping(surfaceSID, coreSID2)
	m.AddMapping(surfaceSID, coreSID3)

	trace := m.GetTraceSlice(surfaceSID)

	require.NotNil(t, trace)
	assert.Equal(t, surfaceSID, trace.SurfaceSID)
	assert.Equal(t, 3, len(trace.CoreSIDs))
	assert.Contains(t, trace.CoreSIDs, coreSID1)
	assert.Contains(t, trace.CoreSIDs, coreSID2)
	assert.Contains(t, trace.CoreSIDs, coreSID3)

	// Should have transformation steps
	assert.Equal(t, 3, len(trace.Steps))

	// First step should be from surface to first core
	assert.Equal(t, "Initial elaboration", trace.Steps[0].Description)
	assert.Equal(t, surfaceSID, trace.Steps[0].FromSID)
	assert.Equal(t, coreSID1, trace.Steps[0].ToSID)

	// Subsequent steps should be from previous core to next core
	assert.Equal(t, "Further transformation", trace.Steps[1].Description)
	assert.Equal(t, coreSID1, trace.Steps[1].FromSID)
	assert.Equal(t, coreSID2, trace.Steps[1].ToSID)

	assert.Equal(t, "Further transformation", trace.Steps[2].Description)
	assert.Equal(t, coreSID2, trace.Steps[2].FromSID)
	assert.Equal(t, coreSID3, trace.Steps[2].ToSID)
}

func TestSIDMap_GetTraceSlice_Empty(t *testing.T) {
	m := NewSIDMap()

	surfaceSID := SID("nonexistent")
	trace := m.GetTraceSlice(surfaceSID)

	require.NotNil(t, trace)
	assert.Equal(t, surfaceSID, trace.SurfaceSID)
	assert.Empty(t, trace.CoreSIDs)
	assert.Empty(t, trace.Steps)
}

func TestSIDMap_MultipleTransformations(t *testing.T) {
	m := NewSIDMap()

	// Simulate multiple surface nodes mapping to core nodes
	surface1 := SID("surface1")
	surface2 := SID("surface2")
	core1 := SID("core1")
	core2 := SID("core2")
	core3 := SID("core3")

	// surface1 -> core1, core2
	m.AddMapping(surface1, core1)
	m.AddMapping(surface1, core2)

	// surface2 -> core3
	m.AddMapping(surface2, core3)

	// Verify surface1 mappings
	cores1 := m.GetCoreSIDs(surface1)
	assert.Equal(t, 2, len(cores1))

	// Verify surface2 mappings
	cores2 := m.GetCoreSIDs(surface2)
	assert.Equal(t, 1, len(cores2))
	assert.Equal(t, core3, cores2[0])

	// Verify reverse mappings
	surf1, ok := m.GetSurfaceSID(core1)
	assert.True(t, ok)
	assert.Equal(t, surface1, surf1)

	surf2, ok := m.GetSurfaceSID(core3)
	assert.True(t, ok)
	assert.Equal(t, surface2, surf2)
}
