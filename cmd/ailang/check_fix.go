package main

import (
	"os"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/quickfix"
)

// checkFixMode is `ailang check --fix`: applies compiler-verified mechanical
// edits (D2=a of M-AI-PREFERRED-LANGUAGE M2) in place before the final check.
var checkFixMode bool

// checkTextEdit aliases quickfix.TextEdit so `ailang check --json`,
// `ailang check --fix`, and `internal/lsp` share a single source of truth.
type checkTextEdit = quickfix.TextEdit

// attachQuickFixEdits enriches errors in out with deterministic mechanical
// TextEdits (read-only; never mutates disk).
func attachQuickFixEdits(out *checkJSONOutput) {
	if out == nil || out.Passed || out.File == "" || len(out.Errors) == 0 {
		return
	}
	raw, err := os.ReadFile(out.File)
	if err != nil {
		return
	}
	src := string(raw)
	for i := range out.Errors {
		if len(out.Errors[i].Edits) > 0 {
			continue
		}
		if edits := computeCheckEdits(out.Errors[i], out.File, src); len(edits) > 0 {
			out.Errors[i].Edits = edits
		}
	}
}

// computeCheckEdits delegates to the shared internal/quickfix engine.
func computeCheckEdits(entry checkJSONError, fallbackFile, src string) []checkTextEdit {
	return quickfix.ComputeEdits(quickfix.Diagnostic{
		Code:    entry.Code,
		Message: entry.Message,
		File:    entry.File,
		Line:    entry.Line,
		Column:  entry.Column,
	}, fallbackFile, src)
}

// applyCheckEdits delegates to the shared internal/quickfix engine.
func applyCheckEdits(src string, edits []checkTextEdit) (string, int) {
	return quickfix.ApplyEdits(src, edits)
}

// runCheckFix iteratively applies mechanical quick-fixes to filename (up to 4
// passes) until either check passes or no further mechanical fixes apply.
func runCheckFix(filename string, cfg pipeline.Config) (int, error) {
	totalApplied := 0
	for pass := 0; pass < 4; pass++ {
		raw, err := os.ReadFile(filename)
		if err != nil {
			return totalApplied, err
		}
		src := string(raw)
		res, runErr := pipeline.Run(cfg, pipeline.Source{
			Code:     src,
			Filename: filename,
			IsREPL:   false,
		})
		var entries []checkJSONError
		if runErr != nil {
			entries = pipelineErrorsToCheckJSON(runErr, filename)
		} else if len(res.Errors) > 0 {
			for _, e := range res.Errors {
				entries = append(entries, errorToCheckJSONError(e, filename))
			}
		} else {
			break // clean!
		}
		var edits []checkTextEdit
		for _, entry := range entries {
			edits = append(edits, computeCheckEdits(entry, filename, src)...)
		}
		if len(edits) == 0 {
			break
		}
		updated, n := applyCheckEdits(src, edits)
		if n == 0 || updated == src {
			break
		}
		if err := os.WriteFile(filename, []byte(updated), 0o644); err != nil {
			return totalApplied, err
		}
		totalApplied += n
	}
	return totalApplied, nil
}
