package types_test

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
)

// Simple ground check helper for tests
func isGround(t types.Type) bool {
	// A type is ground if it contains no type variables
	// This is a simplified check for testing
	typeStr := t.String()
	// Check for type variable patterns (Greek letters, t followed by numbers)
	return !strings.ContainsAny(typeStr, "αβγδ") &&
		!strings.Contains(typeStr, "t1") &&
		!strings.Contains(typeStr, "t2") &&
		!strings.Contains(typeStr, "t3")
}

func TestResolvedConstraints_AreGround(t *testing.T) {
	src := `let r = 1 + 2 in r`

	l := lexer.New(src, "<test>")
	p := parser.New(l)
	surf := p.Parse()
	if errors := p.Errors(); len(errors) > 0 {
		t.Fatalf("parse: %v", errors[0])
	}

	el := elaborate.NewElaborator()
	cp, err := el.Elaborate(surf)
	if err != nil {
		t.Fatalf("elaborate: %v", err)
	}

	tc := types.NewCoreTypeCheckerWithInstances(types.LoadBuiltinInstances())

	// Type check all expressions in the program
	env := types.NewTypeEnvWithBuiltins()
	for _, decl := range cp.Decls {
		_, _, err := tc.CheckCoreExpr(decl, env)
		if err != nil {
			t.Fatalf("typecheck: %v", err)
		}
	}

	rc := tc.GetResolvedConstraints()
	if len(rc) == 0 {
		t.Fatalf("expected resolved constraints, got none")
	}

	for nodeID, c := range rc {
		if !isGround(c.Type) {
			t.Fatalf("constraint at node %d not ground: %s", nodeID, c.Type.String())
		}
		t.Logf("✓ Ground constraint at node %d: %s[%s] -> %s", nodeID, c.ClassName, c.Type.String(), c.Method)
	}
}

func TestResolvedConstraints_OperatorMethods(t *testing.T) {
	testCases := []struct {
		name     string
		src      string
		expected map[string]bool // method names we expect to see
	}{
		{
			name:     "addition",
			src:      `let r = 1 + 2 in r`,
			expected: map[string]bool{"add": true},
		},
		{
			name:     "comparison",
			src:      `let r = 1 < 2 in r`,
			expected: map[string]bool{"lt": true},
		},
		{
			name:     "equality",
			src:      `let r = 1 == 2 in r`,
			expected: map[string]bool{"eq": true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New(tc.src, "<test>")
			p := parser.New(l)
			surf := p.Parse()
			if errors := p.Errors(); len(errors) > 0 {
				t.Fatalf("parse: %v", errors[0])
			}

			el := elaborate.NewElaborator()
			cp, err := el.Elaborate(surf)
			if err != nil {
				t.Fatalf("elaborate: %v", err)
			}

			typechecker := types.NewCoreTypeCheckerWithInstances(types.LoadBuiltinInstances())

			env := types.NewTypeEnvWithBuiltins()
			for _, decl := range cp.Decls {
				_, _, err := typechecker.CheckCoreExpr(decl, env)
				if err != nil {
					t.Fatalf("typecheck: %v", err)
				}
			}

			rc := typechecker.GetResolvedConstraints()
			foundMethods := make(map[string]bool)

			for _, c := range rc {
				if c.Method != "" {
					foundMethods[c.Method] = true
				}
			}

			for expectedMethod := range tc.expected {
				if !foundMethods[expectedMethod] {
					t.Errorf("Expected to find method %q in resolved constraints", expectedMethod)
				}
			}
		})
	}
}
