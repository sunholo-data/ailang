package spikelistrep

import (
	"iter"

	"github.com/sunholo-data/ailang/internal/eval"
)

// ConsList is the C1 persistent cons-cell representation.
type ConsList struct {
	head eval.Value
	tail *ConsList
	n    int
}

func ConsEmpty() List { return (*ConsList)(nil) }

func ConsFromSlice(elements []eval.Value) List {
	var result *ConsList
	for i := len(elements) - 1; i >= 0; i-- {
		result = newConsList(elements[i], result)
	}
	return result
}

func ConsCons(head eval.Value, tail List) List {
	consTail, ok := tail.(*ConsList)
	if !ok && tail.Len() != 0 {
		return ConsFromSlice(append([]eval.Value{head}, tail.ToSlice()...))
	}
	return newConsList(head, consTail)
}

func newConsList(head eval.Value, tail *ConsList) *ConsList {
	n := 1
	if tail != nil {
		n += tail.n
	}
	return &ConsList{head: head, tail: tail, n: n}
}

func (l *ConsList) Len() int {
	if l == nil {
		return 0
	}
	return l.n
}

func (l *ConsList) At(i int) (eval.Value, bool) {
	if i < 0 {
		return nil, false
	}
	for current := l; current != nil; current = current.tail {
		if i == 0 {
			return current.head, true
		}
		i--
	}
	return nil, false
}

func (l *ConsList) All() iter.Seq[eval.Value] {
	return func(yield func(eval.Value) bool) {
		for current := l; current != nil; current = current.tail {
			if !yield(current.head) {
				return
			}
		}
	}
}

func (l *ConsList) ToSlice() []eval.Value {
	result := make([]eval.Value, 0, l.Len())
	for element := range l.All() {
		result = append(result, element)
	}
	return result
}

func (l *ConsList) Uncons() (eval.Value, List, bool) {
	if l == nil {
		return nil, ConsEmpty(), false
	}
	return l.head, l.tail, true
}

func (l *ConsList) DropPrefix(k int) List {
	current := l
	for k > 0 && current != nil {
		current = current.tail
		k--
	}
	return current
}
