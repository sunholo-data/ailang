package types

// FloatEq is AILANG's ONE rule for == on floats: IEEE 754. NaN is unequal to
// everything, itself included; -0.0 == +0.0. Every path that compares floats
// for == uses it: the Eq[Float] dictionary, the eq_Float/ne_Float builtins,
// the evaluator's structural and deferred comparisons, and the bytecode VM's
// OpEq, bare or nested inside a list, tuple, record or ADT.
//
// Before M-FLOAT-EQ-ONE-SEMANTICS (#1274) five implementations disagreed:
// `nan == nan` was true through the dictionary but false through a generic
// helper, and the VM disagreed with the evaluator. IEEE was chosen (Mark,
// 2026-09-25) to match the prior of every model and the reference languages
// benchmark outputs come from (Python, JS, Go, Rust, Haskell). It gives up
// reflexivity for NaN, as Rust and Haskell do. Test for NaN with
// std/math.isNaN, not ==.
func FloatEq(a, b float64) bool { return a == b }
