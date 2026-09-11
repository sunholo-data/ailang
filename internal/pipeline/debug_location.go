package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

// DebugLocationInjector is the "location auto-injected by the compiler" that
// effects.LogEntry.Location has promised since v0.5.1. std/debug's wrappers
// cannot know their caller, so they pass the placeholder "unknown":
//
//	export func log(msg: string) -> () ! {Debug} = _debug_log(msg, "unknown")
//
// This pass runs on lowered Core, after type checking, and rewrites each
// direct call of a wrapper into a call of the underlying builtin with the
// CALL SITE's own position appended:
//
//	std/debug.log(msg)         → $builtin._debug_log(msg, "file.ail:42")
//	std/debug.check(cond, msg) → $builtin._debug_check(cond, msg, "file.ail:42")
//
// Only App nodes whose callee is exactly a std/debug wrapper at the wrapper's
// declared arity are touched. A wrapper used as a first-class value keeps
// going through std/debug (and reports "unknown"), which is the honest answer
// for a call whose site is not syntactically present.
//
// Runs unconditionally; in release mode DebugEraser follows it and erases the
// resulting builtin calls, so the two passes see one shape.
type DebugLocationInjector struct{}

// debugWrappers maps a std/debug export to the builtin it wraps and the
// number of user-visible arguments the wrapper takes.
var debugWrappers = map[string]struct {
	builtin string
	arity   int
}{
	"log":   {"_debug_log", 1},
	"check": {"_debug_check", 2},
}

// Inject returns prog with call-site locations injected into every direct
// std/debug wrapper call.
func (in *DebugLocationInjector) Inject(prog *core.Program) *core.Program {
	out := &core.Program{
		Decls: make([]core.CoreExpr, len(prog.Decls)),
		Meta:  prog.Meta,
		Flags: prog.Flags,
	}
	for i, decl := range prog.Decls {
		out.Decls[i] = in.inject(decl)
	}
	return out
}

func (in *DebugLocationInjector) inject(expr core.CoreExpr) core.CoreExpr {
	app, ok := expr.(*core.App)
	if !ok {
		return mapCoreChildren(expr, in.inject)
	}
	fn, ok := app.Func.(*core.VarGlobal)
	if !ok || fn.Ref.Module != "std/debug" {
		return mapCoreChildren(expr, in.inject)
	}
	w, ok := debugWrappers[fn.Ref.Name]
	if !ok || len(app.Args) != w.arity {
		return mapCoreChildren(expr, in.inject)
	}

	args := make([]core.CoreExpr, 0, w.arity+1)
	for _, a := range app.Args {
		args = append(args, in.inject(a))
	}
	args = append(args, &core.Lit{
		CoreNode: app.CoreNode,
		Kind:     core.StringLit,
		Value:    formatDebugLocation(app),
	})
	return &core.App{
		CoreNode: app.CoreNode,
		Func: &core.VarGlobal{
			CoreNode: fn.CoreNode,
			Ref:      core.GlobalRef{Module: "$builtin", Name: w.builtin},
		},
		Args: args,
	}
}

// formatDebugLocation renders the App's source position as "file.ail:42",
// preferring the surface position; "unknown" when neither span has a line.
func formatDebugLocation(app *core.App) string {
	pos := app.OriginalSpan()
	if pos.Line == 0 {
		pos = app.Span()
	}
	return formatPos(pos)
}

func formatPos(pos ast.Pos) string {
	if pos.Line == 0 {
		return "unknown"
	}
	if pos.File == "" {
		return fmt.Sprintf("<unknown>:%d", pos.Line)
	}
	return fmt.Sprintf("%s:%d", displayPath(pos.File), pos.Line)
}

// displayPath shortens an absolute source path to be relative to the working
// directory when the file lives under it (serve-api loads modules by absolute
// path; a Cloud Run log line wants "api/orders.ail:12", not "/app/api/…").
// Paths outside the working directory, and relative paths, are returned
// as given. Always slash-separated so the location is the same on every OS.
func displayPath(file string) string {
	if filepath.IsAbs(file) {
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, file); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				file = rel
			}
		}
	}
	return filepath.ToSlash(file)
}
