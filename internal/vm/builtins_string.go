package vm

import (
	"encoding/hex"
	"fmt"
	"html"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/sunholo/ailang/internal/bytecode"
)

// M-BYTECODE-STDLIB-BUILTINS M1: Pure string builtins wired to VM OpBuiltinCall.
//
// Each function below matches the semantics of its evaluator counterpart in
// internal/builtins/string*.go. The names in compiler.BuiltinTable use the
// lower-pass convention: "_" + registry name. For example, the evaluator's
// "_str_upper" becomes "__str_upper" in the compiler table.

// --- String primitives -------------------------------------------------------

func builtinStrLen(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__str_len: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_len: expected string, got %s", args[0].Tag)
	}
	return bytecode.NewInt(int64(utf8.RuneCountInString(args[0].AsString()))), nil
}

func builtinStrCompare(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__str_compare: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString || args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_compare: expected strings")
	}
	a, b := args[0].AsString(), args[1].AsString()
	result := int64(0)
	if a < b {
		result = -1
	} else if a > b {
		result = 1
	}
	return bytecode.NewInt(result), nil
}

func builtinStrEq(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__str_eq: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString || args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_eq: expected strings")
	}
	return bytecode.NewBool(args[0].AsString() == args[1].AsString()), nil
}

func builtinStrFind(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__str_find: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString || args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_find: expected strings")
	}
	haystack, needle := args[0].AsString(), args[1].AsString()
	byteIdx := strings.Index(haystack, needle)
	if byteIdx == -1 {
		return bytecode.NewInt(-1), nil
	}
	// ASCII fast path
	if isASCIIvm(haystack) {
		return bytecode.NewInt(int64(byteIdx)), nil
	}
	// UTF-8: convert byte offset to rune index
	runeIdx := utf8.RuneCountInString(haystack[:byteIdx])
	return bytecode.NewInt(int64(runeIdx)), nil
}

func builtinStrSlice(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 3 {
		return bytecode.Value{}, fmt.Errorf("__str_slice: expected 3 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_slice: arg 0 must be string")
	}
	if args[1].Tag != bytecode.TagInt || args[2].Tag != bytecode.TagInt {
		return bytecode.Value{}, fmt.Errorf("__str_slice: args 1,2 must be int")
	}
	str := args[0].AsString()
	start, end := int(args[1].Int), int(args[2].Int)

	// ASCII fast path
	if isASCIIvm(str) {
		length := len(str)
		if start < 0 {
			start = 0
		}
		if end > length {
			end = length
		}
		if start > end {
			start = end
		}
		return bytecode.NewString(str[start:end]), nil
	}

	// Unicode slow path
	runes := []rune(str)
	length := len(runes)
	if start < 0 {
		start = 0
	}
	if end > length {
		end = length
	}
	if start > end {
		start = end
	}
	return bytecode.NewString(string(runes[start:end])), nil
}

func builtinStrTrim(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__str_trim: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_trim: expected string")
	}
	return bytecode.NewString(strings.TrimSpace(args[0].AsString())), nil
}

func builtinStrUpper(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__str_upper: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_upper: expected string")
	}
	return bytecode.NewString(strings.ToUpper(args[0].AsString())), nil
}

func builtinStrLower(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__str_lower: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_lower: expected string")
	}
	return bytecode.NewString(strings.ToLower(args[0].AsString())), nil
}

// --- String splitting and joining --------------------------------------------

func builtinStrSplit(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__str_split: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString || args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_split: expected strings")
	}
	parts := strings.Split(args[0].AsString(), args[1].AsString())
	elems := make([]bytecode.Value, len(parts))
	for i, p := range parts {
		elems[i] = bytecode.NewString(p)
	}
	return bytecode.NewList(elems), nil
}

func builtinStrChars(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__str_chars: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_chars: expected string")
	}
	runes := []rune(args[0].AsString())
	elems := make([]bytecode.Value, len(runes))
	for i, r := range runes {
		elems[i] = bytecode.NewString(string(r))
	}
	return bytecode.NewList(elems), nil
}

