package lsp

import (
	"context"

	"go.lsp.dev/protocol"
)

// References implements LSP textDocument/references. Returns every
// occurrence of the identifier under the cursor across all currently
// open documents in this LSP session.
//
// MVP scope (per design doc): the search is scoped to documents that
// have been didOpen-ed or didSave-ed in this session. Workspace-wide
// scan (parsing every .ail under the project root) is deferred to the
// M-AILANG-LSP-WORKSPACE-SCAN follow-up.
//
// includeDeclaration honors the LSP spec — when true, the declaration
// site is included in the results; when false, only use-sites.
func (s *Server) References(_ context.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	if params == nil {
		return nil, nil
	}
	if !s.initialized.Load() {
		return nil, nil
	}

	docURI := params.TextDocument.URI
	path := uriToPath(docURI)

	v, ok := s.docs.Load(docURI)
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

	line := int(params.Position.Line) + 1
	col := int(params.Position.Character) + 1
	id := idx.Lookup(line, col)
	if id == nil {
		return nil, nil
	}
	target := id.Name

	var out []protocol.Location

	// Walk every open document's PositionIndex for matching identifiers.
	s.docs.Range(func(key, val interface{}) bool {
		du, ok := key.(protocol.DocumentURI)
		if !ok {
			return true
		}
		dpath := uriToPath(du)
		dsrc, _ := val.(string)
		didx := s.indexes.get(dpath)
		if didx == nil {
			didx = BuildPositionIndex(dpath, dsrc)
			if didx == nil {
				return true
			}
			s.indexes.put(dpath, didx)
		}
		for _, ident := range didx.AllIdents() {
			if ident.Name != target {
				continue
			}
			out = append(out, protocol.Location{
				URI:   du,
				Range: identRange(ident),
			})
		}
		return true
	})

	if params.Context.IncludeDeclaration {
		// Try to add the declaration too. Same-file is checked first via
		// findFuncInFile; cross-module via the loaded modules table.
		if loc, ok := s.findFuncInFile(path, src, target); ok {
			if !containsLocation(out, loc) {
				out = append(out, loc)
			}
		}
		// Also check session-open documents for a FuncDecl named target.
		s.docs.Range(func(key, val interface{}) bool {
			du, ok := key.(protocol.DocumentURI)
			if !ok || du == docURI {
				return true
			}
			dpath := uriToPath(du)
			dsrc, _ := val.(string)
			if loc, ok := s.findFuncInFile(dpath, dsrc, target); ok {
				if !containsLocation(out, loc) {
					out = append(out, loc)
				}
			}
			return true
		})
	}

	return out, nil
}

func containsLocation(locs []protocol.Location, want protocol.Location) bool {
	for _, l := range locs {
		if l.URI == want.URI &&
			l.Range.Start == want.Range.Start &&
			l.Range.End == want.Range.End {
			return true
		}
	}
	return false
}
