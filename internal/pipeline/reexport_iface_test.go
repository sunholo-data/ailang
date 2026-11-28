package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/types"
)

// Unit tests for a facade's interface, alias binding, effect ceiling and the
// moved warning (M-PKG-PARALLEL-AGENTS C1). The end-to-end tests are in
// internal/runner and cmd/ailang.

func netFunc() *types.Scheme {
	return &types.Scheme{Type: &types.TFunc2{Return: types.TUnit, EffectRow: &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"Net": types.TUnit}}}}
}

// shapesIface is lib/shapes: function area, ADT Shape = Square | Rect, alias
// Size, and function fetch ! {Net}.
func shapesIface() *iface.Iface {
	s := iface.NewIface("lib/shapes")
	s.Exports["area"] = &iface.IfaceItem{Name: "area", Type: &types.Scheme{Type: types.TInt}, Ref: core.GlobalRef{Module: "lib/shapes", Name: "area"}}
	s.Exports["fetch"] = &iface.IfaceItem{Name: "fetch", Type: netFunc(), Ref: core.GlobalRef{Module: "lib/shapes", Name: "fetch"}}
	s.Types["Shape"] = &iface.TypeExport{Name: "Shape"}
	s.Constructors["Square"] = &iface.ConstructorScheme{TypeName: "Shape", CtorName: "Square", Arity: 1}
	s.Constructors["Rect"] = &iface.ConstructorScheme{TypeName: "Shape", CtorName: "Rect", Arity: 2}
	s.Types["Size"] = &iface.TypeExport{Name: "Size", Arity: 1}
	s.TypeAliases["Size"] = types.TInt
	s.AliasParams["Size"] = []string{"a"}
	return s
}

func ifaces(list ...*iface.Iface) func(string) *iface.Iface {
	m := map[string]*iface.Iface{}
	for _, i := range list {
		m[i.Module] = i
	}
	return func(id string) *iface.Iface { return m[id] }
}

func TestAttachReexports_CopiesEntriesUnderFacadeNames(t *testing.T) {
	f := iface.NewIface("lib/facade")
	attachReexports(f, map[string]loader.Reexport{
		"area":   {From: "lib/shapes", Orig: "area", MovedTo: "new/shapes"},
		"Shape":  {From: "lib/shapes", Orig: "Shape"},
		"Box":    {From: "lib/shapes", Orig: "Square"},
		"Size":   {From: "lib/shapes", Orig: "Size"},
		"gone":   {From: "lib/shapes", Orig: "nothing"},
		"absent": {From: "lib/unloaded", Orig: "x"},
	}, ifaces(shapesIface()))
	rx := f.Reexported
	if rx == nil {
		t.Fatal("no re-export table")
	}
	if len(f.Exports) != 0 || len(f.Constructors) != 0 || len(f.Types) != 0 {
		t.Fatal("re-exports must not enter the facade's own maps")
	}
	if got := rx.Items.Exports["area"]; got == nil || got.Name != "area" || got.Ref.Module != "lib/shapes" {
		t.Errorf("area: %+v", got)
	}
	if rx.Items.Types["Shape"] == nil || rx.Items.Constructors["Rect"] == nil {
		t.Errorf("a re-exported type brings its constructors: %+v", rx.Items.Constructors)
	}
	if box := rx.Items.Constructors["Box"]; box == nil || box.CtorName != "Square" {
		t.Errorf("renamed constructor: %+v", box)
	}
	if rx.Items.TypeAliases["Size"] == nil || rx.Items.AliasParams["Size"] == nil {
		t.Error("alias body and params must travel")
	}
	if o := rx.Origins["area"]; o.Module != "lib/shapes" || o.Name != "area" || o.MovedTo != "new/shapes" {
		t.Errorf("area origin: %+v", o)
	}
	for _, n := range []string{"gone", "absent"} {
		if _, ok := rx.Origins[n]; ok {
			t.Errorf("%s: a name the source lacks must be skipped", n)
		}
	}

	// Nothing to attach, or nothing found: no table.
	empty := iface.NewIface("lib/e")
	attachReexports(empty, nil, ifaces())
	attachReexports(empty, map[string]loader.Reexport{"x": {From: "lib/none", Orig: "x"}}, ifaces())
	if empty.Reexported != nil {
		t.Error("an empty table must stay nil")
	}
}

