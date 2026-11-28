package pipeline

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
func TestOpLowererLowersAllContainerAndQuantifierVariants(t *testing.T) {
	coreTI := types.NewCoreTypeInfo()
	nextID := uint64(100)
	mkAdd := func() *core.Intrinsic {
		xID := nextID
		nextID++
		litID := nextID
		nextID++
		opID := nextID
		nextID++
		coreTI.Set(xID, types.TInt)
		coreTI.Set(litID, types.TInt)
		coreTI.Set(opID, types.TInt)
		return &core.Intrinsic{
			CoreNode: core.CoreNode{NodeID: opID},
			Op:       core.OpAdd,
			Args: []core.CoreExpr{
				&core.Var{CoreNode: core.CoreNode{NodeID: xID}, Name: "x"},
				&core.Lit{CoreNode: core.CoreNode{NodeID: litID}, Kind: core.IntLit, Value: int64(1)},
			},
		}
	}
	mkLambda := func() *core.Lambda {
		id := nextID
		nextID++
		return &core.Lambda{
			CoreNode: core.CoreNode{NodeID: id},
			Params:   []string{"x"},
			Body:     mkAdd(),
		}
	}

	tupleExpr := &core.Tuple{
		CoreNode: core.CoreNode{NodeID: 1},
		Elements: []core.CoreExpr{
			&core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.IntLit, Value: int64(1)},
			mkLambda(),
		},
	}
	arrayExpr := &core.Array{
		CoreNode: core.CoreNode{NodeID: 3},
		Elements: []core.CoreExpr{mkLambda()},
	}
	recUpdateExpr := &core.RecordUpdate{
		CoreNode: core.CoreNode{NodeID: 4},
		Base: &core.Record{
			CoreNode: core.CoreNode{NodeID: 5},
			Fields:   map[string]core.CoreExpr{"f": mkLambda()},
		},
		Updates: map[string]core.CoreExpr{"f": mkLambda()},
	}
	dictAbsExpr := &core.DictAbs{
		CoreNode: core.CoreNode{NodeID: 6},
		Params:   []core.DictParam{{Name: "dict_Num_Int", ClassName: "Num", Type: "Int"}},
		Body:     mkLambda(),
	}
	dictAppExpr := &core.DictApp{
		CoreNode: core.CoreNode{NodeID: 7},
		Dict:     &core.DictRef{CoreNode: core.CoreNode{NodeID: 8}, ClassName: "Num", TypeName: "Int"},
		Method:   "add",
		Args:     []core.CoreExpr{mkAdd(), mkAdd()},
	}

	iID := uint64(50)
	zeroID := uint64(51)
	cmpID := uint64(52)
	coreTI.Set(iID, types.TInt)
	coreTI.Set(zeroID, types.TInt)
	coreTI.Set(cmpID, types.TBool)
	forallExpr := &core.Forall{
		CoreNode: core.CoreNode{NodeID: 9},
		Var:      "i",
		Lo:       mkAdd(),
		Hi:       mkAdd(),
		Body: &core.Intrinsic{
			CoreNode: core.CoreNode{NodeID: cmpID},
			Op:       core.OpGe,
			Args: []core.CoreExpr{
				&core.Var{CoreNode: core.CoreNode{NodeID: iID}, Name: "i"},
				&core.Lit{CoreNode: core.CoreNode{NodeID: zeroID}, Kind: core.IntLit, Value: int64(0)},
			},
		},
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{
			tupleExpr,
			arrayExpr,
			recUpdateExpr,
			dictAbsExpr,
			dictAppExpr,
			forallExpr,
		},
	}

	lowerer := NewOpLowerer(types.NewTypeEnv(), coreTI)
	lowered, err := lowerer.Lower(prog)
	if err != nil {
		t.Fatalf("Lower failed: %v", err)
	}

	for idx, decl := range lowered.Decls {
		leftover := findUnloweredIntrinsics(decl)
		if len(leftover) > 0 {
			t.Errorf("decl[%d] (%T) still has %d un-lowered Intrinsic/BinOp/UnOp nodes: %v", idx, decl, len(leftover), leftover[0])
		}
	}
}

// TST-RC1-04
func TestOpLowererCoversEveryCoreExprSourceType(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, "../core", func(fi os.FileInfo) bool {
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
	if len(wantTypes) < 21 {
		t.Fatalf("expected >= 21 CoreExpr variants in ../core, found %d", len(wantTypes))
	}

	pipePkgs, err := parser.ParseDir(token.NewFileSet(), ".", func(fi os.FileInfo) bool {
		return strings.HasPrefix(fi.Name(), "op_lowering") && !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse op_lowering*.go: %v", err)
	}
	handled := map[string]bool{}
	for _, f := range pipePkgs["pipeline"].Files {
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
	}

	for name := range wantTypes {
		if !handled[name] {
			t.Errorf("CoreExpr variant *core.%s is not handled explicitly in op_lowering*.go", name)
		}
	}
}

func findUnloweredIntrinsics(expr core.CoreExpr) []core.CoreExpr {
	var out []core.CoreExpr
	var walk func(core.CoreExpr)
	walk = func(e core.CoreExpr) {
		if e == nil {
			return
		}
		switch v := e.(type) {
		case *core.Intrinsic, *core.BinOp, *core.UnOp:
			out = append(out, v)
		}
		switch v := e.(type) {
		case *core.Lambda:
			walk(v.Body)
		case *core.Let:
			walk(v.Value)
			walk(v.Body)
		case *core.LetRec:
			for _, b := range v.Bindings {
				walk(b.Value)
			}
			walk(v.Body)
		case *core.App:
			walk(v.Func)
			for _, a := range v.Args {
				walk(a)
			}
		case *core.If:
			walk(v.Cond)
			walk(v.Then)
			walk(v.Else)
		case *core.Match:
			walk(v.Scrutinee)
			for _, arm := range v.Arms {
				walk(arm.Guard)
				walk(arm.Body)
			}
		case *core.Record:
			for _, f := range v.Fields {
				walk(f)
			}
		case *core.RecordAccess:
			walk(v.Record)
		case *core.RecordUpdate:
			walk(v.Base)
			for _, f := range v.Updates {
				walk(f)
			}
		case *core.List:
			for _, el := range v.Elements {
				walk(el)
			}
		case *core.Array:
			for _, el := range v.Elements {
				walk(el)
			}
		case *core.Tuple:
			for _, el := range v.Elements {
				walk(el)
			}
		case *core.DictAbs:
			walk(v.Body)
		case *core.DictApp:
			walk(v.Dict)
			for _, a := range v.Args {
				walk(a)
			}
		case *core.Forall:
			walk(v.Lo)
			walk(v.Hi)
			walk(v.Body)
		case *core.Intrinsic:
			for _, a := range v.Args {
				walk(a)
			}
		case *core.BinOp:
			walk(v.Left)
			walk(v.Right)
		case *core.UnOp:
			walk(v.Operand)
		}
	}
	walk(expr)
	return out
}
