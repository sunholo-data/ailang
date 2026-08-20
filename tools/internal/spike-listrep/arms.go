package spikelistrep

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/eval"
)

// NewArm returns an empty list for a named measurement arm.
func NewArm(name string) (List, error) {
	switch name {
	case "C0":
		return SliceEmpty(), nil
	case "C1":
		return ConsEmpty(), nil
	case "C2K8":
		return ChunkEmpty(8), nil
	case "C2K32":
		return ChunkEmpty(32), nil
	default:
		return nil, fmt.Errorf("unknown arm %q", name)
	}
}

// PrependArm prepends to a list from the named measurement arm.
func PrependArm(name string, value eval.Value, list List) (List, error) {
	switch name {
	case "C0":
		return SliceCons(value, list), nil
	case "C1":
		return ConsCons(value, list), nil
	case "C2K8":
		return ChunkCons(8, value, list), nil
	case "C2K32":
		return ChunkCons(32, value, list), nil
	default:
		return nil, fmt.Errorf("unknown arm %q", name)
	}
}

// SharedSingletonElements returns n references to the same element.
func SharedSingletonElements(n int, element eval.Value) []eval.Value {
	result := make([]eval.Value, n)
	for i := range result {
		result[i] = element
	}
	return result
}
