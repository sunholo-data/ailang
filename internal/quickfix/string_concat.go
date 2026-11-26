package quickfix

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	strConcatLocRe = regexp.MustCompile(`\+\+ operator at ((?:[A-Za-z]:)?[^\s:]+):(\d+):(\d+):`)
)

// computeStringConcatEdits rewrites `s1 ++ s2` (and chained `a ++ b ++ c`) on
// string operands to `"${s1}${s2}"` interpolation, inlining adjacent string
// literals (e.g. `"Hello, " ++ name` → `"Hello, ${name}"`, `"a" ++ "b"` → `"ab"`).
func computeStringConcatEdits(file, src string, diag Diagnostic) []TextEdit {
	if !strings.Contains(diag.Message, "`++` is for lists only") &&
		!strings.Contains(diag.Message, "'++' is for lists only") {
		return nil
	}

	line, col := diag.Line, diag.Column
	if m := strConcatLocRe.FindStringSubmatch(diag.Message); m != nil {
		if l, err := strconv.Atoi(m[2]); err == nil && l > 0 {
			line = l
		}
		if c, err := strconv.Atoi(m[3]); err == nil && c > 0 {
			col = c
		}
	}
	if line < 1 {
		return nil
	}
	if col < 1 {
		col = 1
	}

	lineStarts := computeLineStarts(src)
	opOff, ok := lineColToOffset(src, lineStarts, line, col)
	if !ok {
		return nil
	}
	opOff, ok = locatePlusPlusOp(src, lineStarts, line, opOff)
	if !ok {
		return nil
	}

	chainStart, chainEnd, parts, ok := extractConcatChain(src, opOff)
	if !ok || len(parts) < 2 {
		return nil
	}

	replacement, ok := buildInterpolatedString(parts)
	if !ok {
		return nil
	}

	sLine, sCol := offsetToLineCol(lineStarts, chainStart)
	eLine, eCol := offsetToLineCol(lineStarts, chainEnd)
	return []TextEdit{{
		File:      file,
		StartLine: sLine,
		StartCol:  sCol,
		EndLine:   eLine,
		EndCol:    eCol,
		NewText:   replacement,
		Title:     "Replace string '++' with string interpolation",
	}}
}

// locatePlusPlusOp verifies that `++` begins at opOff, or if opOff points
// earlier on the same line (e.g. start of LHS), finds the first `++` on that
// line outside strings and comments.
func locatePlusPlusOp(src string, lineStarts []int, line, opOff int) (int, bool) {
	if opOff+1 < len(src) && src[opOff] == '+' && src[opOff+1] == '+' {
		return opOff, true
	}
	lineEnd := len(src)
	if line < len(lineStarts) {
		lineEnd = lineStarts[line] - 1
	}
	for i := opOff; i+1 < lineEnd; {
		if src[i] == '"' {
			i = skipStringLiteral(src, i)
			continue
		}
		if src[i] == '-' && i+1 < lineEnd && src[i+1] == '-' {
			break
		}
		if src[i] == '+' && src[i+1] == '+' {
			return i, true
		}
		i++
	}
	return 0, false
}

