package lsp

import (
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// DiagnosticSource is the value placed in protocol.Diagnostic.Source so
// editors can disambiguate AILANG-checker output from other LSP servers
// running against the same file.
const DiagnosticSource = "ailang"

// runPipelineForDiagnostics runs the type-check pipeline on the given source
// and returns LSP diagnostics derived from any structured errors it
// produces. Exhaustiveness warnings come back as DiagnosticSeverityWarning;
// type/elaborator errors as DiagnosticSeverityError.
//
// The path is the on-disk filename (used for span filtering — diagnostics
// for OTHER files are dropped, since LSP diagnostics are URI-scoped).
func runPipelineForDiagnostics(path string, code string) []protocol.Diagnostic {
	cfg := pipeline.Config{
		// Type-check only: we don't want to evaluate the program from the LSP.
		// pipeline.Run defaults are conservative; explicit zero-value Config
		// is fine for M2.
	}
	src := pipeline.Source{
		Code:     code,
		Filename: path,
	}

	result, topErr := pipeline.Run(cfg, src)

	// Many pipeline failures (e.g. typechecker errors that abort the run)
	// surface only via the top-level err return — not in result.Errors.
	// Match cmd/ailang/check.go's behaviour: use the structured error if
	// available, fall back to result.Errors, and dedupe.
	out := make([]protocol.Diagnostic, 0, 1+len(result.Errors)+len(result.Warnings))
	seen := map[string]bool{}
	addError := func(e error) {
		if e == nil {
			return
		}
		if d, ok := errorToDiagnostic(e, path); ok {
			codeStr, _ := d.Code.(string)
			key := codeStr + "|" + d.Message
			if !seen[key] {
				seen[key] = true
				out = append(out, d)
			}
		}
	}
	addError(topErr)
	for _, e := range result.Errors {
		addError(e)
	}
	for _, w := range result.Warnings {
		if w == nil {
			continue
		}
		out = append(out, exhaustivenessWarningToDiagnostic(w))
	}
	return out
}

// errorToDiagnostic converts a pipeline error to an LSP Diagnostic if it
// carries enough position info to be placed in the editor. The bool is false
// when the error has no position (we drop those rather than dumping them at
// {0,0} where they'd cover legitimate code).
func errorToDiagnostic(err error, path string) (protocol.Diagnostic, bool) {
	if err == nil {
		return protocol.Diagnostic{}, false
	}

	if rep, ok := ailerrors.AsReport(err); ok && rep != nil && rep.Span != nil {
		// Drop diagnostics whose span belongs to a *different* file.
		// Cross-file diagnostics belong on the OTHER file's URI; we'll
		// surface them when that file is opened.
		if rep.Span.Start.File != "" && rep.Span.Start.File != path {
			return protocol.Diagnostic{}, false
		}
		return protocol.Diagnostic{
			Range:    spanToRange(rep.Span),
			Severity: protocol.DiagnosticSeverityError,
			Source:   DiagnosticSource,
			Code:     rep.Code,
			Message:  rep.Message,
		}, true
	}

	// Fallback: error without a structured position. Place it at the start
	// of the file so it's still visible, but only for errors we can be sure
	// originated from THIS file (we have no way to tell, so we surface them).
	return protocol.Diagnostic{
		Range:    protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 1}},
		Severity: protocol.DiagnosticSeverityError,
		Source:   DiagnosticSource,
		Code:     "ERROR",
		Message:  err.Error(),
	}, true
}

// exhaustivenessWarningToDiagnostic maps an elaborator exhaustiveness
// warning to LSP Diagnostic{Severity: Warning}. Exhaustiveness warnings are
// the only Result.Warnings shape today; if more variants land, expand here.
func exhaustivenessWarningToDiagnostic(w interface{}) protocol.Diagnostic {
	// Use a typed assertion via a tiny interface so we don't import elaborate
	// (which would create a layering concern). The fields we care about are
	// the location and message.
	type warning interface {
		Position() (line, col int)
		String() string
	}
	if tw, ok := w.(warning); ok {
		line, col := tw.Position()
		return protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(max0(line - 1)), Character: uint32(max0(col - 1))},
				End:   protocol.Position{Line: uint32(max0(line - 1)), Character: uint32(max0(col))},
			},
			Severity: protocol.DiagnosticSeverityWarning,
			Source:   DiagnosticSource,
			Code:     "exhaustiveness",
			Message:  tw.String(),
		}
	}
	// Unknown warning shape — surface at file start as Warning so it's not lost.
	if stringer, ok := w.(interface{ String() string }); ok {
		return protocol.Diagnostic{
			Range:    protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 1}},
			Severity: protocol.DiagnosticSeverityWarning,
			Source:   DiagnosticSource,
			Code:     "warning",
			Message:  stringer.String(),
		}
	}
	return protocol.Diagnostic{
		Range:    protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 1}},
		Severity: protocol.DiagnosticSeverityWarning,
		Source:   DiagnosticSource,
		Code:     "warning",
		Message:  "elaborator warning",
	}
}

// spanToRange maps an AILANG ast.Span (1-indexed line/col) to an LSP Range
// (0-indexed line/character). End defaults to one column past Start when
// the span is degenerate.
func spanToRange(s *ast.Span) protocol.Range {
	startLine := uint32(max0(s.Start.Line - 1))
	startCol := uint32(max0(s.Start.Column - 1))
	endLine := startLine
	endCol := startCol + 1
	if s.End.Line > 0 {
		endLine = uint32(max0(s.End.Line - 1))
	}
	if s.End.Column > 0 {
		endCol = uint32(max0(s.End.Column - 1))
	}
	if endLine < startLine || (endLine == startLine && endCol < startCol) {
		endCol = startCol + 1
		endLine = startLine
	}
	return protocol.Range{
		Start: protocol.Position{Line: startLine, Character: startCol},
		End:   protocol.Position{Line: endLine, Character: endCol},
	}
}

// uriToPath strips the file:// scheme from an LSP DocumentURI, returning
// a filesystem path. Non-file URIs are returned as-is.
func uriToPath(u protocol.DocumentURI) string {
	s := string(u)
	if strings.HasPrefix(s, "file://") {
		return uri.New(s).Filename()
	}
	return s
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
