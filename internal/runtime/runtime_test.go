package runtime

import (
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

func TestNewModuleRuntime(t *testing.T) {
	basePath := "/test/path"

	rt := NewModuleRuntime(basePath)

	// Verify fields are initialized
	if rt.loader == nil {
		t.Error("Expected loader to be initialized")
	}

	if rt.evaluator == nil {
		t.Error("Expected evaluator to be initialized")
	}

	if rt.instances == nil {
		t.Error("Expected instances map to be initialized")
	}

	if rt.basePath != basePath {
		t.Errorf("Expected basePath '%s', got '%s'", basePath, rt.basePath)
	}

	// Verify instances map is empty
	if len(rt.instances) != 0 {
		t.Errorf("Expected instances map to be empty, got %d entries", len(rt.instances))
	}
}

func TestModuleRuntime_GetInstance(t *testing.T) {
	rt := NewModuleRuntime("/test")

	// Test: Instance not found
	inst := rt.GetInstance("test/module")
	if inst != nil {
		t.Error("Expected nil when getting non-existent instance")
	}

	// Add instance to cache
	mockInst := &ModuleInstance{
		Path: "test/module",
	}
	rt.instances["test/module"] = mockInst

	// Test: Instance found
	inst = rt.GetInstance("test/module")
	if inst == nil {
		t.Error("Expected instance to be found")
	}

	if inst.Path != "test/module" {
		t.Errorf("Expected path 'test/module', got '%s'", inst.Path)
	}
}

func TestModuleRuntime_HasInstance(t *testing.T) {
	rt := NewModuleRuntime("/test")

	// Test: Instance doesn't exist
	if rt.HasInstance("test/module") {
		t.Error("Expected HasInstance to return false for non-existent instance")
	}

	// Add instance to cache
	rt.instances["test/module"] = &ModuleInstance{
		Path: "test/module",
	}

	// Test: Instance exists
	if !rt.HasInstance("test/module") {
		t.Error("Expected HasInstance to return true for existing instance")
	}

	// Test: Different instance doesn't exist
	if rt.HasInstance("test/other") {
		t.Error("Expected HasInstance to return false for different instance")
	}
}

func TestModuleRuntime_ListInstances(t *testing.T) {
	rt := NewModuleRuntime("/test")

	// Test: No instances
	instances := rt.ListInstances()
	if len(instances) != 0 {
		t.Errorf("Expected 0 instances, got %d", len(instances))
	}

	// Add instances
	rt.instances["test/a"] = &ModuleInstance{Path: "test/a"}
	rt.instances["test/b"] = &ModuleInstance{Path: "test/b"}
	rt.instances["test/c"] = &ModuleInstance{Path: "test/c"}

	// Test: List instances
	instances = rt.ListInstances()
	if len(instances) != 3 {
		t.Errorf("Expected 3 instances, got %d", len(instances))
	}

	// Verify all instances are present (order doesn't matter)
	found := make(map[string]bool)
	for _, path := range instances {
		found[path] = true
	}

	if !found["test/a"] || !found["test/b"] || !found["test/c"] {
		t.Errorf("Missing instances, got: %v", instances)
	}
}

func TestModuleRuntime_LoadAndEvaluate_CacheHit(t *testing.T) {
	rt := NewModuleRuntime(".")

	// Pre-populate cache with an evaluated instance
	mockInst := &ModuleInstance{
		Path:     "test/cached",
		Bindings: map[string]eval.Value{"foo": &eval.IntValue{Value: 42}}, // Marks as evaluated
	}
	rt.instances["test/cached"] = mockInst

	// Test: LoadAndEvaluate should return cached instance
	inst, err := rt.LoadAndEvaluate("test/cached")
	if err != nil {
		t.Errorf("Expected no error for cached instance, got: %v", err)
	}

	if inst != mockInst {
		t.Error("Expected to get the same cached instance")
	}

	// Verify no additional instances were created
	if len(rt.instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(rt.instances))
	}
}

// Note: Full LoadAndEvaluate tests require actual .ail files and are
// covered in integration tests. These unit tests focus on the caching
// and basic structure.