func TestAttachReexports_ConstructorOnlyNamesItsType(t *testing.T) {
	f := iface.NewIface("lib/facade")
	attachReexports(f, map[string]loader.Reexport{"Square": {From: "lib/shapes", Orig: "Square"}}, ifaces(shapesIface()))
	if f.Reexported.Items.Types["Shape"] == nil {
		t.Fatal("a re-exported constructor must name its type so it renders")
	}
}

func TestAttachReexports_ChainRecordsTheDefinition(t *testing.T) {
	inner := iface.NewIface("lib/inner")
	attachReexports(inner, map[string]loader.Reexport{"area": {From: "lib/shapes", Orig: "area", MovedTo: "first/path"}}, ifaces(shapesIface()))
	outer := iface.NewIface("lib/outer")
	attachReexports(outer, map[string]loader.Reexport{"size": {From: "lib/inner", Orig: "area"}}, ifaces(shapesIface(), inner))
	o := outer.Reexported.Origins["size"]
	if o.Module != "lib/shapes" || o.Name != "area" || o.MovedTo != "first/path" {
		t.Fatalf("chain origin: %+v", o)
	}
}

func TestBindAliasedReexports(t *testing.T) {
	f := iface.NewIface("lib/facade")
	attachReexports(f, map[string]loader.Reexport{
		"area":  {From: "lib/shapes", Orig: "area"},
		"Shape": {From: "lib/shapes", Orig: "Shape"},
	}, ifaces(shapesIface()))
	im := &moduleImports{
		ExternalTypes: map[string]*types.Scheme{},
		GlobalRefs:    map[string]core.GlobalRef{},
		ExplicitTypes: map[string]types.TypeRef{},
	}
	bindAliasedReexports(im, "F", f)
	if ref := im.GlobalRefs["F.area"]; ref.Module != "lib/shapes" || ref.Name != "area" {
		t.Errorf("F.area must bind the origin: %+v", ref)
	}
	if im.ExternalTypes["lib/shapes.area"] == nil || im.IFC.Names["F.area"] != "lib/shapes.area" {
		t.Error("F.area type or IFC entry missing")
	}
	if ref := im.ExplicitTypes["F.Shape"]; ref.Module != "lib/shapes" || ref.Name != "Shape" {
		t.Errorf("F.Shape must resolve to its defining module: %+v", ref)
	}
	before := len(im.GlobalRefs)
	bindAliasedReexports(im, "P", iface.NewIface("lib/plain"))
	if len(im.GlobalRefs) != before {
		t.Error("a module without re-exports binds nothing here")
	}
}

func TestValidateReexportCeiling(t *testing.T) {
	saved := currentPackageManifest
	t.Cleanup(func() { currentPackageManifest = saved })

	f := iface.NewIface("lib/facade")
	attachReexports(f, map[string]loader.Reexport{
		"area":  {From: "lib/shapes", Orig: "area"},
		"fetch": {From: "lib/shapes", Orig: "fetch"},
	}, ifaces(shapesIface()))

	currentPackageManifest = nil
	if err := validateReexportCeiling(f, "lib/facade"); err != nil {
		t.Fatalf("no manifest: %v", err)
	}
	setTestCeiling()
	currentPackageManifest.Effects.Max = []string{} // max = [], not "no ceiling"
	err := validateReexportCeiling(f, "lib/facade")
	if err == nil || !strings.Contains(err.Error(), "re-exported function fetch") || !strings.Contains(err.Error(), "defined in lib/shapes as fetch") {
		t.Fatalf("max = [] with a {Net} re-export: %v", err)
	}
	if err := validateReexportCeiling(f, "std/facade"); err != nil {
		t.Fatalf("a module outside the package is not held to its ceiling: %v", err)
	}
	if err := validateReexportCeiling(iface.NewIface("lib/plain"), "lib/plain"); err != nil {
		t.Fatalf("no re-exports: %v", err)
	}
	setTestCeiling("Net")
	if err := validateReexportCeiling(f, "lib/facade"); err != nil {
		t.Fatalf("max = [Net] covers fetch: %v", err)
	}
}

