package lower

import (
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/types"
)

// ailang#1354: a receiver typed by name (`v: V`) reaches CoreTI as a bare
// TCon, so the lower pass must hand its name to the bytecode compiler —
// otherwise the compiler has no exact type to take the field's slot from.

func TestLowerRecordAccess_NamedReceiverCarriesRecordType(t *testing.T) {
	tests := []struct {
		name string
		typ  types.Type
		want string
	}{
		{"named record", &types.TCon{Name: "V"}, "V"},
		{"generic record", &types.TApp{Constructor: &types.TCon{Name: "Box"}, Args: []types.Type{&types.TCon{Name: "int"}}}, "Box"},
		{"anonymous record", &types.TRecord{Fields: map[string]types.Type{"x": &types.TCon{Name: "float"}}}, ""},
		{"unknown type", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := map[uint64]types.Type{}
			if tt.typ != nil {
				entries[11] = tt.typ
			}
			got := LowerExpr(&core.RecordAccess{
				CoreNode: core.CoreNode{NodeID: 10},
				Record:   coreVar(11, "v"),
				Field:    "x",
			}, makeCTI(entries))
			fa, ok := got.(stmt.FieldAccess)
			if !ok {
				t.Fatalf("expected FieldAccess, got %T", got)
			}
			if fa.RecordType != tt.want {
				t.Errorf("RecordType = %q, want %q", fa.RecordType, tt.want)
			}
		})
	}
}

func TestLowerRecordUpdate_CarriesBaseTypeAndSortsFields(t *testing.T) {
	cti := makeCTI(map[uint64]types.Type{
		11: &types.TRecord{Fields: map[string]types.Type{
			"y": &types.TCon{Name: "int"}, "x": &types.TCon{Name: "int"}, "z": &types.TCon{Name: "int"},
		}},
	})
	got := LowerExpr(&core.RecordUpdate{
		CoreNode: core.CoreNode{NodeID: 10},
		Base:     coreVar(11, "p"),
		Updates:  map[string]core.CoreExpr{"z": litInt(12, 3), "x": litInt(13, 1), "y": litInt(14, 2)},
	}, cti)
	ru, ok := got.(stmt.RecordUpdate)
	if !ok {
		t.Fatalf("expected RecordUpdate, got %T", got)
	}
	if want := []string{"x", "y", "z"}; !reflect.DeepEqual(ru.KnownFields, want) {
		t.Errorf("KnownFields = %v, want %v", ru.KnownFields, want)
	}
	var names []string
	for _, f := range ru.Fields {
		names = append(names, f.Name)
	}
	if want := []string{"x", "y", "z"}; !reflect.DeepEqual(names, want) {
		t.Errorf("update fields = %v, want sorted %v", names, want)
	}

	named := LowerExpr(&core.RecordUpdate{
		CoreNode: core.CoreNode{NodeID: 20},
		Base:     coreVar(21, "v"),
		Updates:  map[string]core.CoreExpr{"x": litInt(22, 1)},
	}, makeCTI(map[uint64]types.Type{21: &types.TCon{Name: "V"}})).(stmt.RecordUpdate)
	if named.RecordType != "V" || named.KnownFields != nil {
		t.Errorf("named base: RecordType=%q KnownFields=%v, want V / nil", named.RecordType, named.KnownFields)
	}
}
