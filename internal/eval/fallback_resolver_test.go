package eval

import (
	"fmt"
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

// mockResolver is a test resolver that resolves from a static map
type mockResolver struct {
	values map[string]Value
}

func (m *mockResolver) ResolveValue(ref core.GlobalRef) (Value, error) {
	key := ref.Module + "." + ref.Name
	if val, ok := m.values[key]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("not found: %s.%s", ref.Module, ref.Name)
}

func TestFallbackResolver_PrimarySucceeds(t *testing.T) {
	primary := &mockResolver{values: map[string]Value{
		"$builtin._io_print": &StringValue{Value: "primary_print"},
	}}
	secondary := &mockResolver{values: map[string]Value{
		"$builtin._io_print": &StringValue{Value: "secondary_print"},
	}}

	resolver := &FallbackResolver{Primary: primary, Secondary: secondary}
	val, err := resolver.ResolveValue(core.GlobalRef{Module: "$builtin", Name: "_io_print"})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	sv, ok := val.(*StringValue)
	if !ok || sv.Value != "primary_print" {
		t.Errorf("expected primary value, got %v", val)
	}
}

func TestFallbackResolver_PrimaryFails_SecondarySucceeds(t *testing.T) {
	primary := &mockResolver{values: map[string]Value{}}
	secondary := &mockResolver{values: map[string]Value{
		"$adt.make_Option_Some": &StringValue{Value: "some_factory"},
	}}

	resolver := &FallbackResolver{Primary: primary, Secondary: secondary}
	val, err := resolver.ResolveValue(core.GlobalRef{Module: "$adt", Name: "make_Option_Some"})

	if err != nil {
		t.Fatalf("expected fallback success, got: %v", err)
	}
	sv, ok := val.(*StringValue)
	if !ok || sv.Value != "some_factory" {
		t.Errorf("expected secondary value, got %v", val)
	}
}

func TestFallbackResolver_BothFail(t *testing.T) {
	primary := &mockResolver{values: map[string]Value{}}
	secondary := &mockResolver{values: map[string]Value{}}

	resolver := &FallbackResolver{Primary: primary, Secondary: secondary}
	_, err := resolver.ResolveValue(core.GlobalRef{Module: "std/json", Name: "asString"})

	if err == nil {
		t.Fatal("expected error when both resolvers fail")
	}
}

func TestFallbackResolver_BuiltinsFromPrimary_ConstructorsFromSecondary(t *testing.T) {
	// Simulates the real scenario: caller's resolver has builtins,
	// function's defining resolver has constructors
	primary := &mockResolver{values: map[string]Value{
		"$builtin._ai_call": &StringValue{Value: "ai_builtin"},
	}}
	secondary := &mockResolver{values: map[string]Value{
		"$adt.make_Option_Some": &StringValue{Value: "some_ctor"},
		"$adt.make_Option_None": &StringValue{Value: "none_ctor"},
	}}

	resolver := &FallbackResolver{Primary: primary, Secondary: secondary}

	// Builtin should come from primary
	val, err := resolver.ResolveValue(core.GlobalRef{Module: "$builtin", Name: "_ai_call"})
	if err != nil {
		t.Fatalf("builtin lookup failed: %v", err)
	}
	if sv, ok := val.(*StringValue); !ok || sv.Value != "ai_builtin" {
		t.Errorf("expected ai_builtin from primary, got %v", val)
	}

	// Constructor should fall through to secondary
	val, err = resolver.ResolveValue(core.GlobalRef{Module: "$adt", Name: "make_Option_Some"})
	if err != nil {
		t.Fatalf("constructor lookup failed: %v", err)
	}
	if sv, ok := val.(*StringValue); !ok || sv.Value != "some_ctor" {
		t.Errorf("expected some_ctor from secondary, got %v", val)
	}
}
