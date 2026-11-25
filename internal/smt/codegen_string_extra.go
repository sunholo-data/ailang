package smt

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
)

// std/string functions without a Z3 string-theory equivalent
// (M-VERIFY-SOUNDNESS-COVERAGE M5). AILANG strings are rune sequences
// (length("héllo") == 5), as Z3 strings are character sequences, so
// rune-level facts carry over exactly.
//
//   - charAt(s, i)         = (str.at s i)          exact (out of range is a runtime error)
//   - trim, toLower, toUpper, split, chars, join, repeat, replace are uninterpreted, each call
//     contributing facts the runtime result always has (Go strings.TrimSpace /
//     ToLower / Split …):
//     trim:   the result is a substring of s, no longer than s
//     case:   rune-by-rune, so the length is preserved
//     split:  delimiter "" → one part per rune; delimiter absent → [s]; else ≥ 1 part
//     chars:  one element per rune
//     join:   [] → "", [x] → x
//     repeat: n <= 0 → ""; otherwise length = len(s) * n
//     replace: unchanged when old does not occur in s (old != ""), or when old == new.
//     (Z3's exact str.replace_all made the solver answer `unknown`, an error.)
//     stringToInt: Go's strconv.ParseInt(s, 10, 64), an optional sign then
//     digits. Unsigned digits within int64 → Some(str.to_int s); no sign and not
//     all digits (or "") → None. A signed or out-of-range s is left open.
//     charCode: a single rune's code point, 0 … 0x10FFFF. Not Z3's str.to_code:
//     Z3's characters stop at 0x2FFFF, so tying the two would prove
//     `charCode(c) < 200000`, which a rune above that breaks. (Any other s is a
//     runtime error, so its value is never observed.)
//
// As for std/math, a model over these is inconclusive (witness replay decides).
var stdlibStringExtra = map[string]string{
	"trim": strTrim, "toLower": strLower, "toUpper": strUpper, "split": strSplit,
	"chars": strChars, "join": strJoin, "repeat": strRepeat, "charAt": strCharAt,
	"replace": strReplace, "stringToInt": strToInt, "charCode": strCharCode,
}

const (
	strTrim     = "$ail.str_trim"
	strLower    = "$ail.str_lower"
	strUpper    = "$ail.str_upper"
	strSplit    = "$ail.str_split"
	strChars    = "$ail.str_chars"
	strJoin     = "$ail.str_join"
	strRepeat   = "$ail.str_repeat"
	strCharAt   = "$ail.str_char_at"
	strReplace  = "$ail.str_replace"
	strToInt    = "$ail.str_to_int"
	strCharCode = "$ail.str_char_code"
)

// stringUF describes one uninterpreted string function: symbol, SMT parameter
// sorts and result sort.
var stringUF = map[string]struct {
	symbol, params, result string
	arity                  int
}{
	strTrim:     {"$str_trim", "String", "String", 1},
	strLower:    {"$str_lower", "String", "String", 1},
	strUpper:    {"$str_upper", "String", "String", 1},
	strSplit:    {"$str_split", "String String", "(Seq String)", 2},
	strChars:    {"$str_chars", "String", "(Seq String)", 1},
	strJoin:     {"$str_join", "String (Seq String)", "String", 2},
	strRepeat:   {"$str_repeat", "String Int", "String", 2},
	strReplace:  {"$str_replace", "String String String", "String", 3},
	strToInt:    {"$str_to_int", "String", "(Option Int)", 1},
	strCharCode: {"$str_char_code", "String", "Int", 1},
}

