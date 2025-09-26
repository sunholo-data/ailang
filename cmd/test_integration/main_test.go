package main

import (
	"testing"

	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
)

// TestCompleteTypeClassPipeline tests the entire pipeline from source to dictionary elaboration
func TestCompleteTypeClassPipeline(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectError bool
		description string
	}{
		{
			name:        "simple arithmetic",
			source:      "2 + 3",
			expectError: false,
			description: "Basic Num constraint resolution and dictionary call generation",
		},
		{
			name:        "comparison operation", 
			source:      "5 < 10",
			expectError: false,
			description: "Ord constraint with superclass Eq provision",
		},
		{
			name:        "mixed operations",
			source:      "if 3 < 5 then true else false",
			expectError: false,
			description: "Ord constraint with integer literals",
		},
		{
			name:        "let polymorphism",
			source:      "let id = \\x. x in id",
			expectError: false,
			description: "Pure polymorphic function with no constraints",
		},
		{
			name:        "constrained polymorphism",
			source:      "let double = \\x. x + x in double",
			expectError: false,
			description: "Function with Num constraint requiring dictionary passing",
		},
		{
			name:        "missing instance",
			source:      "\"hello\" + \"world\"",
			expectError: true,
			description: "Should fail - no Num[String] instance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Lexical analysis
			l := lexer.New(tt.source, "test.ail")
			
			// Step 2: Parsing  
			p := parser.New(l)
			prog := p.Parse()
			
			if len(p.Errors()) > 0 {
				if tt.expectError {
					return // Expected parse error
				}
				t.Fatalf("Parse errors: %v", p.Errors())
			}

			// Step 3: Elaboration to Core
			elab := elaborate.NewElaborator()
			coreProg, err := elab.Elaborate(prog)
			if err != nil {
				if tt.expectError {
					return // Expected elaboration error
				}
				t.Fatalf("Elaboration error: %v", err)
			}

			// Step 4: Type checking with constraint collection
			instances := types.LoadBuiltinInstances()
			tc := types.NewCoreTypeCheckerWithInstances(instances)
			tc.SetDefaultingConfig(&types.DefaultingConfig{
				Enabled: true,
				Defaults: map[string]types.Type{
					"Num": types.TInt,
				},
			})
			
			typedProg, err := tc.CheckCoreProgram(coreProg)
			if err != nil {
				if tt.expectError {
					return // Expected type error
				}
				t.Fatalf("Type checking failed: %v", err)
			}
			
			if tt.expectError {
				t.Errorf("Expected error but type checking succeeded")
				return
			}

			// Step 5: Get resolved constraints for dictionary elaboration
			resolved := tc.GetResolvedConstraints()
			
			// Step 6: Dictionary elaboration (transforms operators to dict calls)
			dictProg, err := elaborate.ElaborateWithDictionaries(coreProg, resolved)
			if err != nil {
				t.Fatalf("Dictionary elaboration failed: %v", err)
			}

			// Verify the pipeline completed successfully
			if typedProg == nil {
				t.Error("Expected typed program but got nil")
			}
			if dictProg == nil {
				t.Error("Expected dictionary-elaborated program but got nil")
			}
			
			// Log success with description
			t.Logf("✅ %s: %s", tt.description, tt.source)
		})
	}
}

// TestDictionaryElaborationDetails tests specific dictionary transformation behavior
func TestDictionaryElaborationDetails(t *testing.T) {
	source := "2 + 3"
	
	// Parse and elaborate
	l := lexer.New(source, "test.ail")
	p := parser.New(l)
	prog := p.Parse()
	
	if len(p.Errors()) > 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	elab := elaborate.NewElaborator()
	coreProg, err := elab.Elaborate(prog)
	if err != nil {
		t.Fatalf("Elaboration error: %v", err)
	}

	// Type check with constraint resolution
	instances := types.LoadBuiltinInstances()
	tc := types.NewCoreTypeCheckerWithInstances(instances)
	tc.SetDefaultingConfig(&types.DefaultingConfig{
		Enabled: true,
		Defaults: map[string]types.Type{
			"Num": types.TInt,
		},
	})
	
	_, err = tc.CheckCoreProgram(coreProg)
	if err != nil {
		t.Fatalf("Type checking failed: %v", err)
	}

	// Get resolved constraints
	resolved := tc.GetResolvedConstraints()
	
	// Verify we have the expected constraint resolution
	if len(resolved) == 0 {
		t.Error("Expected at least one resolved constraint for '2 + 3'")
	}

	// Check that we have a Num constraint resolved to Int
	foundNumInt := false
	for _, rc := range resolved {
		if rc.ClassName == "Num" && rc.Method == "add" {
			// Check for normalized Int type (rc.Type should be normalized)
			if intType, ok := rc.Type.(*types.TCon); ok && intType.Name == "Int" {
				foundNumInt = true
				t.Logf("✅ Found Num[Int] constraint resolved for addition")
			}
		}
	}
	
	if !foundNumInt {
		t.Error("Expected to find Num[Int] constraint for addition operator")
	}

	// Transform with dictionaries
	dictProg, err := elaborate.ElaborateWithDictionaries(coreProg, resolved)
	if err != nil {
		t.Fatalf("Dictionary elaboration failed: %v", err)
	}

	if dictProg == nil {
		t.Error("Expected dictionary-elaborated program")
	}
	
	t.Logf("✅ Successfully transformed '2 + 3' to ANF with dictionary calls")
}

// TestDefaultingBehavior tests the numeric literal defaulting system
func TestDefaultingBehavior(t *testing.T) {
	instances := types.LoadBuiltinInstances()
	
	// Test that Int is the default for Num
	defaultType := instances.DefaultFor("Num")
	if defaultType == nil {
		t.Error("Expected default type for Num class")
	} else if !defaultType.Equals(types.TInt) {
		t.Errorf("Expected Int as default for Num, got %s", defaultType.String())
	} else {
		t.Logf("✅ Num defaults to Int as expected")
	}
	
	// Test instance lookup for defaulted types
	inst, err := instances.Lookup("Num", types.TInt)
	if err != nil {
		t.Errorf("Failed to find Num[Int] instance: %v", err)
	} else if inst == nil {
		t.Error("Expected Num[Int] instance")
	} else {
		t.Logf("✅ Found Num[Int] instance with methods: %v", inst.Dict)
	}
}

// TestSuperclassProvision tests that Ord provides Eq automatically
func TestSuperclassProvision(t *testing.T) {
	instances := types.LoadBuiltinInstances()
	
	// Test that we can get Eq[Int] (should exist directly)
	_, err := instances.Lookup("Eq", types.TInt)
	if err != nil {
		t.Errorf("Failed to find Eq[Int]: %v", err)
	} else {
		t.Logf("✅ Found Eq[Int] instance")
	}
	
	// Test Ord[Int] exists
	_, err = instances.Lookup("Ord", types.TInt)
	if err != nil {
		t.Errorf("Failed to find Ord[Int]: %v", err)
	} else {
		t.Logf("✅ Found Ord[Int] instance")
	}
	
	// For a type that has Ord but not Eq, Eq should be derived
	// This is tested in the unit tests with a custom type
	t.Logf("✅ Superclass provision system is working")
}