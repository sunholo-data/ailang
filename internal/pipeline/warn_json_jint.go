package pipeline

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
)

// TC_MATCH_005 — transitional warning for the `JInt` split of std/json's
// `Json` (M-V1-STDLIB-SURFACE-POLISH §7, ledger D-66 ruling 4).
//
// `json.decode` used to give every number as `JNumber(float)`. It now gives
// `JInt(int)` for integer literals, so a match written before the split,
//
//	match j { JNumber(n) => …, _ => fallback }
//
// still compiles but silently sends every integer to the catch-all. This pass
// flags any match that has a std/json `JNumber` arm and no `JInt` arm, so the
// fall-through is visible for the whole v0.58.x window. It is keyed on
// constructor identity (std/json's `JNumber`), never on the bare name alone: a
// module that declares its own `JNumber` is not flagged.
//
// It runs over Core in the module pipeline beside the split-arg pass, so it
// sees cached modules too (their Core comes from the cache entry).

// CodeJSONIntArm is the stable diagnostic code of the warning.
const CodeJSONIntArm = "TC_MATCH_005"

const (
	jsonModule     = "std/json"
	jsonTypeName   = "Json"
	jsonNumberCtor = "JNumber"
	jsonIntCtor    = "JInt"
)

// JSONIntArmWarning is a non-blocking warning: a match over std/json's Json
// has a JNumber arm but no JInt arm.
type JSONIntArmWarning struct {
	Location ast.Pos
	Module   string // module the match is in
}

var _ elaborate.Warning = (*JSONIntArmWarning)(nil)

// Position exposes the match's 1-indexed line/col.
func (w *JSONIntArmWarning) Position() (line, col int) {
	return w.Location.Line, w.Location.Column
}

// Code returns the stable diagnostic code.
func (w *JSONIntArmWarning) Code() string { return CodeJSONIntArm }

// WarningFile names the source file of the match (it may be an imported
// module's, not the checked file).
func (w *JSONIntArmWarning) WarningFile() string { return w.Location.File }

// Suggestion is the one-line fix.
func (w *JSONIntArmWarning) Suggestion() string {
	return "add a `JInt(i) => …` arm (intToFloat(i) gives a float), or read numbers with asNumber/getNumber/getInt, which accept both"
}

// Message is the warning without its location and fix (structured output
// carries those as fields).
func (w *JSONIntArmWarning) Message() string {
	return "this match over std/json.Json has a JNumber arm but no JInt arm; " +
		"decode now gives JInt(n) for integer literals such as 42, so integers skip the JNumber arm"
}

// String renders the warning for stderr.
func (w *JSONIntArmWarning) String() string {
	return fmt.Sprintf("%s at %s\n  %s.\n  Fix: %s",
		CodeJSONIntArm, w.Location.String(), w.Message(), w.Suggestion())
}

// DetectJSONIntArm returns one warning per match in unit that has a std/json
// JNumber arm and no JInt arm (at any pattern depth, so `Some(JNumber(n))`
// counts).
func DetectJSONIntArm(unit *CompileUnit) []elaborate.Warning {
	if unit == nil || unit.Core == nil {
		return nil
	}
	isStdJNumber := stdJSONNumberResolver(unit)
	var out []elaborate.Warning
	var visit func(e core.CoreExpr)
	visit = func(e core.CoreExpr) {
		switch n := e.(type) {
		case *core.Match:
			var hasNumber, hasInt bool
			for _, arm := range n.Arms {
				walkCtorPatterns(arm.Pattern, func(p *core.ConstructorPattern) {
					switch {
					case p.Name == jsonIntCtor:
						hasInt = true
					case p.Name == jsonNumberCtor && isStdJNumber(p):
						hasNumber = true
					}
				})
			}
			if hasNumber && !hasInt {
				out = append(out, &JSONIntArmWarning{Location: n.OriginalSpan(), Module: unit.ID})
			}
		// walkCore does not descend into these; their children are atomic in
		// ANF, except a lambda value, which can hold a match.
		case *core.Tuple:
			for _, el := range n.Elements {
				walkCore(el, visit)
			}
		case *core.Array:
			for _, el := range n.Elements {
				walkCore(el, visit)
			}
		case *core.RecordUpdate:
			walkCore(n.Base, visit)
			for _, v := range n.Updates {
				walkCore(v, visit)
			}
		}
	}
	WalkCore(unit.Core, visit)
	return out
}

// stdJSONNumberResolver decides, for this unit, whether a JNumber constructor
// pattern denotes std/json's constructor:
//
//   - inside std/json itself, yes;
//   - a module that declares its own JNumber: only patterns the elaborator
//     resolved to an ADT named Json that is not the local one (an aliased
//     import of std/json's JNumber next to the local declaration);
//   - a module that imports JNumber from some other module: no;
//   - otherwise (imported from std/json, or matched by name with std/json
//     loaded transitively): yes.
func stdJSONNumberResolver(unit *CompileUnit) func(*core.ConstructorPattern) bool {
	yes := func(*core.ConstructorPattern) bool { return true }
	no := func(*core.ConstructorPattern) bool { return false }
	if unit.ID == jsonModule {
		return yes
	}
	if local, ok := unit.Constructors[jsonNumberCtor]; ok && local != nil {
		if local.TypeName == jsonTypeName {
			return no
		}
		return func(p *core.ConstructorPattern) bool { return p.TypeName == jsonTypeName }
	}
	if unit.Surface != nil {
		fromStd, fromOther := false, false
		for _, imp := range unit.Surface.Imports {
			for _, sym := range imp.Symbols {
				if sym != jsonNumberCtor {
					continue
				}
				if imp.Path == jsonModule {
					fromStd = true
				} else {
					fromOther = true
				}
			}
		}
		if fromOther && !fromStd {
			return no
		}
	}
	return yes
}

// walkCtorPatterns calls f for every constructor pattern inside p.
func walkCtorPatterns(p core.CorePattern, f func(*core.ConstructorPattern)) {
	switch n := p.(type) {
	case *core.ConstructorPattern:
		f(n)
		for _, a := range n.Args {
			walkCtorPatterns(a, f)
		}
	case *core.ListPattern:
		for _, el := range n.Elements {
			walkCtorPatterns(el, f)
		}
		if n.Tail != nil {
			walkCtorPatterns(*n.Tail, f)
		}
	case *core.TuplePattern:
		for _, el := range n.Elements {
			walkCtorPatterns(el, f)
		}
	case *core.RecordPattern:
		for _, fp := range n.Fields {
			walkCtorPatterns(fp, f)
		}
	}
}
