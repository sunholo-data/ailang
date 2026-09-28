package main

import (
	"regexp"
	"strconv"
	"strings"
)

// Recovering a readable subject when the task's own title is a hash.

// opaqueToken is a run of 16+ hex characters: a content hash or id, not words.
var opaqueToken = regexp.MustCompile(`\b[0-9a-f]{16,}\b`)

// isOpaqueTitle reports whether a title names an id rather than the work.
func isOpaqueTitle(title string) bool {
	return title == "" || opaqueToken.MatchString(title)
}

// stripHandoffWrapping removes the "Handoff: " prefixes (one per stage) and the
// trailing " (approved)" the approval path appends to handoff titles.
func stripHandoffWrapping(title string) string {
	t := strings.TrimSpace(title)
	for strings.HasPrefix(t, "Handoff: ") {
		t = strings.TrimSpace(strings.TrimPrefix(t, "Handoff: "))
	}
	return strings.TrimSpace(strings.TrimSuffix(t, " (approved)"))
}

// requestSubject derives a subject from the ORIGINAL request inside a directive.
//
// A handoff directive carries the root request after "Original Request:",
// truncated at 500 characters — usually mid-JSON, so it cannot be decoded. The
// human field is read tolerantly (jsonFieldPrefix), its first line taken, and
// mail prefixes ("Subject:", "Re:", "Fwd:") removed: Daneel requests arrive as
// emails. "" when no usable subject exists. A pure function of the directive,
// like every other subject rule here.
func requestSubject(directive string) string {
	req := strings.TrimSpace(directive)
	if i := strings.LastIndex(req, "Original Request:"); i >= 0 {
		req = strings.TrimSpace(req[i+len("Original Request:"):])
	}
	if strings.HasPrefix(req, "{") {
		text := subjectFromJSON(req)
		if text == "" {
			for _, k := range jsonSubjectKeys {
				if text = jsonFieldPrefix(req, k); text != "" {
					break
				}
			}
		}
		req = text
	}
	for _, line := range strings.Split(req, "\n") {
		line = stripRequestBoilerplate(stripMailPrefixes(strings.Join(strings.Fields(line), " ")))
		if line != "" {
			return truncateOnWord(line, directiveSubjectMax)
		}
	}
	return ""
}

// jsonFieldPrefix reads a string field from possibly-truncated JSON: the value
// up to its closing quote, or to the end of input when the cut fell inside it.
func jsonFieldPrefix(s, key string) string {
	re := regexp.MustCompile(`"` + regexp.QuoteMeta(key) + `"\s*:\s*"((?:[^"\\]|\\.)*)`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	raw := strings.TrimSuffix(m[1], `\`) // a cut can split an escape
	if v, err := strconv.Unquote(`"` + raw + `"`); err == nil {
		return v
	}
	return strings.ReplaceAll(raw, `\n`, "\n")
}

// stripMailPrefixes removes leading "Subject:", "Re:", "Fwd:" (any case, repeated).
func stripMailPrefixes(s string) string {
	for {
		lower := strings.ToLower(s)
		trimmed := false
		for _, p := range []string{"subject:", "re:", "fwd:", "fw:"} {
			if strings.HasPrefix(lower, p) {
				s = strings.TrimSpace(s[len(p):])
				trimmed = true
				break
			}
		}
		if !trimmed {
			return s
		}
	}
}

// requestLeadIns are the stock openings of templated requests ("Create a design
// document for a salvage/retry policy …"). They say what KIND of work it is —
// which the `[agent] <id>:` prefix and the stage label already say — and spend
// half the title budget before the subject starts.
var requestLeadIns = []string{
	"create a design document for ",
	"create a design doc for ",
	"write a design document for ",
	"write a design doc for ",
}

// stripRequestBoilerplate removes one stock lead-in (any case), then an article.
func stripRequestBoilerplate(s string) string {
	lower := strings.ToLower(s)
	for _, p := range requestLeadIns {
		if strings.HasPrefix(lower, p) {
			rest := strings.TrimSpace(s[len(p):])
			for _, a := range []string{"a ", "an ", "the "} {
				if strings.HasPrefix(strings.ToLower(rest), a) {
					rest = rest[len(a):]
					break
				}
			}
			if rest == "" {
				return s
			}
			return strings.ToUpper(rest[:1]) + rest[1:]
		}
	}
	return s
}
