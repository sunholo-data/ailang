package elaborate

import (
	"fmt"
	"path"
	"strings"
	"unicode"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/errors"
)

// importKind classifies a selectively imported name by the namespace it binds.
type importKind int

const (
	importValue importKind = iota // an exported func: binds a bare value name
	importCtor                    // an exported constructor: binds a bare constructor name
)

// importBinding is one bare name a selective import brings into module scope.
type importBinding struct {
	name string // the name bound here: the alias if present, else the export
	orig string // the exported name in the source module
	path string // the source module path as written
	mod  string // the module alias of `import M as L (x)`, or ""
	pos  ast.Pos
	kind importKind
}

// localBinding is one module-level definition that binds a bare name.
type localBinding struct {
	what string // "func", "let" or "constructor of type T"
	pos  ast.Pos
}

// checkImportCollisions enforces MOD015 (#1467, ruling 2026-10-02): a bare name
// bound by an explicit selective import may not also be bound by a module-level
// func, let or constructor — an ambiguous name, rejected at the definition as in
// Rust E0255 (Haskell/Elm report the same "ambiguous occurrence"). Before this
// the import silently won and the program ran the wrong function with no
// diagnostic.
//
// Only names an import binds bare take part: `import M (x)`, `import M (y as x)`
// and the selective list of `import M as L (x)`. A bare `import M as L` is
// qualified-only and binds no bare names; AILANG has no wildcard import
// (`import M` alone is a parse error), so there is nothing a local could shadow
// silently. Lexical binders (parameters, let, match arms) shadowing an import
// stay legal, and types keep their own rule (M-TYPE-NAME-SHADOW: a local type
// declaration wins over an imported type of the same name). Two imports binding
// the same bare name to different exports are NOT checked here (see the design
// doc's implementation notes).
func checkImportCollisions(file *ast.File, binds []importBinding, lets []*ModuleLet, funcs []*FuncSig) error {
	values := make(map[string]localBinding)
	for _, f := range funcs {
		if f.FuncDecl != nil {
			values[f.Name] = localBinding{"func", f.FuncDecl.Position()}
		}
	}
	for _, l := range lets {
		if _, dup := values[l.Name]; !dup { // let-vs-func is MOD007, reported first
			values[l.Name] = localBinding{"let", l.Pos}
		}
	}
	ctors := localConstructors(file)

	for _, b := range binds {
		locals := values
		if b.kind == importCtor {
			locals = ctors
		}
		if local, clash := locals[b.name]; clash {
			msg := fmt.Sprintf("'%s' is both imported (%s at %s) and defined in this module "+
				"as a %s (at %s) — an imported name and a module-level definition may not share a name.",
				b.name, importText(b), sitePos(file, b.pos), local.what, sitePos(file, local.pos))
			return mod015(file, local.pos, b.name, msg,
				collisionFix(b))
		}
	}
	return nil
}

// importCollisionError renders MOD015 for humans ("Error MOD015: … Fix: …")
// and unwraps to a structured report, so `check --json` and the LSP place it
// at the definition the user acts on.
type importCollisionError struct {
	text string
	rep  error
}

func (e *importCollisionError) Error() string { return e.text }
func (e *importCollisionError) Unwrap() error { return e.rep }

func mod015(file *ast.File, at ast.Pos, name, msg, fix string) error {
	if at.File == "" {
		at.File = file.Path
	}
	rep := &errors.Report{
		Schema:  "ailang.error/v1",
		Code:    errors.MOD015,
		Phase:   "elaborate",
		Message: msg,
		Span:    &ast.Span{Start: at, End: at},
		Data:    map[string]any{"name": name},
		Fix:     &errors.Fix{Suggestion: fix, Confidence: 0.9},
	}
	return &importCollisionError{
		text: fmt.Sprintf("Error %s: %s\n  Fix: %s", errors.MOD015, msg, fix),
		rep:  errors.WrapReport(rep),
	}
}

// localConstructors returns the module's own ADT constructors by name.
func localConstructors(file *ast.File) map[string]localBinding {
	ctors := make(map[string]localBinding)
	for _, decl := range file.Decls {
		td, ok := decl.(*ast.TypeDecl)
		if !ok {
			continue
		}
		if alg, ok := td.Definition.(*ast.AlgebraicType); ok {
			for _, c := range alg.Constructors {
				// The type declaration's position: ast.Constructor.Pos is the
				// parser's cursor AFTER the constructor, not its start.
				ctors[c.Name] = localBinding{"constructor of type " + td.Name, td.Pos}
			}
		}
	}
	return ctors
}

// importText renders the import that binds b, e.g. `import std/list (map)` or
// `import ./a (tock as tick)`.
func importText(b importBinding) string {
	if b.orig != b.name {
		return fmt.Sprintf("import %s (%s as %s)", importHead(b), b.orig, b.name)
	}
	return fmt.Sprintf("import %s (%s)", importHead(b), b.name)
}

// importHead is the import's module part: `std/list` or `std/list as L`.
func importHead(b importBinding) string {
	if b.mod != "" {
		return b.path + " as " + b.mod
	}
	return b.path
}

// collisionFix is the MOD015 fix line. With a module alias the qualified form
// needs no list entry at all.
func collisionFix(b importBinding) string {
	fix := "rename the local definition, or alias the import: " + aliasSuggestion(b)
	if b.mod != "" {
		fix += fmt.Sprintf(" (or drop '%s' from the import list and write %s.%s)", b.orig, b.mod, b.orig)
	}
	return fix
}

// aliasSuggestion proposes an alias prefixed with the module's last path
// segment: `import std/list (map as listMap)`, `import ./a (Red as ARed)`.
func aliasSuggestion(b importBinding) string {
	prefix := identPrefix(path.Base(strings.TrimPrefix(b.path, "./")))
	alias := prefix + upperFirst(b.name)
	if b.kind == importCtor {
		alias = upperFirst(alias)
	}
	return fmt.Sprintf("import %s (%s as %s)", importHead(b), b.orig, alias)
}

// identPrefix keeps the identifier characters of a path segment ("dep2",
// "my_mod"); a segment with none falls back to "imported".
func identPrefix(seg string) string {
	var sb strings.Builder
	for _, r := range seg {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			sb.WriteRune(r)
		}
	}
	s := sb.String()
	if s == "" || !unicode.IsLetter([]rune(s)[0]) {
		return "imported"
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// sitePos renders a position as file:line:col (line:col when the file is unknown).
func sitePos(file *ast.File, p ast.Pos) string {
	name := p.File
	if name == "" {
		name = file.Path
	}
	if name == "" {
		return posString(p)
	}
	return fmt.Sprintf("%s:%d:%d", name, p.Line, p.Column)
}

// importIsConstructor reports whether sym is a constructor exported by the
// module at modPath (GetExport returns nil for both types and constructors).
func (e *Elaborator) importIsConstructor(modPath, sym string) bool {
	mod, err := e.moduleLoader.Load(modPath)
	if err != nil || mod == nil {
		return false
	}
	_, ok := mod.Constructors[sym]
	return ok
}
