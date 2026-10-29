package golang

import (
	"github.com/sunholo-data/ailang/internal/types"
	"testing"
)

func TestBytesUseSliceStorageInGeneratedGo(t *testing.T) {
	got, err := NewTypeMapper().MapType(&types.TCon{Name: "bytes"})
	if err != nil || string(got) != "[]byte" {
		t.Fatalf("bytes must be a primitive slice, got %q (%v)", got, err)
	}
	primitive, ok := MapPrimitiveType("bytes")
	if !ok || string(primitive) != "[]byte" {
		t.Fatalf("primitive bytes = %q,%v", primitive, ok)
	}
	g := NewADTGenerator("test")
	if got := g.mapNamedType("bytes"); got != "[]byte" {
		t.Fatalf("ADT bytes field storage = %q", got)
	}
}