func builtinStrJoin(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__str_join: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__str_join: arg 0 must be list")
	}
	if args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_join: arg 1 must be string")
	}
	elems := args[0].AsList()
	sep := args[1].AsString()
	parts := make([]string, len(elems))
	for i, e := range elems {
		if e.Tag != bytecode.TagString {
			return bytecode.Value{}, fmt.Errorf("__str_join: element %d not a string", i)
		}
		parts[i] = e.AsString()
	}
	return bytecode.NewString(strings.Join(parts, sep)), nil
}

func builtinStrWords(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__str_words: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_words: expected string")
	}
	words := strings.Fields(args[0].AsString())
	elems := make([]bytecode.Value, len(words))
	for i, w := range words {
		elems[i] = bytecode.NewString(w)
	}
	return bytecode.NewList(elems), nil
}

func builtinStrSplitAny(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__str_splitAny: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_splitAny: arg 0 must be string")
	}
	if args[1].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__str_splitAny: arg 1 must be list")
	}
	delimList := args[1].AsList()
	delimRunes := make(map[rune]bool)
	for _, d := range delimList {
		if d.Tag != bytecode.TagString {
			return bytecode.Value{}, fmt.Errorf("__str_splitAny: delimiter must be string")
		}
		for _, r := range d.AsString() {
			delimRunes[r] = true
		}
	}
	parts := strings.FieldsFunc(args[0].AsString(), func(r rune) bool {
		return delimRunes[r]
	})
	elems := make([]bytecode.Value, len(parts))
	for i, p := range parts {
		elems[i] = bytecode.NewString(p)
	}
	return bytecode.NewList(elems), nil
}

// --- String matching ---------------------------------------------------------

func builtinStrStartsWith(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__str_startsWith: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString || args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_startsWith: expected strings")
	}
	return bytecode.NewBool(strings.HasPrefix(args[0].AsString(), args[1].AsString())), nil
}

func builtinStrEndsWith(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__str_endsWith: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString || args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_endsWith: expected strings")
	}
	return bytecode.NewBool(strings.HasSuffix(args[0].AsString(), args[1].AsString())), nil
}

func builtinStrStartsWithIC(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__str_startsWithIC: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString || args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_startsWithIC: expected strings")
	}
	s, prefix := args[0].AsString(), args[1].AsString()
	if len(prefix) > len(s) {
		return bytecode.NewBool(false), nil
	}
	return bytecode.NewBool(strings.EqualFold(s[:len(prefix)], prefix)), nil
}

// --- String replace ----------------------------------------------------------

func builtinStrReplace(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 3 {
		return bytecode.Value{}, fmt.Errorf("__str_replace: expected 3 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString || args[1].Tag != bytecode.TagString || args[2].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_replace: expected strings")
	}
	return bytecode.NewString(strings.ReplaceAll(args[0].AsString(), args[1].AsString(), args[2].AsString())), nil
}

func builtinStrReplaceMany(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__str_replaceMany: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_replaceMany: arg 0 must be string")
	}
	if args[1].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__str_replaceMany: arg 1 must be list")
	}
	s := args[0].AsString()
	pairs := args[1].AsList()
	if len(pairs) == 0 {
		return bytecode.NewString(s), nil
	}
	oldNew := make([]string, 0, len(pairs)*2)
	for i, p := range pairs {
		if p.Tag != bytecode.TagTuple {
			return bytecode.Value{}, fmt.Errorf("__str_replaceMany: element %d must be tuple", i)
		}
		t := p.AsTuple()
		if len(t) != 2 {
			return bytecode.Value{}, fmt.Errorf("__str_replaceMany: element %d must be 2-tuple", i)
		}
		if t[0].Tag != bytecode.TagString || t[1].Tag != bytecode.TagString {
			return bytecode.Value{}, fmt.Errorf("__str_replaceMany: element %d must contain strings", i)
		}
		oldNew = append(oldNew, t[0].AsString(), t[1].AsString())
	}
	replacer := strings.NewReplacer(oldNew...)
	return bytecode.NewString(replacer.Replace(s)), nil
}

// --- Character operations ----------------------------------------------------

