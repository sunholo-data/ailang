package testing

import "testing"

// TestFunctionlessNamedTestsResolveStdlibBuiltins pins the ailang test path for
// modules whose first Core declaration is a named-test body rather than a function.
func TestFunctionlessNamedTestsResolveStdlibBuiltins(t *testing.T) {
	const source = `module test_builtin_resolution

import std/list (any, length)

test "delegating export" { length([1, 2, 3]) == 3 }
test "non-delegating control" { any(\x. x == 2, [1, 2, 3]) }
test "intentional failure control" { false }
`

	result := runInlineTestsOnSource(t, source)
	if len(result.Tests) == 0 {
		t.Fatal("instrument failure")
	}
	if len(result.Tests) != 3 {
		t.Fatalf("expected 3 test results, got %d", len(result.Tests))
	}

	want := map[string]TestStatus{
		"delegating export":           StatusPass,
		"non-delegating control":      StatusPass,
		"intentional failure control": StatusFail,
	}
	for _, got := range result.Tests {
		status, ok := want[got.Name]
		if !ok {
			t.Fatalf("unexpected test result %q", got.Name)
		}
		if got.Status != status {
			t.Errorf("%s: expected %s, got %s: %s", got.Name, status, got.Status, got.Error)
		}
		delete(want, got.Name)
	}
	if len(want) != 0 {
		t.Fatalf("instrument failure: missing test results: %v", want)
	}
}
