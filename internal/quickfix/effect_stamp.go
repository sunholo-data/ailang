package quickfix

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	effectFailFuncRe = regexp.MustCompile(`Effect checking failed for function '([A-Za-z_][A-Za-z0-9_]*)'`)
	effectSuggestRe  = regexp.MustCompile(`Suggested fix:\s+func\s+[A-Za-z_][A-Za-z0-9_]*\(\.\.\.\)\s+->\s+T\s+!\s*\{([^}]*)\}`)
)

// ExtractEffectFailureFunc returns the function name from a validate_effects
// diagnostic message, or "" if the message is not an effect-checking failure.
func ExtractEffectFailureFunc(msg string) string {
	m := effectFailFuncRe.FindStringSubmatch(msg)
	if m == nil {
		return ""
	}
	return m[1]
}

// FindFuncDeclLine returns the 1-based line number of the top-level `func <name>`
// declaration in src, or 0 if not found. Used by LSP diagnostics to place
// effect-checking errors on the function header rather than line 1.
func FindFuncDeclLine(src, funcName string) int {
	hdr, ok := findFuncHeader(src, funcName)
	if !ok {
		return 0
	}
	lineStarts := computeLineStarts(src)
	line, _ := offsetToLineCol(lineStarts, hdr.funcStart)
	return line
}

// computeEffectStampEdits produces deterministic TextEdits that stamp missing
// `! {Effects}` onto a function signature when validate_effects reports a
// label-union suggested fix.
//
// Behaviour (D3=a):
//   - Existing `! { ... }` row: merges suggested effects into the existing row
//     while preserving any `@limit=N` / `@min=N` / parameter annotations.
//   - No existing `! { ... }` row: inserts ` ! { ... }` immediately after the
//     return type (or parameter list if unannotated).
//   - Strips `pure ` from `pure func ...` whenever stamping a non-empty row.
func computeEffectStampEdits(file, src, msg string) []TextEdit {
	funcName := ExtractEffectFailureFunc(msg)
	if funcName == "" {
		return nil
	}
	sm := effectSuggestRe.FindStringSubmatch(msg)
	if sm == nil {
		return nil
	}
	suggestedEntries := parseEffectEntries(sm[1])
	if len(suggestedEntries) == 0 {
		return nil
	}

	hdr, ok := findFuncHeader(src, funcName)
	if !ok {
		return nil
	}

	lineStarts := computeLineStarts(src)
	var edits []TextEdit

	if hdr.hasEffects {
		existingEntries := parseEffectEntries(src[hdr.effectOpen+1 : hdr.effectClose])
		merged := mergeEffectEntries(existingEntries, suggestedEntries)
		newRow := "! {" + strings.Join(merged, ", ") + "}"
		sLine, sCol := offsetToLineCol(lineStarts, hdr.effectBang)
		eLine, eCol := offsetToLineCol(lineStarts, hdr.effectClose+1)
		title := fmt.Sprintf("Update effect annotation on '%s' to '%s'", funcName, newRow)
		if hdr.hasPure {
			title = fmt.Sprintf("Update effect annotation to '%s' and remove 'pure' on '%s'", newRow, funcName)
		}
		edits = append(edits, TextEdit{
			File:      file,
			StartLine: sLine,
			StartCol:  sCol,
			EndLine:   eLine,
			EndCol:    eCol,
			NewText:   newRow,
			Title:     title,
		})
	} else {
		merged := mergeEffectEntries(nil, suggestedEntries)
		newRow := "! {" + strings.Join(merged, ", ") + "}"
		insLine, insCol := offsetToLineCol(lineStarts, hdr.insertOff)
		title := fmt.Sprintf("Add effect annotation '%s' to '%s'", newRow, funcName)
		if hdr.hasPure {
			title = fmt.Sprintf("Add '%s' and remove 'pure' on '%s'", newRow, funcName)
		}
		edits = append(edits, TextEdit{
			File:      file,
			StartLine: insLine,
			StartCol:  insCol,
			EndLine:   insLine,
			EndCol:    insCol,
			NewText:   " " + newRow,
			Title:     title,
		})
	}

	if hdr.hasPure {
		pLine, pCol := offsetToLineCol(lineStarts, hdr.pureStart)
		peLine, peCol := offsetToLineCol(lineStarts, hdr.pureEnd)
		edits = append(edits, TextEdit{
			File:      file,
			StartLine: pLine,
			StartCol:  pCol,
			EndLine:   peLine,
			EndCol:    peCol,
			NewText:   "",
			Title:     fmt.Sprintf("Remove 'pure' from '%s'", funcName),
		})
	}

	return edits
}

type funcHeaderLoc struct {
	funcStart   int  // byte offset of `func` keyword
	hasPure     bool // true if `pure` modifier precedes `func`
	pureStart   int  // start of `pure` keyword
	pureEnd     int  // end of `pure` + following horizontal whitespace
	insertOff   int  // byte offset right after return type (or params `)`)
	hasEffects  bool // true if an existing `! { ... }` clause was found
	effectBang  int  // byte offset of `!`
	effectOpen  int  // byte offset of `{`
	effectClose int  // byte offset of `}`
}

