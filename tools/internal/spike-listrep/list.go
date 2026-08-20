// Package spikelistrep contains throwaway list-representation experiments.
package spikelistrep

import (
	"iter"

	"github.com/sunholo-data/ailang/internal/eval"
)

// List is the read-only accessor surface exercised by the spike.
type List interface {
	Len() int
	At(int) (eval.Value, bool)
	All() iter.Seq[eval.Value]
	ToSlice() []eval.Value
	Uncons() (eval.Value, List, bool)
	DropPrefix(int) List
}
