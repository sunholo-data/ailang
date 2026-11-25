package smt

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// Opaque builtins (M-VERIFY-SOUNDNESS-COVERAGE): Go-backed pure builtins with
// no SMT theory — JSON text, SHA-256, XML escaping. Each call is an
// uninterpreted function of its arguments, which is sound because the builtin
// is pure: equal arguments give equal results, and nothing else is assumed
// beyond the facts listed below, each a property of the Go implementation for
// every input. A contract that needs more of the value than that is refuted
// only by a model over the uninterpreted symbol, which is inconclusive until
// witness replay decides it (uninterpretedInconclusive), as for $str_split.
//
// Before, every function reaching one of them (std/json.encode/decode,
// std/crypto.sha256Hex, std/xml.escapeXml) was skipped, although most of
// their contracts say nothing about the value (sseFrame's "data: " prefix) or
// only its shape (a JSON object starts with "{", a SHA-256 digest is 64 hex
// characters).
type opaqueBuiltin struct {
	symbol string
	params []string // SMT sorts
	result string
	facts  func(r string, a []string) []string
}

var opaqueBuiltins = map[string]opaqueBuiltin{
	// internal/builtins/json_encode.go
	"_json_encode": {"$json_encode", []string{"Json"}, "String", jsonEncodeFacts},
	// internal/builtins/json_decode.go: no fact; Ok and Err are both possible
	"_json_decode": {"$json_decode", []string{"String"}, "(Result Json String)", nil},
	// internal/builtins/crypto.go: hex.EncodeToString of a 32-byte digest
	"_crypto_sha256hex": {"$sha256hex", []string{"String"}, "String", func(r string, _ []string) []string {
		return []string{fmt.Sprintf("(= (str.len %s) 64)", r)}
	}},
	// internal/builtins/xml_serialize.go: html.EscapeString
	"_escapeXml": {"$escape_xml", []string{"String"}, "String", escapeXmlFacts},
}

// encodeOpaqueBuiltin encodes a call to one of opaqueBuiltins. ok is false
// for any other builtin.
func encodeOpaqueBuiltin(name string, args []core.CoreExpr) (string, bool, error) {
	b, isOpaque := opaqueBuiltins[name]
	if !isOpaque {
		return "", false, nil
	}
	if len(args) != len(b.params) {
		return "", true, fmt.Errorf("%w: %s expects %d args, got %d", ErrUnsupportedConstruct, name, len(b.params), len(args))
	}
	a := make([]string, len(args))
	for i, x := range args {
		e, err := EncodeExpr(x)
		if err != nil {
			return "", true, err
		}
		a[i] = e
	}
	decl := fmt.Sprintf("(declare-fun %s (%s) %s)", b.symbol, strings.Join(b.params, " "), b.result)
	if err := useUninterpreted(b.symbol, decl); err != nil {
		return "", true, err
	}
	call := fmt.Sprintf("(%s %s)", b.symbol, strings.Join(a, " "))
	if b.facts != nil {
		for _, fact := range b.facts(call, a) {
			addCallFact(fact)
		}
	}
	return call, true, nil
}

// opaqueBuiltinSorts are the declared (non-primitive) sorts a call to name
// needs, for the per-function demand filter.
func opaqueBuiltinSorts(name string) []string {
	b, ok := opaqueBuiltins[name]
	if !ok {
		return nil
	}
	var out []string
	for _, s := range append(append([]string(nil), b.params...), b.result) {
		for _, ref := range sortRefPattern.FindAllString(s, -1) {
			if !primitiveSorts[ref] {
				out = append(out, ref)
			}
		}
	}
	return out
}

