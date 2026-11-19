package types

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/types/exhaust"
)

// suggestCatchAll is the fix when no single arm is the answer.
const suggestCatchAll = "add a final catch-all arm: _ => ..."

// MatchError is one TC_MATCH_002/004/006/007 diagnostic. It unwraps to a
// structured ailerrors.Report, so `check --json` and `--format agent` carry
// its code, file:line:col and one-line suggestion.
type MatchError struct {
	Code       string
	Pos        ast.Pos
	Message    string   // without location or code
	Suggestion string   // one line, for machine output
	Notes      []string // further human-only lines
	rep        *ailerrors.ReportError
}

func (e *MatchError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s: %s", e.Pos, e.Code, e.Message)
	if e.Suggestion != "" {
		b.WriteString("\n  Suggestion: " + e.Suggestion)
	}
	for _, n := range e.Notes {
		b.WriteString("\n  " + n)
	}
	return b.String()
}

// Unwrap exposes the structured report to ailerrors.AsReport.
func (e *MatchError) Unwrap() error { return e.rep }

func newMatchError(code string, pos ast.Pos, msg, suggestion string, notes []string, data map[string]any) *MatchError {
	span := &ast.Span{Start: pos, End: pos}
	rep := &ailerrors.Report{
		Schema:  "ailang.error/v1",
		Code:    code,
		Phase:   "typecheck",
		Message: msg,
		Span:    span,
		Data:    data,
	}
	if suggestion != "" {
		rep.Fix = &ailerrors.Fix{Suggestion: suggestion, Confidence: 0.9}
	}
	return &MatchError{Code: code, Pos: pos, Message: msg, Suggestion: suggestion, Notes: notes,
		rep: &ailerrors.ReportError{Rep: rep}}
}

// MatchErrors is every match error of one module, sorted by position.
type MatchErrors []*MatchError

func (es MatchErrors) Error() string {
	parts := make([]string, len(es))
	for i, e := range es {
		parts[i] = e.Error()
	}
	return strings.Join(parts, "\n")
}

// Unwrap returns each error, so errors.As finds the first report.
func (es MatchErrors) Unwrap() []error {
	out := make([]error, len(es))
	for i, e := range es {
		out[i] = e
	}
	return out
}

// shownWitnesses caps how many missing patterns a message names; more are
// summarised as "and more".
const shownWitnesses = 5

// newNonExhaustiveError is TC_MATCH_002 with its witnesses (the check asks
// for one more than it shows, to know whether there are more).
func newNonExhaustiveError(pos ast.Pos, typeName string, missing []exhaust.Pattern, guarded bool) *MatchError {
	more := len(missing) > shownWitnesses
	if more {
		missing = missing[:shownWitnesses]
	}
	described := make([]string, len(missing))
	rendered := make([]string, len(missing))
	for i, w := range missing {
		described[i] = exhaust.Describe(w)
		rendered[i] = exhaust.Render(w)
	}
	on := ""
	if typeName != "" {
		on = " on " + typeName
	}
	list := strings.Join(described, ", ")
	if more {
		list += ", and more"
	}
	msg := fmt.Sprintf("non-exhaustive match%s — no arm matches: %s", on, list)
	var suggestion string
	switch {
	case len(rendered) == 1 && rendered[0] == "_":
		suggestion = suggestCatchAll
	case len(rendered) == 1:
		suggestion = fmt.Sprintf("add the missing arm: %s => ...", rendered[0])
	default:
		arms := make([]string, len(rendered))
		for i, r := range rendered {
			arms[i] = r + " => ..."
		}
		suggestion = "add the missing arms: " + strings.Join(arms, ", ") + " (or end with _ => ...)"
	}
	notes := []string{fmt.Sprintf("If this case cannot happen, say so: %s => unreachable(\"<why it cannot happen>\")", rendered[0])}
	if guarded {
		notes = append(notes, "Note: arms with `if` guards do not count toward coverage.")
	}
	return newMatchError(TC_MATCH_002, pos, msg, suggestion, notes, map[string]any{"missing": rendered})
}

