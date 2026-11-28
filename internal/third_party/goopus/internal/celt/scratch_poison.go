//go:build poison

package celt

import "unsafe"

// alloc, poison build. Identical window bookkeeping to the production alloc in
// scratch_alloc.go, except every returned window is stamped with the byte pattern
// 0x5A before being handed back. This is the read-before-write audit prescribed in
// issues #5 and #11: the codec is bit-exact only if every pooled buffer writes its
// entire read window before reading it, so poisoning the window must NOT change any
// output. Run the full battery with -tags poison (add refc for the differential
// harness); any divergence localises a site that reads a buffer before writing it.
//
// 0x5A is a deliberately hostile fill: non-zero (so it differs from make()'s zero
// and from a genuine silence frame), the same in every byte lane (so an int32 reads
// as 0x5A5A5A5A regardless of endianness), and it repeats every call (so a stale
// value cannot happen to match the previous frame's).
func alloc[T any](p *[]T, n int) []T {
	if cap(*p) < n {
		*p = make([]T, n)
	}
	*p = (*p)[:n]
	s := *p
	if n > 0 {
		b := unsafe.Slice((*byte)(unsafe.Pointer(&s[0])), n*int(unsafe.Sizeof(s[0])))
		for i := range b {
			b[i] = 0x5A
		}
	}
	return s
}
