// Package golang provides Go code generation from AILANG Core AST.
package golang

// writeRuntimeHelpers writes utility functions needed by generated code.
func (g *Generator) writeRuntimeHelpers() {
	g.writef("// RecordUpdate creates a new record with specified fields updated.\n")
	g.writef("// AILANG: { base | field1: val1, field2: val2 }\n")
	g.writef("func RecordUpdate(base interface{}, updates map[string]interface{}) interface{} {\n")
	g.indent++
	g.writef("baseMap, ok := base.(map[string]interface{})\n")
	g.writef("if !ok {\n")
	g.indent++
	g.writef("return updates\n")
	g.indent--
	g.writef("}\n")
	g.writef("result := make(map[string]interface{}, len(baseMap)+len(updates))\n")
	g.writef("for k, v := range baseMap {\n")
	g.indent++
	g.writef("result[k] = v\n")
	g.indent--
	g.writef("}\n")
	g.writef("for k, v := range updates {\n")
	g.indent++
	g.writef("result[k] = v\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// Cons prepends an element to a list (cons operator).\n")
	g.writef("// AILANG: head :: tail\n")
	g.writef("func Cons(head interface{}, tail interface{}) []interface{} {\n")
	g.indent++
	g.writef("if tail == nil {\n")
	g.indent++
	g.writef("return []interface{}{head}\n")
	g.indent--
	g.writef("}\n")
	g.writef("list, ok := tail.([]interface{})\n")
	g.writef("if !ok {\n")
	g.indent++
	g.writef("return []interface{}{head}\n")
	g.indent--
	g.writef("}\n")
	g.writef("return append([]interface{}{head}, list...)\n")
	g.indent--
	g.writef("}\n\n")

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

	// CallFunc helper for calling interface{} values as functions
	g.writef("// CallFunc calls an interface{} as a function with the given arguments.\n")
	g.writef("// Used for lambdas stored in variables.\n")
	g.writef("func CallFunc(f interface{}, args ...interface{}) interface{} {\n")
	g.indent++
	g.writef("switch fn := f.(type) {\n")
	g.writef("case func() interface{}:\n")
	g.indent++
	g.writef("return fn()\n")
	g.indent--
	g.writef("case func(interface{}) interface{}:\n")
	g.indent++
	g.writef("if len(args) >= 1 {\n")
	g.indent++
	g.writef("return fn(args[0])\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("case func(interface{}, interface{}) interface{}:\n")
	g.indent++
	g.writef("if len(args) >= 2 {\n")
	g.indent++
	g.writef("return fn(args[0], args[1])\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("case func(interface{}, interface{}, interface{}) interface{}:\n")
	g.indent++
	g.writef("if len(args) >= 3 {\n")
	g.indent++
	g.writef("return fn(args[0], args[1], args[2])\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("case func(...interface{}) interface{}:\n")
	g.indent++
	g.writef("return fn(args...)\n")
	g.indent--
	g.writef("}\n")
	g.writef("panic(\"CallFunc: unsupported function type\")\n")
	g.indent--
	g.writef("}\n\n")

	// ListHead helper for list destructuring
	g.writef("// ListHead returns the first element of a list.\n")
	g.writef("func ListHead(list interface{}) interface{} {\n")
	g.indent++
	g.writef("if l, ok := list.([]interface{}); ok && len(l) > 0 {\n")
	g.indent++
	g.writef("return l[0]\n")
	g.indent--
	g.writef("}\n")
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n\n")

	// ListTail helper for list destructuring
	g.writef("// ListTail returns all but the first element of a list.\n")
	g.writef("func ListTail(list interface{}) interface{} {\n")
	g.indent++
	g.writef("if l, ok := list.([]interface{}); ok && len(l) > 0 {\n")
	g.indent++
	g.writef("return l[1:]\n")
	g.indent--
	g.writef("}\n")
	g.writef("return []interface{}{}\n")
	g.indent--
	g.writef("}\n\n")

	// ListLen helper for list length check
	g.writef("// ListLen returns the length of a list.\n")
	g.writef("func ListLen(list interface{}) int {\n")
	g.indent++
	g.writef("if l, ok := list.([]interface{}); ok {\n")
	g.indent++
	g.writef("return len(l)\n")
	g.indent--
	g.writef("}\n")
	g.writef("return 0\n")
	g.indent--
	g.writef("}\n\n")
}
