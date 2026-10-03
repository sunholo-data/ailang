package types

import "math"

// FloatLt, FloatLte, FloatGt and FloatGte are AILANG's ONE rule for ordered
// comparisons on floats: IEEE 754. Every ordered comparison with a NaN
// operand is false, in both operand orders; -0.0 and +0.0 compare equal.
// They are the ordered counterparts of FloatEq. The users are the Ord[Float]
// dictionary, the lt/le/gt/ge_Float builtins (both registries), the
// evaluator's binop paths and the bytecode VM's OpLt/OpLe.
//
// Before M-FLOAT-ORD-ONE-SEMANTICS (#1419) the Ord[Float] dictionary used a
// total order with NaN greatest, so `nan > 1.0` was true on the evaluator,
// false on the VM, and false on the evaluator through a generic helper. IEEE
// follows the #1274 ruling for ==. It gives up a lawful total order around
// NaN, as Haskell's Ord Double does. Test for NaN with std/math.isNaN; an
// ordered comparison never detects it.
func FloatLt(a, b float64) bool { return a < b }

// FloatLte is IEEE <= (false when either operand is NaN). See FloatLt.
func FloatLte(a, b float64) bool { return a <= b }

// FloatGt is IEEE > (false when either operand is NaN). See FloatLt.
func FloatGt(a, b float64) bool { return a > b }

// FloatGte is IEEE >= (false when either operand is NaN). See FloatLt.
func FloatGte(a, b float64) bool { return a >= b }

// FloatMin and FloatMax back the Ord[Float] dictionary's min/max methods.
// They propagate NaN (math.Min/math.Max), so the answer never depends on
// argument order. Today no surface syntax reaches these methods, but they
// must stay registered for every Ord DictRef to resolve.
func FloatMin(a, b float64) float64 { return math.Min(a, b) }

// FloatMax is the NaN-propagating maximum. See FloatMin.
func FloatMax(a, b float64) float64 { return math.Max(a, b) }
