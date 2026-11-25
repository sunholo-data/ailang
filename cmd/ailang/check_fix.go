package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/stdlibindex"
)

// checkFixMode is `ailang check --fix`: applies compiler-verified mechanical
// edits (D2=a of M-AI-PREFERRED-LANGUAGE M2) in place before the final check.
var checkFixMode bool

// checkTextEdit is a 1-based line/column source replacement emitted in
// `ailang check --json` and applied by `ailang check --fix`.
type checkTextEdit struct {
	File      string `json:"file,omitempty"`
	StartLine int    `json:"start_line"`
	StartCol  int    `json:"start_col"`
	EndLine   int    `json:"end_line"`
	EndCol    int    `json:"end_col"`
	NewText   string `json:"new_text"`
	Title     string `json:"title,omitempty"`
}

var (
	imp010RenameRe  = regexp.MustCompile(`symbol '([^']+)' not exported by '([^']+)' — did you mean '([^']+)'\?`)
	imp010BuiltinRe = regexp.MustCompile(`symbol '([^']+)' not exported by '([^']+)' — '[^']+' is a builtin available in every module`)
	undefVarRe      = regexp.MustCompile(`undefined variable:\s+([A-Za-z_][A-Za-z0-9_]*)`)
)

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

// computeCheckEdits returns 100% deterministic mechanical edits for a single
// diagnostic (D2=a: never guesses structural code rewrites).
func computeCheckEdits(entry checkJSONError, fallbackFile, src string) []checkTextEdit {
	file := entry.File
	if file == "" {
		file = fallbackFile
	}
	lines := strings.Split(src, "\n")

	switch entry.Code {
	case "PAR018":
		// `let rec name = ...` → `letrec name = ...`
		if entry.Line >= 1 && entry.Line <= len(lines) {
			line := lines[entry.Line-1]
			if idx := strings.Index(line, "let rec "); idx >= 0 {
				return []checkTextEdit{{
					File:      file,
					StartLine: entry.Line,
					StartCol:  idx + 1,
					EndLine:   entry.Line,
					EndCol:    idx + 1 + len("let rec"),
					NewText:   "letrec",
					Title:     "Replace 'let rec' with 'letrec'",
				}}
			}
		}

	case "PAR_HASH_COMMENT":
		// `# comment` → `-- comment`
		if entry.Line >= 1 && entry.Line <= len(lines) {
			line := lines[entry.Line-1]
			col := entry.Column - 1
			if col < 0 || col >= len(line) || line[col] != '#' {
				col = strings.Index(line, "#")
			}
			if col >= 0 && col < len(line) && line[col] == '#' {
				return []checkTextEdit{{
					File:      file,
					StartLine: entry.Line,
					StartCol:  col + 1,
					EndLine:   entry.Line,
					EndCol:    col + 2,
					NewText:   "--",
					Title:     "Replace '#' with AILANG comment '--'",
				}}
			}
		}

	case "IMP010":
		if m := imp010RenameRe.FindStringSubmatch(entry.Message); m != nil {
			oldSym, mod, newSym := m[1], m[2], m[3]
			if edit, ok := replaceImportedSymbolEdit(file, lines, mod, oldSym, newSym); ok {
				return []checkTextEdit{edit}
			}
		}
		if m := imp010BuiltinRe.FindStringSubmatch(entry.Message); m != nil {
			sym, mod := m[1], m[2]
			if edit, ok := removeImportedSymbolEdit(file, lines, mod, sym); ok {
				return []checkTextEdit{edit}
			}
		}
	}

	// Unambiguous missing stdlib import for `undefined variable: <sym>`
	if m := undefVarRe.FindStringSubmatch(entry.Message); m != nil {
		sym := m[1]
		mods := stdlibindex.Modules(sym)
		if len(mods) == 1 {
			if edit, ok := insertStdlibImportEdit(file, lines, mods[0], sym); ok {
				return []checkTextEdit{edit}
			}
		}
	}
	return nil
}

func replaceImportedSymbolEdit(file string, lines []string, mod, oldSym, newSym string) (checkTextEdit, bool) {
	prefix := "import " + mod
	for i, line := range lines {
		if !strings.Contains(line, prefix) {
			continue
		}
		open := strings.Index(line, "(")
		closeIdx := strings.Index(line, ")")
		if open < 0 || closeIdx <= open {
			continue
		}
		inner := line[open+1 : closeIdx]
		if pos := findWordIndex(inner, oldSym); pos >= 0 {
			startCol := open + 1 + pos + 1
			return checkTextEdit{
				File:      file,
				StartLine: i + 1,
				StartCol:  startCol,
				EndLine:   i + 1,
				EndCol:    startCol + len(oldSym),
				NewText:   newSym,
				Title:     fmt.Sprintf("Rename '%s' to '%s' in %s import", oldSym, newSym, mod),
			}, true
		}
	}
	return checkTextEdit{}, false
}

