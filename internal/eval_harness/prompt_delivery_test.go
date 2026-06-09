package eval_harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPersistentSystemPromptEnabled pins the post-experiment default: full-prompt
// persistence (the "MOVE" delivery) is OFF unless explicitly re-enabled, because
// the 2026-06-06 delivery experiment showed it bloats context and underperforms
// turn-1 concatenation (1/6 vs 3/6 on local qwen3.5).
func TestPersistentSystemPromptEnabled(t *testing.T) {
	cases := []struct {
		env  string
		want bool
	}{
		{"", false},      // default: OFF (turn-1 concatenation wins)
		{"0", false},     // explicit off
		{"false", false}, //
		{"OFF", false},   // case-insensitive
		{" no ", false},  // trimmed
		{"1", true},      // explicit on (A/B re-enable)
		{"true", true},   //
		{"on", true},     //
		{" Yes ", true},  // trimmed, case-insensitive
	}
	for _, tc := range cases {
		t.Run("env="+tc.env, func(t *testing.T) {
			t.Setenv("AILANG_EVAL_PERSIST_PROMPT", tc.env)
			if got := persistentSystemPromptEnabled(); got != tc.want {
				t.Errorf("persistentSystemPromptEnabled() with %q = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}

func TestMaybePrependTrapsCard(t *testing.T) {
	const directive = "Solve the task."

	// Redirect the default card path at a temp file for the duration of the test
	// so the "default on" behavior is deterministic regardless of cwd.
	writeCard := func(t *testing.T, body string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "card.md")
		if err := os.WriteFile(p, []byte(body+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	withDefaultPath := func(t *testing.T, p string) {
		t.Helper()
		orig := trapsCardDefaultPath
		trapsCardDefaultPath = p
		t.Cleanup(func() { trapsCardDefaultPath = orig })
	}

	assertPrepended := func(t *testing.T, got, card string) {
		t.Helper()
		if !strings.HasPrefix(got, card) {
			t.Errorf("card not at front: %q", got)
		}
		if !strings.HasSuffix(got, directive) {
			t.Errorf("directive not preserved at end: %q", got)
		}
		if !strings.Contains(got, "\n---\n") {
			t.Errorf("missing separator between card and directive: %q", got)
		}
	}

	t.Run("default on: unset env loads the default card path", func(t *testing.T) {
		card := "DEFAULT-CARD: no list indexing."
		withDefaultPath(t, writeCard(t, card))
		t.Setenv("AILANG_EVAL_TRAPS_CARD", "")
		assertPrepended(t, maybePrependTrapsCard(directive), card)
	})

	t.Run("explicit off disables the card", func(t *testing.T) {
		withDefaultPath(t, writeCard(t, "SHOULD-NOT-APPEAR")) // present but must be skipped
		for _, off := range []string{"off", "0", "false", "no", "none", "OFF"} {
			t.Setenv("AILANG_EVAL_TRAPS_CARD", off)
			if got := maybePrependTrapsCard(directive); got != directive {
				t.Errorf("off=%q: got %q, want unchanged", off, got)
			}
		}
	})

	t.Run("explicit path overrides the default", func(t *testing.T) {
		withDefaultPath(t, writeCard(t, "DEFAULT-CARD"))
		override := "OVERRIDE-CARD: imports first."
		t.Setenv("AILANG_EVAL_TRAPS_CARD", writeCard(t, override))
		assertPrepended(t, maybePrependTrapsCard(directive), override)
	})

	t.Run("unreadable default falls back to directive unchanged", func(t *testing.T) {
		withDefaultPath(t, filepath.Join(t.TempDir(), "does-not-exist.md"))
		t.Setenv("AILANG_EVAL_TRAPS_CARD", "")
		if got := maybePrependTrapsCard(directive); got != directive {
			t.Errorf("missing default: got %q, want unchanged", got)
		}
	})

	t.Run("unreadable override path falls back to directive unchanged", func(t *testing.T) {
		t.Setenv("AILANG_EVAL_TRAPS_CARD", filepath.Join(t.TempDir(), "does-not-exist.md"))
		if got := maybePrependTrapsCard(directive); got != directive {
			t.Errorf("bad override path: got %q, want unchanged", got)
		}
	})

	// The real shipped card must exist at the default path and parse as the
	// expected salience card — this guards against the default silently no-op'ing
	// in production because someone moved or renamed the file.
	t.Run("shipped default card exists at repo path", func(t *testing.T) {
		repoCard := filepath.Join("..", "..", "prompts", "agent", "dialect-traps.md")
		body, err := os.ReadFile(repoCard)
		if err != nil {
			t.Fatalf("shipped traps card missing at %s (trapsCardDefaultPath=%q): %v", repoCard, trapsCardDefaultPath, err)
		}
		if !strings.Contains(string(body), "DIALECT TRAPS") {
			t.Errorf("shipped card at %s does not look like the dialect-traps card", repoCard)
		}
	})
}
