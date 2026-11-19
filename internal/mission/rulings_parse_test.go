package mission

import (
	"fmt"
	"strings"
	"testing"
)

// rulingsSeedText is design_docs/planned/m-fleet-rulings-ledger.md §1, verbatim.
const rulingsSeedText = `<!-- decision-ledger:start -->
| ID | Status | Ruling | Evidence |
|---|---|---|---|
| D-FR-1 | RESOLVED | World runs the shared driver; it is not a fork. Text calling it a fork is stale. | Transcribed from D-FLEET-25 (fleet-mission.md, Mark attended 2026-11-16). |
| D-FR-2 | RESOLVED | generator≠judge guard is a PREFERENCE at vendor level, never a blocker (model-level self-review stays refused). | Transcribed from V1 D-62/D-63 (v1-mission.md, Mark attended 2026-09-08); skill fixed in 349e311b9. |
<!-- decision-ledger:end -->

<!-- fleet-rulings-detectors:start -->
D-FR-1 forbid registry "WORLD IS A FORK"
D-FR-1 forbid charters "WORLD IS A FORK"
D-FR-2 forbid skills "generator≠judge guard (HARD"
<!-- fleet-rulings-detectors:end -->
`

const (
	rowResolved1 = "| D-FR-1 | RESOLVED | ruling one | evidence one |"
	rowResolved2 = "| D-FR-2 | RESOLVED | ruling two | evidence two |"
	rowOpen2     = "| D-FR-2 | OPEN | ruling two | evidence two |"
)

// rulingsDoc builds a ledger document from raw ledger-block rows and raw detector lines.
func rulingsDoc(rowLines, detLines []string) string {
	var b strings.Builder
	b.WriteString(ledgerStartMarker + "\n| ID | Status | Ruling | Evidence |\n|---|---|---|---|\n")
	for _, r := range rowLines {
		b.WriteString(r + "\n")
	}
	b.WriteString(ledgerEndMarker + "\n\n" + detectorsStartMarker + "\n")
	for _, d := range detLines {
		b.WriteString(d + "\n")
	}
	b.WriteString(detectorsEndMarker + "\n")
	return b.String()
}

// lineOf returns the 1-based line of the first line whose trimmed text equals want.
func lineOf(t *testing.T, doc, want string) int {
	t.Helper()
	for i, l := range strings.Split(doc, "\n") {
		if strings.TrimSpace(l) == want {
			return i + 1
		}
	}
	t.Fatalf("line %q not found in doc", want)
	return 0
}

func problemLines(ps []rulingProblem) map[int]int {
	m := map[int]int{}
	for _, p := range ps {
		m[p.Line]++
	}
	return m
}

func TestRulingsParse_SeedShapeIsValid(t *testing.T) {
	l, ps := parseRulingsLedger([]byte(rulingsSeedText))
	if len(ps) != 0 {
		t.Fatalf("seed ledger has problems: %s", formatRulingProblems(ps))
	}
	if l == nil {
		t.Fatal("seed ledger is nil with no problems")
	}
	if len(l.Rows) != 2 {
		t.Errorf("rows = %d, want 2 (header and separator must not count)", len(l.Rows))
	}
	if len(l.Forbids) != 3 {
		t.Fatalf("forbids = %d, want 3", len(l.Forbids))
	}
	if got := l.Forbids[2].Literal; got != "generator≠judge guard (HARD" {
		t.Errorf("Forbids[2].Literal = %q", got)
	}
}

func TestRulingsParse_MalformedDetectorsListEveryLine(t *testing.T) {
	valid := `D-FR-1 forbid registry "abcdefgh"`
	bad := []string{
		`D-FR-x forbid skills "abcdefgh"`,
		`D-FR-1 forbid skill "abcdefgh"`,
		`D-FR-1 forbid skills "abcdefg"`,
		`D-FR-1 supersedes D-FR-2`,
		`D-FR-9 forbid skills "abcdefgh"`,
		`D-FR-1 require skills "abcdefgh"`,
	}
	doc := rulingsDoc([]string{rowResolved1, rowResolved2}, append([]string{valid}, bad...))
	l, ps := parseRulingsLedger([]byte(doc))
	if l != nil {
		t.Error("ledger returned despite malformed detectors")
	}
	want := map[int]int{}
	for _, b := range bad {
		want[lineOf(t, doc, b)]++
	}
	got := problemLines(ps)
	if len(ps) != len(bad) {
		t.Errorf("problems = %d, want %d: %s", len(ps), len(bad), formatRulingProblems(ps))
	}
	for line := range want {
		if got[line] != 1 {
			t.Errorf("line %d: %d problems, want 1: %s", line, got[line], formatRulingProblems(ps))
		}
	}
	for line := range got {
		if want[line] == 0 {
			t.Errorf("unexpected problem on line %d: %s", line, formatRulingProblems(ps))
		}
	}
}

