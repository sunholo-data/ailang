package format

// Precedence levels mirror internal/lexer/token.go's Token.Precedence() and the
// parser's binding-power constants (internal/parser/parser.go). The formatter
// reconstructs parentheses purely from these levels plus operand position,
// because the AST retains no ParenExpr node (design V20). Keeping this table in
// lockstep with the parser is a release invariant: if the parser precedences
// change, these must change too, or round-trip breaks.
const (
	precLowest      = 0
	precLambda      = 1  // \x. ...   (lowest binding operator)
	precLogicalOr   = 2  // ||
	precLogicalAnd  = 3  // &&
	precBitwiseXor  = 5  // ^
	precBitwiseAnd  = 6  // &
	precEquals      = 7  // ==, !=
	precLessGreater = 8  // <, >, <=, >=
	precShift       = 9  // <<, >>
	precCons        = 10 // :: (right associative)
	precAppend      = 11 // ++
	precSum         = 12 // +, -
	precProduct     = 13 // *, /, %
	precPrefix      = 14 // unary -, !, not, ~
	precCall        = 15 // f(x), field access, indexing
	precDotAccess   = 16 // r.field
	precAtom        = 17 // literals, identifiers, parenthesised/bracketed forms
)

// binaryPrecedence returns the precedence level of a binary operator spelling.
// Unknown operators return precLowest, which conservatively forces parentheses
// (never fewer than the parser needs).
func binaryPrecedence(op string) int {
	switch op {
	case "||":
		return precLogicalOr
	case "&&":
		return precLogicalAnd
	case "^":
		return precBitwiseXor
	case "&":
		return precBitwiseAnd
	case "==", "!=":
		return precEquals
	case "<", ">", "<=", ">=":
		return precLessGreater
	case "<<", ">>":
		return precShift
	case "::":
		return precCons
	case "++":
		return precAppend
	case "+", "-":
		return precSum
	case "*", "/", "%":
		return precProduct
	default:
		return precLowest
	}
}

// rightAssociative reports whether a binary operator associates to the right.
// Only cons (::) is right-associative in AILANG; every other binary operator is
// left-associative. Associativity decides whether the same-precedence operand on
// the associative side needs parentheses.
func rightAssociative(op string) bool {
	return op == "::"
}
