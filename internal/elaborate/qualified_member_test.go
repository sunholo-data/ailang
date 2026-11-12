package elaborate

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/errors"
)

// `L.member` where L aliases a module that lacks member used to report
// "undefined variable: L" (the alias), not the member. It is now IMP010 naming
// the member and the module, with the import hint: after the stdlib aliases are
// removed, `L.head_or` names headOr (M-V1-STDLIB-SURFACE-POLISH §6.3).
func TestQualifiedAccessToMissingMember(t *testing.T) {
	e := NewElaborator()
	e.MergeGlobalEnv(map[string]core.GlobalRef{
		"L.headOr": {Module: "std/list", Name: "headOr"},
		"L.map":    {Module: "std/list", Name: "map"},
	})
	acc := func(rec, field string) *ast.RecordAccess {
		return &ast.RecordAccess{Record: &ast.Identifier{Name: rec}, Field: field, Pos: ast.Pos{File: "u.ail", Line: 3, Column: 5}}
	}

	// Present member: a VarGlobal, as before.
	got, err := e.normalize(acc("L", "headOr"))
	if err != nil {
		t.Fatalf("L.headOr: %v", err)
	}
	if vg, ok := got.(*core.VarGlobal); !ok || vg.Ref.Name != "headOr" {
		t.Fatalf("L.headOr resolved to %v", got)
	}

	// A removed std spelling names its replacement.
	_, err = e.normalize(acc("L", "head_or"))
	if err == nil {
		t.Fatal("L.head_or with no head_or export: no error")
	}
	rep, ok := errors.AsReport(err)
	if !ok || rep.Code != errors.IMP010 {
		t.Fatalf("want a structured IMP010, got %T %v", err, err)
	}
	for _, want := range []string{"'head_or' not exported by 'std/list'", "L.head_or", "use 'headOr'"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q: %s", want, err)
		}
	}
	if strings.Contains(err.Error(), "undefined variable") {
		t.Errorf("still blames the alias: %s", err)
	}

	// A typo names the member too (the did-you-mean line needs the stdlib
	// index the CLI wires into importhint; covered end to end by the CLI).
	_, err = e.normalize(acc("L", "mapp"))
	if err == nil || !strings.Contains(err.Error(), "'mapp' not exported by 'std/list'") {
		t.Errorf("L.mapp: %v", err)
	}

	// Control: a record variable that is not a module alias is a plain record
	// access (no IMP010).
	e2 := NewElaborator()
	if _, err := e2.normalize(acc("cfg", "port")); err != nil {
		if _, isRep := errors.AsReport(err); isRep {
			t.Errorf("record access on a non-alias produced a report: %v", err)
		}
	}
}
