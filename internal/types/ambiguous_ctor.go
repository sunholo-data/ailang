package types

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// TC_AMBIGUOUS_CTOR_001 codes a constructor name that two or more imported
// modules offer through the constructor auto-import (M-CTOR-AUTO), that no
// import names explicitly and that nothing in the program decides
// (M-V1-NOMINAL-TYPE-IDENTITY). It used to mean whichever module was imported
// first.
const TC_AMBIGUOUS_CTOR_001 = "TC_AMBIGUOUS_CTOR_001"

// AmbiguousCtorError is TC_AMBIGUOUS_CTOR_001.
type AmbiguousCtorError struct {
	Name       string
	Candidates []TypeRef // the ADTs offering it, sorted by module
	Where      string
}

// NewAmbiguousCtorError builds TC_AMBIGUOUS_CTOR_001.
func NewAmbiguousCtorError(name string, cands []TypeRef, where string) *AmbiguousCtorError {
	return &AmbiguousCtorError{Name: name, Candidates: cands, Where: where}
}

func (e *AmbiguousCtorError) Error() string {
	parts := make([]string, len(e.Candidates))
	for i, c := range e.Candidates {
		parts[i] = fmt.Sprintf("%s's %s", c.Module, c.Name)
	}
	first := ""
	if len(e.Candidates) > 0 {
		first = e.Candidates[0].Module
	}
	return fmt.Sprintf("%s %s: constructor '%s' is ambiguous: it belongs to %s, all imported, and nothing here says which one is meant.\n"+
		"  Suggestion: import the constructor by name from the module you mean, e.g. import %s (%s).",
		TC_AMBIGUOUS_CTOR_001, e.Where, e.Name, strings.Join(parts, " and "), first, e.Name)
}

// SetAmbiguousConstructors installs the auto-imported constructor names that
// two or more modules offer (constructor -> candidate ADTs).
func (tc *CoreTypeChecker) SetAmbiguousConstructors(m map[string][]TypeRef) {
	tc.ambiguousCtors = m
}

// resolveAmbiguousPattern decides which ADT an ambiguous constructor pattern
// names: the candidate the scrutinee's type already is. The scrutinee's type
// is read through the equalities collected so far in this declaration (a
// speculative solve; nothing is committed). When the type is not known, or is
// none of the candidates, the pattern is TC_AMBIGUOUS_CTOR_001 rather than a
// silent pick.
func (tc *CoreTypeChecker) resolveAmbiguousPattern(p *core.ConstructorPattern, scrutType Type, ctx *InferenceContext) error {
	cands := tc.ambiguousCtors[p.Name]
	t := scrutType
	if sub, err := ctx.solveLocalEqualities(0); err == nil {
		t = ApplySubstitution(sub, scrutType)
	}
	if app, ok := t.(*TApp); ok {
		t, _ = decomposeApp(app)
	}
	if head, ok := t.(*TCon); ok {
		for _, c := range cands {
			if c.Name == head.Name && c.Module == head.Module {
				p.TypeName, p.TypeModule = c.Name, c.Module
				ObserveAmbiguity(AmbiguityEvent{Kind: "ctor-pattern-resolved", Name: p.Name, Candidates: cands})
				return nil
			}
		}
	}
	ObserveAmbiguity(AmbiguityEvent{Kind: "ctor-pattern-unresolved", Name: p.Name, Candidates: cands})
	if !nominalAmbiguityIsError {
		p.TypeName, p.TypeModule = cands[0].Name, cands[0].Module
		return nil
	}
	return NewAmbiguousCtorError(p.Name, cands, "in a pattern")
}
