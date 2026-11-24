package types

import "testing"

func TestSubsumptionEdgesAreExactlyRatifiedEdges(t *testing.T) {
	if len(subsumptionEdges) != 3 {
		t.Fatalf("got %d edges, want exactly 3: %#v", len(subsumptionEdges), subsumptionEdges)
	}
	want := map[subsumptionEdge]bool{
		{Effect: "Rand", Declared: "seeded", Required: "os"}:     true,
		{Effect: "Rand", Declared: "crypto", Required: "os"}:     true,
		{Effect: "AI", Declared: "routeable", Required: "fixed"}: true,
	}
	for _, edge := range subsumptionEdges {
		if !want[edge] {
			t.Errorf("unexpected edge: %#v", edge)
		}
		delete(want, edge)
	}
	if len(want) != 0 {
		t.Errorf("missing edges: %#v", want)
	}
	if !ModeSubsumes("AI", "routeable", "fixed") {
		t.Error("AI routeable must subsume fixed")
	}
	if ModeSubsumes("AI", "fixed", "routeable") {
		t.Error("AI fixed must not subsume routeable (edge is one-way)")
	}
	if ModeSubsumes("AI", "replay-only", "fixed") {
		t.Error("AI replay-only must not subsume fixed without an explicit edge")
	}
	for _, effect := range []string{"Clock", "Net", "FS"} {
		if ModeSubsumes(effect, "routeable", "fixed") {
			t.Errorf("%s unexpectedly gained AI's routeable->fixed edge", effect)
		}
	}
	for _, effect := range []string{"AI", "Clock", "Net", "FS"} {
		// The edge lookup must key on the EFFECT NAME, not just the mode pair.
		if ModeSubsumes(effect, "seeded", "os") {
			t.Errorf("%s wrongly inherited Rand's seeded->os edge (effect name ignored in lookup)", effect)
		}
		if ModeSubsumes(effect, "crypto", "os") {
			t.Errorf("%s wrongly inherited Rand's crypto->os edge (effect name ignored in lookup)", effect)
		}
	}
}

func TestBareAndExplicitDefaultAreEquivalent(t *testing.T) {
	bare := effectTestRow("Rand", nil)
	os := effectTestRow("Rand", map[string]string{"mode": "os"})
	seeded := effectTestRow("Rand", map[string]string{"mode": "seeded"})
	crypto := effectTestRow("Rand", map[string]string{"mode": "crypto"})
	if !SubsumeEffectRows(bare, os) || !SubsumeEffectRows(os, bare) {
		t.Fatal("bare Rand and explicit mode=os must cover each other")
	}
	if SubsumeEffectRows(seeded, bare) || SubsumeEffectRows(crypto, bare) {
		t.Fatal("bare Rand declaration must not cover seeded or crypto")
	}
}

func TestNonModeParametersRemainInvariant(t *testing.T) {
	required := effectTestRow("AI", map[string]string{"mode": "fixed", "scope": "byok"})
	declared := effectTestRow("AI", map[string]string{"mode": "fixed", "scope": "managed"})
	if SubsumeEffectRows(required, declared) {
		t.Fatal("AI scope mismatch unexpectedly subsumed")
	}
}

func TestDefaultModeDoesNotGrantSubsumption(t *testing.T) {
	required := effectTestRow("AI", map[string]string{"mode": "fixed"})
	declared := effectTestRow("AI", map[string]string{"mode": "replay-only"})
	if SubsumeEffectRows(required, declared) {
		t.Fatal("AI replay-only must not cover its registered fixed default without an edge")
	}
}

func TestStructuredEffectRowDiff(t *testing.T) {
	required := effectTestRow("Rand", map[string]string{"mode": "seeded"})
	declared := effectTestRow("Rand", map[string]string{"mode": "os"})
	diff := DiffEffectRows(required, declared)
	if len(diff.Missing) != 0 || len(diff.ParamMismatches) != 1 {
		t.Fatalf("unexpected diff: %#v", diff)
	}
	want := (EffectParamMismatch{Effect: "Rand", Key: "mode", RequiredValue: "seeded", DeclaredValue: "os"})
	if diff.ParamMismatches[0] != want {
		t.Fatalf("got %#v, want %#v", diff.ParamMismatches[0], want)
	}
}

func effectTestRow(effect string, params map[string]string) *Row {
	row := &Row{Kind: EffectRow, Labels: map[string]Type{effect: Unit()}}
	if params != nil {
		row.Params = map[string]map[string]string{effect: params}
	}
	return row
}
