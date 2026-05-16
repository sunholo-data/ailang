package lsp

import (
	"context"
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/types"
	"go.lsp.dev/protocol"
)

// Hover implements LSP textDocument/hover. Returns the inferred type and
// effect row for the identifier under the cursor when available, or just
// the identifier name + a deferred-types note for local bindings.
//
// MVP scope: hover answers for top-level identifiers (function decls and
// other module-exported names) by consulting Result.Interface. Local
// bindings (let-bound vars, lambda params) get a basic name+location
// response without a type — full local-type plumbing through Core node IDs
// is the M-AILANG-LSP-LOCAL-TYPES follow-up.
func (s *Server) Hover(_ context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	if params == nil {
		return nil, nil
	}
	if !s.initialized.Load() {
		return nil, nil
	}

	uri := params.TextDocument.URI
	path := uriToPath(uri)

	v, ok := s.docs.Load(uri)
	if !ok {
		return nil, nil
	}
	src, _ := v.(string)
	if src == "" {
		return nil, nil
	}

	idx := s.indexes.get(path)
	if idx == nil {
		idx = BuildPositionIndex(path, src)
		if idx == nil {
			return nil, nil
		}
		s.indexes.put(path, idx)
	}

	// LSP positions are 0-indexed; ast.Pos is 1-indexed.
	line := int(params.Position.Line) + 1
	col := int(params.Position.Character) + 1

	id := idx.Lookup(line, col)
	if id == nil {
		return nil, nil
	}

	// Try to look up the inferred type. Fall back to a name-only hover
	// when the identifier is a local binding (covered by future milestone).
	contents := s.formatHoverContents(id, path, src)
	rng := identRange(id)

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: contents,
		},
		Range: &rng,
	}, nil
}

// formatHoverContents produces the hover body for the given identifier.
// Tries (in order): the file's own iface for top-level exports; the
// imported modules' ifaces for cross-module references; finally a
// name-and-location-only fallback.
func (s *Server) formatHoverContents(id *ast.Identifier, path string, src string) string {
	// Re-run the pipeline against the in-memory buffer so the type info we
	// hand back reflects the user's current edits, not stale on-disk state.
	cfg := pipeline.Config{}
	srcBundle := pipeline.Source{Code: src, Filename: path}
	result, _ := pipeline.Run(cfg, srcBundle)

	if scheme := lookupTypeInResult(result, id.Name); scheme != nil {
		return formatScheme(id.Name, scheme)
	}

	// Fallback: name + position only. Document the limit so users know
	// it's a deferred capability, not a bug.
	return fmt.Sprintf("**%s**\n\n_local binding — type info for non-exported names is deferred to M-AILANG-LSP-LOCAL-TYPES_", id.Name)
}

// lookupTypeInResult searches the root module's iface and any loaded
// dependent modules' ifaces for an export matching name.
func lookupTypeInResult(result pipeline.Result, name string) *types.Scheme {
	if result.Interface != nil {
		if item, ok := result.Interface.GetExport(name); ok && item != nil {
			return item.Type
		}
	}
	for _, mod := range result.Modules {
		if mod == nil || mod.Iface == nil {
			continue
		}
		if item, ok := mod.Iface.GetExport(name); ok && item != nil {
			return item.Type
		}
	}
	return nil
}

// formatScheme renders a Scheme as a Markdown hover body. Includes the
// effect row when present so AI agents see the full callable signature.
func formatScheme(name string, scheme *types.Scheme) string {
	if scheme == nil {
		return fmt.Sprintf("**%s**", name)
	}
	typeStr := "?"
	if scheme.Type != nil {
		typeStr = scheme.Type.String()
	}
	return fmt.Sprintf("**%s**\n\n```ailang\n%s : %s\n```", name, name, typeStr)
}

// identRange returns the LSP Range covering an Identifier (start at its
// Pos, end at start + len(name)).
func identRange(id *ast.Identifier) protocol.Range {
	startLine := uint32(max0(id.Pos.Line - 1))
	startCol := uint32(max0(id.Pos.Column - 1))
	endCol := startCol + uint32(len(id.Name))
	return protocol.Range{
		Start: protocol.Position{Line: startLine, Character: startCol},
		End:   protocol.Position{Line: startLine, Character: endCol},
	}
}

// asserting that *iface.Iface stays in our import set even if the iface
// lookup path is stripped by a future change.
var _ = (*iface.Iface)(nil)