// placeUninterpretedDeclarations splits the declare-fun lines of the
// uninterpreted symbols used: those over primitive sorts go in front of every
// declaration (as before), those over a datatype must follow its declaration.
// A datatype this function does not declare is a skip naming the builtin,
// never a script Z3 rejects.
func placeUninterpretedDeclarations(declared map[string]bool) (front, afterTypes []string, err error) {
	for _, s := range usedUninterpreted() {
		decl := activeUninterpreted[s]
		var missing []string
		needsType := false
		for _, ref := range sortRefPattern.FindAllString(strings.TrimPrefix(decl, "(declare-fun "+s), -1) {
			if primitiveSorts[ref] {
				continue
			}
			needsType = true
			if !declared[ref] {
				missing = append(missing, ref)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			return nil, nil, fmt.Errorf("%w: %s needs sort %s, which this function does not declare", ErrUnresolvableTypes, opaqueBuiltinName(s), strings.Join(missing, ", "))
		}
		if needsType {
			afterTypes = append(afterTypes, decl)
		} else {
			front = append(front, decl)
		}
	}
	return front, afterTypes, nil
}

// opaqueBuiltinName is the builtin behind an uninterpreted symbol, for messages.
func opaqueBuiltinName(symbol string) string {
	for name, b := range opaqueBuiltins {
		if b.symbol == symbol {
			return name
		}
	}
	return symbol
}

// stdJSONVariants is std/json's Json, constructor → field sorts. The shape
// facts name its testers and selectors, so they are emitted only when the
// declared Json is this one and no other declared type reuses a name.
var stdJSONVariants = map[string]string{
	"JNull": "", "JBool": "Bool", "JNumber": "Real", "JString": "String",
	"JArray": "(Seq Json)", "JObject": "(Seq Record_key_value)", "JInt": "Int",
}

func isStdJSONDeclared() bool {
	if activeADTContext == nil {
		return false
	}
	variants := activeADTContext.adts["Json"]
	if len(variants) != len(stdJSONVariants) {
		return false
	}
	for _, v := range variants {
		want, ok := stdJSONVariants[v.Name]
		if !ok || len(v.Fields) > 1 || (want == "") != (len(v.Fields) == 0) || (want != "" && v.Fields[0].Sort != want) {
			return false
		}
	}
	for owner, vs := range activeADTContext.adts {
		if owner == "Json" {
			continue
		}
		for _, v := range vs {
			if _, clash := stdJSONVariants[v.Name]; clash {
				return false
			}
		}
	}
	return true
}

// jsonEncodeFacts: the text's outer shape follows the top constructor
// (encodeValue in internal/builtins/json_encode.go), and is never empty.
func jsonEncodeFacts(r string, a []string) []string {
	facts := []string{fmt.Sprintf("(>= (str.len %s) 1)", r)}
	if !isStdJSONDeclared() {
		return facts
	}
	j := a[0]
	is := func(ctor string) string { return fmt.Sprintf("((_ is %s) %s)", ctor, j) }
	bracketed := func(ctor, open, closed string) []string {
		o, c := encodeStringLiteral(open), encodeStringLiteral(closed)
		return []string{
			fmt.Sprintf("(=> %s (and (str.prefixof %s %s) (str.suffixof %s %s) (>= (str.len %s) 2)))", is(ctor), o, r, c, r, r),
			fmt.Sprintf("(=> (and %s (= (seq.len (%s_0 %s)) 0)) (= %s %s))", is(ctor), ctor, j, r, encodeStringLiteral(open+closed)),
		}
	}
	quote := encodeStringLiteral(`"`)
	facts = append(facts,
		fmt.Sprintf(`(=> %s (= %s "null"))`, is("JNull"), r),
		fmt.Sprintf(`(=> %s (= %s (ite (JBool_0 %s) "true" "false")))`, is("JBool"), r, j),
		fmt.Sprintf("(=> %s (and (str.prefixof %s %s) (str.suffixof %s %s) (>= (str.len %s) 2)))", is("JString"), quote, r, quote, r, r),
	)
	facts = append(facts, bracketed("JArray", "[", "]")...)
	return append(facts, bracketed("JObject", "{", "}")...)
}

// escapeXmlFacts: html.EscapeString replaces each of & ' < > " with a longer
// entity (&amp; &#39; &lt; &gt; &#34;) and copies every other rune.
func escapeXmlFacts(r string, a []string) []string {
	s := a[0]
	special := []string{"&", "'", "<", ">", `"`}
	var inS []string
	facts := []string{fmt.Sprintf("(>= (str.len %s) (str.len %s))", r, s)}
	for _, c := range special {
		lit := encodeStringLiteral(c)
		inS = append(inS, fmt.Sprintf("(str.contains %s %s)", s, lit))
		if c != "&" {
			facts = append(facts, fmt.Sprintf("(not (str.contains %s %s))", r, lit))
		}
	}
	return append(facts, fmt.Sprintf("(=> (not (or %s)) (= %s %s))", strings.Join(inS, " "), r, s))
}
