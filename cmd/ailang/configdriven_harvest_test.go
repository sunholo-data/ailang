package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// TestHarvestAndRegisterFromDir_RootManifestOnly verifies the load-bearing
// startup wiring: an ailang.toml with [[ai_provider]] in the project root
// is automatically harvested. No lock file, no dependencies — just the
// project's own config.
func TestHarvestAndRegisterFromDir_RootManifestOnly(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	dir := t.TempDir()
	tomlContent := `
[package]
name = "test/harvest_root"
version = "0.1.0"
edition = "1"

[exports]
modules = ["test/harvest_root/core"]

[[ai_provider]]
schema_version = 1
name = "harvest-test"
endpoint = "http://localhost:9999/v1/chat/completions"
request_shape = "openai_chat"
response_path = "$.choices[0].message.content"
auth = { type = "none" }
cost = { input_per_1m_usd = 0.5, output_per_1m_usd = 1.5 }
`
	if err := os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := HarvestAndRegisterFromDir(dir); err != nil {
		t.Fatalf("HarvestAndRegisterFromDir failed: %v", err)
	}

	provider, ok := ai.GlobalProviderRegistry.Lookup("harvest-test")
	if !ok {
		t.Fatal("harvest-test provider not registered after harvest")
	}
	if provider.Name() != "harvest-test" {
		t.Errorf("provider name = %q", provider.Name())
	}
	source := ai.GlobalProviderRegistry.SourceOf("harvest-test")
	if !strings.HasSuffix(source, "ailang.toml") {
		t.Errorf("source path should end with ailang.toml, got %q", source)
	}
}

// TestHarvestAndRegisterFromDir_NoManifestIsHarmless verifies that running
// the harvest in a directory without ailang.toml is a quiet no-op — bare
// projects continue to work.
func TestHarvestAndRegisterFromDir_NoManifestIsHarmless(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	dir := t.TempDir() // empty
	if err := HarvestAndRegisterFromDir(dir); err != nil {
		t.Errorf("bare-project harvest should not error, got: %v", err)
	}
	if names := ai.GlobalProviderRegistry.Names(); len(names) != 0 {
		t.Errorf("expected empty registry for bare project, got: %v", names)
	}
}

// TestHarvestAndRegisterFromDir_MalformedManifestIsHarmless: if ailang.toml
// exists but is invalid, harvest silently skips (the pipeline reports the
// real error from validation; we don't double-report).
func TestHarvestAndRegisterFromDir_MalformedManifestIsHarmless(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte("this is not valid toml at all"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := HarvestAndRegisterFromDir(dir); err != nil {
		t.Errorf("malformed manifest should not crash harvest, got: %v", err)
	}
}

// TestHarvestAndRegisterFromDir_DispatchEndToEnd: full happy path —
// ailang.toml on disk → harvest → registry → dispatch → mock HTTP →
// response text. This is the integration test for the load-bearing
// "startup wiring" piece of M4.
func TestHarvestAndRegisterFromDir_DispatchEndToEnd(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"harvest path works"}}]}`))
	}))
	defer server.Close()

	dir := t.TempDir()
	tomlContent := `
[package]
name = "test/harvest_dispatch"
version = "0.1.0"
edition = "1"

[exports]
modules = ["test/harvest_dispatch/core"]

[[ai_provider]]
schema_version = 1
name = "harvest-dispatch-test"
endpoint = "` + server.URL + `"
request_shape = "openai_chat"
response_path = "$.choices[0].message.content"
auth = { type = "none" }
`
	if err := os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := HarvestAndRegisterFromDir(dir); err != nil {
		t.Fatalf("harvest failed: %v", err)
	}

	provider, ok := ai.GlobalProviderRegistry.Lookup("harvest-dispatch-test")
	if !ok {
		t.Fatal("provider not registered")
	}
	resp, err := provider.Generate(t.Context(), &ai.Request{
		Model:      "any-model",
		UserPrompt: "Hello",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp.Text != "harvest path works" {
		t.Errorf("Text = %q", resp.Text)
	}
}

// TestHarvestAndRegisterFromDir_Idempotent verifies that calling harvest
// twice for the same directory is a no-op (no duplicate-name error). This
// matters because setupAIHandler may be called multiple times in the same
// process (e.g. tests, REPL).
func TestHarvestAndRegisterFromDir_Idempotent(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	dir := t.TempDir()
	tomlContent := `
[package]
name = "test/idempotent"
version = "0.1.0"
edition = "1"

[exports]
modules = ["test/idempotent/core"]

[[ai_provider]]
schema_version = 1
name = "idempotent-test"
endpoint = "http://localhost:9999"
request_shape = "openai_chat"
response_path = "$.x"
auth = { type = "none" }
`
	if err := os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := HarvestAndRegisterFromDir(dir); err != nil {
		t.Fatalf("first harvest: %v", err)
	}
	// Second call must not error — re-registering the same provider+source
	// pair is intentionally a no-op (per registry.Register idempotency).
	if err := HarvestAndRegisterFromDir(dir); err != nil {
		t.Errorf("second harvest should be idempotent, got: %v", err)
	}
}

// TestHarvestAndRegisterFromDir_FindsAncestorManifest verifies that the
// harvest function walks UP from the working directory to find the
// nearest ancestor ailang.toml — same lookup behaviour as the rest of
// the toolchain (FindManifest is the shared mechanism).
func TestHarvestAndRegisterFromDir_FindsAncestorManifest(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	rootDir := t.TempDir()
	subDir := filepath.Join(rootDir, "src", "core")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	tomlContent := `
[package]
name = "test/ancestor"
version = "0.1.0"
edition = "1"

[exports]
modules = ["test/ancestor/core"]

[[ai_provider]]
schema_version = 1
name = "ancestor-test"
endpoint = "http://localhost:9999"
request_shape = "openai_chat"
response_path = "$.x"
auth = { type = "none" }
`
	if err := os.WriteFile(filepath.Join(rootDir, "ailang.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}
	// Harvest invoked from a subdirectory must still find the ancestor manifest.
	if err := HarvestAndRegisterFromDir(subDir); err != nil {
		t.Fatalf("harvest from subdir failed: %v", err)
	}
	if _, ok := ai.GlobalProviderRegistry.Lookup("ancestor-test"); !ok {
		t.Error("ancestor manifest not harvested when invoked from subdirectory")
	}
}
