package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/builtins"
)

// TestBuiltinTypes_GoldenSnapshot ensures builtin type signatures don't change accidentally.
//
// This is a CRITICAL regression guard that catches:
// - Lost effect rows (e.g., ! {IO} → pure)
// - Signature changes (e.g., string -> int instead of string -> string)
// - Arity changes (e.g., 1 param → 2 params)
// - New builtins added without review
//
// The golden file is a single consolidated snapshot of all builtin signatures.
// Any change to builtin types will cause this test to fail with a clear diff.
//
// To update the golden file (after intentional changes):
//
//	UPDATE_GOLDEN=1 go test -v ./internal/pipeline -run TestBuiltinTypes_GoldenSnapshot
func TestBuiltinTypes_GoldenSnapshot(t *testing.T) {
	specs := builtins.AllSpecs()

	// Generate current snapshot
	var current strings.Builder
	current.WriteString("# AILANG Builtin Type Signatures\n")
	current.WriteString("# Auto-generated golden file - DO NOT EDIT MANUALLY\n")
	current.WriteString("# To update: UPDATE_GOLDEN=1 go test -v ./internal/pipeline -run TestBuiltinTypes_GoldenSnapshot\n")
	current.WriteString("#\n")
	current.WriteString("# Format: <name> : <type_signature>\n")
	current.WriteString("#\n\n")

	// Sort names for deterministic output
	names := make([]string, 0, len(specs))
	for name := range specs {
		names = append(names, name)
	}
	sort.Strings(names)

	// Generate type signatures
	for _, name := range names {
		spec := specs[name]
		typ := spec.Type()

		// Use the type's String() method for canonical representation
		signature := typ.String()

		current.WriteString(fmt.Sprintf("%s : %s\n", name, signature))
	}

	// Path to golden file
	goldenPath := filepath.Join("testdata", "builtin_types.golden")

	// Update mode
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		// Ensure directory exists
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create testdata directory: %v", err)
		}

		// Write golden file
		err := os.WriteFile(goldenPath, []byte(current.String()), 0644)
		require.NoError(t, err, "Failed to write golden file")

		t.Logf("✅ Golden file updated: %s", goldenPath)
		t.Logf("📝 %d builtin signatures saved", len(names))
		return
	}

	// Verify mode - compare with golden
	goldenContent, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Fatalf("Golden file missing: %s\n\n"+
			"Run this to generate it:\n"+
			"  UPDATE_GOLDEN=1 go test -v ./internal/pipeline -run TestBuiltinTypes_GoldenSnapshot\n",
			goldenPath)
	}
	require.NoError(t, err, "Failed to read golden file")

	expectedStr := string(goldenContent)
	actualStr := current.String()

	if expectedStr != actualStr {
		// Generate a helpful diff
		diff := generateDiff(expectedStr, actualStr)

		t.Fatalf("❌ Builtin type signatures have changed!\n\n"+
			"This means:\n"+
			"  - A builtin signature was modified (effect added/removed, type changed)\n"+
			"  - A builtin was added or removed\n"+
			"  - Arity changed (number of parameters)\n\n"+
			"Diff:\n%s\n\n"+
			"If this change is intentional, update the golden file:\n"+
			"  UPDATE_GOLDEN=1 go test -v ./internal/pipeline -run TestBuiltinTypes_GoldenSnapshot\n\n"+
			"If this change is NOT intentional, you may have introduced a regression!\n",
			diff)
	}

	t.Logf("✅ All %d builtin type signatures match golden file", len(names))
}

// generateDiff creates a simple line-by-line diff showing what changed
func generateDiff(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	var diff strings.Builder
	diff.WriteString("Expected vs Actual:\n\n")

	// Build maps for quick lookup
	expectedMap := make(map[string]bool)
	actualMap := make(map[string]bool)

	for _, line := range expectedLines {
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		expectedMap[line] = true
	}

	for _, line := range actualLines {
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		actualMap[line] = true
	}

	// Find removed lines
	for _, line := range expectedLines {
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		if !actualMap[line] {
			diff.WriteString(fmt.Sprintf("  - %s\n", line))
		}
	}

	// Find added lines
	for _, line := range actualLines {
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		if !expectedMap[line] {
			diff.WriteString(fmt.Sprintf("  + %s\n", line))
		}
	}

	// Show unchanged count
	unchanged := 0
	for _, line := range actualLines {
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		if expectedMap[line] {
			unchanged++
		}
	}

	diff.WriteString(fmt.Sprintf("\n  (%d signatures unchanged)\n", unchanged))

	return diff.String()
}

// TestBuiltinTypes_CriticalSignatures is a focused test for the most important builtins.
// This provides a fast smoke test that runs even if the golden file is missing.
func TestBuiltinTypes_CriticalSignatures(t *testing.T) {
	specs := builtins.AllSpecs()

	critical := map[string]string{
		"_io_print":        "String -> () ! {IO}",
		"_io_println":      "String -> () ! {IO}",
		"_io_readLine":     "() -> String ! {IO}",
		"_net_httpRequest": "(String, String, List[{name: String, value: String}], String) -> Result[{body: String, headers: List[{name: String, value: String}], ok: Bool, status: Int}, NetError] ! {Net}",
		"_str_len":         "String -> Int",
		"concat_String":    "(String, String) -> String",
	}

	for name, expectedSig := range critical {
		t.Run(name, func(t *testing.T) {
			spec, ok := specs[name]
			require.True(t, ok, "Critical builtin %s is missing!", name)

			typ := spec.Type()
			actualSig := typ.String()

			require.Equal(t, expectedSig, actualSig,
				"CRITICAL REGRESSION: %s signature changed!\n"+
					"Expected: %s\n"+
					"Got:      %s\n\n"+
					"This may indicate lost effect rows or type changes.",
				name, expectedSig, actualSig)
		})
	}

	t.Logf("✅ All %d critical builtin signatures are correct", len(critical))
}