// findFuncHeader locates `func <name>` outside strings/comments and scans its
// type parameters, parameter list, optional return type (including record types
// `{ ... }`), and optional existing `! { ... }` effect row.
func findFuncHeader(src, name string) (funcHeaderLoc, bool) {
	n := len(src)
	i := 0
	for i < n {
		// Skip whitespace and comments.
		next := skipWSAndComments(src, i)
		if next > i {
			i = next
			continue
		}
		// Skip string literals so `func foo` inside a string is never matched.
		if src[i] == '"' {
			i = skipStringLiteral(src, i)
			continue
		}
		// Check for `func` keyword at a word boundary.
		if isKeywordAt(src, i, "func") {
			funcStart := i
			pos := skipWSAndComments(src, i+len("func"))
			if isKeywordAt(src, pos, name) {
				afterName := pos + len(name)
				if loc, ok := scanFuncHeaderAfterName(src, funcStart, afterName); ok {
					return loc, true
				}
			}
			i += len("func")
			continue
		}
		// Advance past identifier or single byte.
		if isIdentByte(src[i]) {
			for i < n && isIdentByte(src[i]) {
				i++
			}
		} else {
			i++
		}
	}
	return funcHeaderLoc{}, false
}

func scanFuncHeaderAfterName(src string, funcStart, pos int) (funcHeaderLoc, bool) {
	n := len(src)
	pos = skipWSAndComments(src, pos)

	// Optional type parameters `[T, U: Eq]`.
	if pos < n && src[pos] == '[' {
		end, ok := scanBalanced(src, pos, '[', ']')
		if !ok {
			return funcHeaderLoc{}, false
		}
		pos = skipWSAndComments(src, end)
	}

	// Required parameter list `(...)`.
	if pos >= n || src[pos] != '(' {
		return funcHeaderLoc{}, false
	}
	paramsEnd, ok := scanBalanced(src, pos, '(', ')')
	if !ok {
		return funcHeaderLoc{}, false
	}

	loc := funcHeaderLoc{
		funcStart: funcStart,
		insertOff: paramsEnd,
	}

	// Check for a `pure` modifier immediately before `func`.
	if pStart, pEnd, ok := findPrecedingPure(src, funcStart); ok {
		loc.hasPure = true
		loc.pureStart = pStart
		loc.pureEnd = pEnd
	}

	pos = skipWSAndComments(src, paramsEnd)

	// Optional return type: `-> <Type>`.
	if pos+1 < n && src[pos] == '-' && src[pos+1] == '>' {
		retEnd, ok := scanReturnType(src, pos+2)
		if !ok {
			return funcHeaderLoc{}, false
		}
		loc.insertOff = retEnd
		pos = skipWSAndComments(src, retEnd)
	}

	// Optional existing effect row: `! { ... }`.
	if pos < n && src[pos] == '!' {
		bangOff := pos
		bracePos := skipWSAndComments(src, bangOff+1)
		if bracePos < n && src[bracePos] == '{' {
			closeEnd, ok := scanBalanced(src, bracePos, '{', '}')
			if ok {
				loc.hasEffects = true
				loc.effectBang = bangOff
				loc.effectOpen = bracePos
				loc.effectClose = closeEnd - 1
			}
		}
	}

	return loc, true
}

// findPrecedingPure checks if the token immediately preceding funcStart (on the
// same declaration) is the `pure` keyword, returning the byte range of `pure `
// (including trailing horizontal spaces/tabs up to `func`).
func findPrecedingPure(src string, funcStart int) (int, int, bool) {
	wsEnd := funcStart
	i := funcStart
	for i > 0 && (src[i-1] == ' ' || src[i-1] == '\t') {
		i--
	}
	if i == funcStart {
		return 0, 0, false
	}
	wordEnd := i
	for i > 0 && isIdentByte(src[i-1]) {
		i--
	}
	if src[i:wordEnd] != "pure" {
		return 0, 0, false
	}
	return i, wsEnd, true
}

// scanReturnType scans a return type expression starting immediately after `->`,
// returning the byte offset right after the last token of the return type.
// Handles record types `{ a: int }`, tuples/unit `()`, lists `[T]`, generic
// types `Result[{ a: int }, string]`, and curried function types `a -> b`.
func scanReturnType(src string, start int) (int, bool) {
	pos := skipWSAndComments(src, start)
	end, ok := scanTypeAtom(src, pos)
	if !ok {
		return 0, false
	}
	// Handle curried arrow return types like `-> (int) -> int` or `-> int -> int`.
	for {
		next := skipWSAndComments(src, end)
		if next+1 < len(src) && src[next] == '-' && src[next+1] == '>' {
			nextAtomPos := skipWSAndComments(src, next+2)
			nextEnd, ok := scanTypeAtom(src, nextAtomPos)
			if !ok {
				return 0, false
			}
			end = nextEnd
			continue
		}
		break
	}
	return end, true
}