func TestRulingsParse_MalformedRowFailsWholeLedger(t *testing.T) {
	cases := map[string]string{
		"three-column row": "| D-FR-3 | RESOLVED |",
		"unknown status":   "| D-FR-3 | PARKED | r | e |",
		"prose line":       "some prose inside the ledger",
	}
	for name, row := range cases {
		t.Run(name, func(t *testing.T) {
			doc := rulingsDoc([]string{rowResolved1, row}, []string{`D-FR-1 forbid registry "abcdefgh"`})
			l, ps := parseRulingsLedger([]byte(doc))
			if l != nil {
				t.Fatal("partial ledger returned")
			}
			if len(ps) != 1 || ps[0].Line != lineOf(t, doc, row) {
				t.Fatalf("want one problem on the row line, got: %s", formatRulingProblems(ps))
			}
		})
	}
}

func genDetectors(n int) []string {
	out := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, fmt.Sprintf(`D-FR-1 forbid registry "literal-%04d"`, i))
	}
	return out
}

func TestRulingsParse_DetectorCap(t *testing.T) {
	// 64 real lines, with 3 comments and 2 blank lines interleaved: must pass.
	dets := genDetectors(64)
	withNoise := []string{"# c1", dets[0], "", dets[1], "# c2", dets[2], dets[3], "", "# c3"}
	withNoise = append(withNoise, dets[4:]...)
	doc := rulingsDoc([]string{rowResolved1}, withNoise)
	if _, ps := parseRulingsLedger([]byte(doc)); len(ps) != 0 {
		t.Fatalf("64 detectors + comments/blank lines must pass: %s", formatRulingProblems(ps))
	}

	// 65 real lines: exactly one cap problem, on the file line of the 65th.
	dets = genDetectors(65)
	doc = rulingsDoc([]string{rowResolved1}, dets)
	_, ps := parseRulingsLedger([]byte(doc))
	if len(ps) != 1 {
		t.Fatalf("65 detectors: want exactly 1 problem, got: %s", formatRulingProblems(ps))
	}
	line := lineOf(t, doc, dets[64])
	if ps[0].Line != line || !strings.Contains(ps[0].Why, "64-line cap") {
		t.Fatalf("problem = %+v, want line %d with 64-line cap", ps[0], line)
	}
	if detail := formatRulingProblems(ps); !strings.Contains(detail, fmt.Sprintf("line %d:", line)) {
		t.Errorf("formatted detail %q lacks line %d", detail, line)
	}
}

func TestRulingsParse_LiteralBytes(t *testing.T) {
	cases := []struct {
		name, literal string
		wantOK        bool
	}{
		{"8 bytes, 6 runes", "≠abcde", true},
		{"7 bytes", "abcdefg", false},
		{"200 bytes", strings.Repeat("a", 200), true},
		{"201 bytes", strings.Repeat("a", 201), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := rulingsDoc([]string{rowResolved1}, []string{`D-FR-1 forbid registry "` + c.literal + `"`})
			_, ps := parseRulingsLedger([]byte(doc))
			if c.wantOK && len(ps) != 0 {
				t.Errorf("want accepted, got: %s", formatRulingProblems(ps))
			}
			if !c.wantOK && len(ps) != 1 {
				t.Errorf("want rejected, got %d problems", len(ps))
			}
		})
	}
	t.Run("embedded quote", func(t *testing.T) {
		doc := rulingsDoc([]string{rowResolved1}, []string{`D-FR-1 forbid registry "ab"cdefghij"`})
		_, ps := parseRulingsLedger([]byte(doc))
		if len(ps) != 1 || !strings.Contains(ps[0].Why, "double quote") {
			t.Errorf("want one double-quote problem, got: %s", formatRulingProblems(ps))
		}
	})
}

