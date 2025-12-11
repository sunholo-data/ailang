package types

import (
	"testing"
)

func TestTypeReport_NotFound(t *testing.T) {
	tc := NewCoreTypeChecker()

	report := tc.TypeReport(999)

	if report.Found {
		t.Error("expected Found=false for non-existent node")
	}
	if report.Raw != nil {
		t.Error("expected Raw=nil for non-existent node")
	}
	if report.Resolved != nil {
		t.Error("expected Resolved=nil for non-existent node")
	}
}

func TestTypeReport_BasicType(t *testing.T) {
	tc := NewCoreTypeChecker()

	// Add a type to CoreTI
	tc.CoreTI[42] = TInt

	report := tc.TypeReport(42)

	if !report.Found {
		t.Fatal("expected Found=true for existing node")
	}
	if report.Raw != TInt {
		t.Errorf("expected Raw=int, got %v", report.Raw)
	}
	if report.Resolved != TInt {
		t.Errorf("expected Resolved=int, got %v", report.Resolved)
	}
}

func TestTypeReport_WithTVar(t *testing.T) {
	tc := NewCoreTypeChecker()

	// Add a type variable to CoreTI
	tvar := &TVar{Name: "α1"}
	tc.CoreTI[42] = tvar

	report := tc.TypeReport(42)

	if !report.Found {
		t.Fatal("expected Found=true")
	}
	if report.Raw != tvar {
		t.Errorf("expected Raw=α1, got %v", report.Raw)
	}
	// Without substitution, resolved equals raw
	if report.Resolved.String() != "α1" {
		t.Errorf("expected Resolved=α1 (no sub), got %v", report.Resolved)
	}
}

func TestTypeReport_WithConstraint(t *testing.T) {
	tc := NewCoreTypeChecker()

	// Add a type and a resolved constraint
	tc.CoreTI[42] = TInt
	tc.resolvedConstraints[42] = &ResolvedConstraint{
		NodeID:    42,
		ClassName: "Num",
		Type:      TInt,
		Method:    "add",
	}

	report := tc.TypeReport(42)

	if !report.Found {
		t.Fatal("expected Found=true")
	}
	if len(report.Constraints) != 1 {
		t.Fatalf("expected 1 constraint, got %d", len(report.Constraints))
	}

	c := report.Constraints[0]
	if c.ClassName != "Num" {
		t.Errorf("expected ClassName=Num, got %s", c.ClassName)
	}
	if c.Type != TInt {
		t.Errorf("expected Type=int, got %v", c.Type)
	}
	if c.Method != "add" {
		t.Errorf("expected Method=add, got %s", c.Method)
	}
	if !c.Resolved {
		t.Error("expected Resolved=true")
	}
}

func TestTypeReport_String(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.CoreTI[42] = TFloat

	report := tc.TypeReport(42)
	str := report.String()

	if str == "" {
		t.Error("expected non-empty string")
	}
	t.Logf("TypeReport.String(): %s", str)

	// Check it contains expected parts
	if !contains(str, "NodeID:42") {
		t.Error("expected string to contain NodeID:42")
	}
	if !contains(str, "float") {
		t.Error("expected string to contain float")
	}
}

func TestTypeReport_NotFound_String(t *testing.T) {
	tc := NewCoreTypeChecker()

	report := tc.TypeReport(999)
	str := report.String()

	if str != "TypeReport{not found}" {
		t.Errorf("expected 'TypeReport{not found}', got '%s'", str)
	}
}

func TestOriginKind_String(t *testing.T) {
	tests := []struct {
		kind OriginKind
		want string
	}{
		{OriginUnknown, "unknown"},
		{OriginAnnotation, "annotation"},
		{OriginLiteral, "literal"},
		{OriginInferred, "inferred"},
		{OriginDefaulted, "defaulted"},
		{OriginFromUse, "from_use"},
		{OriginFromPattern, "from_pattern"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("OriginKind(%d).String() = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