// encodeStringExtra encodes one of stdlibStringExtra. ok is false otherwise.
func encodeStringExtra(name string, args []core.CoreExpr) (string, bool, error) {
	arity := map[string]int{strCharAt: 2}[name]
	uf, isUF := stringUF[name]
	if isUF {
		arity = uf.arity
	}
	if arity == 0 {
		return "", false, nil
	}
	if len(args) != arity {
		return "", true, fmt.Errorf("%w: %s expects %d args, got %d", ErrUnsupportedConstruct, name, arity, len(args))
	}
	a := make([]string, len(args))
	for i, x := range args {
		e, err := EncodeExpr(x)
		if err != nil {
			return "", true, err
		}
		a[i] = e
	}
	if name == strCharAt {
		return fmt.Sprintf("(str.at %s %s)", a[0], a[1]), true, nil
	}
	if err := useUninterpreted(uf.symbol, fmt.Sprintf("(declare-fun %s (%s) %s)", uf.symbol, uf.params, uf.result)); err != nil {
		return "", true, err
	}
	var call string
	switch name {
	case strSplit: // std/string.split(s, delimiter)
		call = fmt.Sprintf("(%s %s %s)", uf.symbol, a[0], a[1])
	case strJoin: // std/string.join(delimiter, xs)
		call = fmt.Sprintf("(%s %s %s)", uf.symbol, a[0], a[1])
	case strRepeat:
		call = fmt.Sprintf("(%s %s %s)", uf.symbol, a[0], a[1])
	case strReplace: // replace(s, old, new)
		call = fmt.Sprintf("(%s %s %s %s)", uf.symbol, a[0], a[1], a[2])
	default:
		call = fmt.Sprintf("(%s %s)", uf.symbol, a[0])
	}
	for _, fact := range stringFacts(name, call, a) {
		addCallFact(fact)
	}
	return call, true, nil
}

// stringFacts are properties every runtime result has, for every input.
func stringFacts(name, r string, a []string) []string {
	s := a[0]
	switch name {
	case strTrim:
		return []string{
			fmt.Sprintf("(str.contains %s %s)", s, r),
			fmt.Sprintf("(<= (str.len %s) (str.len %s))", r, s),
		}
	case strLower, strUpper:
		return []string{fmt.Sprintf("(= (str.len %s) (str.len %s))", r, s)}
	case strSplit:
		d := a[1]
		return []string{
			fmt.Sprintf("(=> (= %s \"\") (= (seq.len %s) (str.len %s)))", d, r, s),
			fmt.Sprintf("(=> (and (not (= %[1]s \"\")) (not (str.contains %[2]s %[1]s))) (= %[3]s (seq.unit %[2]s)))", d, s, r),
			fmt.Sprintf("(=> (not (= %s \"\")) (>= (seq.len %s) 1))", d, r),
		}
	case strChars:
		return []string{fmt.Sprintf("(= (seq.len %s) (str.len %s))", r, s)}
	case strJoin:
		xs := a[1]
		return []string{
			fmt.Sprintf("(=> (= (seq.len %s) 0) (= %s \"\"))", xs, r),
			fmt.Sprintf("(=> (= (seq.len %[1]s) 1) (= %[2]s (seq.nth %[1]s 0)))", xs, r),
		}
	case strReplace:
		old, nw := a[1], a[2]
		return []string{
			fmt.Sprintf("(=> (and (not (= %[1]s \"\")) (not (str.contains %[2]s %[1]s))) (= %[3]s %[2]s))", old, s, r),
			fmt.Sprintf("(=> (= %s %s) (= %s %s))", old, nw, r, s),
		}
	case strCharCode:
		return []string{fmt.Sprintf("(=> (= (str.len %[1]s) 1) (and (<= 0 %[2]s) (<= %[2]s 1114111)))", s, r)}
	case strToInt:
		k := fmt.Sprintf("(str.to_int %s)", s)
		return []string{
			fmt.Sprintf("(=> (and (<= 0 %[1]s) (<= %[1]s 9223372036854775807)) (= %[2]s ((as Some (Option Int)) %[1]s)))", k, r),
			fmt.Sprintf("(=> (and (= %[1]s (- 1)) (not (str.prefixof \"-\" %[2]s)) (not (str.prefixof \"+\" %[2]s))) (= %[3]s (as None (Option Int))))", k, s, r),
		}
	case strRepeat:
		n := a[1]
		return []string{
			fmt.Sprintf("(=> (<= %s 0) (= %s \"\"))", n, r),
			fmt.Sprintf("(=> (> %[1]s 0) (= (str.len %[2]s) (* (str.len %[3]s) %[1]s)))", n, r, s),
		}
	}
	return nil
}
