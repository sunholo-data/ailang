// Package mathx is AILANG's architecture-independent transcendental math core.
//
// AILANG promises determinism: a pure builtin must return the same bits on
// every machine. Go's standard math package does not make that promise across
// GOARCH. On arm64 it runs an assembly Exp written with fused multiply-add,
// and the compiler fuses x*y+z inside the pure-Go Log, Sin, Cos, Tan, Atan,
// Asin and Pow (the Go spec allows this, "possibly across statements"). On
// amd64 it runs assembly Exp and Log, and Exp switches to FMA instructions when
// the CPU has them. The same program therefore printed 1.229317398921793 on
// arm64 and 1.2293173989217931 on x86_64 (issue #1465).
//
// This package carries the fdlibm-derived pure-Go algorithms from the Go
// standard library (go1.26.6, src/math; BSD licence in LICENSE-GO) with one
// mechanical change: every floating-point product is wrapped in an explicit
// float64(...) conversion. The Go spec says an explicit conversion rounds to
// the target precision and prevents fusion, so each operation is individually
// rounded IEEE 754 binary64 on every architecture. Coefficients, branches and
// special-value handling are unchanged.
//
// Canonical values are therefore "the Go pure-Go algorithm with no fusion" —
// the values Go's own math package returns on GOARCH=wasm, where there is no
// assembly and no fusion. golden_test.go pins the exact bits; running it on any
// architecture checks the same table.
//
// Exact operations stay on the host math package: Sqrt (IEEE 754 requires a
// correctly rounded result), Floor, Ceil, Round, Abs, Frexp, Ldexp, Modf and
// the classification helpers have a single correct answer, so no architecture
// can disagree on them.
package mathx
