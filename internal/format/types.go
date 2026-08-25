package format

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// types.go prints type-annotation nodes to canonical source. Types appear in
// parameter annotations, return types, let/binder annotations, method
// signatures, type aliases, and constructor fields.

// formatEffectRow renders an effect annotation slice for the formatter,
// preserving the nil-vs-empty distinction that ast.FormatEffects collapses:
//
//   - nil          → ""       (no annotation was present in the source)
//   - non-nil, len 0 → "! {}"  (an explicit empty/pure row `! {}`)
//   - len > 0      → "! {e1, e2, ...}"
//
// Element rendering reuses ast.EffectAnnotation.String() to stay byte-identical
// to ast.FormatEffects for the non-empty case, so existing goldens stay green.
// This exists (rather than reusing ast.FormatEffects) because round-trip fmt
// must round-trip `! {}`: the parser produces a non-nil empty slice for `! {}`
// but nil for no annotation, and ast.FormatEffects returns "" for both — which
// dropped `! {}` and broke round-trip verification (controller-found defect).
func formatEffectRow(effects []ast.EffectAnnotation) string {
	if effects == nil {
		return ""
	}
	if len(effects) == 0 {
		return "! {}"
	}
	parts := make([]string, len(effects))
	for i := range effects {
		parts[i] = effects[i].String()
	}
	return "! {" + strings.Join(parts, ", ") + "}"
}

// typeString renders any ast.Type to a canonical source string, or errors on a
// nil-required child or unknown concrete type.
func (p *printer) typeString(t ast.Type) (string, error) {
	if t == nil {
		return "", fmt.Errorf("nil type node")
	}
	switch n := t.(type) {
	case *ast.SimpleType:
		return n.Name, nil
	case *ast.TypeVar:
		return n.Name, nil
	case *ast.ListType:
		el, err := p.typeString(n.Element)
		if err != nil {
			return "", err
		}
		return "[" + el + "]", nil
	case *ast.ArrayType:
		el, err := p.typeString(n.Element)
		if err != nil {
			return "", err
		}
		return "Array[" + el + "]", nil
	case *ast.TupleType:
		parts, err := p.typeList(n.Elements)
		if err != nil {
			return "", err
		}
		return "(" + strings.Join(parts, ", ") + ")", nil
	case *ast.TypeApp:
		args, err := p.typeList(n.Args)
		if err != nil {
			return "", err
		}
		// `[T]` is normalized to TypeApp{Constructor:"list"} at parse time
		// (parser_type.go, DX-17 Phase 2) so the compiler has one internal
		// representation. The surface sugar is erased there, which meant the
		// formatter printed the generic `Constructor[args]` form and turned
		// every `[int]` a user wrote into `list[int]`.
		//
		// That is not cosmetic. `ailang prompt` — the canonical teaching text
		// every eval model is given — uses `[int]` 64 times and `list[...]`
		// ZERO times. So `ailang fmt` was telling models their correct code was
		// non-canonical and handing them a dialect the prompt never taught.
		// Measured 2026-07-30: a weak local model followed that advice, switched
		// dialect mid-run, broke working code, and burned ~60% more tokens.
		//
		// The ast.ListType case above is unreachable for parsed input for the
		// same reason; it is kept for hand-built ASTs.
		if n.Constructor == "list" && len(args) == 1 {
			return "[" + args[0] + "]", nil
		}
		return n.Constructor + "[" + strings.Join(args, ", ") + "]", nil
	case *ast.FuncType:
		return p.funcTypeString(n)
	case *ast.RecordType:
		return p.recordTypeString(n)
	case *ast.LabelledType:
		return p.labelledTypeString(n)
	default:
		return "", fmt.Errorf("unsupported type node: %T", t)
	}
}

// typeList renders a slice of types, propagating the first error.
func (p *printer) typeList(ts []ast.Type) ([]string, error) {
	parts := make([]string, len(ts))
	for i, t := range ts {
		s, err := p.typeString(t)
		if err != nil {
			return nil, err
		}
		parts[i] = s
	}
	return parts, nil
}

