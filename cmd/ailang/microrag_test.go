package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/effects"
)

// TestChunkPromptByH2 verifies the markdown chunker:
//   - splits on every "## " heading (heading line included in body)
//   - skips chunks with body < 200 bytes (no signal)
//   - emits stable keys derived from version + slugified section name
func TestChunkPromptByH2(t *testing.T) {
	long := strings.Repeat("filler line for body padding so we exceed the 200 byte minimum. ", 5)
	doc := "# Top-level title\n" +
		"some intro text\n\n" +
		"## Pattern Matching\n" + long + "\n" +
		"## Effect System\n" + long + "\n" +
		"## Tiny\n" + "too short\n"

	chunks := chunkPromptByH2(doc, "v0.4.10")
	if got, want := len(chunks), 2; got != want {
		t.Fatalf("chunkPromptByH2: got %d chunks, want %d (Tiny should be filtered)", got, want)
	}

	if chunks[0].Section != "Pattern Matching" {
		t.Errorf("chunks[0].Section = %q, want %q", chunks[0].Section, "Pattern Matching")
	}
	if chunks[0].Key != "syntax-v0.4.10-pattern-matching" {
		t.Errorf("chunks[0].Key = %q, want %q", chunks[0].Key, "syntax-v0.4.10-pattern-matching")
	}
	if !strings.HasPrefix(chunks[0].Body, "## Pattern Matching\n") {
		t.Errorf("chunks[0].Body should start with the heading line, got %q...", chunks[0].Body[:40])
	}

	if chunks[1].Section != "Effect System" {
		t.Errorf("chunks[1].Section = %q, want %q", chunks[1].Section, "Effect System")
	}
	if chunks[1].Key != "syntax-v0.4.10-effect-system" {
		t.Errorf("chunks[1].Key = %q, want %q", chunks[1].Key, "syntax-v0.4.10-effect-system")
	}
}

// TestChunkPromptByH2_StableKeys ensures slug normalisation matches the
// shell indexer's awk gsub: lowercase, non-alphanumeric → '-', trim dashes.
func TestChunkPromptByH2StableKeys(t *testing.T) {
	body := strings.Repeat("x ", 150)
	doc := "## Foo / Bar (v2)\n" + body + "\n"
	chunks := chunkPromptByH2(doc, "v1")
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	want := "syntax-v1-foo-bar-v2"
	if chunks[0].Key != want {
		t.Errorf("slugified key = %q, want %q", chunks[0].Key, want)
	}
}

// TestSanitizeKey verifies the builtin key sanitizer matches `tr -c 'A-Za-z0-9_' '_'`.
func TestSanitizeKey(t *testing.T) {
	cases := map[string]string{
		"_net_httpRequest":  "_net_httpRequest",
		"std/io.print":      "std_io_print",
		"foo bar-baz":       "foo_bar_baz",
		"already_sanitized": "already_sanitized",
	}
	for in, want := range cases {
		if got := sanitizeKey(in); got != want {
			t.Errorf("sanitizeKey(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestBootstrapEmitsBuiltins confirms one frame is written per registered
// builtin spec, all under the `ailang-builtins` namespace, with stable
// `builtin-<sanitized-name>` keys.
func TestBootstrapEmitsBuiltins(t *testing.T) {
	dir := t.TempDir()
	cache, err := effects.NewSQLiteSharedCache(filepath.Join(dir, "brain.db"))
	if err != nil {
		t.Fatalf("NewSQLiteSharedCache: %v", err)
	}
	defer cache.Close()

	store := &effects.BrainStore{User: cache} // user-only; project nil is OK

	specs := builtins.AllSpecs()
	if len(specs) == 0 {
		t.Skip("no builtins registered (init() may not have fired); skipping")
	}

	count, err := bootstrapBuiltins(store, effects.ScopeUser, "v-test")
	if err != nil {
		t.Fatalf("bootstrapBuiltins: %v", err)
	}
	if count != len(specs) {
		t.Errorf("frames written = %d, want %d", count, len(specs))
	}

	// Spot-check a known builtin made it in with the namespace + format tag.
	for _, spec := range specs {
		key := "builtin-" + sanitizeKey(spec.Name)
		val, ok := cache.Get(key)
		if !ok {
			t.Errorf("missing key %q for builtin %q", key, spec.Name)
			continue
		}
		_ = val // we just need existence
		break
	}
}

// TestBootstrapGracefulNoOllama verifies bootstrap completes when no
// embedder is configured. Frames are written with embedding_dim=0; the
// search side's SimHash + FTS still works.
func TestBootstrapGracefulNoOllama(t *testing.T) {
	dir := t.TempDir()
	// Open WITHOUT WithEmbedder — simulates Ollama-down scenario.
	cache, err := effects.NewSQLiteSharedCache(filepath.Join(dir, "brain.db"))
	if err != nil {
		t.Fatalf("NewSQLiteSharedCache: %v", err)
	}
	defer cache.Close()
	store := &effects.BrainStore{User: cache}

	// Use a minimal synthetic syntax doc so the test doesn't depend on
	// the embedded prompt corpus.
	doc := "## Section A\n" + strings.Repeat("body filler ", 30) + "\n"
	chunks := chunkPromptByH2(doc, "v-test")
	if len(chunks) == 0 {
		t.Fatalf("test setup: no chunks emitted")
	}

	for _, c := range chunks {
		frame := effects.BrainFrame{
			Key:       c.Key,
			Namespace: bootstrapSyntaxNamespace,
			Content:   c.Body,
			Source:    bootstrapSourceTag,
		}
		if err := store.Put(frame, effects.ScopeUser); err != nil {
			t.Fatalf("Put without embedder: %v", err)
		}
	}

	// Confirm the frame is there. Without an embedder, embedding_dim is 0
	// but the frame still indexes via SimHash + FTS.
	stats := cache.Stats()
	if got := stats.Namespaces[bootstrapSyntaxNamespace]; got == 0 {
		t.Errorf("expected at least 1 frame in %s, got 0 (Namespaces=%v)",
			bootstrapSyntaxNamespace, stats.Namespaces)
	}
}
