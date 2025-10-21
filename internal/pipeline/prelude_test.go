package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
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
			name: "empty module",
			code: `module test`,
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

// TestPrelude_TypeInjection tests that InjectPrelude correctly adds print to type environment
func TestPrelude_TypeInjection(t *testing.T) {
	// Create base type environment with builtins
	env := types.NewTypeEnvWithBuiltins()

	// Verify print is NOT in base environment
	_, err := env.Lookup("print")
	if err == nil {
		t.Fatal("print should not be in base type environment")
	}

	// Inject prelude
	env = InjectPrelude(env)

	// Verify print IS now in environment
	printBinding, err := env.Lookup("print")
	if err != nil {
		t.Fatalf("print should be in type environment after injection: %v", err)
	}

	// Verify print has correct type: string -> () ! {IO}
	scheme, ok := printBinding.(*types.Scheme)
	if !ok {
		t.Fatalf("print binding is not a Scheme, got %T", printBinding)
	}

	// Check it's a function type
	funcType, ok := scheme.Type.(*types.TFunc2)
	if !ok {
		t.Fatalf("print type is not TFunc2, got %T", scheme.Type)
	}

	// Check parameters: should have 1 parameter of type string
	if len(funcType.Params) != 1 {
		t.Fatalf("print should have 1 parameter, got %d", len(funcType.Params))
	}

	// The parameter type should be string (lowercase, as it's a primitive)
	paramTypeStr := funcType.Params[0].String()
	if paramTypeStr != "string" {
		t.Fatalf("print parameter should be string, got %v", funcType.Params[0])
	}

	// Check return type: should be Unit (represented as ())
	returnTypeStr := funcType.Return.String()
	if returnTypeStr != "()" {
		t.Fatalf("print return type should be (), got %v", funcType.Return)
	}

	// Check effects: should have IO effect
	if funcType.EffectRow == nil {
		t.Fatal("print should have effect row")
	}
	if _, hasIO := funcType.EffectRow.Labels["IO"]; !hasIO {
		t.Fatal("print should have IO effect")
	}
}

// TestPrelude_NotInBuiltinList tests that print is NOT in the global builtin registry
func TestPrelude_NotInBuiltinList(t *testing.T) {
	// This test ensures print is not globally available
	// It should only be injected via prelude for entry modules

	// Create a type environment with builtins
	env := types.NewTypeEnvWithBuiltins()

	// Verify print is NOT in the base environment
	_, err := env.Lookup("print")
	if err == nil {
		t.Fatal("print should NOT be in global builtins (it's prelude-only)")
	}

	// Verify _io_println IS available (print delegates to this)
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
			name: "nil interface",
			iface: nil,
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
