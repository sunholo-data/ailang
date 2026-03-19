package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// M-DX12: Test ADT slice converter generation
func TestGenerateADTSliceConverter(t *testing.T) {
	// Register an ADT type for slice conversion
	prog := &core.Program{
		Decls: []core.CoreExpr{},
	}

	gen := New("game")
	// Register ADT slice type - this is what happens when [DrawCmd] is encountered
	gen.RegisterADTSliceType("DrawCmd")
	gen.RegisterADTSliceType("Camera")

	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should have DrawCmd converter (M-BUGFIX: exported with capital C)
	if !strings.Contains(codeStr, "func ConvertToDrawCmdSlice(v interface{}) []*DrawCmd") {
		t.Errorf("Missing ConvertToDrawCmdSlice function, got:\n%s", codeStr)
	}

	// Should have Camera converter (M-BUGFIX: exported with capital C)
	if !strings.Contains(codeStr, "func ConvertToCameraSlice(v interface{}) []*Camera") {
		t.Errorf("Missing ConvertToCameraSlice function, got:\n%s", codeStr)
	}

	// Should have fail-fast panic
	if !strings.Contains(codeStr, "panic(fmt.Sprintf") {
		t.Errorf("Missing panic for fail-fast, got:\n%s", codeStr)
	}

	// Should have empty slice handling
	if !strings.Contains(codeStr, "[]*DrawCmd{}") {
		t.Errorf("Missing empty slice return, got:\n%s", codeStr)
	}

	// Should be deterministic (sorted alphabetically)
	cameraIdx := strings.Index(codeStr, "ConvertToCameraSlice")
	drawCmdIdx := strings.Index(codeStr, "ConvertToDrawCmdSlice")
	if cameraIdx == -1 || drawCmdIdx == -1 {
		t.Errorf("Missing converters")
	}
	if cameraIdx > drawCmdIdx {
		t.Errorf("Converters should be sorted alphabetically (Camera before DrawCmd), got Camera at %d, DrawCmd at %d", cameraIdx, drawCmdIdx)
	}
}

// TestRecordReturnType tests M-BUGFIX: Functions returning record types should
// generate correct Go return types (e.g., *BridgeState instead of struct{}).
func TestRecordReturnType(t *testing.T) {
	// Create a Lambda that "returns a record type"
	// We need to set up CoreTypeInfo to indicate the return type is a TRecord
	lam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 1},
		Params:   []string{"x"},
		Body:     &core.Lit{Kind: core.IntLit, Value: int64(42)}, // placeholder body
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name:  "initBridge",
				Value: lam,
				Body:  &core.Var{Name: "initBridge"},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"initBridge": {IsExport: true},
		},
	}

	gen := New("test")

	// Register a record type (simulating type BridgeState = { x: int, y: int })
	gen.RegisterRecordType("BridgeState", []string{"X", "Y"}, map[string]string{
		"X": "int64",
		"Y": "int64",
	})

	// Set up CoreTypeInfo with the Lambda returning a TRecord with matching fields
	cti := types.CoreTypeInfo{
		1: &types.TFunc2{
			Params: []types.Type{&types.TCon{Name: "int"}},
			Return: &types.TRecord{
				Fields: map[string]types.Type{
					"x": &types.TCon{Name: "int"},
					"y": &types.TCon{Name: "int"},
				},
			},
		},
	}
	gen.SetCoreTypeInfo(cti)

	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// The typed wrapper should have *BridgeState as return type
	// Check for the specific function signature
	if !strings.Contains(codeStr, "func InitBridge(x int64) *BridgeState") {
		t.Errorf("Expected 'func InitBridge(x int64) *BridgeState', got:\n%s", codeStr)
	}
	// Make sure the return type is not struct{} in the InitBridge function
	if strings.Contains(codeStr, "func InitBridge(x int64) struct{}") {
		t.Errorf("Expected *BridgeState return type, but got struct{} in InitBridge")
	}
}

// TestADTConstructorQualifiedLookup tests M-DX22: Same-named constructors in
// different ADTs should be disambiguated by type-qualified keys.
func TestADTConstructorQualifiedLookup(t *testing.T) {
	gen := New("test")

	// Register two ADTs with same-named constructor "Viewport"
	// DrawCmd.Viewport has 5 fields, WindowType.Viewport has 0 fields
	gen.RegisterADTConstructorFull("DrawCmd", "Viewport",
		[]string{"string", "int64", "int64", "int64", "int64"},
		[]string{"name", "x", "y", "w", "h"})
	gen.RegisterADTConstructorFull("WindowType", "Viewport", nil, nil)

	// Qualified lookup should find the correct one
	drawInfo, ok := gen.LookupADTConstructor("DrawCmd", "Viewport")
	if !ok {
		t.Fatal("Expected to find DrawCmd.Viewport")
	}
	if drawInfo.TypeName != "DrawCmd" {
		t.Errorf("Expected TypeName=DrawCmd, got %s", drawInfo.TypeName)
	}
	if len(drawInfo.FieldTypes) != 5 {
		t.Errorf("Expected 5 field types for DrawCmd.Viewport, got %d", len(drawInfo.FieldTypes))
	}

	windowInfo, ok := gen.LookupADTConstructor("WindowType", "Viewport")
	if !ok {
		t.Fatal("Expected to find WindowType.Viewport")
	}
	if windowInfo.TypeName != "WindowType" {
		t.Errorf("Expected TypeName=WindowType, got %s", windowInfo.TypeName)
	}
	if len(windowInfo.FieldTypes) != 0 {
		t.Errorf("Expected 0 field types for WindowType.Viewport, got %d", len(windowInfo.FieldTypes))
	}

	// Unqualified lookup should still work (returns first match, but both exist)
	anyInfo, ok := gen.LookupADTConstructor("", "Viewport")
	if !ok {
		t.Fatal("Expected unqualified lookup to find a Viewport constructor")
	}
	if anyInfo.CtorName != "Viewport" {
		t.Errorf("Expected CtorName=Viewport, got %s", anyInfo.CtorName)
	}

	// LookupADTConstructorByQualifiedName should work with full key
	drawByKey, ok := gen.LookupADTConstructorByQualifiedName("DrawCmd.Viewport")
	if !ok {
		t.Fatal("Expected LookupADTConstructorByQualifiedName to find DrawCmd.Viewport")
	}
	if drawByKey.TypeName != "DrawCmd" {
		t.Errorf("Expected TypeName=DrawCmd from qualified name lookup, got %s", drawByKey.TypeName)
	}

	windowByKey, ok := gen.LookupADTConstructorByQualifiedName("WindowType.Viewport")
	if !ok {
		t.Fatal("Expected LookupADTConstructorByQualifiedName to find WindowType.Viewport")
	}
	if windowByKey.TypeName != "WindowType" {
		t.Errorf("Expected TypeName=WindowType from qualified name lookup, got %s", windowByKey.TypeName)
	}
}