// extractConcatChain scans left and right from the `++` at opOff to collect
// the full flat `e1 ++ e2 ++ ... ++ ek` operand chain on the same nesting level.
func extractConcatChain(src string, opOff int) (start, end int, parts []string, ok bool) {
	lhsStart, lhsEnd, ok := scanOperandBackward(src, opOff)
	if !ok {
		return 0, 0, nil, false
	}
	rhsStart, rhsEnd, ok := scanOperandForward(src, opOff+2)
	if !ok {
		return 0, 0, nil, false
	}

	leftParts := []string{strings.TrimSpace(src[lhsStart:lhsEnd])}
	chainStart := lhsStart
	for {
		prevOp, hasPrev := findPrevPlusPlus(src, chainStart)
		if !hasPrev {
			break
		}
		pStart, pEnd, ok := scanOperandBackward(src, prevOp)
		if !ok {
			break
		}
		leftParts = append(leftParts, strings.TrimSpace(src[pStart:pEnd]))
		chainStart = pStart
	}
	// Reverse leftParts so they are in source order.
	for i, j := 0, len(leftParts)-1; i < j; i, j = i+1, j-1 {
		leftParts[i], leftParts[j] = leftParts[j], leftParts[i]
	}

	rightParts := []string{strings.TrimSpace(src[rhsStart:rhsEnd])}
	chainEnd := rhsEnd
	for {
		nextOp, hasNext := findNextPlusPlus(src, chainEnd)
		if !hasNext {
			break
		}
		nStart, nEnd, ok := scanOperandForward(src, nextOp+2)
		if !ok {
			break
		}
		rightParts = append(rightParts, strings.TrimSpace(src[nStart:nEnd]))
		chainEnd = nEnd
	}

	parts = append(leftParts, rightParts...)
	return chainStart, chainEnd, parts, true
}

func findPrevPlusPlus(src string, before int) (int, bool) {
	i := before
	for i > 0 && (src[i-1] == ' ' || src[i-1] == '\t') {
		i--
	}
	if i >= 2 && src[i-2] == '+' && src[i-1] == '+' {
		return i - 2, true
	}
	return 0, false
}

func findNextPlusPlus(src string, after int) (int, bool) {
	n := len(src)
	i := after
	for i < n && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	if i+1 < n && src[i] == '+' && src[i+1] == '+' {
		return i, true
	}
	return 0, false
}

// scanOperandBackward scans a single primary/postfix expression immediately
// preceding `before` (skipping horizontal whitespace).
func scanOperandBackward(src string, before int) (int, int, bool) {
	end := before
	for end > 0 && (src[end-1] == ' ' || src[end-1] == '\t') {
		end--
	}
	if end == 0 {
		return 0, 0, false
	}

	pos := end
	consumed := false
	for pos > 0 {
		c := src[pos-1]
		switch {
		case c == '"':
			if consumed {
				return pos, end, true
			}
			strStart, ok := scanStringBackward(src, pos-1)
			if !ok {
				return 0, 0, false
			}
			return strStart, end, true

		case c == ')':
			open, ok := scanBalancedBackward(src, pos-1, '(', ')')
			if !ok {
				return 0, 0, false
			}
			pos = open
			consumed = true
			// A call `foo(x)` or `a.b(x)` continues backward into the callee.
			if pos > 0 && (isIdentByte(src[pos-1]) || src[pos-1] == ')' || src[pos-1] == ']') {
				continue
			}
			return pos, end, true

		case c == ']':
			open, ok := scanBalancedBackward(src, pos-1, '[', ']')
			if !ok {
				return 0, 0, false
			}
			pos = open
			consumed = true
			if pos > 0 && (isIdentByte(src[pos-1]) || src[pos-1] == ')' || src[pos-1] == ']') {
				continue
			}
			return pos, end, true

		case isIdentByte(c):
			idStart := pos - 1
			for idStart > 0 && isIdentByte(src[idStart-1]) {
				idStart--
			}
			word := src[idStart:pos]
			if isBoundaryKeyword(word) {
				if consumed {
					return pos, end, true
				}
				return 0, 0, false
			}
			pos = idStart
			consumed = true
			// Qualified name or field access: `a.b.c` or `f(x).name`.
			if pos > 0 && src[pos-1] == '.' {
				pos--
				continue
			}
			return pos, end, true

		default:
			if consumed {
				return pos, end, true
			}
			return 0, 0, false
		}
	}
	if consumed {
		return pos, end, true
	}
	return 0, 0, false
}