func TestDetectMovedImports(t *testing.T) {
	saved := currentPackageManifest
	t.Cleanup(func() { currentPackageManifest = saved })
	setTestCeiling()

	moved := &ast.ImportDecl{
		Path: "lib/shapes", Symbols: []string{"area", "Shape"},
		SymbolPos: []ast.Pos{{File: "app/main.ail", Line: 3, Column: 20}},
		Pos:       ast.Pos{Line: 3, Column: 1},
		ViaModule: "lib/facade", ViaMovedTo: "new/shapes",
	}
	units := map[string]*CompileUnit{
		"app/main":          {ID: "app/main", Surface: &ast.File{Path: "app/main.ail", Imports: []*ast.ImportDecl{nil, {Path: "lib/x", Symbols: []string{"y"}}, moved}}},
		"pkg/other/dep/mod": {ID: "pkg/other/dep/mod", Surface: &ast.File{Imports: []*ast.ImportDecl{moved}}},
		"app/nil":           nil,
		"app/nosurface":     {ID: "app/nosurface"},
	}
	ws := DetectMovedImports(units)
	if len(ws) != 2 {
		t.Fatalf("want one warning per name in the project's own module, got %d: %v", len(ws), ws)
	}
	w := ws[0].(*MovedImportWarning)
	if line, col := w.Position(); line != 3 || col != 20 {
		t.Errorf("position: %d:%d", line, col)
	}
	if w.Code() != CodeDeprecated || w.WarningFile() != "app/main.ail" {
		t.Errorf("code/file: %s %s", w.Code(), w.WarningFile())
	}
	if w.Message() != "lib/facade.area has moved to new/shapes" || w.Suggestion() != "import area from new/shapes" {
		t.Errorf("text: %q / %q", w.Message(), w.Suggestion())
	}
	if s := w.String(); !strings.Contains(s, "TC_DEPRECATED_001") || !strings.Contains(s, "Fix: import area from new/shapes") {
		t.Errorf("rendered: %q", s)
	}
	// The second name has no recorded position: it falls back to the import's,
	// in the unit's file.
	w2 := ws[1].(*MovedImportWarning)
	if line, _ := w2.Position(); line != 3 || w2.WarningFile() != "app/main.ail" || w2.Name != "Shape" {
		t.Errorf("fallback position: %+v", w2)
	}
}

func TestCacheIface_RoundTripsReexports(t *testing.T) {
	src := shapesIface()
	for _, item := range src.Exports {
		item.IFC = types.IFCSummaryFromScheme(item.Type)
	}
	f := iface.NewIface("lib/facade")
	attachReexports(f, map[string]loader.Reexport{
		"area":  {From: "lib/shapes", Orig: "area", MovedTo: "new/shapes"},
		"Shape": {From: "lib/shapes", Orig: "Shape"},
	}, ifaces(src))
	raw, err := marshalIfaceFull(f)
	if err != nil {
		t.Fatal(err)
	}
	back, err := unmarshalIfaceFull(raw)
	if err != nil {
		t.Fatal(err)
	}
	rx := back.Reexported
	if rx == nil || rx.Origins["area"].MovedTo != "new/shapes" || rx.Origins["Shape"].Module != "lib/shapes" {
		t.Fatalf("origins lost in the cache: %+v", rx)
	}
	if rx.Items.Exports["area"] == nil || rx.Items.Types["Shape"] == nil || rx.Items.Constructors["Rect"] == nil {
		t.Fatalf("items lost in the cache: %+v", rx.Items)
	}
	if len(back.Exports) != 0 {
		t.Fatal("re-exports leaked into the facade's own exports")
	}
	// A module without re-exports serializes without the section.
	plain, _ := marshalIfaceFull(iface.NewIface("lib/plain"))
	if strings.Contains(string(plain), `"reexports"`) {
		t.Fatalf("plain interface carries a reexports section: %s", plain)
	}
}
