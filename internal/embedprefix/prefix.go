// Package embedprefix applies model- and role-aware task-instruction prefixes to
// text before it is embedded.
//
// Modern instruction-tuned embedders require a task prefix prepended to every
// input, and use DIFFERENT prefixes for documents vs queries:
//
//   - EmbeddingGemma (Google):
//     document:    "title: none | text: {text}"
//     query:       "task: search result | query: {text}"
//     code query:  "task: code retrieval | query: {text}"
//   - nomic-embed-text (Nomic):
//     document:    "search_document: {text}"
//     query:       "search_query: {text}"
//
// Without the prefix, retrieval relevance degrades sharply (EmbeddingGemma
// especially). The role must therefore be threaded from each call site — the
// corpus-document-embed path uses RoleDocument, the query/retrieval path uses
// RoleQuery — so this package exposes a tiny pure helper plus a wrapper.
//
// It is a leaf package (no dependency on internal/effects or internal/messaging),
// so both embedder subsystems can use it without an import cycle.
package embedprefix

import "strings"

// Role distinguishes how an embedded text will be used.
type Role int

const (
	// RoleNone applies no prefix (default; used by subsystems that should keep
	// raw-text behaviour, e.g. message-routing envelope vectors).
	RoleNone Role = iota
	// RoleDocument is a corpus document stored for later retrieval.
	RoleDocument
	// RoleQuery is a retrieval query (general "search result" task).
	RoleQuery
	// RoleCodeQuery is a retrieval query specifically for code.
	RoleCodeQuery
)

// ApplyTaskPrefix returns text with the model+role-appropriate instruction prefix
// prepended. Model matching is case-insensitive substring, so tags like
// "embeddinggemma:latest" or "ollama:nomic-embed-text" match. Unknown models or
// RoleNone return text unchanged (safe no-op) — preserving behaviour for any
// embedder/subsystem not explicitly handled.
func ApplyTaskPrefix(model string, role Role, text string) string {
	if role == RoleNone {
		return text
	}
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "embeddinggemma"):
		switch role {
		case RoleDocument:
			return "title: none | text: " + text
		case RoleQuery:
			return "task: search result | query: " + text
		case RoleCodeQuery:
			return "task: code retrieval | query: " + text
		}
	case strings.Contains(m, "nomic-embed"):
		switch role {
		case RoleDocument:
			return "search_document: " + text
		case RoleQuery, RoleCodeQuery:
			return "search_query: " + text
		}
	}
	return text
}

// roleEmbedder is the minimal surface needed to prefix-then-embed. Both
// effects.Embedder and messaging.Embedder satisfy it structurally, so this leaf
// package depends on neither.
type roleEmbedder interface {
	Embed(text string) ([]float32, error)
	ModelName() string
}

// EmbedWithRole prefixes text for the embedder's model and the given role, then
// delegates to the embedder's Embed.
func EmbedWithRole(e roleEmbedder, role Role, text string) ([]float32, error) {
	return e.Embed(ApplyTaskPrefix(e.ModelName(), role, text))
}
