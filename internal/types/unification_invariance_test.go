package types

import "testing"

func TestEffectRowUnificationRemainsInvariant(t *testing.T) {
	seeded := effectTestRow("Rand", map[string]string{"mode": "seeded"})
	bare := effectTestRow("Rand", nil)
	u := NewUnifier()
	if _, err := u.unifyRows(seeded, bare, Substitution{}); err == nil {
		t.Fatal("seeded function effect row unexpectedly unified with bare/os Rand")
	}
}
