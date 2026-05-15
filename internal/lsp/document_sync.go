package lsp

import (
	"context"

	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// DidOpen stores the document text and runs the pipeline immediately so the
// agent gets diagnostics the moment it opens a .ail file (no save required).
func (s *Server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	if params == nil {
		return nil
	}
	uri := params.TextDocument.URI
	s.docs.Store(uri, params.TextDocument.Text)
	s.publishDiagnosticsFor(ctx, uri, params.TextDocument.Text)
	return nil
}

// DidChange stores the latest buffer text but does NOT re-run the pipeline.
// MVP scope per the design doc: incremental re-typecheck on every keystroke
// is deferred to M-AILANG-LSP-INCREMENTAL. didSave is the diagnostic trigger.
func (s *Server) DidChange(_ context.Context, params *protocol.DidChangeTextDocumentParams) error {
	if params == nil || len(params.ContentChanges) == 0 {
		return nil
	}
	// With TextDocumentSyncKindFull, the last change carries the full text.
	last := params.ContentChanges[len(params.ContentChanges)-1]
	s.docs.Store(params.TextDocument.URI, last.Text)
	return nil
}

// DidSave runs the pipeline on the saved buffer and publishes diagnostics.
// If the client included Text in the save (we advertised IncludeText: true),
// we use that; otherwise we fall back to the in-memory buffer from the most
// recent didOpen/didChange.
func (s *Server) DidSave(ctx context.Context, params *protocol.DidSaveTextDocumentParams) error {
	if params == nil {
		return nil
	}
	uri := params.TextDocument.URI
	var code string
	if params.Text != "" {
		code = params.Text
		s.docs.Store(uri, code)
	} else if v, ok := s.docs.Load(uri); ok {
		code = v.(string)
	}
	s.publishDiagnosticsFor(ctx, uri, code)
	return nil
}

// DidClose forgets the buffer so a stale closed file doesn't keep memory.
// We also clear its diagnostics so the editor's gutter doesn't keep red
// marks from a now-closed file.
func (s *Server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	if params == nil {
		return nil
	}
	uri := params.TextDocument.URI
	s.docs.Delete(uri)
	if s.client != nil {
		_ = s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: []protocol.Diagnostic{},
		})
	}
	return nil
}

// publishDiagnosticsFor runs the pipeline on the given content and pushes
// the resulting diagnostics to the LSP client. No-op when no client is wired
// (e.g., during construction before Run, or in tests that bypass NewServer).
func (s *Server) publishDiagnosticsFor(ctx context.Context, uri protocol.DocumentURI, code string) {
	if s.client == nil {
		return
	}
	path := uriToPath(uri)
	diags := runPipelineForDiagnostics(path, code)
	if diags == nil {
		diags = []protocol.Diagnostic{}
	}
	if err := s.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diags,
	}); err != nil {
		s.logger.Warn("publishDiagnostics failed", zap.Error(err))
	}
}
