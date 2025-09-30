package parser

import (
	"flag"
	"testing"
)

// TestMain provides setup/teardown for all parser tests
func TestMain(m *testing.M) {
	// Parse flags
	flag.Parse()

	// Run tests
	m.Run()
}

// TestSmoke is a minimal smoke test to verify test infrastructure works
func TestSmoke(t *testing.T) {
	input := "42"

	prog := mustParse(t, input)
	output := parseAndPrint(t, input)

	// Verify we got a program back
	if prog == nil {
		t.Fatal("Expected non-nil program")
	}

	// Verify output is non-empty
	if output == "" {
		t.Fatal("Expected non-empty output")
	}

	// Test golden compare (will create file on first run with -update)
	goldenCompare(t, "smoke/int_literal", output)

	t.Logf("Smoke test passed. Infrastructure is working.")
	t.Logf("Run 'go test -update ./internal/parser' to update golden files")
}
