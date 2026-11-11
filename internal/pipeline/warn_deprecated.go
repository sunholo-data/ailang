package pipeline

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
)

// TC_DEPRECATED_001 — a use of a function marked @deprecated("replacement")
// (M-V1-STDLIB-SURFACE-POLISH §6.1 item 2).
//
// The stdlib renames keep their old names as one-line forwarders, e.g.
//
//	@deprecated("headOr")
//	export pure func head_or[a](xs: [a], default: a) -> a = headOr(xs, default)
//
// so 0.x programs keep compiling, while every use names the new spelling.
//
// The pass is keyed on identity, like the split-arg and TC_MATCH_005 passes:
// a use is a Core VarGlobal{Module, Name} whose target declaration carries
// @deprecated. Every VarGlobal is visited, not only call heads, so
// `map(abs_Int, xs)` warns as well as `abs_Int(x)`. An import line is not a
// use, and neither is the definition: a module's own references to its
// top-level functions are not VarGlobals, so the defining module never warns
// about itself.
//
// Warm cache: the deprecation set is read from each module's PARSED source
// (CompileUnit.Surface), which the loader re-parses on every run, and the uses
// from the unit's Core, which a cache hit restores. Nothing is read from the
// cached interface, so the warning cannot be lost on a second run, and adding
// @deprecated to a dependency warns at once even though it leaves the
// dependency's interface digest unchanged.

// CodeDeprecated is the stable diagnostic code of the warning.
const CodeDeprecated = "TC_DEPRECATED_001"

// DeprecatedUseWarning is a non-blocking warning: a deprecated function is
// used.
type DeprecatedUseWarning struct {
	Location    ast.Pos // the use
	Module      string  // module that defines the deprecated function
	Name        string  // its name
	Replacement string  // @deprecated's first argument
	Note        string  // @deprecated's optional second argument
}

var _ elaborate.Warning = (*DeprecatedUseWarning)(nil)

// Position exposes the use's 1-indexed line/col.
func (w *DeprecatedUseWarning) Position() (line, col int) {
	return w.Location.Line, w.Location.Column
}

// Code returns the stable diagnostic code.
func (w *DeprecatedUseWarning) Code() string { return CodeDeprecated }

// WarningFile names the file holding the use (an imported module's file when
// the use is there, not the checked file).
func (w *DeprecatedUseWarning) WarningFile() string { return w.Location.File }

// Message is the warning without its location and fix.
func (w *DeprecatedUseWarning) Message() string {
	return fmt.Sprintf("%s.%s is deprecated; use %s", w.Module, w.Name, w.Replacement)
}

// Suggestion is the one-line fix.
func (w *DeprecatedUseWarning) Suggestion() string {
	s := fmt.Sprintf("replace %s with %s (import it from %s)", w.Name, w.Replacement, w.Module)
	if w.Note != "" {
		s += "; " + w.Note
	}
	return s
}

// String renders the warning for stderr.
func (w *DeprecatedUseWarning) String() string {
	return fmt.Sprintf("%s at %s\n  %s.\n  Fix: %s", CodeDeprecated, w.Location.String(), w.Message(), w.Suggestion())
}

type deprecation struct{ replacement, note string }

// deprecatedIndex collects every exported @deprecated function of the units,
// keyed by its global identity.
func deprecatedIndex(units map[string]*CompileUnit) map[core.GlobalRef]deprecation {
	idx := map[core.GlobalRef]deprecation{}
	for id, u := range units {
		if u == nil || u.Surface == nil {
			continue
		}
		for _, fd := range u.Surface.Funcs {
			if fd == nil || !fd.IsExport {
				continue
			}
			if repl, note, ok := fd.Deprecation(); ok {
				idx[core.GlobalRef{Module: id, Name: fd.Name}] = deprecation{repl, note}
			}
		}
	}
	return idx
}

