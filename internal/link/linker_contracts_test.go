package link

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// TST-RC1-04
func TestLinkerCollectsContractAndContainerDictRefs(t *testing.T) {
	mkMissingRef := func(typeName string) *core.DictRef {
		return &core.DictRef{
			CoreNode:  core.CoreNode{NodeID: 1},
			ClassName: "Eq",
			TypeName:  typeName,
		}
	}

	cases := []struct {
		name     string
		prog     *core.Program
		wantType string
	}{
		{
			name: "contract requires Expr",
			prog: &core.Program{
				Decls: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(1)}},
				Meta: map[string]*core.DeclMeta{
					"checkFn": {
						Name: "checkFn",
						Contracts: []*core.Contract{
							{Kind: core.RequiresKind, Expr: mkMissingRef("MissingContractEq")},
						},
					},
				},
			},
			wantType: "MissingContractEq",
		},
		{
			name: "match guard",
			prog: &core.Program{
				Decls: []core.CoreExpr{
					&core.Match{
						Scrutinee: &core.Lit{Kind: core.IntLit, Value: int64(1)},
						Arms: []core.MatchArm{
							{
								Pattern: &core.WildcardPattern{},
								Guard:   mkMissingRef("MissingGuardEq"),
								Body:    &core.Lit{Kind: core.BoolLit, Value: true},
							},
						},
					},
				},
			},
			wantType: "MissingGuardEq",
		},
		{
			name: "tuple element",
			prog: &core.Program{
				Decls: []core.CoreExpr{
					&core.Tuple{Elements: []core.CoreExpr{mkMissingRef("MissingTupleEq")}},
				},
			},
			wantType: "MissingTupleEq",
		},
		{
			name: "array element",
			prog: &core.Program{
				Decls: []core.CoreExpr{
					&core.Array{Elements: []core.CoreExpr{mkMissingRef("MissingArrayEq")}},
				},
			},
			wantType: "MissingArrayEq",
		},
		{
			name: "record update",
			prog: &core.Program{
				Decls: []core.CoreExpr{
					&core.RecordUpdate{
						Base:    &core.Var{Name: "r"},
						Updates: map[string]core.CoreExpr{"x": mkMissingRef("MissingRecUpdateEq")},
					},
				},
			},
			wantType: "MissingRecUpdateEq",
		},
		{
			name: "intrinsic arg",
			prog: &core.Program{
				Decls: []core.CoreExpr{
					&core.Intrinsic{
						Op:   core.OpNot,
						Args: []core.CoreExpr{mkMissingRef("MissingIntrinsicEq")},
					},
				},
			},
			wantType: "MissingIntrinsicEq",
		},
		{
			name: "forall body",
			prog: &core.Program{
				Decls: []core.CoreExpr{
					&core.Forall{
						Var:  "i",
						Lo:   &core.Lit{Kind: core.IntLit, Value: int64(0)},
						Hi:   &core.Lit{Kind: core.IntLit, Value: int64(5)},
						Body: mkMissingRef("MissingForallEq"),
					},
				},
			},
			wantType: "MissingForallEq",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg := types.NewDictionaryRegistry()
			linker := NewLinkerWithRegistry(reg)
			_, err := linker.LinkProgram(tc.prog, LinkOptions{Namespace: "prelude"})
			if err == nil {
				t.Fatalf("expected unresolved dictionary error for %s, got nil", tc.wantType)
			}
			if !strings.Contains(err.Error(), tc.wantType) {
				t.Fatalf("expected error mentioning %q, got: %v", tc.wantType, err)
			}
		})
	}
}

// TST-RC1-04
func TestLinkerCoversEveryCoreExprSourceType(t *testing.T) {
	pkgs, err := parser.ParseDir(token.NewFileSet(), "../core", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse ../core: %v", err)
	}
	wantTypes := map[string]bool{}
	for _, f := range pkgs["core"].Files {
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "coreExpr" || fn.Recv == nil {
				continue
			}
			if star, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
				if id, ok := star.X.(*ast.Ident); ok {
					wantTypes[id.Name] = true
				}
			}
		}
	}

	f, err := parser.ParseFile(token.NewFileSet(), "linker.go", nil, 0)
	if err != nil {
		t.Fatalf("parse linker.go: %v", err)
	}
	handled := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSwitchStmt)
		if !ok {
			return true
		}
		for _, stmt := range ts.Body.List {
			cc, ok := stmt.(*ast.CaseClause)
			if !ok {
				continue
			}
			for _, expr := range cc.List {
				if star, ok := expr.(*ast.StarExpr); ok {
					if sel, ok := star.X.(*ast.SelectorExpr); ok {
						if pkgIdent, ok := sel.X.(*ast.Ident); ok && pkgIdent.Name == "core" {
							handled[sel.Sel.Name] = true
						}
					}
				}
			}
		}
		return true
	})

	for name := range wantTypes {
		if !handled[name] {
			t.Errorf("CoreExpr variant *core.%s is not handled explicitly in linker.go", name)
		}
	}
}
