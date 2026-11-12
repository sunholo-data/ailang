package elaborate

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// #2221: the row of an annotated open record `{x: int | r}` carries the same
// kind VALUE as every checker-made record row. It was a *KRow, and
// KRow.Equals only matches the value type, so RecordRow.Equals(kind) was false
// ("r has kind Row Record, but r has kind Row Record") and MarshalKind could
// not encode it (CACHE_WRITE_FAILED).
func TestOpenRecordAnnotationRowHasRecordRowKind(t *testing.T) {
	e := NewElaborator()
	got := e.astTypeToInternalType(&ast.RecordType{
		Fields: []*ast.RecordField{{Name: "x", Type: &ast.SimpleType{Name: "int"}}},
		Row:    &ast.TypeVar{Name: "r"},
	})
	open, ok := got.(*types.TRecordOpen)
	if !ok {
		t.Fatalf("want *types.TRecordOpen, got %T", got)
	}
	row, ok := open.Row.(*types.RowVar)
	if !ok {
		t.Fatalf("want a *types.RowVar row, got %T", open.Row)
	}
	if !types.RecordRow.Equals(row.Kind) || !row.Kind.Equals(types.RecordRow) {
		t.Fatalf("row kind %#v does not equal types.RecordRow both ways", row.Kind)
	}
}