// DetectDeprecatedUses returns one warning per use site of a deprecated
// function across the compiled units, in module order then source order.
func DetectDeprecatedUses(units map[string]*CompileUnit) []elaborate.Warning {
	idx := deprecatedIndex(units)
	if len(idx) == 0 {
		return nil
	}
	ids := make([]string, 0, len(units))
	for id := range units {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	pkgName := ""
	if currentPackageManifest != nil {
		pkgName = currentPackageManifest.Package.Name
	}
	var out []elaborate.Warning
	for _, id := range ids {
		u := units[id]
		if u == nil || u.Core == nil {
			continue
		}
		// A use inside an installed dependency (pkg/<other package>/...) is
		// that package's to fix, and its own `check` reports it. Reporting it
		// here buried a downstream project under warnings it cannot act on
		// (measured: docparse got 15 sites from sunholo/ailang_parse, repeated
		// in 21 files). The project's own modules and std still report.
		if strings.HasPrefix(id, "pkg/") && !isOwnPackageModule(id, pkgName) {
			continue
		}
		// One warning per use site: a node reachable twice (a shared subtree)
		// must not report twice.
		seen := map[string]bool{}
		var found []*DeprecatedUseWarning
		visit := func(e core.CoreExpr) {
			vg, ok := e.(*core.VarGlobal)
			if !ok || vg.Ref.Module == id {
				return
			}
			d, ok := idx[vg.Ref]
			if !ok {
				return
			}
			pos := vg.OriginalSpan()
			key := fmt.Sprintf("%s|%d|%d|%s|%s", pos.File, pos.Line, pos.Column, vg.Ref.Module, vg.Ref.Name)
			if seen[key] {
				return
			}
			seen[key] = true
			found = append(found, &DeprecatedUseWarning{
				Location: pos, Module: vg.Ref.Module, Name: vg.Ref.Name,
				Replacement: d.replacement, Note: d.note,
			})
		}
		for _, decl := range u.Core.Decls {
			walkEveryCore(decl, visit)
		}
		for _, meta := range u.Core.Meta {
			if meta == nil {
				continue
			}
			for _, c := range meta.Contracts {
				if c != nil {
					walkEveryCore(c.Expr, visit)
				}
			}
		}
		sort.SliceStable(found, func(a, b int) bool {
			pa, pb := found[a].Location, found[b].Location
			if pa.File != pb.File {
				return pa.File < pb.File
			}
			if pa.Line != pb.Line {
				return pa.Line < pb.Line
			}
			return pa.Column < pb.Column
		})
		for _, w := range found {
			out = append(out, w)
		}
	}
	return out
}

// walkEveryCore visits every Core node, including the children walkCore
// treats as atomic in ANF (tuple, array and record-update elements can still
// hold a VarGlobal value such as `(abs_Int, 1)`), and contract quantifiers.
func walkEveryCore(e core.CoreExpr, visit func(core.CoreExpr)) {
	if e == nil {
		return
	}
	visit(e)
	switch n := e.(type) {
	case *core.Let:
		walkEveryCore(n.Value, visit)
		walkEveryCore(n.Body, visit)
	case *core.LetRec:
		for _, b := range n.Bindings {
			walkEveryCore(b.Value, visit)
		}
		walkEveryCore(n.Body, visit)
	case *core.Lambda:
		walkEveryCore(n.Body, visit)
	case *core.App:
		walkEveryCore(n.Func, visit)
		for _, a := range n.Args {
			walkEveryCore(a, visit)
		}
	case *core.If:
		walkEveryCore(n.Cond, visit)
		walkEveryCore(n.Then, visit)
		walkEveryCore(n.Else, visit)
	case *core.Match:
		walkEveryCore(n.Scrutinee, visit)
		for _, arm := range n.Arms {
			walkEveryCore(arm.Guard, visit)
			walkEveryCore(arm.Body, visit)
		}
	case *core.BinOp:
		walkEveryCore(n.Left, visit)
		walkEveryCore(n.Right, visit)
	case *core.UnOp:
		walkEveryCore(n.Operand, visit)
	case *core.Intrinsic:
		for _, a := range n.Args {
			walkEveryCore(a, visit)
		}
	case *core.Record:
		// Map order is irrelevant: findings are sorted by position.
		for _, v := range n.Fields {
			walkEveryCore(v, visit)
		}
	case *core.RecordAccess:
		walkEveryCore(n.Record, visit)
	case *core.RecordUpdate:
		walkEveryCore(n.Base, visit)
		for _, v := range n.Updates {
			walkEveryCore(v, visit)
		}
	case *core.List:
		for _, el := range n.Elements {
			walkEveryCore(el, visit)
		}
	case *core.Array:
		for _, el := range n.Elements {
			walkEveryCore(el, visit)
		}
	case *core.Tuple:
		for _, el := range n.Elements {
			walkEveryCore(el, visit)
		}
	case *core.DictAbs:
		walkEveryCore(n.Body, visit)
	case *core.DictApp:
		walkEveryCore(n.Dict, visit)
		for _, a := range n.Args {
			walkEveryCore(a, visit)
		}
	case *core.Forall:
		walkEveryCore(n.Lo, visit)
		walkEveryCore(n.Hi, visit)
		walkEveryCore(n.Body, visit)
	}
}