func builtinStrCharAt(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("__str_charAt: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_charAt: arg 0 must be string")
	}
	if args[1].Tag != bytecode.TagInt {
		return bytecode.Value{}, fmt.Errorf("__str_charAt: arg 1 must be int")
	}
	runes := []rune(args[0].AsString())
	idx := int(args[1].Int)
	if idx < 0 || idx >= len(runes) {
		return bytecode.Value{}, fmt.Errorf("__str_charAt: index %d out of bounds for string of length %d", idx, len(runes))
	}
	return bytecode.NewString(string(runes[idx])), nil
}

func builtinStrCharCode(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__str_charCode: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_charCode: expected string")
	}
	runes := []rune(args[0].AsString())
	if len(runes) != 1 {
		return bytecode.Value{}, fmt.Errorf("__str_charCode: expected single-character string, got %d characters", len(runes))
	}
	return bytecode.NewInt(int64(runes[0])), nil
}

// --- String encoding ---------------------------------------------------------

func builtinStrDecodeQP(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__str_decodeQP: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__str_decodeQP: expected string")
	}
	s := args[0].AsString()
	var buf strings.Builder
	buf.Grow(len(s))

	i := 0
	for i < len(s) {
		if s[i] != '=' {
			buf.WriteByte(s[i])
			i++
			continue
		}
		remaining := len(s) - i - 1
		if remaining < 1 {
			buf.WriteByte('=')
			i++
			continue
		}
		if s[i+1] == '\n' {
			i += 2
			continue
		}
		if s[i+1] == '\r' {
			if remaining >= 2 && s[i+2] == '\n' {
				i += 3
			} else {
				i += 2
			}
			continue
		}
		if remaining >= 2 {
			hexStr := s[i+1 : i+3]
			decoded, hexErr := hex.DecodeString(hexStr)
			if hexErr == nil {
				buf.Write(decoded)
				i += 3
				continue
			}
		}
		buf.WriteByte('=')
		i++
	}
	return bytecode.NewString(buf.String()), nil
}

func builtinEscapeXml(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__escapeXml: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__escapeXml: expected string")
	}
	return bytecode.NewString(html.EscapeString(args[0].AsString())), nil
}

// --- String conversion (type conversion, NOT parsing) ------------------------

func builtinStringIntToStr(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__string_intToStr: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagInt {
		return bytecode.Value{}, fmt.Errorf("__string_intToStr: expected int")
	}
	return bytecode.NewString(strconv.FormatInt(args[0].Int, 10)), nil
}

func builtinStringFloatToStr(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__string_floatToStr: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagFloat {
		return bytecode.Value{}, fmt.Errorf("__string_floatToStr: expected float")
	}
	return bytecode.NewString(strconv.FormatFloat(args[0].Flt, 'g', -1, 64)), nil
}

// --- String parsing (returns Option ADT) -------------------------------------
//
// Option is defined as: type Option[a] = Some(a) | None
// Constructor tag ordinals (source order): Some=0, None=1

const (
	optionTagSome = 0
	optionTagNone = 1
)

func builtinStringToInt(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__stringToInt: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__stringToInt: expected string")
	}
	n, err := strconv.ParseInt(args[0].AsString(), 10, 64)
	if err != nil {
		return bytecode.NewADT(optionTagNone, nil), nil
	}
	return bytecode.NewADT(optionTagSome, []bytecode.Value{bytecode.NewInt(n)}), nil
}

func builtinStringToFloat(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__stringToFloat: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("__stringToFloat: expected string")
	}
	s := args[0].AsString()
	// Reject underscores — Go's ParseFloat accepts them silently
	if strings.ContainsRune(s, '_') {
		return bytecode.NewADT(optionTagNone, nil), nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return bytecode.NewADT(optionTagNone, nil), nil
	}
	return bytecode.NewADT(optionTagSome, []bytecode.Value{bytecode.NewFloat(f)}), nil
}

// --- helpers -----------------------------------------------------------------

// isASCIIvm returns true if every byte in s is < 128 (pure ASCII).
func isASCIIvm(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 128 {
			return false
		}
	}
	return true
}
