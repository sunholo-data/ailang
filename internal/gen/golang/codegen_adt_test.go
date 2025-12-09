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
		1: &types.TFunc{
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
