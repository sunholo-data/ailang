package protocol

import (
	"fmt"
	"strings"
)

// ToolAnnotations is the MCP `annotations` object on a tool: behaviour hints
// clients use to decide whether a call needs confirmation. Directory listings
// (Anthropic, OpenAI) require every tool to declare readOnlyHint or
// destructiveHint, so a tool without annotations cannot be listed.
//
// DestructiveHint and OpenWorldHint are pointers because the MCP spec defaults
// both to true when absent; nil means "not stated", not false.
type ToolAnnotations struct {
	ReadOnlyHint    bool  `json:"readOnlyHint"`
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	IdempotentHint  bool  `json:"idempotentHint"`
	OpenWorldHint   *bool `json:"openWorldHint,omitempty"`
}

// ToolHintWords is the closed vocabulary of @mcp_hints("...") arguments.
var ToolHintWords = []string{"readOnly", "destructive", "idempotent", "openWorld"}

// ValidateToolHints rejects unknown hint words, duplicates, and the one
// contradictory pair. The parser calls it so `ailang check` catches a typo
// before a server ever starts.
func ValidateToolHints(words []string) error {
	seen := make(map[string]bool, len(words))
	for _, w := range words {
		if !isToolHintWord(w) {
			return fmt.Errorf("unknown MCP hint %q; supported: %s", w, strings.Join(ToolHintWords, ", "))
		}
		if seen[w] {
			return fmt.Errorf("duplicate MCP hint %q", w)
		}
		seen[w] = true
	}
	if seen["readOnly"] && seen["destructive"] {
		return fmt.Errorf("MCP hints readOnly and destructive contradict each other")
	}
	return nil
}

// ResolveToolHints turns a function's declared hints into wire annotations.
//
//   - declared: the @mcp_hints list is the COMPLETE statement — a word that is
//     absent is false, including destructive and openWorld (whose MCP defaults
//     are true). Writing @mcp_hints("openWorld") therefore means "additive,
//     not destructive", which is what an author listing hints intends.
//   - not declared, noEffects (an empty declared effect row): the checker
//     proves it touches nothing, so it is read-only and closed-world.
//   - not declared, effectful: nil. The caller must not guess; a wrong
//     readOnlyHint is worse than none.
func ResolveToolHints(words []string, declared, noEffects bool) (*ToolAnnotations, error) {
	if !declared {
		if noEffects {
			return &ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: boolPtr(false)}, nil
		}
		return nil, nil
	}
	if err := ValidateToolHints(words); err != nil {
		return nil, err
	}
	has := func(w string) bool {
		for _, x := range words {
			if x == w {
				return true
			}
		}
		return false
	}
	a := &ToolAnnotations{
		ReadOnlyHint:   has("readOnly"),
		IdempotentHint: has("idempotent"),
		OpenWorldHint:  boolPtr(has("openWorld")),
	}
	// destructiveHint is only meaningful when the tool writes.
	if !a.ReadOnlyHint {
		a.DestructiveHint = boolPtr(has("destructive"))
	}
	return a, nil
}

func isToolHintWord(w string) bool {
	for _, x := range ToolHintWords {
		if x == w {
			return true
		}
	}
	return false
}

func boolPtr(b bool) *bool { return &b }
