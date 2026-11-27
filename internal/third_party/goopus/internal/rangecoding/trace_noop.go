//go:build !ectrace

package rangecoding

// Default build: the symbol trace is compiled out. These methods are empty and
// inline to nothing, so the primitives' trace(...) calls disappear entirely and
// the coder carries no trace state. Build with `-tags ectrace` for the real
// implementation in trace_on.go.

func (_this *Encoder) trace(kind traceKind, a, b, c uint32) {}

func (_this *Decoder) trace(kind traceKind, a, b, c uint32) {}
