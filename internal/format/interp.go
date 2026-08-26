package format

import (
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// interp.go reconstructs the string-interpolation spelling of a desugared
// concat_String chain.
//
// # WHY THIS EXISTS
//
// `"a ${x} b"` is desugared by parseInterpolatedString (parser_literals.go) into
// a left-associative chain of concat_String calls whose holes are wrapped in
// show(). Nothing in the AST records that the source was an interpolation, so
// without re-sugaring the formatter answers a model's
//
//	println("${name} is ${show(age)}")
//
// with
//
//	println(concat_String(concat_String(show(name), " is "), show(show(age))))
//
// `ailang prompt` — the canonical teaching text handed to every eval model —
// uses `"${x}"` in nearly every example and never teaches concat_String chains.
// So fmt was telling models their correct code was non-canonical, on every
// single write. Measured 2026-07-30 on arms differing only by the fmt extension:
// +62% output tokens, median +2,693, 18/23 pairs, Wilcoxon p=0.0112. A model
// wrote, in its own context, "the file type-checks but the formatter wants
// canonical style", switched dialect, and broke working code.
//
// No type-check can catch that: both spellings parse and mean the same thing.
// See TestFmtDoesNotDriftFromTeachingPrompt for the dialect gate.
//
// # WHY IT IS THIS CONSERVATIVE
//
// Re-sugaring must be exactly reversible or it breaks round-trip, which is a far
// worse failure than leaving a concat chain unsugared. Every guard below marks a
// shape where Parse(fmt(x)) would differ from Parse(x); when any fires, the
// caller falls through to the ordinary call printer and emits the chain verbatim.

// interpolationString renders a concat_String chain back to interpolated-string
// source, reporting ok=false for any chain whose interpolated spelling would not
// re-parse to the identical AST.
func (p *printer) interpolationString(n *ast.FuncCall) (string, bool) {
	parts, ok := flattenConcatChain(n)
	if !ok {
		return "", false
	}

	var b strings.Builder
	b.WriteByte('"')
	sawHole := false
	prevWasText := false

	for _, part := range parts {
		if lit, isLit := part.(*ast.Literal); isLit && lit.Kind == ast.StringLit {
			s, isStr := lit.Value.(string)
			if !isStr {
				return "", false
			}
			// The parser ELIDES empty string parts, so an empty literal here was
			// hand-written; re-emitting it as interpolation text would drop it.
			if s == "" {
				return "", false
			}
			// Two adjacent literals came from a hand-written
			// concat_String("a", "b"), not from an interpolation. As text they
			// would fuse into a single "ab" part and re-parse to a different tree.
			if prevWasText {
				return "", false
			}
			txt, escOK := escapeInterpText(s)
			if !escOK {
				return "", false
			}
			b.WriteString(txt)
			prevWasText = true
			continue
		}

		// The only other shape an interpolation produces is a hole: show(expr).
		// A bare non-show operand means the chain was hand-written (concat_String
		// takes strings directly, with no show wrapper) and must be left alone.
		call, isCall := part.(*ast.FuncCall)
		if !isCall {
			return "", false
		}
		id, isID := call.Func.(*ast.Identifier)
		if !isID || id.Name != "show" || len(call.Args) != 1 {
			return "", false
		}
		hole, holeOK := p.holeText(call.Args[0])
		if !holeOK {
			return "", false
		}
		b.WriteString("${" + hole + "}")
		sawHole = true
		prevWasText = false
	}

	// A chain of pure literals is a hand-written concat_String, not an
	// interpolation: `concat_String("a", "b")` must not become `"ab"`.
	if !sawHole {
		return "", false
	}
	b.WriteByte('"')
	return b.String(), true
}

// flattenConcatChain returns the operands of a concat_String chain in source
// order, reporting ok=false if n is not such a chain.
//
// It descends the LEFT spine only. The parser folds interpolation segments
// left-associatively, so a right-nested `concat_String(a, concat_String(b, c))`
// is necessarily hand-written; flattening it and re-emitting the flat form would
// silently re-associate the tree and break round-trip.
func flattenConcatChain(n *ast.FuncCall) ([]ast.Expr, bool) {
	if !isConcatString(n) {
		return nil, false
	}
	var parts []ast.Expr
	if inner, ok := n.Args[0].(*ast.FuncCall); ok && isConcatString(inner) {
		sub, subOK := flattenConcatChain(inner)
		if !subOK {
			return nil, false
		}
		parts = append(parts, sub...)
	} else {
		parts = append(parts, n.Args[0])
	}
	return append(parts, n.Args[1]), true
}

func isConcatString(n *ast.FuncCall) bool {
	id, ok := n.Func.(*ast.Identifier)
	return ok && id.Name == "concat_String" && len(n.Args) == 2
}

// holeText renders an interpolation hole's expression to a single line of source,
// reporting ok=false when the rendered text cannot safely sit inside `${...}`.
//
// The hole lexer is brace- AND string-aware, so nested record literals, blocks,
// match expressions and quoted strings are all fine inside `${...}` — verified
// against the lexer, and the active prompt teaches exactly that ("Nested braces
// work (record literals, field access, let-blocks)"). Only three shapes are
// refused, and all three are about the RE-LEXER rather than the expression:
//
//   - a newline, which no string literal may contain;
//   - `--`, which would comment out the rest of the line and swallow the closing
//     quote. This is the fail-closed guard for a comment rendered inside a hole:
//     the sub-printer shares the attachment index, so such a comment is emitted
//     and then caught here rather than being silently dropped;
//   - `${`, which the lexer does not accept nested inside a hole's own string.
//
// Any refusal falls back to printing the chain in ordinary call form, which is
// always correct — just not the taught spelling.
func (p *printer) holeText(e ast.Expr) (string, bool) {
	sub := &printer{
		w:                newWriter(p.w.indent),
		att:              p.att,
		maxWidth:         p.maxWidth,
		measuring:        p.measuring,
		measurementDepth: p.measurementDepth,
	}
	if err := sub.expr(e, precLowest); err != nil {
		return "", false
	}
	// The preceding error path deliberately discards sub.expr's concrete error;
	// only this path propagates a successful sub.expr with measurementErr set.
	// TestMeasurementErrorAlwaysAccompaniesRenderError measured that live cell
	// empty across 312 injected corpus sites. Keep the propagation as fail-closed
	// defence; M2's continuation layout is the change most likely to make it live.
	if sub.measurementErr != nil {
		p.measurementErr = sub.measurementErr
		return "", false
	}
	s := sub.w.buf.String()
	if strings.ContainsAny(s, "\n") || strings.Contains(s, "--") || strings.Contains(s, "${") {
		return "", false
	}
	return s, true
}

// escapeInterpText escapes a decoded string segment for use as interpolation
// text, reporting ok=false if the segment cannot be represented there.
//
// It reuses escapeString — the single canonical escaping routine (literal.go) —
// and strips its surrounding quotes, so interpolation text and plain string
// literals can never disagree about how a byte is spelled. That is what makes
// this total: escapeString already emits `\"` for a quote and `\${` for a
// literal interpolation marker, and a bare `{` or `}` in a text segment is
// ordinary text to the lexer. The ok result is retained so a future escaping
// change has somewhere to refuse rather than corrupt.
func escapeInterpText(s string) (string, bool) {
	esc := escapeString(s)
	// escapeString always brackets its result in ASCII double quotes.
	return esc[1 : len(esc)-1], true
}
