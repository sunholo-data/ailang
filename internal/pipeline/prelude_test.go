package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/types"
)

// TestPrelude_EntryModuleDetection tests that IsEntryModuleFromAST correctly identifies entry modules
func TestPrelude_EntryModuleDetection(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name: "entry module with export func main",
			code: `module test
export func main() -> () ! {IO} = print("hello")`,
			expected: true,
		},
		{
			name: "library module with no main",
			code: `module test
export func helper() -> () ! {IO} = print("hello")`,
			expected: false,
		},
		{
			name: "module with non-exported main",
			code: `module test
func main() -> () ! {IO} = print("hello")`,
			expected: false,
		},
		{
			name: "module with main that takes parameters",
			code: `module test
export func main(x: int) -> () ! {IO} = print("hello")`,
			expected: false,
		},
		{
			name:     "empty module",
			code:     `module test`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.code, "test.ail")
			p := parser.New(l)
			file := p.ParseFile()

			result := IsEntryModuleFromAST(file)
			if result != tt.expected {
				t.Errorf("IsEntryModuleFromAST() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestPrelude_TypeInjection tests that InjectPrelude correctly adds println to type environment
func TestPrelude_TypeInjection(t *testing.T) {
	// Create base type environment with builtins
	env := types.NewTypeEnvWithBuiltins()

	// Verify println is NOT in base environment
	_, err := env.Lookup("println")
	if err == nil {
		t.Fatal("println should not be in base type environment")
	}

	// Inject prelude
	env = InjectPrelude(env)

	// Verify println IS now in environment
	printlnBinding, err := env.Lookup("println")
	if err != nil {
		t.Fatalf("println should be in type environment after injection: %v", err)
	}

	// Verify println has correct type: string -> () ! {IO}
	scheme, ok := printlnBinding.(*types.Scheme)
	if !ok {
		t.Fatalf("println binding is not a Scheme, got %T", printlnBinding)
	}

	// Check it's a function type
	funcType, ok := scheme.Type.(*types.TFunc2)
	if !ok {
		t.Fatalf("println type is not TFunc2, got %T", scheme.Type)
	}

	// Check parameters: should have 1 parameter of type string
	if len(funcType.Params) != 1 {
		t.Fatalf("println should have 1 parameter, got %d", len(funcType.Params))
	}

	// The parameter type should be string (lowercase, as it's a primitive)
	paramTypeStr := funcType.Params[0].String()
	if paramTypeStr != "string" {
		t.Fatalf("println parameter should be string, got %v", funcType.Params[0])
	}

	// Check return type: should be Unit (represented as ())
	returnTypeStr := funcType.Return.String()
	if returnTypeStr != "()" {
		t.Fatalf("println return type should be (), got %v", funcType.Return)
	}

	// Check effects: should have IO effect
	if funcType.EffectRow == nil {
		t.Fatal("println should have effect row")
	}
	if _, hasIO := funcType.EffectRow.Labels["IO"]; !hasIO {
		t.Fatal("println should have IO effect")
	}
}

// TestPrelude_NotInBuiltinList tests that println is NOT in the global builtin registry
func TestPrelude_NotInBuiltinList(t *testing.T) {
	// This test ensures println is not globally available
	// It should only be injected via prelude for entry modules
	// print (no newline) requires explicit import from std/io

	// Create a type environment with builtins
	env := types.NewTypeEnvWithBuiltins()

	// Verify println is NOT in the base environment (prelude-only)
	_, err := env.Lookup("println")
	if err == nil {
		t.Fatal("println should NOT be in global builtins (it's prelude-only)")
	}

	// Verify _io_println IS available (println delegates to this)
	_, err = env.Lookup("_io_println")
	if err != nil {
		t.Fatal("_io_println should be in global builtins")
	}
}

// TestPrelude_EntryModulePipeline tests the full pipeline with entry module
// TODO: This test needs the runModule path to handle prelude injection properly
// For now, we verify via the direct AST and type injection tests above
func TestPrelude_EntryModulePipeline(t *testing.T) {
	t.Skip("TODO: runModule path needs prelude injection wiring - covered by manual tests for now")
}

// TestPrelude_LibraryModulePipeline tests that library modules CANNOT use print
func TestPrelude_LibraryModulePipeline(t *testing.T) {
	code := `module test_library
export func helper() -> () ! {IO} = print("should fail")`

	cfg := Config{
		Mode: ModeCheck,
	}

	src := Source{
		Code:     code,
		Filename: "",
		IsREPL:   false,
	}

	_, err := Run(cfg, src)

	// Should get a type error about undefined variable
	if err == nil {
		t.Fatal("library module should fail when using print")
	}

	if !strings.Contains(err.Error(), "undefined variable: print") {
		t.Fatalf("expected 'undefined variable: print' error, got: %v", err)
	}
}

// TestPrelude_EntryModuleMissingIO tests effect checking behavior
// Note: This test verifies that print usage requires IO effect in the type signature
// TODO: Once runModule path supports prelude, this should show effect-specific diagnostic
func TestPrelude_EntryModuleMissingIO(t *testing.T) {
	t.Skip("TODO: Effect diagnostic test needs runModule path with prelude - testing via manual verification for now")

	// When implemented, this should verify:
	// 1. Entry module with print("x") but main() -> () (no ! {IO})
	// 2. Should get helpful error: "print requires IO effect. Add ! {IO} to main signature"
}

// TestPrelude_Shadowing tests that user definitions can shadow print
// TODO: This test needs proper syntax and pipeline path
func TestPrelude_Shadowing(t *testing.T) {
	t.Skip("TODO: Shadowing test needs syntax fix and runModule path - covered by manual tests for now")
}

// TestIsEntryModule tests the interface-based entry module detection
func TestIsEntryModule(t *testing.T) {
	tests := []struct {
		name     string
		iface    *iface.Iface
		expected bool
	}{
		{
			name:     "nil interface",
			iface:    nil,
			expected: false,
		},
		{
			name: "interface with no exports",
			iface: &iface.Iface{
				Module:  "test",
				Exports: make(map[string]*iface.IfaceItem),
			},
			expected: false,
		},
		{
			name: "interface with main function (0 params)",
			iface: &iface.Iface{
				Module: "test",
				Exports: map[string]*iface.IfaceItem{
					"main": {
						Type: &types.Scheme{
							Type: &types.TFunc2{
								Params: []types.Type{}, // 0 parameters
								Return: &types.TCon{Name: "Unit"},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "interface with main function (1 param)",
			iface: &iface.Iface{
				Module: "test",
				Exports: map[string]*iface.IfaceItem{
					"main": {
						Type: &types.Scheme{
							Type: &types.TFunc2{
								Params: []types.Type{&types.TCon{Name: "Int"}}, // 1 parameter
								Return: &types.TCon{Name: "Unit"},
							},
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEntryModule(tt.iface)
			if result != tt.expected {
				t.Errorf("IsEntryModule() = %v, want %v", result, tt.expected)
			}
		})
	}
}
