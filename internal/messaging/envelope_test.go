package messaging

import (
	"encoding/json"
	"testing"
)

func TestNewEnvelope(t *testing.T) {
	env := NewEnvelope()
	if env == nil {
		t.Fatal("NewEnvelope returned nil")
	}
	if !env.IsEmpty() {
		t.Error("new envelope should be empty")
	}
	if env.Slots == nil {
		t.Error("slots map should be initialized")
	}
}

func TestEnvelopeSetGet(t *testing.T) {
	env := NewEnvelope()
	vec := []float32{0.1, 0.2, 0.3}
	env.Set(SlotIntent, vec, "ollama:nomic-embed-text")

	if env.IsEmpty() {
		t.Error("envelope should not be empty after Set")
	}

	got := env.Get(SlotIntent)
	if got == nil {
		t.Fatal("Get returned nil for populated slot")
	}
	if got.Model != "ollama:nomic-embed-text" {
		t.Errorf("model = %q, want %q", got.Model, "ollama:nomic-embed-text")
	}
	if got.Dimension != 3 {
		t.Errorf("dimension = %d, want 3", got.Dimension)
	}
	if len(got.Vector) != 3 {
		t.Errorf("vector length = %d, want 3", len(got.Vector))
	}

	// Get non-existent slot
	if env.Get(SlotCode) != nil {
		t.Error("Get should return nil for unpopulated slot")
	}
}

func TestEnvelopeGetVector(t *testing.T) {
	env := NewEnvelope()
	vec := []float32{1.0, 2.0}
	env.Set(SlotCode, vec, "test")

	got := env.GetVector(SlotCode)
	if len(got) != 2 || got[0] != 1.0 || got[1] != 2.0 {
		t.Errorf("GetVector = %v, want [1.0, 2.0]", got)
	}

	if env.GetVector(SlotSkill) != nil {
		t.Error("GetVector should return nil for missing slot")
	}

	// Nil envelope
	var nilEnv *Envelope
	if nilEnv.GetVector(SlotCode) != nil {
		t.Error("GetVector on nil envelope should return nil")
	}
}

func TestEnvelopeIsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		env   *Envelope
		empty bool
	}{
		{"nil envelope", nil, true},
		{"new envelope", NewEnvelope(), true},
		{"envelope with nil slots", &Envelope{}, true},
		{"populated envelope", func() *Envelope {
			e := NewEnvelope()
			e.Set(SlotIntent, []float32{1.0}, "m")
			return e
		}(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.env.IsEmpty(); got != tt.empty {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.empty)
			}
		})
	}
}

func TestEnvelopeMerge(t *testing.T) {
	base := NewEnvelope()
	base.Set(SlotIntent, []float32{1.0}, "model-a")

	other := NewEnvelope()
	other.Set(SlotIntent, []float32{9.0}, "model-b")
	other.Set(SlotCode, []float32{2.0}, "model-b")

	base.Merge(other)

	// Intent should be preserved (not overwritten)
	if base.Get(SlotIntent).Vector[0] != 1.0 {
		t.Error("Merge should not overwrite existing slots")
	}

	// Code should be added
	if base.Get(SlotCode) == nil || base.Get(SlotCode).Vector[0] != 2.0 {
		t.Error("Merge should add missing slots")
	}
}

func TestEnvelopeMergeOverwrite(t *testing.T) {
	base := NewEnvelope()
	base.Set(SlotIntent, []float32{1.0}, "model-a")

	other := NewEnvelope()
	other.Set(SlotIntent, []float32{9.0}, "model-b")

	base.MergeOverwrite(other)

	if base.Get(SlotIntent).Vector[0] != 9.0 {
		t.Error("MergeOverwrite should overwrite existing slots")
	}
}

func TestEnvelopeMergeNil(t *testing.T) {
	base := NewEnvelope()
	base.Set(SlotIntent, []float32{1.0}, "m")

	base.Merge(nil)
	base.MergeOverwrite(nil)

	if base.Get(SlotIntent).Vector[0] != 1.0 {
		t.Error("merging nil should not affect envelope")
	}
}

func TestEnvelopePopulatedSlots(t *testing.T) {
	env := NewEnvelope()
	if len(env.PopulatedSlots()) != 0 {
		t.Error("empty envelope should have no populated slots")
	}

	env.Set(SlotIntent, []float32{1.0}, "m")
	env.Set(SlotCode, []float32{2.0}, "m")

	slots := env.PopulatedSlots()
	if len(slots) != 2 {
		t.Errorf("populated slots = %d, want 2", len(slots))
	}

	// Nil envelope
	var nilEnv *Envelope
	if len(nilEnv.PopulatedSlots()) != 0 {
		t.Error("nil envelope should have no populated slots")
	}
}

