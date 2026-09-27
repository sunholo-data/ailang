package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/types"
)

// Unit tests for the ceiling walker: an authority entry point must be found
// under EVERY Core node kind (an unvisited child would be a silent soundness
// hole), and a missing type must fail loudly.

type walkerFixture struct {
	next  uint64
	coreT types.CoreTypeInfo
}

func newWalkerFixture() *walkerFixture {
	return &walkerFixture{coreT: types.CoreTypeInfo{}}
}

func (f *walkerFixture) node() core.CoreNode {
	f.next++
	f.coreT[f.next] = types.TInt
	return core.CoreNode{NodeID: f.next}
}

// processRef is a reference to std/stream.asyncExecProcess typed with
// {Stream, Process} — the authority the walker must find.
func (f *walkerFixture) processRef() *core.VarGlobal {
	n := f.node()
	f.coreT[n.NodeID] = &types.TFunc2{
		Params: []types.Type{types.TString},
		EffectRow: &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{
			"Stream": types.TUnit, "Process": types.TUnit,
		}},
		Return: types.TUnit,
	}
	return &core.VarGlobal{CoreNode: n, Ref: core.GlobalRef{Module: "std/stream", Name: "asyncExecProcess"}}
}

func (f *walkerFixture) lit() core.CoreExpr {
	return &core.Lit{CoreNode: f.node(), Kind: core.IntLit, Value: 1}
}

func TestCeilingWalker_FindsEntryUnderEveryNodeKind(t *testing.T) {
	wrappers := map[string]func(f *walkerFixture, inner core.CoreExpr) core.CoreExpr{
		"Lambda": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.Lambda{CoreNode: f.node(), Params: []string{"x"}, Body: in}
		},
		"Let value": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.Let{CoreNode: f.node(), Name: "g", Value: in, Body: f.lit()}
		},
		"Let body": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.Let{CoreNode: f.node(), Name: "g", Value: f.lit(), Body: in}
		},
		"LetRec binding": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.LetRec{CoreNode: f.node(), Bindings: []core.RecBinding{{Name: "r", Value: in}}, Body: f.lit()}
		},
		"App arg": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.App{CoreNode: f.node(), Func: f.lit(), Args: []core.CoreExpr{in}}
		},
		"If else": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.If{CoreNode: f.node(), Cond: f.lit(), Then: f.lit(), Else: in}
		},
		"Match guard": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.Match{CoreNode: f.node(), Scrutinee: f.lit(), Arms: []core.MatchArm{{Guard: in, Body: f.lit()}}}
		},
		"Match body": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.Match{CoreNode: f.node(), Scrutinee: f.lit(), Arms: []core.MatchArm{{Body: in}}}
		},
		"BinOp": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.BinOp{CoreNode: f.node(), Op: "+", Left: f.lit(), Right: in}
		},
		"UnOp": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.UnOp{CoreNode: f.node(), Op: "-", Operand: in}
		},
		"Intrinsic arg": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.Intrinsic{CoreNode: f.node(), Op: core.OpEq, Args: []core.CoreExpr{f.lit(), in}}
		},
		"Record field": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.Record{CoreNode: f.node(), Fields: map[string]core.CoreExpr{"a": f.lit(), "b": in}}
		},
		"RecordAccess": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.RecordAccess{CoreNode: f.node(), Record: in, Field: "a"}
		},
		"RecordUpdate": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.RecordUpdate{CoreNode: f.node(), Base: f.lit(), Updates: map[string]core.CoreExpr{"a": in}}
		},
		"List": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.List{CoreNode: f.node(), Elements: []core.CoreExpr{in}}
		},
		"Array": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.Array{CoreNode: f.node(), Elements: []core.CoreExpr{in}}
		},
		"Tuple": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.Tuple{CoreNode: f.node(), Elements: []core.CoreExpr{f.lit(), in}}
		},
		"DictAbs": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.DictAbs{CoreNode: f.node(), Body: in}
		},
		"DictApp arg": func(f *walkerFixture, in core.CoreExpr) core.CoreExpr {
			return &core.DictApp{CoreNode: f.node(), Dict: &core.DictRef{CoreNode: f.node(), ClassName: "Eq", TypeName: "Int"}, Method: "eq", Args: []core.CoreExpr{in}}
		},
	}
	for name, wrap := range wrappers {
		t.Run(name, func(t *testing.T) {
			f := newWalkerFixture()
			top := &core.Let{CoreNode: f.node(), Name: "fn", Value: wrap(f, f.processRef()), Body: f.lit()}
			w := &ceilingWalker{coreTI: f.coreT, pkgName: "t/a"}
			w.walkTopLevel(top)
			if w.err != nil {
				t.Fatalf("walk error: %v", w.err)
			}
			if len(w.hits) != 1 || !strings.Contains(strings.Join(w.hits[0].effects, ","), "Process") {
				t.Fatalf("want exactly one hit carrying Process under %s; got %+v", name, w.hits)
			}
			if w.hits[0].owner != "function fn" || !strings.Contains(w.hits[0].what, "std/stream.asyncExecProcess") {
				t.Fatalf("attribution: %+v", w.hits[0])
			}
		})
	}
}

