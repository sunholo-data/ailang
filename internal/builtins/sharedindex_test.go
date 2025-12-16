package builtins

import (
	"testing"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// createSharedIndexContext creates a test context with SharedIndex enabled
func createSharedIndexContext() *effects.EffContext {
	return &effects.EffContext{
		SharedIndex: effects.NewSharedIndexContext(nil),
	}
}

func TestSharedIndexUpsert(t *testing.T) {
	ctx := createSharedIndexContext()

	args := []eval.Value{
		&eval.StringValue{Value: "beliefs"},
		&eval.StringValue{Value: "belief1"},
		&eval.IntValue{Value: 12345},
		&eval.IntValue{Value: 1},
		&eval.IntValue{Value: 1000},
	}

	result, err := sharedIndexUpsertImpl(ctx, args)
	if err != nil {
		t.Fatalf("sharedIndexUpsertImpl failed: %v", err)
	}

	if _, ok := result.(*eval.UnitValue); !ok {
		t.Errorf("expected UnitValue, got %T", result)
	}

	// Verify entry was stored
	count := ctx.SharedIndex.Index.EntryCount("beliefs")
	if count != 1 {
		t.Errorf("expected 1 entry, got %d", count)
	}

	// Verify upsert count incremented
	if ctx.SharedIndex.UpsertCount != 1 {
		t.Errorf("expected UpsertCount=1, got %d", ctx.SharedIndex.UpsertCount)
	}
}

func TestSharedIndexUpsert_NoCapability(t *testing.T) {
	ctx := &effects.EffContext{
		SharedIndex: nil, // Not enabled
	}

	args := []eval.Value{
		&eval.StringValue{Value: "beliefs"},
		&eval.StringValue{Value: "belief1"},
		&eval.IntValue{Value: 12345},
		&eval.IntValue{Value: 1},
		&eval.IntValue{Value: 1000},
	}

	_, err := sharedIndexUpsertImpl(ctx, args)
	if err == nil {
		t.Fatal("expected error when SharedIndex not enabled")
	}
	if err.Error() != "_sharedindex_upsert: SharedIndex effect not enabled (use --caps SharedIndex)" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSharedIndexDelete(t *testing.T) {
	ctx := createSharedIndexContext()

	// First upsert an entry
	ctx.SharedIndex.Index.Upsert("beliefs", "belief1", 12345, 1, 1000)

	args := []eval.Value{
		&eval.StringValue{Value: "beliefs"},
		&eval.StringValue{Value: "belief1"},
	}

	result, err := sharedIndexDeleteImpl(ctx, args)
	if err != nil {
		t.Fatalf("sharedIndexDeleteImpl failed: %v", err)
	}

	if _, ok := result.(*eval.UnitValue); !ok {
		t.Errorf("expected UnitValue, got %T", result)
	}

	// Verify entry was deleted
	count := ctx.SharedIndex.Index.EntryCount("beliefs")
	if count != 0 {
		t.Errorf("expected 0 entries after delete, got %d", count)
	}
}

func TestSharedIndexFindSimHash(t *testing.T) {
	ctx := createSharedIndexContext()

	// Add entries with known SimHashes
	ctx.SharedIndex.Index.Upsert("beliefs", "belief1", 0, 1, 1000)    // Distance 0
	ctx.SharedIndex.Index.Upsert("beliefs", "belief2", 1, 2, 2000)    // Distance 1
	ctx.SharedIndex.Index.Upsert("beliefs", "belief3", 3, 3, 3000)    // Distance 2
	ctx.SharedIndex.Index.Upsert("beliefs", "belief4", 0xFF, 4, 4000) // Distance 8

	args := []eval.Value{
		&eval.StringValue{Value: "beliefs"},
		&eval.IntValue{Value: 0},     // query_simhash
		&eval.IntValue{Value: 3},     // top_k
		&eval.IntValue{Value: 0},     // max_scan (0 = unlimited)
		&eval.BoolValue{Value: true}, // deterministic
	}

	result, err := sharedIndexFindSimHashImpl(ctx, args)
	if err != nil {
		t.Fatalf("sharedIndexFindSimHashImpl failed: %v", err)
	}

	listVal, ok := result.(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", result)
	}

	if len(listVal.Elements) != 3 {
		t.Fatalf("expected 3 results, got %d", len(listVal.Elements))
	}

	// Check first result
	first, ok := listVal.Elements[0].(*eval.RecordValue)
	if !ok {
		t.Fatalf("expected RecordValue, got %T", listVal.Elements[0])
	}

	keyVal, ok := first.Fields["key"].(*eval.StringValue)
	if !ok || keyVal.Value != "belief1" {
		t.Errorf("expected first result key=belief1, got %v", first.Fields["key"])
	}

	scoreVal, ok := first.Fields["score"].(*eval.FloatValue)
	if !ok || scoreVal.Value != 1.0 {
		t.Errorf("expected first result score=1.0, got %v", first.Fields["score"])
	}
}

func TestSharedIndexFindSimHash_DeterministicOrdering(t *testing.T) {
	ctx := createSharedIndexContext()

	// Add entries with same SimHash (same score)
	ctx.SharedIndex.Index.Upsert("test", "key_c", 0, 1, 1000)
	ctx.SharedIndex.Index.Upsert("test", "key_a", 0, 2, 2000)
	ctx.SharedIndex.Index.Upsert("test", "key_b", 0, 3, 3000)

	args := []eval.Value{
		&eval.StringValue{Value: "test"},
		&eval.IntValue{Value: 0},     // query_simhash
		&eval.IntValue{Value: 3},     // top_k
		&eval.IntValue{Value: 0},     // max_scan
		&eval.BoolValue{Value: true}, // deterministic
	}

	// Run multiple times and verify identical results
	var firstKeys []string
	for run := 0; run < 5; run++ {
		result, err := sharedIndexFindSimHashImpl(ctx, args)
		if err != nil {
			t.Fatalf("run %d: failed: %v", run, err)
		}

		listVal := result.(*eval.ListValue)
		keys := make([]string, len(listVal.Elements))
		for i, elem := range listVal.Elements {
			record := elem.(*eval.RecordValue)
			keys[i] = record.Fields["key"].(*eval.StringValue).Value
		}

		if run == 0 {
			firstKeys = keys
		} else {
			for i, key := range keys {
				if key != firstKeys[i] {
					t.Errorf("run %d: result %d differs: %s vs %s", run, i, key, firstKeys[i])
				}
			}
		}
	}

	// In Strict mode, should be sorted by key ASC when scores equal
	if firstKeys[0] != "key_a" {
		t.Errorf("Strict mode should sort by key ASC, got first key: %s", firstKeys[0])
	}
}

func TestSharedIndexEntryCount(t *testing.T) {
	ctx := createSharedIndexContext()

	ctx.SharedIndex.Index.Upsert("beliefs", "b1", 0, 1, 1000)
	ctx.SharedIndex.Index.Upsert("beliefs", "b2", 0, 1, 2000)

	args := []eval.Value{
		&eval.StringValue{Value: "beliefs"},
	}

	result, err := sharedIndexEntryCountImpl(ctx, args)
	if err != nil {
		t.Fatalf("sharedIndexEntryCountImpl failed: %v", err)
	}

	intVal, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}

	if intVal.Value != 2 {
		t.Errorf("expected count=2, got %d", intVal.Value)
	}
}

func TestSharedIndexNamespaces(t *testing.T) {
	ctx := createSharedIndexContext()

	ctx.SharedIndex.Index.Upsert("beliefs", "b1", 0, 1, 1000)
	ctx.SharedIndex.Index.Upsert("goals", "g1", 0, 1, 2000)
	ctx.SharedIndex.Index.Upsert("actions", "a1", 0, 1, 3000)

	args := []eval.Value{
		&eval.UnitValue{}, // unit parameter
	}

	result, err := sharedIndexNamespacesImpl(ctx, args)
	if err != nil {
		t.Fatalf("sharedIndexNamespacesImpl failed: %v", err)
	}

	listVal, ok := result.(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", result)
	}

	if len(listVal.Elements) != 3 {
		t.Errorf("expected 3 namespaces, got %d", len(listVal.Elements))
	}

	// Should be sorted alphabetically
	expected := []string{"actions", "beliefs", "goals"}
	for i, elem := range listVal.Elements {
		strVal, ok := elem.(*eval.StringValue)
		if !ok {
			t.Errorf("expected StringValue at %d, got %T", i, elem)
			continue
		}
		if strVal.Value != expected[i] {
			t.Errorf("expected namespace %d=%s, got %s", i, expected[i], strVal.Value)
		}
	}
}