// funcTypeString renders a function type `(P1, P2) -> R ! {E}`, or the bare
// arrow form `P -> R ! {E}` for a single unambiguous parameter.
//
// `int -> int` and `(int) -> int` parse to the IDENTICAL FuncType
// (parser_type.go: the S-ARROWTYPE sugar path and the parenthesised path both
// yield Params:[int]), so both round-trip. Only one of them is taught: the
// active prompt writes `int -> int` and never `(int) -> int`. Always emitting
// the parenthesised form made fmt rewrite 10 of the prompt's own examples —
// the same class of contradiction as `[int]`→`list[int]`, and the second-largest
// dialect divergence after string interpolation. See
// TestFmtDoesNotDriftFromTeachingPrompt.
func (p *printer) funcTypeString(n *ast.FuncType) (string, error) {
	params, err := p.typeList(n.Params)
	if err != nil {
		return "", err
	}
	ret, err := p.typeString(n.Return)
	if err != nil {
		return "", err
	}
	eff := formatEffectRow(n.Effects)
	if eff != "" {
		eff = " " + eff
	}
	if len(params) == 1 && bareArrowSafe(n.Params[0]) {
		return params[0] + " -> " + ret + eff, nil
	}
	return "(" + strings.Join(params, ", ") + ") -> " + ret + eff, nil
}

// bareArrowSafe reports whether a sole parameter type can drop its parentheses
// without changing what the result re-parses to.
//
// Three shapes must keep them:
//   - a FuncType parameter, because the arrow is RIGHT-associative: `(int ->
//     int) -> int` and `int -> int -> int` are different types;
//   - a TupleType parameter, because `(int, string) -> bool` is read as a
//     TWO-parameter function type, not as one tuple parameter — the only
//     spelling for the latter is `((int, string)) -> bool`;
//   - a RecordType parameter, because the emitted `{ a: int } -> ()` opens with
//     `{`, which the parser reads as a BLOCK and not as a type. This is the
//     defect that made `std/cognition.ail` (whose `subscribeMsg` takes a record-
//     shaped callback) emit source that failed to re-parse.
func bareArrowSafe(t ast.Type) bool {
	switch n := t.(type) {
	case *ast.FuncType, *ast.TupleType, *ast.RecordType:
		return false
	case *ast.LabelledType:
		// A label/refinement prints as `<base><suffix>`, so safety is decided
		// entirely by the base: `{a: int}<L>` still opens with `{`.
		return bareArrowSafe(n.Base)
	case *ast.SimpleType, *ast.TypeVar, *ast.ListType, *ast.ArrayType, *ast.TypeApp:
		return true
	default:
		// Whitelist, not blacklist: an ast.Type node added later must default to
		// KEEPING its parentheses. Verbose output is a cosmetic defect; dropping
		// parens that turn out to be load-bearing emits source that does not
		// re-parse, which is a soundness defect.
		return false
	}
}

// recordTypeString renders a record type `{ a: T, b: U }`, an open record in the
// taught `...` sugar (`{ a: T, ... }`), or an explicit row variable
// (`{ a: T | r }`).
func (p *printer) recordTypeString(n *ast.RecordType) (string, error) {
	fields := make([]string, len(n.Fields))
	for i, f := range n.Fields {
		ft, err := p.typeString(f.Type)
		if err != nil {
			return "", err
		}
		fields[i] = f.Name + ": " + ft
	}
	body := strings.Join(fields, ", ")
	if n.Row != nil {
		// `{a: T, ...}` desugars to a row variable with a COMPILER-GENERATED name
		// (parser.go:freshRowVarName → `_r0`, `_r1`, …). Printing that name raw
		// leaked an internal identifier into user-facing source: fmt answered the
		// prompt's own `{email: string, ...}` with `{ email: string | _r0 }`.
		// Re-emitting the `...` sugar round-trips because the names are assigned in
		// source order by a per-parse counter, so a re-parse regenerates the same
		// sequence. An explicitly-written row variable is never `_r<digits>` and is
		// printed as-is.
		if generatedRowVar(n.Row.Name) {
			if body == "" {
				return "{ ... }", nil
			}
			return "{ " + body + ", ... }", nil
		}
		if body == "" {
			return "{ | " + n.Row.Name + " }", nil
		}
		return "{ " + body + " | " + n.Row.Name + " }", nil
	}
	if body == "" {
		return "{}", nil
	}
	return "{ " + body + " }", nil
}

// generatedRowVarRe matches the compiler-generated row-variable names the parser
// synthesizes for the `...` open-record sugar (parser.go:freshRowVarName).
var generatedRowVarRe = regexp.MustCompile(`^_r[0-9]+$`)

// generatedRowVar reports whether a row-variable name was synthesized by the
// parser rather than written by the user.
func generatedRowVar(name string) bool {
	return generatedRowVarRe.MatchString(name)
}

// labelledTypeString renders IFC label / refinement syntax `T<label>` / `T{not L}`.
func (p *printer) labelledTypeString(n *ast.LabelledType) (string, error) {
	base, err := p.typeString(n.Base)
	if err != nil {
		return "", err
	}
	if n.Label != nil {
		return base + "<" + n.Label.Name + ">", nil
	}
	if n.Refinement != nil {
		return base + "{not " + n.Refinement.NotLabel + "}", nil
	}
	return base, nil
}
