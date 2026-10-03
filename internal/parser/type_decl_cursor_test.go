package parser

import (
	"fmt"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// Regression tests for #1466 (M-PARSER-NULLARY-SINGLE-CTOR-CURSOR).
//
// A single nullary constructor with a leading pipe (`type J = | Idle`) and no
// deriving clause over-advanced the cursor past its own last token. ParseFile
// advances one token between declarations, so the NEXT declaration lost its
// first token: `export type W` parsed as non-exported (IMP010 at import),
// `pure func` parsed as non-pure, and `func` produced cascading parse errors.

func parseNoErrors(t *testing.T, input string) *ast.File {
	t.Helper()
	p := New(lexer.New(input, "test://unit"))
	file := p.ParseFile()
	for _, e := range p.Errors() {
		t.Errorf("parse error: %v", e)
	}
	if len(p.Errors()) > 0 {
		t.Fatalf("unexpected parse errors")
	}
	return file
}

func findTypeDecl(file *ast.File, name string) *ast.TypeDecl {
	for _, d := range file.Decls {
		if td, ok := d.(*ast.TypeDecl); ok && td.Name == name {
			return td
		}
	}
	return nil
}

func findFuncDecl(file *ast.File, name string) *ast.FuncDecl {
	for _, d := range file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name == name {
			return fd
		}
	}
	return nil
}

func TestNullaryLeadingPipeSingleCtor_FollowerFlagsPreserved(t *testing.T) {
	tests := []struct {
		name       string
		follower   string
		typeName   string // set when the follower is a type decl
		funcName   string // set when the follower is a func decl
		wantExport bool
		wantPure   bool
	}{
		{name: "export_type", follower: "export type W = { j: J, n: int }", typeName: "W", wantExport: true},
		{name: "export_pure_func", follower: "export pure func mk(n: int) -> int = n", funcName: "mk", wantExport: true, wantPure: true},
		{name: "pure_func", follower: "pure func mk(n: int) -> int = n", funcName: "mk", wantPure: true},
		{name: "plain_func", follower: "func mk() -> int = 1", funcName: "mk"},
		{name: "export_func", follower: "export func mk() -> int = 1", funcName: "mk", wantExport: true},
	}
	for li, lead := range []string{"export type J = | Idle", "type J = | Idle", "export type J =\n  | Idle"} {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("lead%d/%s", li, tt.name), func(t *testing.T) {
				input := "module test\n\n" + lead + "\n" + tt.follower + "\n"
				file := parseNoErrors(t, input)

				j := findTypeDecl(file, "J")
				if j == nil {
					t.Fatalf("type J not found")
				}
				alg, ok := j.Definition.(*ast.AlgebraicType)
				if !ok || len(alg.Constructors) != 1 || alg.Constructors[0].Name != "Idle" {
					t.Fatalf("type J: want single-constructor ADT Idle, got %#v", j.Definition)
				}

				if tt.typeName != "" {
					td := findTypeDecl(file, tt.typeName)
					if td == nil {
						t.Fatalf("type %s not found", tt.typeName)
					}
					if td.Exported != tt.wantExport {
						t.Errorf("type %s: Exported=%v, want %v", tt.typeName, td.Exported, tt.wantExport)
					}
				}
				if tt.funcName != "" {
					fd := findFuncDecl(file, tt.funcName)
					if fd == nil {
						t.Fatalf("func %s not found", tt.funcName)
					}
					if fd.IsExport != tt.wantExport {
						t.Errorf("func %s: IsExport=%v, want %v", tt.funcName, fd.IsExport, tt.wantExport)
					}
					if fd.IsPure != tt.wantPure {
						t.Errorf("func %s: IsPure=%v, want %v", tt.funcName, fd.IsPure, tt.wantPure)
					}
				}
			})
		}
	}
}

// The other type-body shapes must keep their current AST and cursor behaviour.
func TestTypeBodyShapes_FollowerExportPreserved(t *testing.T) {
	tests := []struct {
		name      string
		decl      string
		wantCtors []string // nil => TypeAlias
		wantDeriv int
	}{
		{name: "leading_pipe_multi", decl: "export type J = | Idle | Busy", wantCtors: []string{"Idle", "Busy"}},
		{name: "leading_pipe_deriving", decl: "export type J = | Idle deriving (Eq)", wantCtors: []string{"Idle"}, wantDeriv: 1},
		{name: "leading_pipe_multi_deriving", decl: "export type J = | Idle | Busy deriving (Eq)", wantCtors: []string{"Idle", "Busy"}, wantDeriv: 1},
		{name: "leading_pipe_fields", decl: "export type J = | Wrap(int)", wantCtors: []string{"Wrap"}},
		{name: "no_pipe_multi", decl: "export type J = Idle | Busy", wantCtors: []string{"Idle", "Busy"}},
		{name: "no_pipe_alias", decl: "export type J = Idle"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "module test\n\ntype Idle = int\n" + tt.decl + "\nexport type W = { n: int }\n"
			file := parseNoErrors(t, input)
			j := findTypeDecl(file, "J")
			if j == nil {
				t.Fatalf("type J not found")
			}
			if len(j.Deriving) != tt.wantDeriv {
				t.Errorf("type J: %d deriving entries, want %d", len(j.Deriving), tt.wantDeriv)
			}
			if tt.wantCtors == nil {
				if _, ok := j.Definition.(*ast.TypeAlias); !ok {
					t.Errorf("type J: want TypeAlias, got %T", j.Definition)
				}
			} else {
				alg, ok := j.Definition.(*ast.AlgebraicType)
				if !ok {
					t.Fatalf("type J: want AlgebraicType, got %T", j.Definition)
				}
				if len(alg.Constructors) != len(tt.wantCtors) {
					t.Fatalf("type J: %d ctors, want %d", len(alg.Constructors), len(tt.wantCtors))
				}
				for i, c := range alg.Constructors {
					if c.Name != tt.wantCtors[i] {
						t.Errorf("ctor %d: %s, want %s", i, c.Name, tt.wantCtors[i])
					}
				}
			}
			w := findTypeDecl(file, "W")
			if w == nil || !w.Exported {
				t.Errorf("type W after %q: want exported, got %#v", tt.decl, w)
			}
		})
	}
}