func TestValidateSlot(t *testing.T) {
	for _, slot := range AllSlots {
		if err := ValidateSlot(slot); err != nil {
			t.Errorf("ValidateSlot(%q) unexpected error: %v", slot, err)
		}
	}

	if err := ValidateSlot("bogus"); err == nil {
		t.Error("ValidateSlot should reject unknown slot names")
	}
}

func TestEnvelopeJSONRoundTrip(t *testing.T) {
	env := NewEnvelope()
	env.Set(SlotIntent, []float32{0.1, 0.2, 0.3}, "ollama:nomic-embed-text")
	env.Set(SlotResolution, []float32{0.4, 0.5}, "openai:text-embedding-3-small")

	jsonStr := env.ToJSON()

	// Verify it's valid JSON
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		t.Fatalf("ToJSON produced invalid JSON: %v", err)
	}

	// Round-trip
	restored := EnvelopeFromJSON(jsonStr)
	if restored.IsEmpty() {
		t.Fatal("restored envelope should not be empty")
	}

	intent := restored.Get(SlotIntent)
	if intent == nil {
		t.Fatal("intent slot missing after round-trip")
	}
	if intent.Dimension != 3 {
		t.Errorf("intent dimension = %d, want 3", intent.Dimension)
	}
	if intent.Model != "ollama:nomic-embed-text" {
		t.Errorf("intent model = %q, want %q", intent.Model, "ollama:nomic-embed-text")
	}

	resolution := restored.Get(SlotResolution)
	if resolution == nil {
		t.Fatal("resolution slot missing after round-trip")
	}
	if resolution.Dimension != 2 {
		t.Errorf("resolution dimension = %d, want 2", resolution.Dimension)
	}

	// Slots not set should be nil
	if restored.Get(SlotCode) != nil {
		t.Error("code slot should be nil after round-trip")
	}
}

func TestEnvelopeFromJSONEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		empty bool
	}{
		{"empty string", "", true},
		{"empty object", "{}", true},
		{"invalid json", "not json", true},
		{"null", "null", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := EnvelopeFromJSON(tt.input)
			if env == nil {
				t.Fatal("EnvelopeFromJSON should never return nil")
			}
			if env.IsEmpty() != tt.empty {
				t.Errorf("IsEmpty() = %v, want %v", env.IsEmpty(), tt.empty)
			}
		})
	}
}

func TestEnvelopeToJSONEmpty(t *testing.T) {
	env := NewEnvelope()
	if got := env.ToJSON(); got != "{}" {
		t.Errorf("empty envelope ToJSON = %q, want %q", got, "{}")
	}

	var nilEnv *Envelope
	if got := nilEnv.ToJSON(); got != "{}" {
		t.Errorf("nil envelope ToJSON = %q, want %q", got, "{}")
	}
}

func TestDimensionMatch(t *testing.T) {
	a := &EnvelopeVector{Vector: make([]float32, 768), Model: "ollama:x", Dimension: 768}
	b := &EnvelopeVector{Vector: make([]float32, 768), Model: "ollama:y", Dimension: 768}
	c := &EnvelopeVector{Vector: make([]float32, 1536), Model: "openai:z", Dimension: 1536}

	if err := DimensionMatch(a, b); err != nil {
		t.Errorf("same dimension should match: %v", err)
	}
	if err := DimensionMatch(a, c); err == nil {
		t.Error("different dimensions should not match")
	}
	if err := DimensionMatch(nil, a); err != nil {
		t.Error("nil should match anything")
	}
	if err := DimensionMatch(a, nil); err != nil {
		t.Error("nil should match anything")
	}
}

// --- Mock embedder for builder tests ---

type mockEmbedder struct {
	dim       int
	callCount int
}

func (m *mockEmbedder) Embed(text string) ([]float32, error) {
	m.callCount++
	vec := make([]float32, m.dim)
	// Use hash of text to produce deterministic but distinct vectors
	for i := range vec {
		vec[i] = float32(len(text)%10) * 0.1
	}
	return vec, nil
}

func (m *mockEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, t := range texts {
		v, err := m.Embed(t)
		if err != nil {
			return nil, err
		}
		results[i] = v
	}
	return results, nil
}

func (m *mockEmbedder) Dimension() int    { return m.dim }
func (m *mockEmbedder) ModelName() string { return "mock:test" }

// --- Builder tests ---