func scanTypeAtom(src string, pos int) (int, bool) {
	n := len(src)
	if pos >= n {
		return 0, false
	}
	switch src[pos] {
	case '(':
		return scanBalanced(src, pos, '(', ')')
	case '[':
		return scanBalanced(src, pos, '[', ']')
	case '{':
		// Record return type `{ field: Type, ... }` — balanced braces distinguish
		// it from the function body `{ ... }` that follows.
		return scanBalanced(src, pos, '{', '}')
	default:
		if !isIdentStart(src[pos]) {
			return 0, false
		}
		end := pos + 1
		for end < n && (isIdentByte(src[end]) || src[end] == '.') {
			end++
		}
		// Optional generic type arguments `Name[T1, T2]`.
		after := skipWSAndComments(src, end)
		if after < n && src[after] == '[' {
			genEnd, ok := scanBalanced(src, after, '[', ']')
			if !ok {
				return 0, false
			}
			end = genEnd
		}
		return end, true
	}
}

// parseEffectEntries splits the interior of `! { ... }` on top-level commas
// (respecting `[...]` inside parameterized effects like `Rand[mode=crypto]`).
func parseEffectEntries(inner string) []string {
	parts := splitTopLevelCommas(inner)
	var out []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func effectBaseName(entry string) string {
	entry = strings.TrimSpace(entry)
	i := 0
	for i < len(entry) && isIdentByte(entry[i]) {
		i++
	}
	return entry[:i]
}

// mergeEffectEntries merges existing and suggested effect entries by base name,
// preserving existing annotations (such as `@limit=N`, `@min=N`, or params) and
// sorting deterministically (concrete effects alphabetically, lowercase row vars last).
func mergeEffectEntries(existing, suggested []string) []string {
	byName := make(map[string]string, len(existing)+len(suggested))
	for _, e := range suggested {
		if name := effectBaseName(e); name != "" {
			byName[name] = e
		}
	}
	for _, e := range existing {
		if name := effectBaseName(e); name != "" {
			byName[name] = e // existing wins to preserve @limit/@min/params
		}
	}
	var concrete, rowVars []string
	for name, full := range byName {
		if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' {
			rowVars = append(rowVars, full)
		} else {
			concrete = append(concrete, full)
		}
	}
	sort.Strings(concrete)
	sort.Strings(rowVars)
	return append(concrete, rowVars...)
}

func isKeywordAt(src string, pos int, kw string) bool {
	if pos < 0 || pos+len(kw) > len(src) {
		return false
	}
	if src[pos:pos+len(kw)] != kw {
		return false
	}
	if pos > 0 && isIdentByte(src[pos-1]) {
		return false
	}
	end := pos + len(kw)
	if end < len(src) && isIdentByte(src[end]) {
		return false
	}
	return true
}

func isIdentStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

func skipWSAndComments(src string, i int) int {
	n := len(src)
	for i < n {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			i++
			continue
		}
		if c == '-' && i+1 < n && src[i+1] == '-' {
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < n && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			if i+1 < n {
				i += 2
			} else {
				i = n
			}
			continue
		}
		break
	}
	return i
}

// skipStringLiteral advances past a double-quoted string literal starting at i,
// handling escapes (`\"`, `\\`) and `${...}` interpolation blocks.
func skipStringLiteral(src string, i int) int {
	n := len(src)
	if i >= n || src[i] != '"' {
		return i
	}
	i++
	escaped := false
	for i < n {
		c := src[i]
		if escaped {
			escaped = false
			i++
			continue
		}
		if c == '\\' {
			escaped = true
			i++
			continue
		}
		if c == '$' && i+1 < n && src[i+1] == '{' {
			end, ok := scanBalanced(src, i+1, '{', '}')
			if !ok {
				return n
			}
			i = end
			continue
		}
		if c == '"' {
			return i + 1
		}
		i++
	}
	return n
}

// scanBalanced scans from an opening delimiter openCh at start to its matching
// closeCh, returning the byte offset immediately AFTER closeCh.
func scanBalanced(src string, start int, openCh, closeCh byte) (int, bool) {
	n := len(src)
	if start >= n || src[start] != openCh {
		return 0, false
	}
	depth := 0
	for i := start; i < n; {
		c := src[i]
		if c == '"' {
			i = skipStringLiteral(src, i)
			continue
		}
		if c == '-' && i+1 < n && src[i+1] == '-' {
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < n && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			if i+1 < n {
				i += 2
			} else {
				return 0, false
			}
			continue
		}
		if c == openCh {
			depth++
		} else if c == closeCh {
			depth--
			if depth == 0 {
				return i + 1, true
			}
		}
		i++
	}
	return 0, false
}