func TestCeilingWalker_TopLevelShapes(t *testing.T) {
	f := newWalkerFixture()
	rec := &core.LetRec{CoreNode: f.node(), Bindings: []core.RecBinding{{Name: "loop", Value: f.processRef()}}, Body: f.processRef()}
	w := &ceilingWalker{coreTI: f.coreT, pkgName: "t/a"}
	w.walkTopLevel(rec)
	if w.err != nil || len(w.hits) != 2 {
		t.Fatalf("want 2 hits (binding + trailing expression); err=%v hits=%+v", w.err, w.hits)
	}
	if w.hits[0].owner != "function loop" || w.hits[1].owner != "top-level expression" {
		t.Fatalf("owners: %q, %q", w.hits[0].owner, w.hits[1].owner)
	}
}

func TestCeilingWalker_OwnAndCompilerGlobals(t *testing.T) {
	f := newWalkerFixture()
	own := f.processRef()
	own.Ref.Module = "pkg/t/a/exec" // own sibling: its body is checked in its own module
	builtin := f.processRef()
	builtin.Ref.Module = "$builtin"
	w := &ceilingWalker{coreTI: f.coreT, pkgName: "t/a"}
	w.walk(own, "function a")
	w.walk(builtin, "function b")
	if len(w.hits) != 1 || w.hits[0].owner != "function b" {
		t.Fatalf("want only the $builtin reference charged; got %+v", w.hits)
	}
}

func TestCeilingWalker_MissingTypeFailsLoudly(t *testing.T) {
	f := newWalkerFixture()
	ref := f.processRef()
	delete(f.coreT, ref.ID())
	w := &ceilingWalker{coreTI: f.coreT, pkgName: "t/a"}
	w.walk(ref, "function f")
	if w.err == nil || !strings.Contains(w.err.Error(), "no type information") {
		t.Fatalf("an entry point without CoreTI must be an error, got %v", w.err)
	}
}

func TestCeilingWalker_DescribeAndLabels(t *testing.T) {
	f := newWalkerFixture()
	d := &core.DictRef{CoreNode: f.node(), ClassName: "Show", TypeName: "Int"}
	if got := describeCeilingExpr(d); !strings.Contains(got, "dictionary Show[Int]") {
		t.Errorf("DictRef description: %q", got)
	}
	if got := describeCeilingExpr(f.lit()); !strings.Contains(got, "*core.Lit") {
		t.Errorf("default description: %q", got)
	}
	// Nested: a function type inside a record field inside a list.
	nested := &types.TList{Element: &types.TRecord{Fields: map[string]types.Type{
		"run": &types.TFunc2{Return: types.TUnit, EffectRow: &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"Net": types.TUnit}}},
	}}}
	if got := typeEffectLabels(nested); len(got) != 1 || got[0] != "Net" {
		t.Errorf("nested labels: %v", got)
	}
	if got := typeEffectLabels(types.TInt); len(got) != 0 {
		t.Errorf("pure type labels: %v", got)
	}
}

func TestCeilingCacheDigest(t *testing.T) {
	saved := currentPackageManifest
	t.Cleanup(func() { currentPackageManifest = saved })

	currentPackageManifest = nil
	if _, ok := ceilingCacheDigest("m"); ok {
		t.Fatal("no manifest: no digest")
	}
	setTestCeiling("Stream", "IO")
	d1, ok := ceilingCacheDigest("m")
	if !ok || d1 != "t/a:[IO,Stream]" {
		t.Fatalf("own module digest = %q, %v", d1, ok)
	}
	if _, ok := ceilingCacheDigest("std/stream"); ok {
		t.Fatal("std modules carry no ceiling digest")
	}
	setTestCeiling("Stream")
	if d2, _ := ceilingCacheDigest("m"); d2 == d1 {
		t.Fatal("narrowing max must change the digest")
	}
}

func setTestCeiling(maxEffects ...string) {
	m := &pkg.PackageManifest{}
	m.Package.Name = "t/a"
	m.Effects.Max = maxEffects
	currentPackageManifest = m
}