func removeImportedSymbolEdit(file string, lines []string, mod, sym string) (checkTextEdit, bool) {
	prefix := "import " + mod
	for i, line := range lines {
		if !strings.Contains(line, prefix) {
			continue
		}
		open := strings.Index(line, "(")
		closeIdx := strings.Index(line, ")")
		if open < 0 || closeIdx <= open {
			continue
		}
		parts := strings.Split(line[open+1:closeIdx], ",")
		var kept []string
		found := false
		for _, p := range parts {
			t := strings.TrimSpace(p)
			if t == sym {
				found = true
				continue
			}
			if t != "" {
				kept = append(kept, t)
			}
		}
		if !found {
			continue
		}
		if len(kept) == 0 {
			return checkTextEdit{
				File:      file,
				StartLine: i + 1,
				StartCol:  1,
				EndLine:   i + 2,
				EndCol:    1,
				NewText:   "",
				Title:     fmt.Sprintf("Remove redundant builtin import '%s'", sym),
			}, true
		}
		newLine := line[:open+1] + strings.Join(kept, ", ") + line[closeIdx:]
		return checkTextEdit{
			File:      file,
			StartLine: i + 1,
			StartCol:  1,
			EndLine:   i + 1,
			EndCol:    len(line) + 1,
			NewText:   newLine,
			Title:     fmt.Sprintf("Remove builtin '%s' from %s import", sym, mod),
		}, true
	}
	return checkTextEdit{}, false
}

func insertStdlibImportEdit(file string, lines []string, mod, sym string) (checkTextEdit, bool) {
	// If the module is already imported with a symbol list, append sym to it.
	prefix := "import " + mod
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		open := strings.Index(line, "(")
		closeIdx := strings.Index(line, ")")
		if open >= 0 && closeIdx > open {
			if findWordIndex(line[open+1:closeIdx], sym) >= 0 {
				return checkTextEdit{}, false // already imported
			}
			return checkTextEdit{
				File:      file,
				StartLine: i + 1,
				StartCol:  closeIdx + 1,
				EndLine:   i + 1,
				EndCol:    closeIdx + 1,
				NewText:   ", " + sym,
				Title:     fmt.Sprintf("Add '%s' to import %s", sym, mod),
			}, true
		}
	}

	// Otherwise insert a new `import <mod> (<sym>)` after the last import or module line.
	insertAfter := 0
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "module ") || strings.HasPrefix(t, "import ") {
			insertAfter = i + 1
		}
	}
	return checkTextEdit{
		File:      file,
		StartLine: insertAfter + 1,
		StartCol:  1,
		EndLine:   insertAfter + 1,
		EndCol:    1,
		NewText:   fmt.Sprintf("import %s (%s)\n", mod, sym),
		Title:     fmt.Sprintf("Add import %s (%s)", mod, sym),
	}, true
}

func findWordIndex(s, word string) int {
	for start := 0; start < len(s); {
		idx := strings.Index(s[start:], word)
		if idx < 0 {
			return -1
		}
		abs := start + idx
		leftOK := abs == 0 || !isIdentByte(s[abs-1])
		rightPos := abs + len(word)
		rightOK := rightPos == len(s) || !isIdentByte(s[rightPos])
		if leftOK && rightOK {
			return abs
		}
		start = abs + 1
	}
	return -1
}

func isIdentByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// applyCheckEdits applies a set of non-overlapping 1-based line/col TextEdits
// to src, returning the updated source and the count of applied edits.
func applyCheckEdits(src string, edits []checkTextEdit) (string, int) {
	if len(edits) == 0 {
		return src, 0
	}
	lineStarts := computeLineStarts(src)
	type byteEdit struct {
		start, end int
		newText    string
	}
	var resolved []byteEdit
	for _, e := range edits {
		s, ok1 := lineColToOffset(src, lineStarts, e.StartLine, e.StartCol)
		end, ok2 := lineColToOffset(src, lineStarts, e.EndLine, e.EndCol)
		if !ok1 || !ok2 || end < s {
			continue
		}
		resolved = append(resolved, byteEdit{start: s, end: end, newText: e.NewText})
	}
	// Sort descending by start offset so earlier offsets stay valid; skip overlaps.
	sort.Slice(resolved, func(i, j int) bool {
		if resolved[i].start != resolved[j].start {
			return resolved[i].start > resolved[j].start
		}
		return resolved[i].end > resolved[j].end
	})
	out := src
	applied := 0
	lastStart := len(src) + 1
	for _, e := range resolved {
		if e.end > lastStart {
			continue // overlapping edit — skip safely
		}
		out = out[:e.start] + e.newText + out[e.end:]
		lastStart = e.start
		applied++
	}
	return out, applied
}

func computeLineStarts(src string) []int {
	starts := []int{0}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func lineColToOffset(src string, lineStarts []int, line, col int) (int, bool) {
	if line < 1 || col < 1 {
		return 0, false
	}
	if line == len(lineStarts)+1 && col == 1 {
		return len(src), true
	}
	if line > len(lineStarts) {
		return 0, false
	}
	off := lineStarts[line-1] + (col - 1)
	if off > len(src) {
		off = len(src)
	}
	return off, true
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
