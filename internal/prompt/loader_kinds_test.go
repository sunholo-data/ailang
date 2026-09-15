package prompt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The three shipped series all load through the one loader, from disk.
func TestKinds_LoadFromRepo(t *testing.T) {
	for _, k := range []Kind{Syntax, Agent, DevTools} {
		content, ver, err := k.LoadPromptWithVersion("latest")
		if err != nil {
			t.Fatalf("%q: %v", k, err)
		}
		if content == "" || ver == "" {
			t.Fatalf("%q: empty content or version", k)
		}
		active, err := k.GetActiveVersion()
		if err != nil || active != ver {
			t.Fatalf("%q: active %q (err %v) != resolved %q", k, active, err, ver)
		}
		versions, err := k.ListVersions()
		if err != nil || len(versions) == 0 {
			t.Fatalf("%q: ListVersions %v / %v", k, versions, err)
		}
	}
}

// Each series keeps its pre-consolidation error text: the syntax prompt
// explains the version namespace, the others say which manifest was searched.
func TestKinds_UnknownVersionText(t *testing.T) {
	_, err := Agent.LoadPrompt("nope")
	if err == nil || err.Error() != `version "nope" not found in agent versions.json` {
		t.Fatalf("agent: %v", err)
	}
	_, err = DevTools.GetVersionMetadata("nope")
	if err == nil || err.Error() != `version "nope" not found in devtools versions.json` {
		t.Fatalf("devtools: %v", err)
	}
	_, err = Syntax.LoadPrompt("nope")
	if err == nil || !strings.Contains(err.Error(), `"nope" is not a known prompt version`) {
		t.Fatalf("syntax: %v", err)
	}
}

func writeManifest(t *testing.T, dir string, m VersionsManifest) string {
	t.Helper()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "prompts", "versions.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoader_VerifyHash(t *testing.T) {
	dir := t.TempDir()
	body := []byte("# prompt\n")
	if err := os.MkdirAll(filepath.Join(dir, "prompts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "prompts", "p.md"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	good := SHA256Hex(body)
	bad := strings.Repeat("0", 64)
	frozen := &FrozenMarker{At: "2026-01-01", Reason: "banked", EvidenceCount: 1, EvidenceExample: "x"}
	manifestPath := writeManifest(t, dir, VersionsManifest{
		Versions: map[string]VersionMetadata{
			"ok":            {File: "prompts/p.md", Hash: good},
			"placeholder":   {File: "prompts/p.md", Hash: "PLACEHOLDER"},
			"mutable-bad":   {File: "prompts/p.md", Hash: bad},
			"frozen-ok":     {File: "prompts/p.md", Hash: good, Frozen: frozen},
			"frozen-bad":    {File: "prompts/p.md", Hash: bad, Frozen: frozen},
			"frozen-nohash": {File: "prompts/p.md", Hash: "PLACEHOLDER", Frozen: frozen},
		},
		Active: "ok",
	})

	verified := NewLoader(Syntax, WithManifestFile(manifestPath), WithRoot(dir), WithVerify())
	unverified := NewLoader(Syntax, WithManifestFile(manifestPath), WithRoot(dir))

	for _, id := range []string{"ok", "placeholder", "frozen-ok", ""} {
		if got, err := verified.LoadPrompt(id); err != nil || got != string(body) {
			t.Errorf("verified %q: %q / %v", id, got, err)
		}
	}
	cases := map[string]string{
		"mutable-bad":   "hash mismatch",
		"frozen-bad":    "is FROZEN",
		"frozen-nohash": "not a 64-hex sha256",
	}
	for id, want := range cases {
		_, err := verified.LoadPrompt(id)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("verified %q: want %q, got %v", id, want, err)
		}
		// Without WithVerify the same entries load — the hash is informational.
		if _, err := unverified.LoadPrompt(id); err != nil {
			t.Errorf("unverified %q: %v", id, err)
		}
	}
}

// A manifest whose active field is the "latest" sentinel resolves to the
// newest version tagged production (the eval registry convention).
func TestLoader_LatestSentinel(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeManifest(t, dir, VersionsManifest{
		Versions: map[string]VersionMetadata{
			"v1":      {File: "prompts/v1.md", Hash: "PLACEHOLDER", Created: "2025-01-01", Tags: []string{"production"}},
			"v2":      {File: "prompts/v2.md", Hash: "PLACEHOLDER", Created: "2025-02-01", Tags: []string{"production"}},
			"v3-beta": {File: "prompts/v3.md", Hash: "PLACEHOLDER", Created: "2025-03-01", Tags: []string{"experimental"}},
		},
		Active: "latest",
	})
	l := NewLoader(Syntax, WithManifestFile(manifestPath), WithRoot(dir))
	if v, err := l.GetActiveVersion(); err != nil || v != "v2" {
		t.Fatalf("active = %q, %v; want v2", v, err)
	}
	empty := writeManifest(t, t.TempDir(), VersionsManifest{Versions: map[string]VersionMetadata{}})
	if _, err := NewLoader(Syntax, WithManifestFile(empty)).LoadPrompt(""); err == nil ||
		!strings.Contains(err.Error(), "no active prompt version") {
		t.Fatalf("empty active: %v", err)
	}
}

// WithManifestFile is a request for THAT file: the embedded FS must not shadow it.
func TestLoader_ManifestFileBypassesEmbedded(t *testing.T) {
	prev := embeddedPrompts
	defer func() { embeddedPrompts = prev }()
	embeddedPrompts = os.DirFS(findProjectRoot())

	dir := t.TempDir()
	manifestPath := writeManifest(t, dir, VersionsManifest{
		Versions: map[string]VersionMetadata{"only": {File: "prompts/only.md", Hash: "PLACEHOLDER"}},
		Active:   "only",
	})
	if err := os.WriteFile(filepath.Join(dir, "prompts", "only.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := NewLoader(Syntax, WithManifestFile(manifestPath), WithRoot(dir)).LoadPrompt("")
	if err != nil || got != "mine" {
		t.Fatalf("got %q, %v", got, err)
	}
}
