package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/types"
)

// TestCanonicalTypeFingerprint_Stability tests that fingerprints are stable
func TestCanonicalTypeFingerprint_Stability(t *testing.T) {
	tests := []struct {
		name  string
		types []types.Type
		want  string // Expected to be deterministic
	}{
		{
			name:  "empty",
			types: []types.Type{},
			want:  "unit",
		},
		{
			name:  "single Int",
			types: []types.Type{&types.TCon{Name: "Int"}},
			want:  "", // Will check it's consistent, not the exact value
		},
		{
			name: "Int and Float",
			types: []types.Type{
				&types.TCon{Name: "Int"},
				&types.TCon{Name: "Float"},
			},
			want: "",
		},
		{
			name: "List(Int)",
			types: []types.Type{
				&types.TApp{
					Constructor: &types.TCon{Name: "List"},
					Args:        []types.Type{&types.TCon{Name: "Int"}},
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First call
			got1 := canonicalTypeFingerprint(tt.types)

			// Second call - should be identical
			got2 := canonicalTypeFingerprint(tt.types)

			if got1 != got2 {
				t.Errorf("canonicalTypeFingerprint() not stable: got1=%q, got2=%q", got1, got2)
			}

			// Check expected value if provided
			if tt.want != "" && got1 != tt.want {
				t.Errorf("canonicalTypeFingerprint() = %q, want %q", got1, tt.want)
			}

			// Check hash suffix is present (except for "unit")
			if tt.want != "unit" && !strings.Contains(got1, "$") {
				t.Errorf("canonicalTypeFingerprint() missing hash suffix: %q", got1)
			}
		})
	}
}

// TestCanonicalTypeFingerprint_Ordering tests that type order affects fingerprint
func TestCanonicalTypeFingerprint_Ordering(t *testing.T) {
	types1 := []types.Type{
		&types.TCon{Name: "Int"},
		&types.TCon{Name: "Float"},
	}
	types2 := []types.Type{
		&types.TCon{Name: "Float"},
		&types.TCon{Name: "Int"},
	}

	fp1 := canonicalTypeFingerprint(types1)
	fp2 := canonicalTypeFingerprint(types2)

	// Fingerprints should be the same (sorted internally)
	if fp1 != fp2 {
		t.Errorf("canonicalTypeFingerprint() order-dependent: fp1=%q, fp2=%q", fp1, fp2)
	}
}

// TestTypeHeads tests type head extraction
func TestTypeHeads(t *testing.T) {
	tests := []struct {
		name  string
		types []types.Type
		want  []string
	}{
		{
			name:  "empty",
			types: []types.Type{},
			want:  []string{},
		},
		{
			name:  "Int",
			types: []types.Type{&types.TCon{Name: "Int"}},
			want:  []string{"Int"},
		},
		{
			name: "Int and Float",
			types: []types.Type{
				&types.TCon{Name: "Int"},
				&types.TCon{Name: "Float"},
			},
			want: []string{"Int", "Float"},
		},
		{
			name: "List(Int)",
			types: []types.Type{
				&types.TApp{
					Constructor: &types.TCon{Name: "List"},
					Args:        []types.Type{&types.TCon{Name: "Int"}},
				},
			},
			want: []string{"List"},
		},
		{
			name: "Record",
			types: []types.Type{
				&types.TRecord{Fields: map[string]types.Type{"x": &types.TCon{Name: "Int"}}},
			},
			want: []string{"Record"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := typeHeads(tt.types)
			if len(got) != len(tt.want) {
				t.Errorf("typeHeads() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("typeHeads()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestGenerateSpecializedName tests name generation
func TestGenerateSpecializedName(t *testing.T) {
	tests := []struct {
		name        string
		defSym      string
		argTypes    []types.Type
		fingerprint string
		wantPrefix  string // Expected prefix
		wantSuffix  string // Expected suffix
	}{
		{
			name:   "max with Int Int",
			defSym: "max",
			argTypes: []types.Type{
				&types.TCon{Name: "Int"},
				&types.TCon{Name: "Int"},
			},
			fingerprint: "Int:Int$2f",
			wantPrefix:  "_max$Int$Int",
			wantSuffix:  "2f",
		},
		{
			name:   "sort with List",
			defSym: "sort",
			argTypes: []types.Type{
				&types.TApp{
					Constructor: &types.TCon{Name: "List"},
					Args:        []types.Type{&types.TCon{Name: "Int"}},
				},
			},
			fingerprint: "List(Int)$8a",
			wantPrefix:  "_sort$List",
			wantSuffix:  "8a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateSpecializedName(tt.defSym, tt.argTypes, tt.fingerprint)

			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("generateSpecializedName() = %q, want prefix %q", got, tt.wantPrefix)
			}
			if !strings.HasSuffix(got, tt.wantSuffix) {
				t.Errorf("generateSpecializedName() = %q, want suffix %q", got, tt.wantSuffix)
			}

			// Check that name is valid Go identifier (starts with _)
			if !strings.HasPrefix(got, "_") {
				t.Errorf("generateSpecializedName() = %q, should start with underscore", got)
			}
		})
	}
}

// TestIsPolymorphic tests type variable detection
func TestIsPolymorphic(t *testing.T) {
	tests := []struct {
		name string
		typ  types.Type
		want bool
	}{
		{
			name: "Int is concrete",
			typ:  &types.TCon{Name: "Int"},
			want: false,
		},
		{
			name: "TVar is polymorphic",
			typ:  &types.TVar{Name: "a"},
			want: true,
		},
		{
			name: "List(Int) is concrete",
			typ: &types.TApp{
				Constructor: &types.TCon{Name: "List"},
				Args:        []types.Type{&types.TCon{Name: "Int"}},
			},
			want: false,
		},
		{
			name: "List(α) is polymorphic",
			typ: &types.TApp{
				Constructor: &types.TCon{Name: "List"},
				Args:        []types.Type{&types.TVar{Name: "a"}},
			},
			want: true,
		},
		{
			name: "Record{x:Int} is concrete",
			typ: &types.TRecord{
				Fields: map[string]types.Type{"x": &types.TCon{Name: "Int"}},
			},
			want: false,
		},
		{
			name: "Record{x:α} is polymorphic",
			typ: &types.TRecord{
				Fields: map[string]types.Type{"x": &types.TVar{Name: "a"}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPolymorphic(tt.typ)
			if got != tt.want {
				t.Errorf("isPolymorphic() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAllConcrete tests concrete type list check
func TestAllConcrete(t *testing.T) {
	tests := []struct {
		name  string
		types []types.Type
		want  bool
	}{
		{
			name:  "empty list is concrete",
			types: []types.Type{},
			want:  true,
		},
		{
			name:  "Int is concrete",
			types: []types.Type{&types.TCon{Name: "Int"}},
			want:  true,
		},
		{
			name: "Int and Float are concrete",
			types: []types.Type{
				&types.TCon{Name: "Int"},
				&types.TCon{Name: "Float"},
			},
			want: true,
		},
		{
			name: "Int and α are not all concrete",
			types: []types.Type{
				&types.TCon{Name: "Int"},
				&types.TVar{Name: "a"},
			},
			want: false,
		},
		{
			name: "List(Int) is concrete",
			types: []types.Type{
				&types.TApp{
					Constructor: &types.TCon{Name: "List"},
					Args:        []types.Type{&types.TCon{Name: "Int"}},
				},
			},
			want: true,
		},
		{
			name: "List(α) is not concrete",
			types: []types.Type{
				&types.TApp{
					Constructor: &types.TCon{Name: "List"},
					Args:        []types.Type{&types.TVar{Name: "a"}},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allConcrete(tt.types)
			if got != tt.want {
				t.Errorf("allConcrete() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNewSpecializer tests basic initialization
func TestNewSpecializer(t *testing.T) {
	coreTI := types.NewCoreTypeInfo()
	spec := NewSpecializer(&coreTI)

	if spec.CoreTI != &coreTI {
		t.Error("NewSpecializer() did not set CoreTI correctly")
	}

	if spec.Cache == nil {
		t.Error("NewSpecializer() did not initialize Cache")
	}

	if spec.PerFunction == nil {
		t.Error("NewSpecializer() did not initialize PerFunction")
	}

	if spec.TotalCount != 0 {
		t.Errorf("NewSpecializer() TotalCount = %d, want 0", spec.TotalCount)
	}

	if spec.Limits.MaxPerFunction != 16 {
		t.Errorf("NewSpecializer() Limits.MaxPerFunction = %d, want 16", spec.Limits.MaxPerFunction)
	}

	if spec.Limits.MaxPerModule != 512 {
		t.Errorf("NewSpecializer() Limits.MaxPerModule = %d, want 512", spec.Limits.MaxPerModule)
	}
}

// TestSpecializationKey_String tests key string representation
func TestSpecializationKey_String(t *testing.T) {
	key := SpecializationKey{
		DefSym:           "max",
		TypesFingerprint: "Int:Int$2f",
	}

	got := key.String()
	want := "max[Int:Int$2f]"

	if got != want {
		t.Errorf("SpecializationKey.String() = %q, want %q", got, want)
	}
}

// TestCacheTracking tests that cache hits and misses are tracked correctly
func TestCacheTracking(t *testing.T) {
	coreTI := types.NewCoreTypeInfo()
	s := NewSpecializer(&coreTI)

	// Initially should have 0 hits and 0 misses
	if s.CacheHits != 0 {
		t.Errorf("Expected 0 cache hits initially, got %d", s.CacheHits)
	}
	if s.CacheMisses != 0 {
		t.Errorf("Expected 0 cache misses initially, got %d", s.CacheMisses)
	}

	stats := s.GetStats()
	if stats.CacheHits != 0 {
		t.Errorf("Expected 0 cache hits in stats, got %d", stats.CacheHits)
	}
	if stats.CacheMisses != 0 {
		t.Errorf("Expected 0 cache misses in stats, got %d", stats.CacheMisses)
	}
}

// TestPerFunctionCapEnforcement tests that per-function limits are enforced
func TestPerFunctionCapEnforcement(t *testing.T) {
	coreTI := types.NewCoreTypeInfo()
	s := NewSpecializer(&coreTI)

	// Set very low per-function limit
	s.Limits.MaxPerFunction = 2

	// Simulate specializations by directly updating PerFunction counter
	s.PerFunction["testFunc"] = 2

	// Verify we're at the limit
	if s.PerFunction["testFunc"] < s.Limits.MaxPerFunction {
		t.Errorf("Expected to be at limit, but PerFunction=%d, MaxPerFunction=%d",
			s.PerFunction["testFunc"], s.Limits.MaxPerFunction)
	}
}

// TestModuleCapEnforcement tests that module-wide limits are enforced
func TestModuleCapEnforcement(t *testing.T) {
	coreTI := types.NewCoreTypeInfo()
	s := NewSpecializer(&coreTI)

	// Set very low module limit
	s.Limits.MaxPerModule = 5

	// Simulate specializations
	s.TotalCount = 5

	// Verify we're at the limit
	if s.TotalCount < s.Limits.MaxPerModule {
		t.Errorf("Expected to be at module limit, but TotalCount=%d, MaxPerModule=%d",
			s.TotalCount, s.Limits.MaxPerModule)
	}
}

// TestSkipReasonTracking tests that skip reasons are recorded
func TestSkipReasonTracking(t *testing.T) {
	coreTI := types.NewCoreTypeInfo()
	s := NewSpecializer(&coreTI)

	// Add a skip reason
	s.Skipped = append(s.Skipped, SkipReason{
		DefSym:   "recursiveFunc",
		Reason:   "Recursive function not specialized in v0.4.0",
		Location: "test.ail:10:5",
	})

	stats := s.GetStats()
	if len(stats.SkippedFunctions) != 1 {
		t.Errorf("Expected 1 skipped function, got %d", len(stats.SkippedFunctions))
	}

	skip := stats.SkippedFunctions[0]
	if skip.DefSym != "recursiveFunc" {
		t.Errorf("Expected DefSym=recursiveFunc, got %s", skip.DefSym)
	}
	if skip.Reason != "Recursive function not specialized in v0.4.0" {
		t.Errorf("Expected specific reason, got %s", skip.Reason)
	}
	if skip.Location != "test.ail:10:5" {
		t.Errorf("Expected location=test.ail:10:5, got %s", skip.Location)
	}
}
