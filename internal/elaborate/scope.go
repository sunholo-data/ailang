package elaborate

import "github.com/sunholo-data/ailang/internal/core"

// Lexical scope for identifier resolution (M-ELABORATOR-LEXICAL-SCOPE, #1467).
//
// Expression-position identifiers used to resolve constructor table →
// globalEnv (every import AND every builtin) → local, with no notion of which
// local binders were visible. A lambda parameter, function parameter, let,
// letrec, block statement-let, match binder or forall variable named like an
// import, a builtin or a constructor was therefore captured by it: false type
// errors at best, a silently wrong program at worst. Each binder construct now
// pushes a frame while its scope region is normalized, and a name bound in any
// open frame resolves to the local before the global tables are consulted.
//
// Pattern position is untouched: constructor-first classification (#323)
// still decides whether an identifier in a pattern is a binder.
//
// Module-level declarations are NOT a frame here: a module func that shares
// a name with an import still loses to the import (design doc row 7, a
// separate decision).

// pushScope opens a frame binding names. "_" and "" bind nothing.
func (e *Elaborator) pushScope(names ...string) {
	frame := make(map[string]bool, len(names))
	for _, n := range names {
		if n != "" && n != "_" {
			frame[n] = true
		}
	}
	e.scope = append(e.scope, frame)
}

// popScope closes the innermost frame. Callers defer it right after pushScope
// so an elaboration error cannot leak a frame into the next declaration.
func (e *Elaborator) popScope() {
	e.scope = e.scope[:len(e.scope)-1]
}

// inScope reports whether name is bound by an enclosing local binder.
func (e *Elaborator) inScope(name string) bool {
	for i := len(e.scope) - 1; i >= 0; i-- {
		if e.scope[i][name] {
			return true
		}
	}
	return false
}

// corePatternBinders lists the variables an elaborated pattern binds. It reads
// the CORE pattern, so it agrees with elaboratePattern's constructor-vs-binder
// classification by construction.
func corePatternBinders(pat core.CorePattern) []string {
	switch p := pat.(type) {
	case *core.VarPattern:
		return []string{p.Name}
	case *core.ConstructorPattern:
		return corePatternListBinders(p.Args)
	case *core.TuplePattern:
		return corePatternListBinders(p.Elements)
	case *core.ListPattern:
		out := corePatternListBinders(p.Elements)
		if p.Tail != nil {
			out = append(out, corePatternBinders(*p.Tail)...)
		}
		return out
	case *core.RecordPattern:
		var out []string
		for _, fp := range p.Fields {
			out = append(out, corePatternBinders(fp)...)
		}
		return out
	default: // *core.LitPattern, *core.WildcardPattern, nil
		return nil
	}
}

func corePatternListBinders(pats []core.CorePattern) []string {
	var out []string
	for _, sub := range pats {
		out = append(out, corePatternBinders(sub)...)
	}
	return out
}
