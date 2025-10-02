package runtime

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
)

func TestModuleGlobalResolver_ResolveLocal(t *testing.T) {
	// Create a module instance with some bindings
	inst := &ModuleInstance{
		Path: "test/module",
		Bindings: map[string]eval.Value{
			"foo": &eval.IntValue{Value: 42},
			"bar": &eval.IntValue{Value: 100},
		},
		Exports: map[string]eval.Value{
			"foo": &eval.IntValue{Value: 42},
		},
		Imports: make(map[string]*ModuleInstance),
	}

	resolver := newModuleGlobalResolver(inst)

	// Test: Resolve local binding with empty module path
	ref := core.GlobalRef{Module: "", Name: "foo"}
	val, err := resolver.ResolveValue(ref)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	intVal, ok := val.(*eval.IntValue)
	if !ok {
		t.Errorf("Expected IntValue, got %T", val)
	}

	if intVal.Value != 42 {
		t.Errorf("Expected value 42, got %d", intVal.Value)
	}

	// Test: Resolve local binding with module path matching current
	ref = core.GlobalRef{Module: "test/module", Name: "bar"}
	val, err = resolver.ResolveValue(ref)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	intVal, ok = val.(*eval.IntValue)
	if !ok {
		t.Errorf("Expected IntValue, got %T", val)
	}

	if intVal.Value != 100 {
		t.Errorf("Expected value 100, got %d", intVal.Value)
	}
}

func TestModuleGlobalResolver_ResolveLocal_Undefined(t *testing.T) {
	inst := &ModuleInstance{
		Path: "test/module",
		Bindings: map[string]eval.Value{
			"foo": &eval.IntValue{Value: 42},
		},
		Exports: make(map[string]eval.Value),
		Imports: make(map[string]*ModuleInstance),
	}

	resolver := newModuleGlobalResolver(inst)

	// Test: Resolve undefined local binding
	ref := core.GlobalRef{Module: "", Name: "undefined"}
	_, err := resolver.ResolveValue(ref)
	if err == nil {
		t.Error("Expected error when resolving undefined binding")
	}

	// Error should mention available bindings
	if err.Error() != "undefined binding 'undefined' in module test/module (available: [foo])" {
		t.Errorf("Expected error to contain available bindings, got: %v", err)
	}
}

func TestModuleGlobalResolver_ResolveImported(t *testing.T) {
	// Create a dependency module
	depInst := &ModuleInstance{
		Path: "test/dep",
		Bindings: map[string]eval.Value{
			"helper": &eval.IntValue{Value: 10},
			"public": &eval.IntValue{Value: 20},
		},
		Exports: map[string]eval.Value{
			"public": &eval.IntValue{Value: 20},
		},
		Imports: make(map[string]*ModuleInstance),
	}

	// Create current module that imports the dependency
	inst := &ModuleInstance{
		Path:     "test/module",
		Bindings: make(map[string]eval.Value),
		Exports:  make(map[string]eval.Value),
		Imports: map[string]*ModuleInstance{
			"test/dep": depInst,
		},
	}

	resolver := newModuleGlobalResolver(inst)

	// Test: Resolve exported binding from imported module
	ref := core.GlobalRef{Module: "test/dep", Name: "public"}
	val, err := resolver.ResolveValue(ref)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	intVal, ok := val.(*eval.IntValue)
	if !ok {
		t.Errorf("Expected IntValue, got %T", val)
	}

	if intVal.Value != 20 {
		t.Errorf("Expected value 20, got %d", intVal.Value)
	}

	// Test: Cannot access private (non-exported) binding from imported module
	ref = core.GlobalRef{Module: "test/dep", Name: "helper"}
	_, err = resolver.ResolveValue(ref)
	if err == nil {
		t.Error("Expected error when accessing non-exported binding from imported module")
	}
}

func TestModuleGlobalResolver_ResolveImported_ModuleNotImported(t *testing.T) {
	inst := &ModuleInstance{
		Path:     "test/module",
		Bindings: make(map[string]eval.Value),
		Exports:  make(map[string]eval.Value),
		Imports: map[string]*ModuleInstance{
			"test/dep1": &ModuleInstance{Path: "test/dep1"},
		},
	}

	resolver := newModuleGlobalResolver(inst)

	// Test: Resolve from module that is not imported
	ref := core.GlobalRef{Module: "test/dep2", Name: "foo"}
	_, err := resolver.ResolveValue(ref)
	if err == nil {
		t.Error("Expected error when resolving from non-imported module")
	}

	// Error should mention available imports
	if err.Error() != "module test/dep2 not imported by test/module (available imports: [test/dep1])" {
		t.Errorf("Expected error to contain available imports, got: %v", err)
	}
}

func TestModuleGlobalResolver_ResolveImported_ExportNotFound(t *testing.T) {
	depInst := &ModuleInstance{
		Path: "test/dep",
		Bindings: map[string]eval.Value{
			"foo": &eval.IntValue{Value: 42},
		},
		Exports: map[string]eval.Value{
			"foo": &eval.IntValue{Value: 42},
		},
		Imports: make(map[string]*ModuleInstance),
	}

	inst := &ModuleInstance{
		Path:     "test/module",
		Bindings: make(map[string]eval.Value),
		Exports:  make(map[string]eval.Value),
		Imports: map[string]*ModuleInstance{
			"test/dep": depInst,
		},
	}

	resolver := newModuleGlobalResolver(inst)

	// Test: Resolve export that doesn't exist
	ref := core.GlobalRef{Module: "test/dep", Name: "bar"}
	_, err := resolver.ResolveValue(ref)
	if err == nil {
		t.Error("Expected error when resolving non-existent export")
	}
}

func TestModuleGlobalResolver_EmptyModule(t *testing.T) {
	inst := &ModuleInstance{
		Path:     "test/empty",
		Bindings: make(map[string]eval.Value),
		Exports:  make(map[string]eval.Value),
		Imports:  make(map[string]*ModuleInstance),
	}

	resolver := newModuleGlobalResolver(inst)

	// Test: Resolve from empty module
	ref := core.GlobalRef{Module: "", Name: "foo"}
	_, err := resolver.ResolveValue(ref)
	if err == nil {
		t.Error("Expected error when resolving from empty module")
	}

	// Error should indicate module has no bindings
	if err.Error() != "undefined binding 'foo' in module test/empty (module has no bindings)" {
		t.Errorf("Expected error to indicate module has no bindings, got: %v", err)
	}
}