func TestRulingsParse_BlockStructure(t *testing.T) {
	base := rulingsDoc([]string{rowResolved1}, []string{`D-FR-1 forbid registry "abcdefgh"`})
	cases := map[string]struct {
		doc, wantSub string
	}{
		"missing detector start": {
			strings.Replace(base, detectorsStartMarker+"\n", "", 1),
			"expected exactly one <!-- fleet-rulings-detectors:start -->",
		},
		"duplicated ledger start": {
			strings.Replace(base, ledgerStartMarker, ledgerStartMarker+"\n"+ledgerStartMarker, 1),
			"expected exactly one <!-- decision-ledger:start --> (found 2)",
		},
		"ledger end before start": {
			func() string {
				s := strings.Replace(base, ledgerStartMarker, "\x02", 1)
				s = strings.Replace(s, ledgerEndMarker, ledgerStartMarker, 1)
				return strings.Replace(s, "\x02", ledgerEndMarker, 1)
			}(),
			"must come before",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			l, ps := parseRulingsLedger([]byte(c.doc))
			if l != nil {
				t.Fatal("ledger returned despite block-structure problem")
			}
			if len(ps) == 0 || !strings.Contains(formatRulingProblems(ps), c.wantSub) {
				t.Fatalf("want problem containing %q, got: %s", c.wantSub, formatRulingProblems(ps))
			}
		})
	}
}

func TestRulingsActive_Supersession(t *testing.T) {
	forbid1 := `D-FR-1 forbid registry "abcdefgh"`
	cases := []struct {
		name        string
		rows        []string
		dets        []string
		want        []string
		wantForbids []string
	}{
		{"superseder RESOLVED", []string{rowResolved1, rowResolved2}, []string{"D-FR-2 supersedes D-FR-1", forbid1}, []string{"D-FR-2"}, nil},
		{"superseder OPEN", []string{rowResolved1, rowOpen2}, []string{"D-FR-2 supersedes D-FR-1", forbid1}, []string{"D-FR-1"}, []string{"D-FR-1"}},
		{"target OPEN alone", []string{"| D-FR-1 | OPEN | r | e |"}, []string{forbid1}, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, ps := parseRulingsLedger([]byte(rulingsDoc(c.rows, c.dets)))
			if len(ps) != 0 {
				t.Fatalf("unexpected problems: %s", formatRulingProblems(ps))
			}
			got := l.active()
			if len(got) != len(c.want) {
				t.Fatalf("active = %v, want %v", got, c.want)
			}
			for _, id := range c.want {
				if !got[id] {
					t.Errorf("active = %v, want %v", got, c.want)
				}
			}
			var gotForbids []string
			for _, d := range l.activeForbids() {
				gotForbids = append(gotForbids, d.ID)
			}
			if strings.Join(gotForbids, ",") != strings.Join(c.wantForbids, ",") {
				t.Errorf("activeForbids = %v, want %v", gotForbids, c.wantForbids)
			}
		})
	}
}

func TestRulingsParse_SupersedesOrderAndDuplicates(t *testing.T) {
	dets := []string{"D-FR-1 supersedes D-FR-1", "D-FR-1 supersedes D-FR-2"}
	doc := rulingsDoc([]string{rowResolved1, rowResolved2}, dets)
	l, ps := parseRulingsLedger([]byte(doc))
	if l != nil {
		t.Error("ledger returned despite self/forward supersedes")
	}
	if len(ps) != 2 {
		t.Errorf("want 2 problems (self, forward), got: %s", formatRulingProblems(ps))
	}
	for _, d := range dets {
		if !strings.Contains(formatRulingProblems(ps), "LOWER-numbered") {
			t.Errorf("missing LOWER-numbered problem for %q", d)
		}
	}

	dup := rulingsDoc([]string{rowResolved1, rowResolved1}, nil)
	if _, ps := parseRulingsLedger([]byte(dup)); len(ps) != 1 || !strings.Contains(ps[0].Why, "duplicate ID") {
		t.Errorf("want one duplicate-ID problem, got: %s", formatRulingProblems(ps))
	}
}
