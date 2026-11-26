// Package quickfix implements deterministic, compiler-verified mechanical
// source edits for AILANG diagnostics. Both the CLI (`ailang check --json` /
// `--fix`) and the LSP server (`textDocument/codeAction`) call this package
// so they share a single source of truth with zero drift.
package quickfix

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/stdlibindex"
)

// Diagnostic is the minimal compiler-diagnostic shape consumed by ComputeEdits.
// Line and Column are 1-based (0 if unknown).
type Diagnostic struct {
	Code    string
	Message string
	File    string
	Line    int
	Column  int
}

// TextEdit is a 1-based line/column source replacement emitted in
// `ailang check --json`, applied by `ailang check --fix`, and converted
// to LSP protocol.TextEdit by internal/lsp.
type TextEdit struct {
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
	matchMissingRe  = regexp.MustCompile(`no arm matches:\s*(.+)$`)
	wildDomainRe    = regexp.MustCompile(`^_\s+\(any .+ other than .+\)$`)
)

// ComputeEdits returns 100% deterministic mechanical edits for a single
// diagnostic (never guesses structural code rewrites).
func ComputeEdits(diag Diagnostic, fallbackFile, src string) []TextEdit {
	file := diag.File
	if file == "" {
		file = fallbackFile
	}
	lines := strings.Split(src, "\n")

	switch diag.Code {
	case "PAR017":
		// `func f() -> T = { ... }` → `func f() -> T { ... }`
		if edit, ok := removeEqualsBeforeBraceEdit(file, lines, diag.Line); ok {
			return []TextEdit{edit}
		}

	case "PAR018":
		// `let rec name = ...` → `letrec name = ...`
		if diag.Line >= 1 && diag.Line <= len(lines) {
			line := lines[diag.Line-1]
			if idx := strings.Index(line, "let rec "); idx >= 0 {
				return []TextEdit{{
					File:      file,
					StartLine: diag.Line,
					StartCol:  idx + 1,
					EndLine:   diag.Line,
					EndCol:    idx + 1 + len("let rec"),
					NewText:   "letrec",
					Title:     "Replace 'let rec' with 'letrec'",
				}}
			}
		}

	case "PAR_HASH_COMMENT":
		// `# comment` → `-- comment`
		if diag.Line >= 1 && diag.Line <= len(lines) {
			line := lines[diag.Line-1]
			col := diag.Column - 1
			if col < 0 || col >= len(line) || line[col] != '#' {
				col = strings.Index(line, "#")
			}
			if col >= 0 && col < len(line) && line[col] == '#' {
				return []TextEdit{{
					File:      file,
					StartLine: diag.Line,
					StartCol:  col + 1,
					EndLine:   diag.Line,
					EndCol:    col + 2,
					NewText:   "--",
					Title:     "Replace '#' with AILANG comment '--'",
				}}
			}
		}

	case "IMP010":
		if m := imp010RenameRe.FindStringSubmatch(diag.Message); m != nil {
			oldSym, mod, newSym := m[1], m[2], m[3]
			if edit, ok := replaceImportedSymbolEdit(file, lines, mod, oldSym, newSym); ok {
				return []TextEdit{edit}
			}
		}
		if m := imp010BuiltinRe.FindStringSubmatch(diag.Message); m != nil {
			sym, mod := m[1], m[2]
			if edit, ok := removeImportedSymbolEdit(file, lines, mod, sym); ok {
				return []TextEdit{edit}
			}
		}

	case "TC_MATCH_002":
		if edit, ok := insertMissingMatchArmsEdit(file, src, diag.Line, diag.Column, diag.Message); ok {
			return []TextEdit{edit}
		}
	}

	// Unambiguous missing stdlib import for `undefined variable: <sym>`:
	// either exported by a single stdlib module, or by exactly one stdlib module
	// that is already imported with a symbol list in this file.
	if m := undefVarRe.FindStringSubmatch(diag.Message); m != nil {
		sym := m[1]
		mods := stdlibindex.Modules(sym)
		targetMod := ""
		if len(mods) == 1 {
			targetMod = mods[0]
		} else if len(mods) > 1 {
			var importedMatches []string
			for _, mod := range mods {
				if fileHasSelectiveImport(lines, mod) {
					importedMatches = append(importedMatches, mod)
				}
			}
			if len(importedMatches) == 1 {
				targetMod = importedMatches[0]
			}
		}
		if targetMod != "" {
			if edit, ok := insertStdlibImportEdit(file, lines, targetMod, sym); ok {
				return []TextEdit{edit}
			}
		}
	}

	// Phase 2 (M-AI-GO-TO-LANGUAGE D3=a): auto-stamp missing `! {Effects}`
	// onto function signatures and rewrite `s1 ++ s2` on strings to `"${s1}${s2}"`.
	if edits := computeEffectStampEdits(file, src, diag.Message); len(edits) > 0 {
		return edits
	}
	if edits := computeStringConcatEdits(file, src, diag); len(edits) > 0 {
		return edits
	}
	return nil
}