// newUnresolvedCtorError is TC_MATCH_007 (#2303), located at the pattern
// when it has a position. The fix that works is to import the type — its
// constructors then come into scope and the pattern fixes the value's type —
// or to annotate the value matched.
func newUnresolvedCtorError(matchPos ast.Pos, err error) *MatchError {
	var e *unresolvedCtorError
	if !errors.As(err, &e) {
		return newUndecidedError(matchPos, err)
	}
	pos := matchPos
	if e.pattern.Pos.Line > 0 {
		pos = e.pattern.Pos
		if pos.File == "" {
			pos.File = matchPos.File
		}
	}
	imports := make([]string, len(e.cands))
	described := make([]string, len(e.cands))
	for i, key := range e.cands {
		mod, name := SplitNominalKey(key)
		imports[i] = fmt.Sprintf("import %s (%s)", mod, name)
		described[i] = fmt.Sprintf("%s's %s", mod, name)
	}
	// Each fix is one step that decides the pattern's type.
	var fix string
	switch {
	case e.fixedType != "":
		fix = "remove the arm"
	case e.scrutinee != "" && len(e.ctors) > 0:
		_, name := SplitNominalKey(e.scrutinee)
		fix = fmt.Sprintf("remove the arm, or match one of %s's constructors: %s", name, strings.Join(e.ctors, ", "))
	case e.scrutinee != "":
		fix = "remove the arm"
	case e.rigid != "" && len(described) == 1:
		fix = fmt.Sprintf("write the value's type instead of %s, %s, and import it: %s", e.rigid, described[0], imports[0])
	case e.rigid != "" && len(described) > 1:
		fix = fmt.Sprintf("write the value's type instead of %s, and import it: %s", e.rigid, strings.Join(imports, " or "))
	case e.rigid != "":
		fix = fmt.Sprintf("write the value's type instead of %s, and import it", e.rigid)
	case len(imports) == 1:
		// Imported, the type's constructors are in scope and the pattern
		// itself fixes the value's type.
		fix = imports[0]
	case len(imports) > 1:
		fix = "import the type you mean: " + strings.Join(imports, " or ")
	default:
		fix = "import the type that declares " + e.pattern.Name
	}
	var notes []string
	if e.scrutinee == "" && e.rigid == "" && e.fixedType == "" && len(described) > 0 {
		notes = append(notes, fmt.Sprintf("Loaded types declaring %s: %s. A catch-all arm does not decide which one is meant.", e.pattern.Name, strings.Join(described, ", ")))
	}
	return newMatchError(TC_MATCH_007, pos, e.Error(), fix, notes,
		map[string]any{"constructor": e.pattern.Name, "candidates": described})
}

// newBudgetError is TC_MATCH_004.
func newBudgetError(pos ast.Pos, cause error) *MatchError {
	return newMatchError(TC_MATCH_004, pos,
		fmt.Sprintf("this match is too large to check for exhaustiveness (%v)", cause),
		suggestCatchAll, nil, nil)
}

// newUndecidedError is TC_MATCH_002 for a match whose coverage the checker
// cannot decide; failing closed, it names the reason.
func newUndecidedError(pos ast.Pos, cause error) *MatchError {
	return newMatchError(TC_MATCH_002, pos,
		fmt.Sprintf("this match cannot be checked for exhaustiveness: %s", strings.TrimPrefix(cause.Error(), errUndecidable.Error()+": ")),
		suggestCatchAll, nil, nil)
}

// patternArityError is a constructor pattern with the wrong number of arguments.
type patternArityError struct {
	pattern *core.ConstructorPattern
	arity   int
}

func (e *patternArityError) Error() string {
	return fmt.Sprintf("constructor pattern %s has %d argument(s), but %s takes %d", e.pattern.Name, len(e.pattern.Args), e.pattern.Name, e.arity)
}

// newPatternArityError is TC_MATCH_006, located at the pattern when it has a position.
func newPatternArityError(matchPos ast.Pos, e *patternArityError) *MatchError {
	pos := matchPos
	if e.pattern.Pos.Line > 0 {
		pos = e.pattern.Pos
		if pos.File == "" {
			pos.File = matchPos.File
		}
	}
	// One `_` per declared field, appended rather than pre-sized: the arity
	// comes from the program's own declaration.
	var holes []string
	for i := 0; i < e.arity; i++ {
		holes = append(holes, "_")
	}
	fix := e.pattern.Name
	if len(holes) > 0 {
		fix += "(" + strings.Join(holes, ", ") + ")"
	}
	return newMatchError(TC_MATCH_006, pos, e.Error(), "write "+fix, nil, nil)
}

// MatchWarning is TC_MATCH_003: an arm no value can reach, because the
// unguarded arms before it match everything it matches.
type MatchWarning struct {
	Pos     ast.Pos // the match
	Arm     int     // 1-based
	Pattern string
}

func (w *MatchWarning) Code() string { return TC_MATCH_003 }

// Position is the match's line and column.
func (w *MatchWarning) Position() (int, int) { return w.Pos.Line, w.Pos.Column }

// WarningFile is the match's file.
func (w *MatchWarning) WarningFile() string { return w.Pos.File }

func (w *MatchWarning) Message() string {
	return fmt.Sprintf("arm %d (%s) of this match can never be reached: earlier arms match every value it matches", w.Arm, w.Pattern)
}

func (w *MatchWarning) Suggestion() string {
	return "remove the arm, or move it above the arm that already covers it"
}

func (w *MatchWarning) String() string {
	return fmt.Sprintf("warning: %s at %s: %s", TC_MATCH_003, w.Pos, w.Message())
}
