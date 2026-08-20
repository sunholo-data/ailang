package spikelistrep

import (
	"iter"

	"github.com/sunholo-data/ailang/internal/eval"
)

// ChunkList is the C2 persistent, unrolled-cons representation. Published
// chunks are immutable; prepend copies at most the leading K-element chunk.
type ChunkList struct {
	elems []eval.Value
	tail  *ChunkList
	total int
	k     int
}

func ChunkEmpty(k int) List {
	checkChunkCapacity(k)
	return newChunkList(nil, nil, 0, k)
}

func ChunkFromSlice(k int, elements []eval.Value) List {
	checkChunkCapacity(k)
	var tail *ChunkList
	for end := len(elements); end > 0; {
		start := max(0, end-k)
		tail = newChunkList(elements[start:end], tail, len(elements)-start, k)
		end = start
	}
	if tail == nil {
		return newChunkList(nil, nil, 0, k)
	}
	return tail
}

func ChunkCons(k int, head eval.Value, tail List) List {
	checkChunkCapacity(k)
	chunkTail, ok := tail.(*ChunkList)
	if !ok || chunkTail.k != k {
		return ChunkFromSlice(k, append([]eval.Value{head}, tail.ToSlice()...))
	}
	if chunkTail.total == 0 {
		return newChunkList([]eval.Value{head}, nil, 1, k)
	}
	if len(chunkTail.elems) < k {
		elems := make([]eval.Value, 1, len(chunkTail.elems)+1)
		elems[0] = head
		elems = append(elems, chunkTail.elems...)
		return newChunkList(elems, chunkTail.tail, chunkTail.total+1, k)
	}
	return newChunkList([]eval.Value{head}, chunkTail, chunkTail.total+1, k)
}

func newChunkList(elems []eval.Value, tail *ChunkList, total, k int) *ChunkList {
	return &ChunkList{elems: append([]eval.Value(nil), elems...), tail: tail, total: total, k: k}
}

func checkChunkCapacity(k int) {
	if k <= 0 {
		panic("spikelistrep: chunk capacity must be positive")
	}
}

func (l *ChunkList) Len() int { return l.total }

func (l *ChunkList) At(i int) (eval.Value, bool) {
	if i < 0 || i >= l.total {
		return nil, false
	}
	for current := l; current != nil; current = current.tail {
		if i < len(current.elems) {
			return current.elems[i], true
		}
		i -= len(current.elems)
	}
	return nil, false
}

func (l *ChunkList) All() iter.Seq[eval.Value] {
	return func(yield func(eval.Value) bool) {
		for current := l; current != nil; current = current.tail {
			for _, element := range current.elems {
				if !yield(element) {
					return
				}
			}
		}
	}
}

func (l *ChunkList) ToSlice() []eval.Value {
	result := make([]eval.Value, 0, l.total)
	for element := range l.All() {
		result = append(result, element)
	}
	return result
}

func (l *ChunkList) Uncons() (eval.Value, List, bool) {
	if l.total == 0 {
		return nil, l, false
	}
	head := l.elems[0]
	if len(l.elems) == 1 {
		if l.tail == nil {
			return head, ChunkEmpty(l.k), true
		}
		return head, l.tail, true
	}
	return head, newChunkList(l.elems[1:], l.tail, l.total-1, l.k), true
}

func (l *ChunkList) DropPrefix(count int) List {
	if count <= 0 {
		return l
	}
	if count >= l.total {
		return ChunkEmpty(l.k)
	}
	current := l
	for count >= len(current.elems) {
		count -= len(current.elems)
		current = current.tail
	}
	if count == 0 {
		return current
	}
	return newChunkList(current.elems[count:], current.tail, current.total-count, current.k)
}