// scanOperandForward scans a single primary/postfix expression immediately
// following `after` (skipping horizontal whitespace).
func scanOperandForward(src string, after int) (int, int, bool) {
	n := len(src)
	start := after
	for start < n && (src[start] == ' ' || src[start] == '\t') {
		start++
	}
	if start >= n {
		return 0, 0, false
	}

	pos := start
	switch {
	case src[pos] == '"':
		end := skipStringLiteral(src, pos)
		if end <= pos+1 || src[end-1] != '"' {
			return 0, 0, false
		}
		return start, end, true

	case src[pos] == '(':
		end, ok := scanBalanced(src, pos, '(', ')')
		if !ok {
			return 0, 0, false
		}
		pos = end

	case src[pos] == '[':
		end, ok := scanBalanced(src, pos, '[', ']')
		if !ok {
			return 0, 0, false
		}
		pos = end

	case isIdentStart(src[pos]):
		idEnd := pos + 1
		for idEnd < n && isIdentByte(src[idEnd]) {
			idEnd++
		}
		if isBoundaryKeyword(src[pos:idEnd]) {
			return 0, 0, false
		}
		pos = idEnd

	default:
		return 0, 0, false
	}

	// Consume postfix continuations on the same chain: `.field`, `(args)`, `[idx]`.
	for pos < n {
		switch src[pos] {
		case '.':
			if pos+1 < n && isIdentStart(src[pos+1]) {
				pos++
				for pos < n && isIdentByte(src[pos]) {
					pos++
				}
				continue
			}
			return start, pos, true
		case '(':
			end, ok := scanBalanced(src, pos, '(', ')')
			if !ok {
				return 0, 0, false
			}
			pos = end
		case '[':
			end, ok := scanBalanced(src, pos, '[', ']')
			if !ok {
				return 0, 0, false
			}
			pos = end
		default:
			return start, pos, true
		}
	}
	return start, pos, true
}

func isBoundaryKeyword(word string) bool {
	switch word {
	case "let", "letrec", "in", "if", "then", "else", "match", "func", "pure", "export", "return":
		return true
	}
	return false
}

// scanStringBackward finds the opening `"` of the string literal whose closing
// `"` is at closeQuote. Scans from the start of the line forward to handle
// escapes and `${...}` interpolations accurately.
func scanStringBackward(src string, closeQuote int) (int, bool) {
	lineStart := closeQuote
	for lineStart > 0 && src[lineStart-1] != '\n' {
		lineStart--
	}
	for i := lineStart; i <= closeQuote; {
		if src[i] == '"' {
			end := skipStringLiteral(src, i)
			if end-1 == closeQuote {
				return i, true
			}
			if end <= i {
				return 0, false
			}
			i = end
			continue
		}
		i++
	}
	return 0, false
}

// scanBalancedBackward finds the matching openCh for closeCh at closeOff.
func scanBalancedBackward(src string, closeOff int, openCh, closeCh byte) (int, bool) {
	if closeOff < 0 || closeOff >= len(src) || src[closeOff] != closeCh {
		return 0, false
	}
	depth := 0
	for i := closeOff; i >= 0; i-- {
		c := src[i]
		if c == '"' {
			strStart, ok := scanStringBackward(src, i)
			if !ok {
				return 0, false
			}
			i = strStart
			continue
		}
		if c == closeCh {
			depth++
		} else if c == openCh {
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

// buildInterpolatedString combines operand fragments into a single `"..."`
// interpolated string literal. String-literal operands are inlined directly;
// non-literal operands are wrapped in `${...}`. Returns false if any operand
// contains a newline or (for non-literals) a double quote that would produce
// invalid interpolation syntax.
func buildInterpolatedString(parts []string) (string, bool) {
	var b strings.Builder
	b.WriteByte('"')
	for _, part := range parts {
		if part == "" || strings.ContainsRune(part, '\n') {
			return "", false
		}
		if isSingleStringLiteral(part) {
			b.WriteString(part[1 : len(part)-1])
			continue
		}
		if strings.ContainsRune(part, '"') {
			return "", false
		}
		b.WriteString("${")
		b.WriteString(part)
		b.WriteByte('}')
	}
	b.WriteByte('"')
	return b.String(), true
}

func isSingleStringLiteral(s string) bool {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return false
	}
	return skipStringLiteral(s, 0) == len(s)
}
