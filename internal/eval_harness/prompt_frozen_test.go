package eval_harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func frozenFixture(t *testing.T, marker *FrozenMarker, hash string, content string) (*PromptLoader, string) {
	t.Helper()
	root := t.TempDir()
	if hash == "" {
		h := sha256.Sum256([]byte(content))
		hash = hex.EncodeToString(h[:])
	}
	version := PromptVersion{File: "prompts/x.md", Hash: hash, Frozen: marker}
	reg := PromptRegistry{Versions: map[string]PromptVersion{"x": version}, Active: "x"}
	data, _ := json.Marshal(reg)
	os.MkdirAll(filepath.Join(root, "prompts"), 0o755)
	os.WriteFile(filepath.Join(root, "prompts", "x.md"), []byte(content), 0o644)
	os.WriteFile(filepath.Join(root, "prompts", "versions.json"), data, 0o644)
	loader, err := NewPromptLoader(filepath.Join(root, "prompts", "versions.json"))
	if err != nil {
		t.Fatal(err)
	}
	return loader, filepath.Join(root, "prompts", "x.md")
}

func TestLoadPrompt_FrozenTamperEmitsTeachingError(t *testing.T) {
	m := &FrozenMarker{At: "2026-08-27", Reason: "banked", EvidenceCount: 4242, EvidenceExample: "eval_results/baselines/x/y.json"}
	l, path := frozenFixture(t, m, "", "original")
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString("!")
	f.Close()
	_, err := l.LoadPrompt("x")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"is FROZEN", "cited by 4242 banked baseline files", "create_prompt_version.sh"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q: %v", want, err)
		}
	}
}
func TestLoadPrompt_NeverBankedTamperRegenSucceeds(t *testing.T) {
	l, path := frozenFixture(t, nil, "", "new bytes")
	got, err := l.LoadPrompt("x")
	if err != nil || got != "new bytes" {
		t.Fatalf("got %q, %v (%s)", got, err, path)
	}
}
func TestLoadPrompt_NeverBankedStaleHashTeachesRegen(t *testing.T) {
	l, path := frozenFixture(t, nil, "", "old")
	os.WriteFile(path, []byte("new"), 0o644)
	_, err := l.LoadPrompt("x")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"hash mismatch", "not yet banked", "in-place editing is allowed", "shasum -a 256"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q: %v", want, err)
		}
	}
}
func TestLoadPrompt_FrozenPlaceholderRefused(t *testing.T) {
	l, _ := frozenFixture(t, &FrozenMarker{}, "PLACEHOLDER", "x")
	_, err := l.LoadPrompt("x")
	if err == nil || !strings.Contains(err.Error(), "is FROZEN") || !strings.Contains(err.Error(), "unenforceable hash") {
		t.Fatalf("unexpected: %v", err)
	}
}
func TestLoadPrompt_UnfrozenPlaceholderStillLoads(t *testing.T) {
	l, _ := frozenFixture(t, nil, "PLACEHOLDER", "x")
	got, err := l.LoadPrompt("x")
	if err != nil || got != "x" {
		t.Fatalf("got %q, %v", got, err)
	}
}
func TestPromptRegistry_FrozenFieldRoundTrips(t *testing.T) {
	r := PromptRegistry{Versions: map[string]PromptVersion{"f": {Frozen: &FrozenMarker{At: "a", Reason: "banked", EvidenceCount: 2, EvidenceExample: "e"}}, "m": {}}}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var got PromptRegistry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Versions["f"].Frozen.EvidenceCount != 2 {
		t.Fatalf("marker lost: %#v", got)
	}
	if strings.Contains(string(data), `"m":{"file":"","hash":"","description":"","created":"","tags":null,"notes":"","frozen"`) {
		t.Fatalf("nil marker emitted: %s", data)
	}
}
