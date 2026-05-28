package notify

import (
	"context"
	"testing"
)

// stubChannel is a no-op Channel for registry tests.
type stubChannel struct {
	name string
	sent []Notification
}

func (s *stubChannel) Name() string { return s.name }
func (s *stubChannel) Send(_ context.Context, n Notification) error {
	s.sent = append(s.sent, n)
	return nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	ch := &stubChannel{name: "stub"}
	if err := reg.Register(ch); err != nil {
		t.Fatalf("register: %v", err)
	}
	got, err := reg.Get("stub")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != ch {
		t.Fatal("Get returned a different channel instance")
	}
}

func TestRegistrySameInstanceIdempotent(t *testing.T) {
	reg := NewRegistry()
	ch := &stubChannel{name: "stub"}
	if err := reg.Register(ch); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := reg.Register(ch); err != nil {
		t.Fatalf("re-register same instance should be idempotent, got: %v", err)
	}
}

func TestRegistryDifferentInstanceErrors(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(&stubChannel{name: "stub"}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := reg.Register(&stubChannel{name: "stub"}); err == nil {
		t.Fatal("registering a different instance for the same name should error")
	}
}

func TestRegistryEmptyNameErrors(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(&stubChannel{name: ""}); err == nil {
		t.Fatal("registering a channel with empty Name should error")
	}
}

func TestRegistryGetUnknownErrors(t *testing.T) {
	reg := NewRegistry()
	if _, err := reg.Get("nope"); err == nil {
		t.Fatal("Get on an unknown name should error")
	}
}

func TestRegistryNamesSorted(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(&stubChannel{name: "zulu"})
	_ = reg.Register(&stubChannel{name: "alpha"})
	_ = reg.Register(&stubChannel{name: "mike"})
	got := reg.Names()
	want := []string{"alpha", "mike", "zulu"}
	if len(got) != len(want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Names() = %v, want %v", got, want)
		}
	}
}