func TestEnvelopeBuilderIntentOnly(t *testing.T) {
	emb := &mockEmbedder{dim: 4}
	msg := &InboxMessage{
		Title:   "Fix the type error",
		Payload: "The unifier crashes on recursive types",
	}

	env, err := NewEnvelopeBuilder(emb).Build(msg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if env.Get(SlotIntent) == nil {
		t.Error("intent slot should be populated")
	}
	if env.Get(SlotIntent).Model != "mock:test" {
		t.Errorf("model = %q, want %q", env.Get(SlotIntent).Model, "mock:test")
	}

	// Other slots should not be populated
	for _, slot := range []string{SlotCode, SlotContext, SlotSkill, SlotResolution} {
		if env.Get(slot) != nil {
			t.Errorf("slot %q should not be populated without With* call", slot)
		}
	}

	if emb.callCount != 1 {
		t.Errorf("expected 1 embed call (intent), got %d", emb.callCount)
	}
}

func TestEnvelopeBuilderAllSlots(t *testing.T) {
	emb := &mockEmbedder{dim: 4}
	msg := &InboxMessage{
		Title:   "Add parser support for records",
		Payload: "Need to handle record types in the parser",
	}

	env, err := NewEnvelopeBuilder(emb).
		WithCodeContext(
			[]string{"internal/parser/parser.go", "internal/ast/ast.go"},
			[]string{"func parseRecord() { ... }"},
		).
		WithSessionContext(
			[]string{"parser.go", "ast.go"},
			[]string{"unexpected token RBRACE"},
			[]string{"Read", "Grep"},
		).
		WithSkillHints(
			[]string{"parser", "elaboration"},
			[]string{"RecordExpr", "RecordType"},
			[]string{"internal/parser/*.go"},
		).
		WithResolution("fix: handle record parsing", "diff --git a/parser.go...").
		Build(msg)

	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	for _, slot := range AllSlots {
		if env.Get(slot) == nil {
			t.Errorf("slot %q should be populated", slot)
		}
	}

	// 5 embed calls: intent + code + context + skill + resolution
	if emb.callCount != 5 {
		t.Errorf("expected 5 embed calls, got %d", emb.callCount)
	}
}

func TestEnvelopeBuilderNilEmbedder(t *testing.T) {
	msg := &InboxMessage{Title: "test"}
	_, err := NewEnvelopeBuilder(nil).Build(msg)
	if err == nil {
		t.Error("Build with nil embedder should fail")
	}
}

func TestEnvelopeBuilderIntentTruncatesPayload(t *testing.T) {
	emb := &mockEmbedder{dim: 4}
	longPayload := ""
	for i := 0; i < 500; i++ {
		longPayload += "x"
	}
	msg := &InboxMessage{
		Title:   "Test",
		Payload: longPayload,
	}

	env, err := NewEnvelopeBuilder(emb).Build(msg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if env.Get(SlotIntent) == nil {
		t.Error("intent should be populated even with long payload")
	}
}

func TestBuildCodeText(t *testing.T) {
	got := buildCodeText([]string{"a.go", "b.go"}, []string{"func foo() {}"})
	if got == "" {
		t.Error("buildCodeText should not return empty")
	}
	if !contains(got, "a.go") || !contains(got, "func foo") {
		t.Errorf("unexpected output: %q", got)
	}
}

func TestBuildContextText(t *testing.T) {
	got := buildContextText([]string{"x.go"}, []string{"err1"}, []string{"Read"})
	if got == "" {
		t.Error("buildContextText should not return empty")
	}
}

func TestBuildSkillText(t *testing.T) {
	got := buildSkillText([]string{"parser"}, []string{"RecordExpr"}, []string{"*.go"})
	if got == "" {
		t.Error("buildSkillText should not return empty")
	}
}

func TestBuildResolutionText(t *testing.T) {
	got := buildResolutionText("fix bug", "diff content")
	if got == "" {
		t.Error("buildResolutionText should not return empty")
	}

	// Test truncation of long diffs
	longDiff := ""
	for i := 0; i < 3000; i++ {
		longDiff += "x"
	}
	got = buildResolutionText("msg", longDiff)
	if len(got) > 2200 { // commit msg + "Diff:\n" + 2000 chars
		t.Error("long diffs should be truncated")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestNewEmbedderFromConfig(t *testing.T) {
	// Test "none" provider
	emb, err := NewEmbedderFromConfig(EmbedConfig{Provider: "none"})
	if err != nil {
		t.Fatalf("none provider should not error: %v", err)
	}
	if emb != nil {
		t.Error("none provider should return nil embedder")
	}

	// Test empty provider
	emb, err = NewEmbedderFromConfig(EmbedConfig{Provider: ""})
	if err != nil {
		t.Fatalf("empty provider should not error: %v", err)
	}
	if emb != nil {
		t.Error("empty provider should return nil embedder")
	}

	// Test unknown provider
	_, err = NewEmbedderFromConfig(EmbedConfig{Provider: "bogus"})
	if err == nil {
		t.Error("unknown provider should error")
	}

	// Test openai without key
	_, err = NewEmbedderFromConfig(EmbedConfig{Provider: "openai"})
	if err == nil {
		t.Error("openai without key should error")
	}

	// Test gemini without key
	_, err = NewEmbedderFromConfig(EmbedConfig{Provider: "gemini"})
	if err == nil {
		t.Error("gemini without key should error")
	}
}
