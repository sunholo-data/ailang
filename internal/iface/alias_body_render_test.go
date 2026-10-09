package iface

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// A closed record alias carries its own name as TypeName (M-ALIAS-BODY-CLOSURE).
// Its rendered BODY must stay structural; only nested aliases render by name.
func TestRenderTypeAlias_ClosedRecordRootIsStructural(t *testing.T) {
	inner := &types.TRecord{Fields: map[string]types.Type{"mime": &types.TCon{Name: "string"}}, TypeName: "ImagePart"}
	root := &types.TRecord{
		Fields: map[string]types.Type{
			"role":   &types.TCon{Name: "string"},
			"images": &types.TList{Element: inner},
		},
		TypeName: "Message",
	}
	got := renderTypeAlias(root)
	want := "{images: [ImagePart], role: string}"
	if got != want {
		t.Fatalf("renderTypeAlias = %q, want %q", got, want)
	}
	if root.TypeName != "Message" {
		t.Fatalf("renderTypeAlias mutated the alias: TypeName = %q", root.TypeName)
	}
}