// insertMissingMatchArmsEdit computes a deterministic stub edit inserting
// `<missing_pattern> => _?` for each missing witness reported by TC_MATCH_002.
func insertMissingMatchArmsEdit(file, src string, line, col int, msg string) (TextEdit, bool) {
	if line < 1 {
		return TextEdit{}, false
	}
	patterns := extractMissingMatchPatterns(msg)
	if len(patterns) == 0 {
		return TextEdit{}, false
	}
	lineStarts := computeLineStarts(src)
	if line > len(lineStarts) {
		return TextEdit{}, false
	}
	startCol := col
	if startCol < 1 {
		startCol = 1
	}
	startOff, ok := lineColToOffset(src, lineStarts, line, startCol)
	if !ok {
		return TextEdit{}, false
	}

	openBrace, closeBrace, ok := findMatchBraces(src, startOff)
	if !ok {
		return TextEdit{}, false
	}

	// Find last non-whitespace byte before closeBrace (skipping line comments).
	prev := findPrevCodeByte(src, openBrace+1, closeBrace)
	openLine, _ := offsetToLineCol(lineStarts, openBrace)
	closeLine, _ := offsetToLineCol(lineStarts, closeBrace)
	multiline := closeLine > openLine

	var title string
	if len(patterns) == 1 {
		title = fmt.Sprintf("Add missing match arm '%s => _?'", patterns[0])
	} else {
		title = fmt.Sprintf("Add %d missing match arms", len(patterns))
	}

	if !multiline {
		var stubs []string
		for _, p := range patterns {
			stubs = append(stubs, p+" => _?")
		}
		joined := strings.Join(stubs, ", ")
		if prev < 0 {
			// Empty `match x {}`
			insLine, insCol := offsetToLineCol(lineStarts, openBrace+1)
			return TextEdit{
				File:      file,
				StartLine: insLine,
				StartCol:  insCol,
				EndLine:   insLine,
				EndCol:    insCol,
				NewText:   " " + joined + " ",
				Title:     title,
			}, true
		}
		sep := ", "
		if src[prev] == ',' {
			sep = " "
		}
		insLine, insCol := offsetToLineCol(lineStarts, prev+1)
		return TextEdit{
			File:      file,
			StartLine: insLine,
			StartCol:  insCol,
			EndLine:   insLine,
			EndCol:    insCol,
			NewText:   sep + joined,
			Title:     title,
		}, true
	}

	// Multiline match: infer indentation from the previous arm line, or fall
	// back to closing-brace indentation + two spaces.
	armIndent := leadingWhitespace(lineTextAt(src, lineStarts, closeLine)) + "  "
	if prev >= 0 {
		prevLine, _ := offsetToLineCol(lineStarts, prev)
		if prevLine > openLine {
			armIndent = leadingWhitespace(lineTextAt(src, lineStarts, prevLine))
		}
	}

	var stubLines []string
	for _, p := range patterns {
		stubLines = append(stubLines, armIndent+p+" => _?")
	}
	joined := strings.Join(stubLines, ",\n")

	if prev < 0 {
		// Empty multiline `match x {\n}`
		insLine, insCol := offsetToLineCol(lineStarts, openBrace+1)
		return TextEdit{
			File:      file,
			StartLine: insLine,
			StartCol:  insCol,
			EndLine:   insLine,
			EndCol:    insCol,
			NewText:   "\n" + joined,
			Title:     title,
		}, true
	}

	prefix := ",\n"
	if src[prev] == ',' {
		prefix = "\n"
	}
	insLine, insCol := offsetToLineCol(lineStarts, prev+1)
	return TextEdit{
		File:      file,
		StartLine: insLine,
		StartCol:  insCol,
		EndLine:   insLine,
		EndCol:    insCol,
		NewText:   prefix + joined,
		Title:     title,
	}, true
}

// extractMissingMatchPatterns parses the missing witness patterns from a
// TC_MATCH_002 diagnostic message.
func extractMissingMatchPatterns(msg string) []string {
	m := matchMissingRe.FindStringSubmatch(msg)
	if m == nil {
		if strings.Contains(msg, "cannot be checked for exhaustiveness") {
			return []string{"_"}
		}
		return nil
	}
	raw := strings.TrimSpace(m[1])
	// Strip any trailing notes on subsequent lines (if full Error() was passed).
	if nl := strings.IndexByte(raw, '\n'); nl >= 0 {
		raw = strings.TrimSpace(raw[:nl])
	}
	parts := splitTopLevelCommas(raw)
	var out []string
	hasMore := false
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "and more" {
			hasMore = true
			continue
		}
		if wildDomainRe.MatchString(p) {
			p = "_"
		}
		out = append(out, p)
	}
	if hasMore {
		// Replace truncated witness tail with catch-all `_` so the stub is exhaustive.
		out = append(out, "_")
	}
	return out
}

