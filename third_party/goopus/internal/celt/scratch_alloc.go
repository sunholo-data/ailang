//go:build !poison

package celt

// alloc is the Go stand-in for ALLOC(x, n, T): it hands out an n-element window of
// a pooled buffer, growing the backing array only when a call asks for more than
// any previous call did. Sizes are a function of (mode, channels, frameSize) alone,
// so after the first largest-frame Encode or Decode no call reallocates.
//
// The window is NOT zeroed: see the zeroing contract in scratch.go's file comment.
// A build-tagged twin in scratch_poison.go deliberately fills it with 0x5A instead,
// so the -tags poison test build surfaces any read-before-write dependency.
func alloc[T any](p *[]T, n int) []T {
	if cap(*p) < n {
		*p = make([]T, n)
	}
	*p = (*p)[:n]
	return *p
}
