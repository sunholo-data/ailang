package embedprefix

import "testing"

func TestApplyTaskPrefix(t *testing.T) {
	tests := []struct {
		name  string
		model string
		role  Role
		text  string
		want  string
	}{
		// EmbeddingGemma — the model card's required prefixes.
		{"gemma document", "embeddinggemma", RoleDocument, "let x = 1", "title: none | text: let x = 1"},
		{"gemma query", "embeddinggemma", RoleQuery, "roman numeral", "task: search result | query: roman numeral"},
		{"gemma code query", "embeddinggemma", RoleCodeQuery, "parse args", "task: code retrieval | query: parse args"},
		{"gemma tag suffix matches", "ollama:embeddinggemma:latest", RoleQuery, "q", "task: search result | query: q"},

		// nomic-embed-text — search_document/search_query.
		{"nomic document", "nomic-embed-text", RoleDocument, "let x = 1", "search_document: let x = 1"},
		{"nomic query", "nomic-embed-text", RoleQuery, "roman numeral", "search_query: roman numeral"},
		{"nomic code query falls back to query", "nomic-embed-text", RoleCodeQuery, "q", "search_query: q"},
		{"nomic prefixed tag matches", "ollama:nomic-embed-text", RoleDocument, "d", "search_document: d"},

		// Unknown model + RoleNone → no-op (safety: never corrupt unhandled paths).
		{"unknown model no-op", "text-embedding-3-small", RoleQuery, "q", "q"},
		{"role none no-op even for gemma", "embeddinggemma", RoleNone, "q", "q"},
		{"empty model no-op", "", RoleDocument, "d", "d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ApplyTaskPrefix(tt.model, tt.role, tt.text); got != tt.want {
				t.Errorf("ApplyTaskPrefix(%q, %v, %q) = %q, want %q", tt.model, tt.role, tt.text, got, tt.want)
			}
		})
	}
}

// stubEmbedder records the text it was asked to embed.
type stubEmbedder struct {
	model string
	got   string
}

func (s *stubEmbedder) Embed(text string) ([]float32, error) { s.got = text; return []float32{1}, nil }
func (s *stubEmbedder) ModelName() string                    { return s.model }

func TestEmbedWithRole_PrefixesBeforeEmbedding(t *testing.T) {
	s := &stubEmbedder{model: "embeddinggemma"}
	if _, err := EmbedWithRole(s, RoleQuery, "fizzbuzz"); err != nil {
		t.Fatalf("EmbedWithRole error: %v", err)
	}
	if want := "task: search result | query: fizzbuzz"; s.got != want {
		t.Errorf("embedded %q, want %q", s.got, want)
	}
}
