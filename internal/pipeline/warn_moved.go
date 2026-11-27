package pipeline

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/elaborate"
)

// The "moved" variant of TC_DEPRECATED_001 (M-PKG-PARALLEL-AGENTS C1).
//
// A facade marks a re-export line with the module's new path:
//
//	export import pkg/acme/core/types/document (Block, Heading) @deprecated("pkg/acme/core/types/document")
//
// Every name a module imports through that line warns at the name in its
// import list, whether the name is a function, a type or a constructor. The
// warning is import-based rather than use-based: the loader rewrote the import
// to the name's origin and recorded the facade line it went through
// (ImportDecl.ViaModule / ViaMovedTo), and types and constructors have no
// Core use to key on. The loader re-parses and rewrites on every run, so the
// warning survives a warm cache.

// MovedImportWarning is a non-blocking warning: a name is imported through a
// facade line that says the module moved.
type MovedImportWarning struct {
	Location ast.Pos // the name in the import list
	Facade   string  // the module imported through
	Name     string  // the name as the facade's source exports it
	MovedTo  string  // the new path
}

var _ elaborate.Warning = (*MovedImportWarning)(nil)

// Position exposes the name's 1-indexed line/col.
func (w *MovedImportWarning) Position() (line, col int) {
	return w.Location.Line, w.Location.Column
}

// Code is TC_DEPRECATED_001: a moved name is a deprecation whose replacement
// is a module path.
func (w *MovedImportWarning) Code() string { return CodeDeprecated }

// WarningFile names the file holding the import.
func (w *MovedImportWarning) WarningFile() string { return w.Location.File }

// Message is the warning without its location and fix.
func (w *MovedImportWarning) Message() string {
	return fmt.Sprintf("%s.%s has moved to %s", w.Facade, w.Name, w.MovedTo)
}

// Suggestion is the one-line fix.
func (w *MovedImportWarning) Suggestion() string {
	return fmt.Sprintf("import %s from %s", w.Name, w.MovedTo)
}

// String renders the warning for stderr.
func (w *MovedImportWarning) String() string {
	return fmt.Sprintf("%s at %s\n  %s.\n  Fix: %s", CodeDeprecated, w.Location.String(), w.Message(), w.Suggestion())
}

// DetectMovedImports reports every name imported through a moved re-export
// line, in the project's own modules (an installed dependency's imports are
// its own to fix, as for DetectDeprecatedUses).
func DetectMovedImports(units map[string]*CompileUnit) []elaborate.Warning {
	pkgName := ""
	if currentPackageManifest != nil {
		pkgName = currentPackageManifest.Package.Name
	}
	ids := make([]string, 0, len(units))
	for id := range units {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var out []elaborate.Warning
	for _, id := range ids {
		u := units[id]
		if u == nil || u.Surface == nil {
			continue
		}
		if strings.HasPrefix(id, "pkg/") && !isOwnPackageModule(id, pkgName) {
			continue
		}
		for _, imp := range u.Surface.Imports {
			if imp == nil || imp.ViaMovedTo == "" {
				continue
			}
			for i, name := range imp.Symbols {
				pos := imp.Pos
				if i < len(imp.SymbolPos) {
					pos = imp.SymbolPos[i]
				}
				if pos.File == "" {
					pos.File = u.Surface.Path
				}
				out = append(out, &MovedImportWarning{Location: pos, Facade: imp.ViaModule, Name: name, MovedTo: imp.ViaMovedTo})
			}
		}
	}
	return out
}