// splitTopLevelCommas splits s on commas that are not nested inside (), [], {},
// or double-quoted string literals.
func splitTopLevelCommas(s string) []string {
	var parts []string
	depth := 0
	inStr := false
	escaped := false
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// findMatchBraces scans forward from startOff to find the `{` opening the
// match body and its balanced closing `}`, ignoring strings and comments.
func findMatchBraces(src string, startOff int) (openBrace, closeBrace int, ok bool) {
	n := len(src)
	openBrace = -1
	depth := 0
	inStr := false
	escaped := false
	for i := startOff; i < n; i++ {
		c := src[i]
		if inStr {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		// Skip `--` line comments
		if c == '-' && i+1 < n && src[i+1] == '-' {
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}
		// Skip `/* ... */` block comments
		if c == '/' && i+1 < n && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i++
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{':
			if openBrace < 0 {
				openBrace = i
			}
			depth++
		case '}':
			if openBrace >= 0 {
				depth--
				if depth == 0 {
					return openBrace, i, true
				}
			}
		}
	}
	return -1, -1, false
}

// findPrevCodeByte returns the index of the last non-whitespace byte in
// src[lo:hi], or -1 if src[lo:hi] is all whitespace.
func findPrevCodeByte(src string, lo, hi int) int {
	for i := hi - 1; i >= lo; i-- {
		switch src[i] {
		case ' ', '\t', '\r', '\n':
			continue
		default:
			return i
		}
	}
	return -1
}

func lineTextAt(src string, lineStarts []int, line int) string {
	if line < 1 || line > len(lineStarts) {
		return ""
	}
	start := lineStarts[line-1]
	end := len(src)
	if line < len(lineStarts) {
		end = lineStarts[line] - 1
	}
	if end < start {
		return ""
	}
	return src[start:end]
}

func leadingWhitespace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[:i]
}

func fileHasSelectiveImport(lines []string, mod string) bool {
	prefix := "import " + mod
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		open := strings.Index(line, "(")
		closeIdx := strings.Index(line, ")")
		if open >= 0 && closeIdx > open {
			return true
		}
	}
	return false
}

// removeEqualsBeforeBraceEdit fixes the `= {` trap (`func f() -> T = { ... }`)
// by removing `= ` on the function header line at or just above errLine.
func removeEqualsBeforeBraceEdit(file string, lines []string, errLine int) (TextEdit, bool) {
	if errLine < 1 || errLine > len(lines) {
		return TextEdit{}, false
	}
	start := errLine
	minLine := errLine - 3
	if minLine < 1 {
		minLine = 1
	}
	for l := start; l >= minLine; l-- {
		line := lines[l-1]
		if !strings.Contains(line, "func ") {
			continue
		}
		if idx := strings.Index(line, "= {"); idx >= 0 {
			return TextEdit{
				File:      file,
				StartLine: l,
				StartCol:  idx + 1,
				EndLine:   l,
				EndCol:    idx + 3, // remove "= "
				NewText:   "",
				Title:     "Remove '=' before '{' in block-body function",
			}, true
		}
	}
	return TextEdit{}, false
}

func replaceImportedSymbolEdit(file string, lines []string, mod, oldSym, newSym string) (TextEdit, bool) {
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
			return TextEdit{
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
	return TextEdit{}, false
}

func removeImportedSymbolEdit(file string, lines []string, mod, sym string) (TextEdit, bool) {
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
			return TextEdit{
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
		return TextEdit{
			File:      file,
			StartLine: i + 1,
			StartCol:  1,
			EndLine:   i + 1,
			EndCol:    len(line) + 1,
			NewText:   newLine,
			Title:     fmt.Sprintf("Remove builtin '%s' from %s import", sym, mod),
		}, true
	}
	return TextEdit{}, false
}

func insertStdlibImportEdit(file string, lines []string, mod, sym string) (TextEdit, bool) {
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
				return TextEdit{}, false // already imported
			}
			return TextEdit{
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
	return TextEdit{
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

// ApplyEdits applies a set of non-overlapping 1-based line/col TextEdits
// to src, returning the updated source and the count of applied edits.
func ApplyEdits(src string, edits []TextEdit) (string, int) {
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

func offsetToLineCol(lineStarts []int, off int) (line, col int) {
	// Binary search for the largest lineStarts[i] <= off.
	idx := sort.Search(len(lineStarts), func(i int) bool {
		return lineStarts[i] > off
	}) - 1
	if idx < 0 {
		idx = 0
	}
	return idx + 1, (off - lineStarts[idx]) + 1
}
