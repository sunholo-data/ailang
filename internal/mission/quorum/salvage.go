package quorum

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// SalvagedVerdict contains complete tokens recovered from an invalid response.
// A reject is a veto; a pass is audit only. Neither is a validated vote or seat.
type SalvagedVerdict struct {
	Verdict            Verdict `json:"verdict"`
	StrongestObjection string  `json:"strongest_objection,omitempty"`
	Offset             int64   `json:"offset"` // -1 for syntactically valid gate violations
}

// salvageVerdict reads only strict tokens preceding the break, never repairs
// JSON, and never inspects nested verdicts or searches past leading prose.
func salvageVerdict(raw string) *SalvagedVerdict {
	b := []byte(stripCodeFences(strings.TrimSpace(raw)))
	dec := json.NewDecoder(strings.NewReader(string(b)))
	first, err := dec.Token()
	if err != nil || first != json.Delim('{') {
		return nil
	}
	got := &SalvagedVerdict{Offset: -1}
	depth := 1
	expectKey := true
	key := ""
	var pending *string
	confirm := func() {
		if pending != nil {
			got.StrongestObjection = *pending
			pending = nil
		}
	}
	for depth > 0 {
		token, err := dec.Token()
		if err != nil {
			break
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{', '[':
				if depth == 1 {
					expectKey = true
				}
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					confirm()
				}
			}
			continue
		}
		if depth != 1 {
			continue
		}
		if expectKey {
			confirm()
			key, _ = token.(string)
			expectKey = false
			continue
		}
		expectKey = true
		value, ok := token.(string)
		if !ok {
			continue
		}
		switch key {
		case "verdict":
			if value == string(VerdictReject) || (value == string(VerdictPass) && got.Verdict != VerdictReject) {
				got.Verdict = Verdict(value)
			}
		case "strongest_objection":
			pending = &value
		}
	}
	if got.Verdict == "" {
		return nil
	}
	// Token() uses different EOF wording/offsets. Preserve Unmarshal's offset,
	// the same one emitted by ParseReviewResult, including trailing JSON errors.
	var decoded any
	var syntax *json.SyntaxError
	if err := json.Unmarshal(b, &decoded); errors.As(err, &syntax) {
		got.Offset = syntax.Offset
	}
	return got
}

func recordInvalid(out *ReviewerOutcome, raw string) {
	out.AbsentReason = ReasonInvalid
	out.Salvaged = salvageVerdict(raw)
}

const diagHalfWindow = 120

// describeJSONError locates syntax breaks in the actual bytes parsed. Bounds
// snap backwards to rune starts so the quoted window contains valid UTF-8.
func describeJSONError(err error, b []byte) string {
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) {
		return fmt.Sprintf("%v (raw: %.200q)", err, b)
	}
	offset := int(min(max(syntax.Offset, 0), int64(len(b))))
	a, z := max(0, offset-diagHalfWindow), min(len(b), offset+diagHalfWindow)
	for a > 0 && a < len(b) && !utf8.RuneStart(b[a]) {
		a--
	}
	for z > 0 && z < len(b) && !utf8.RuneStart(b[z]) {
		z--
	}
	return fmt.Sprintf("%v at byte %d of %d (window [%d:%d]: %q)", err, syntax.Offset, len(b), a, z, b[a:z])
}
