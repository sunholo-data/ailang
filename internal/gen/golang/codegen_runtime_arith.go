// Package golang provides Go code generation from AILANG Core AST.
package golang

// writeRuntimeArithmeticHelpers writes arithmetic, comparison, and type conversion functions.
func (g *Generator) writeRuntimeArithmeticHelpers() {
	// Integer arithmetic helpers (from dictionary resolution)
	g.writef("// AddInt adds two integers.\n")
	g.writef("func AddInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) + toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// SubInt subtracts two integers.\n")
	g.writef("func SubInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) - toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// MulInt multiplies two integers.\n")
	g.writef("func MulInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) * toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// DivInt divides two integers.\n")
	g.writef("func DivInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) / toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// ModInt returns integer modulo.\n")
	g.writef("func ModInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) %% toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	// Float arithmetic helpers
	g.writef("// AddFloat adds two floats.\n")
	g.writef("func AddFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) + toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// SubFloat subtracts two floats.\n")
	g.writef("func SubFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) - toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// MulFloat multiplies two floats.\n")
	g.writef("func MulFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) * toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// DivFloat divides two floats.\n")
	g.writef("func DivFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) / toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	// Type conversion helpers
	g.writef("// IntToFloat converts int to float.\n")
	g.writef("func IntToFloat(a interface{}) interface{} {\n")
	g.indent++
	g.writef("return float64(toInt64(a))\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// FloatToInt converts float to int.\n")
	g.writef("func FloatToInt(a interface{}) interface{} {\n")
	g.indent++
	g.writef("return int64(toFloat64(a))\n")
	g.indent--
	g.writef("}\n\n")

	// Comparison helpers
	g.writef("// EqInt compares two integers for equality.\n")
	g.writef("func EqInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) == toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	// M-VERIFY: Integer inequality comparison helper
	g.writef("// NeInt compares two integers for inequality.\n")
	g.writef("func NeInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) != toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	// M-DX30: String equality comparison helper
	g.writef("// EqString compares two strings for equality.\n")
	g.writef("func EqString(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return a.(string) == b.(string)\n")
	g.indent--
	g.writef("}\n\n")

	// M-DX30: Float equality comparison helper
	g.writef("// EqFloat compares two floats for equality.\n")
	g.writef("func EqFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) == toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// LtInt compares two integers (less than).\n")
	g.writef("func LtInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) < toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// LeInt compares two integers (less or equal).\n")
	g.writef("func LeInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) <= toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// GtInt compares two integers (greater than).\n")
	g.writef("func GtInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) > toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// GeInt compares two integers (greater or equal).\n")
	g.writef("func GeInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) >= toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	// Negation helper
	g.writef("// NegInt negates an integer.\n")
	g.writef("func NegInt(a interface{}) interface{} {\n")
	g.indent++
	g.writef("return -toInt64(a)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// NegFloat negates a float.\n")
	g.writef("func NegFloat(a interface{}) interface{} {\n")
	g.indent++
	g.writef("return -toFloat64(a)\n")
	g.indent--
	g.writef("}\n\n")

	// M-BUGFIX: Float comparison helpers (missing in previous versions)
	g.writef("// LtFloat compares two floats (less than).\n")
	g.writef("func LtFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) < toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// LeFloat compares two floats (less or equal).\n")
	g.writef("func LeFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) <= toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// GtFloat compares two floats (greater than).\n")
	g.writef("func GtFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) > toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// GeFloat compares two floats (greater or equal).\n")
	g.writef("func GeFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) >= toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	// Type conversion utility functions
	g.writef("// toInt64 converts interface{} to int64.\n")
	g.writef("func toInt64(v interface{}) int64 {\n")
	g.indent++
	g.writef("switch x := v.(type) {\n")
	g.writef("case int64:\n")
	g.indent++
	g.writef("return x\n")
	g.indent--
	g.writef("case int:\n")
	g.indent++
	g.writef("return int64(x)\n")
	g.indent--
	g.writef("case float64:\n")
	g.indent++
	g.writef("return int64(x)\n")
	g.indent--
	g.writef("default:\n")
	g.indent++
	g.writef("return 0\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// toFloat64 converts interface{} to float64.\n")
	g.writef("func toFloat64(v interface{}) float64 {\n")
	g.indent++
	g.writef("switch x := v.(type) {\n")
	g.writef("case float64:\n")
	g.indent++
	g.writef("return x\n")
	g.indent--
	g.writef("case int64:\n")
	g.indent++
	g.writef("return float64(x)\n")
	g.indent--
	g.writef("case int:\n")
	g.indent++
	g.writef("return float64(x)\n")
	g.indent--
	g.writef("default:\n")
	g.indent++
	g.writef("return 0\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n\n")
}
