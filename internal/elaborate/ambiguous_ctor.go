package elaborate

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// MarkAmbiguousConstructor records that the bare constructor name is offered
// by two or more imported modules (M-CTOR-AUTO) and imported by name from
// none (M-V1-NOMINAL-TYPE-IDENTITY). A pattern using it leaves its ADT open
// for the typechecker, which resolves it by the scrutinee's type; an
// expression using it is TC_AMBIGUOUS_CTOR_001. A local declaration of the
// name clears the mark (the local wins).
func (e *Elaborator) MarkAmbiguousConstructor(name string, candidates []types.TypeRef) {
	if e.ambiguousCtors == nil {
		e.ambiguousCtors = make(map[string][]types.TypeRef)
	}
	e.ambiguousCtors[name] = candidates
}

// ambiguousCtorError is TC_AMBIGUOUS_CTOR_001 for a constructor expression.
func (e *Elaborator) ambiguousCtorError(name string, pos ast.Pos) error {
	cands := e.ambiguousCtors[name]
	types.ObserveAmbiguity(types.AmbiguityEvent{Kind: "ctor-expr", Module: e.ModuleID(), Name: name, Candidates: cands})
	if !types.AmbiguityIsError() {
		return nil
	}
	return types.NewAmbiguousCtorError(name, cands, "at "+e.sourcePos(pos))
}

// sourcePos renders pos as file:line:col, naming the module when the position
// carries no file.
func (e *Elaborator) sourcePos(pos ast.Pos) string {
	if pos.File != "" {
		return fmt.Sprintf("%s:%s", pos.File, posString(pos))
	}
	return fmt.Sprintf("%s:%s", e.ModuleID(), posString(pos))
}

// ImportSite is one explicit import that binds a bare name.
type ImportSite struct {
	Path string // the module path as written
	Sym  string // the exported name it imports
	Pos  ast.Pos
}

// ImportImportCollision is MOD015 for two explicit selective imports that bind
// one bare type or constructor name to different modules' declarations
// (M-V1-NOMINAL-TYPE-IDENTITY; D6 of the design doc). The import order no
// longer decides which one the name means.
func ImportImportCollision(file *ast.File, kind, name string, first, second ImportSite) error {
	msg := fmt.Sprintf("%s '%s' is imported twice: from %s (at %s) and from %s (at %s) — two imports may not bind one bare %s name to different %ss.",
		kind, name, first.Path, sitePos(file, first.Pos), second.Path, sitePos(file, second.Pos), kind, kind)
	alias := identPrefix(lastPathSeg(second.Path))
	alias = strings.ToUpper(alias[:1]) + alias[1:] + second.Sym
	fix := fmt.Sprintf("alias one of them: import %s (%s as %s)", second.Path, second.Sym, alias)
	return mod015(file, second.Pos, name, msg, fix)
}

func lastPathSeg(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}
