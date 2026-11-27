package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

// `export import M (names) [@deprecated("path")]` in the import header
// (M-PKG-PARALLEL-AGENTS C1).
func TestReexportImport_Parses(t *testing.T) {
	input := `module lib/facade

import std/io (println)
export import lib/color (Color, Red as Crimson)
export import pkg/demo/core/types/document (Block, Heading) @deprecated("pkg/demo/core/types/document")
import std/list (map)

export func hello() -> string = "hi"
`
	p := New(lexer.New(input, "facade.ail"))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	if len(file.Imports) != 4 {
		t.Fatalf("imports = %d, want 4", len(file.Imports))
	}
	plain, color, doc, list := file.Imports[0], file.Imports[1], file.Imports[2], file.Imports[3]
	if plain.Reexport || list.Reexport {
		t.Error("a plain import must not be a re-export")
	}
	if !color.Reexport || color.Path != "lib/color" || strings.Join(color.Symbols, ",") != "Color,Red" {
		t.Errorf("color = %+v", color)
	}
	if color.SymbolAliases["Red"] != "Crimson" || color.MovedTo != "" {
		t.Errorf("color alias/moved = %v %q", color.SymbolAliases, color.MovedTo)
	}
	if color.Pos.Line != 4 || color.Pos.Column != 1 {
		t.Errorf("a re-export's position is its 'export' token, got %v", color.Pos)
	}
	if !doc.Reexport || !doc.IsPackage || doc.MovedTo != "pkg/demo/core/types/document" {
		t.Errorf("doc = %+v", doc)
	}
	if len(file.Funcs) != 1 || file.Funcs[0].Name != "hello" {
		t.Errorf("the declarations after the header must still parse: %d funcs", len(file.Funcs))
	}
}

func TestReexportImport_Errors(t *testing.T) {
	cases := map[string]struct{ src, code string }{
		"no symbol list":          {`export import lib/color`, "PAR_REEXPORT_FORM"},
		"module alias":            {`export import lib/color as C`, "PAR_REEXPORT_FORM"},
		"alias and symbols":       {`export import lib/color as C (Red)`, "PAR_REEXPORT_FORM"},
		"other annotation":        {`export import lib/color (Red) @verify(depth: 1)`, "PAR_REEXPORT_ANNOTATION"},
		"empty deprecated path":   {`export import lib/color (Red) @deprecated("")`, "PAR_DEPRECATED_ARG"},
		"annotation before":       {"@deprecated(\"x\")\nexport import lib/color (Red)", "PAR_ATTR_REQUIRES_FUNC"},
		"after a declaration":     {"export func f() -> int = 1\nexport import lib/color (Red)", "PAR_IMPORT_PLACEMENT"},
		"export without follower": {`export 42`, "PAR_EXPORT_REQUIRES_FUNC"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := New(lexer.New("module lib/facade\n\n"+tc.src+"\n", "facade.ail"))
			p.ParseFile()
			var got []string
			for _, e := range p.Errors() {
				got = append(got, e.Error())
			}
			joined := strings.Join(got, "\n")
			if !strings.Contains(joined, tc.code) {
				t.Fatalf("want %s, got:\n%s", tc.code, joined)
			}
		})
	}
}

// Annotations after a rejected `export import` are consumed, so the form error
// is the only one reported.
func TestReexportImport_RejectedFormSkipsTrailingAnnotations(t *testing.T) {
	src := "module lib/facade\n\nexport import lib/color as C @deprecated(\"a\") @deprecated(\"b\")\n\nexport func f() -> int = 1\n"
	p := New(lexer.New(src, "facade.ail"))
	p.ParseFile()
	errs := p.Errors()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "PAR_REEXPORT_FORM") {
		var got []string
		for _, e := range errs {
			got = append(got, e.Error())
		}
		t.Fatalf("want exactly PAR_REEXPORT_FORM, got:\n%s", strings.Join(got, "\n"))
	}
}

// The export-follower message names `import` now that `export import` exists.
func TestExportRequiresFunc_NamesImport(t *testing.T) {
	p := New(lexer.New("module m\n\nexport 42\n", "m.ail"))
	p.ParseFile()
	joined := ""
	for _, e := range p.Errors() {
		joined += e.Error()
	}
	if !strings.Contains(joined, "'import'") {
		t.Errorf("PAR_EXPORT_REQUIRES_FUNC should list 'import': %s", joined)
	}
}

// Conflict Surface: files that use `export` in the positions `export import`
// extends must parse exactly as before (V28).
func TestReexportImport_ConflictSurfaceFixturesStillParse(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, rel := range []string{
		"examples/wasm_friendly_patterns.ail", "examples/progress_bar.ail",
		"std/json.ail", "std/sem.ail", "std/ai.ail", "std/array.ail", "std/bytes.ail",
	} {
		src, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		p := New(lexer.New(string(src), rel))
		file := p.ParseFile()
		if errs := p.Errors(); len(errs) > 0 {
			t.Errorf("%s: %v", rel, errs)
			continue
		}
		for _, imp := range file.Imports {
			if imp.Reexport {
				t.Errorf("%s: import %s parsed as a re-export", rel, imp.Path)
			}
		}
	}
}
