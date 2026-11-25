package main

import (
	"regexp"
	"strings"
)

// A skip reason is a chain: each layer of the encoder that the failure
// unwound through adds a prefix ("resolving cross-function calls: ", "let
// body: ", "calls user function "kv" that is not SMT-encodable in this
// context: " …), and the construct that actually blocks the proof is at the
// end. blockers strips the layers and splits a list of rejections, so a skip is
// blamed on what would have to change for it to encode, not on the helper it
// happened to surface in.

// sortClosure wraps a whole reason in parentheses; it is stripped first.
const sortClosure = "SMT declaration closure could not resolve a required sort ("

var chainLayer = regexp.MustCompile(`^(?:` + strings.Join([]string{
	`unsupported construct in SMT encoding: `,
	`unresolvable SMT declaration types: `,
	`resolving cross-function calls: `,
	`cannot encode (?:function body|ensures clause|requires clause): `,
	`let (?:body|value): `,
	`constructor arg \d+: `,
	`match pattern: `,
	`callee "[^"]*": `,
	`the specialised \w+ chain does not encode: `,
	`encoding unrolled body at depth \d+: `,
	`unrolling recursive function: `,
	`calls user function "[^"]*" that is not SMT-encodable in this context: `,
}, "|") + `)`)

// mutualRecursion is one bucket for the two ways a helper cycle is reported:
// topoSort's "circular function call detected: a → b → a", and a cycle through
// the verified function itself, which ends the helper chain.
const mutualRecursion = "mutually recursive helpers"

// anonymousRecord buckets every "record construction: no declared record type
// has fields [...]", whatever the fields.
const anonymousRecord = "record construction with no declared record type"

// rootReason strips every encoder layer off the front of a reason.
func rootReason(reason string) string {
	s := strings.TrimSpace(reason)
	if strings.HasPrefix(s, sortClosure) {
		s = strings.TrimSuffix(strings.TrimPrefix(s, sortClosure), ")")
	}
	for {
		loc := chainLayer.FindStringIndex(s)
		if loc == nil {
			return s
		}
		s = s[loc[1]:]
	}
}

// blockers names the distinct constructs a skip is blamed on, normalised so
// the same limitation hit by different functions lands in one bucket. A
// callee-signature rejection keeps the type's head, since that is the lever.
func blockers(reason string) []string {
	var out []string
	seen := map[string]bool{}
	for _, part := range strings.Split(rootReason(reason), "; ") {
		part = rootReason(part)
		if part == "" {
			continue
		}
		b := normaliseReason(part)
		switch m := calleeRe.FindStringSubmatch(part); {
		case m != nil:
			b = "callee signature uses an unencodable type: " + typeHead(m[2])
		case strings.HasPrefix(part, "record construction: no declared record type has fields"):
			// One lever (an anonymous record has no sort) whatever the fields.
			b = anonymousRecord
		case strings.HasPrefix(part, "circular function call detected"),
			strings.Contains(part, "the helpers are mutually recursive"):
			b = mutualRecursion
		}
		if !seen[b] {
			seen[b] = true
			out = append(out, b)
		}
	}
	return out
}

// typeHead folds a type to its constructor: "Result[a, e]" → "Result",
// "(int, string)" → "tuple", "a" → "type variable".
func typeHead(t string) string {
	switch {
	case strings.HasPrefix(t, "("):
		return "tuple"
	case len(t) == 1 && t >= "a" && t <= "z":
		return "type variable"
	}
	if i := strings.IndexAny(t, "[ "); i > 0 {
		return t[:i]
	}
	return t
}

// blockerCounts counts, over skipped results, each blocker a skip is blamed on
// (any) and the skips it is the only blocker of (sole): removing a sole
// blocker would let that result encode, as far as the reason can tell.
func blockerCounts(rows []row) (anyOf, sole map[string]int) {
	anyOf, sole = map[string]int{}, map[string]int{}
	for _, r := range rows {
		if r.Status != "skipped" {
			continue
		}
		bs := r.Blockers
		if bs == nil {
			bs = blockers(r.Reason) // a snapshot banked before blockers were recorded
		}
		for _, b := range bs {
			anyOf[b]++
		}
		if len(bs) == 1 {
			sole[bs[0]]++
		}
	}
	return anyOf, sole
}
