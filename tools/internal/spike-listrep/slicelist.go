package spikelistrep

import (
	"iter"

	"github.com/sunholo-data/ailang/internal/eval"
)

// SliceList is the C0 control and wraps the production slice representation.
type SliceList struct {
	value *eval.ListValue
}

func SliceEmpty() List { return newSliceList(nil) }

func SliceFromSlice(elements []eval.Value) List {
	copyOfElements := append([]eval.Value(nil), elements...)
	return newSliceList(copyOfElements)
}

// SliceCons mirrors listConsImpl's preallocated shallow copy.
func SliceCons(head eval.Value, tail List) List {
	sliceTail, ok := tail.(*SliceList)
	if !ok {
		return SliceFromSlice(append([]eval.Value{head}, tail.ToSlice()...))
	}
	elements := sliceTail.value.Elements
	result := make([]eval.Value, 0, 1+len(elements))
	result = append(result, head)
	result = append(result, elements...)
	return newSliceList(result)
}

func newSliceList(elements []eval.Value) *SliceList {
	return &SliceList{value: &eval.ListValue{Elements: elements}}
}

func (l *SliceList) Len() int { return len(l.value.Elements) }

func (l *SliceList) At(i int) (eval.Value, bool) {
	if i < 0 || i >= l.Len() {
		return nil, false
	}
	return l.value.Elements[i], true
}

func (l *SliceList) All() iter.Seq[eval.Value] {
	return func(yield func(eval.Value) bool) {
		for _, element := range l.value.Elements {
			if !yield(element) {
				return
			}
		}
	}
}

func (l *SliceList) ToSlice() []eval.Value {
	return append([]eval.Value(nil), l.value.Elements...)
}

func (l *SliceList) Uncons() (eval.Value, List, bool) {
	if l.Len() == 0 {
		return nil, SliceEmpty(), false
	}
	return l.value.Elements[0], newSliceList(l.value.Elements[1:]), true
}

func (l *SliceList) DropPrefix(k int) List {
	if k <= 0 {
		return l
	}
	if k >= l.Len() {
		return SliceEmpty()
	}
	return newSliceList(l.value.Elements[k:])
}
