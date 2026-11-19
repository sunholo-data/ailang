package mission

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// M-FLEET-RULINGS-LEDGER M1: a pure parser for design_docs/fleet-rulings.md.
// It returns the ledger or the complete list of problems, never a partial ledger.
const (
	rulingsLedgerRel          = "design_docs/fleet-rulings.md"
	rulingsMission            = "rulings" // Finding.Mission for fleet-wide findings (not "fleet": that is a mission name)
	rulingsMaxDetectors       = 64
	rulingsMinLiteral         = 8   // bytes
	rulingsMaxLiteral         = 200 // bytes
	rulingsMaxFileBytes       = 4 << 20
	rulingsMaxHitsPerDetector = 20
	ledgerStartMarker         = "<!-- decision-ledger:start -->"
	ledgerEndMarker           = "<!-- decision-ledger:end -->"
	detectorsStartMarker      = "<!-- fleet-rulings-detectors:start -->"
	detectorsEndMarker        = "<!-- fleet-rulings-detectors:end -->"
	rulingsLinkText           = "fleet-rulings.md"
)

type rulingRow struct {
	ID     string
	Num    int
	Status string // "OPEN" | "RESOLVED"
	Line   int
}

type forbidDetector struct {
	ID      string
	Scope   string // "registry" | "skills" | "charters"
	Literal string
	Line    int
}

type supersession struct {
	By     string
	Target string
	Line   int
}

type rulingsLedger struct {
	Rows       map[string]rulingRow
	Forbids    []forbidDetector // ledger order
	Supersedes []supersession
}

// rulingProblem is one malformation; Line is 1-based, 0 = whole-file problem.
type rulingProblem struct {
	Line int
	Why  string
}

var (
	rulingIDRe      = regexp.MustCompile(`^D-FR-([1-9][0-9]*)$`)
	rulingSepCellRe = regexp.MustCompile(`^:?-+:?$`)
)

// rulingNum returns the number of a well-formed D-FR-<n> ID.
func rulingNum(id string) (int, bool) {
	m := rulingIDRe.FindStringSubmatch(id)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return n, true
}

// blockMarkerLine returns the 0-based index of the single line containing marker,
// or appends a block problem and returns -1.
func blockMarkerLine(lines []string, marker string, problems *[]rulingProblem) int {
	found, first := 0, -1
	for i, l := range lines {
		if strings.Contains(l, marker) {
			if first < 0 {
				first = i
			}
			found++
		}
	}
	if found != 1 {
		line := 0
		if first >= 0 {
			line = first + 1
		}
		*problems = append(*problems, rulingProblem{line, fmt.Sprintf("expected exactly one %s (found %d)", marker, found)})
		return -1
	}
	return first
}

// parseRulingsLedger parses the decision-ledger and detector blocks. It returns
// (nil, problems) whenever any problem exists.
func parseRulingsLedger(data []byte) (*rulingsLedger, []rulingProblem) {
	lines := strings.Split(string(data), "\n")
	var problems []rulingProblem

	ls := blockMarkerLine(lines, ledgerStartMarker, &problems)
	le := blockMarkerLine(lines, ledgerEndMarker, &problems)
	ds := blockMarkerLine(lines, detectorsStartMarker, &problems)
	de := blockMarkerLine(lines, detectorsEndMarker, &problems)
	if len(problems) > 0 {
		return nil, problems
	}
	if ls > le {
		problems = append(problems, rulingProblem{le + 1, ledgerStartMarker + " must come before " + ledgerEndMarker})
	}
	if ds > de {
		problems = append(problems, rulingProblem{de + 1, detectorsStartMarker + " must come before " + detectorsEndMarker})
	}
	if len(problems) > 0 {
		return nil, problems
	}
	if !(le < ds || de < ls) {
		problems = append(problems, rulingProblem{ds + 1, "the decision-ledger and detector blocks overlap"})
		return nil, problems
	}

	l := &rulingsLedger{Rows: map[string]rulingRow{}}
	for i := ls + 1; i < le; i++ {
		parseLedgerRow(lines[i], i+1, l, &problems)
	}
	if len(l.Rows) == 0 {
		problems = append(problems, rulingProblem{0, "decision ledger has no rows"})
	}

	ordinal := 0
	for i := ds + 1; i < de; i++ {
		text := strings.TrimSpace(lines[i])
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		ordinal++
		if ordinal > rulingsMaxDetectors {
			problems = append(problems, rulingProblem{i + 1, fmt.Sprintf("detector line %d exceeds the %d-line cap", ordinal, rulingsMaxDetectors)})
		}
		fd, sup, why := parseDetectorLine(text, l.Rows)
		if why != "" {
			problems = append(problems, rulingProblem{i + 1, why})
			continue
		}
		if fd != nil {
			fd.Line = i + 1
			l.Forbids = append(l.Forbids, *fd)
		}
		if sup != nil {
			sup.Line = i + 1
			l.Supersedes = append(l.Supersedes, *sup)
		}
	}

	if len(problems) > 0 {
		return nil, problems
	}
	return l, nil
}

// parseLedgerRow validates one line of the decision-ledger block, appending a problem
// on failure and recording the row on success.
func parseLedgerRow(raw string, lineNo int, l *rulingsLedger, problems *[]rulingProblem) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return
	}
	if !strings.HasPrefix(text, "|") {
		*problems = append(*problems, rulingProblem{lineNo, "non-table line inside the decision-ledger block"})
		return
	}
	cells := strings.Split(strings.ReplaceAll(text, `\|`, "\x01"), "|")
	for i := range cells {
		cells[i] = strings.ReplaceAll(strings.TrimSpace(cells[i]), "\x01", "|")
	}
	if len(cells) > 1 && cells[1] == "ID" {
		return // header
	}
	if isSeparatorRow(cells) {
		return
	}
	if len(cells) != 6 {
		*problems = append(*problems, rulingProblem{lineNo, fmt.Sprintf("row has %d fields, want a 4-column row", len(cells))})
		return
	}
	id, status := cells[1], cells[2]
	num, ok := rulingNum(id)
	if !ok {
		*problems = append(*problems, rulingProblem{lineNo, fmt.Sprintf("bad ID %q (want D-FR-<n>)", id)})
		return
	}
	if status != "OPEN" && status != "RESOLVED" {
		*problems = append(*problems, rulingProblem{lineNo, fmt.Sprintf("status %q (want OPEN or RESOLVED)", status)})
		return
	}
	if cells[3] == "" || cells[4] == "" {
		*problems = append(*problems, rulingProblem{lineNo, "empty ruling or evidence cell"})
		return
	}
	if _, dup := l.Rows[id]; dup {
		*problems = append(*problems, rulingProblem{lineNo, fmt.Sprintf("duplicate ID %q", id)})
		return
	}
	l.Rows[id] = rulingRow{ID: id, Num: num, Status: status, Line: lineNo}
}

// isSeparatorRow reports whether every cell is empty or a dash rule, with at least one dash.
// The "at least one dash" clause is stricter than the plan: an all-empty "|||||" row is a
// malformed row, not a separator.
func isSeparatorRow(cells []string) bool {
	dashes := false
	for _, c := range cells {
		if c == "" {
			continue
		}
		if !rulingSepCellRe.MatchString(c) {
			return false
		}
		dashes = true
	}
	return dashes
}

// parseDetectorLine parses one non-blank, non-comment detector line. It returns exactly one
// of: a forbid, a supersession, or a non-empty problem text.
func parseDetectorLine(text string, rows map[string]rulingRow) (*forbidDetector, *supersession, string) {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return nil, nil, `malformed detector line (want <ID> forbid <scope> "<literal>" or <ID> supersedes <ID>)`
	}
	id := fields[0]
	idNum, ok := rulingNum(id)
	if !ok {
		return nil, nil, fmt.Sprintf("bad ID %q (want D-FR-<n>)", id)
	}
	if _, isRow := rows[id]; !isRow {
		return nil, nil, fmt.Sprintf("unknown ID %q", id)
	}
	switch fields[1] {
	case "forbid":
		fd, why := parseForbid(text, id)
		return fd, nil, why
	case "supersedes":
		sup, why := parseSupersedes(fields, id, idNum, rows)
		return nil, sup, why
	default:
		return nil, nil, fmt.Sprintf("unknown verb %q (want forbid or supersedes)", fields[1])
	}
}

func parseForbid(text, id string) (*forbidDetector, string) {
	const want = `malformed detector line (want <ID> forbid <scope> "<literal>")`
	a := strings.Index(text, `"`)
	b := strings.LastIndex(text, `"`)
	if a < 0 || a == b {
		return nil, "missing quoted literal"
	}
	head := strings.Fields(text[:a])
	if len(head) != 3 || strings.TrimSpace(text[b+1:]) != "" {
		return nil, want
	}
	literal := text[a+1 : b]
	if strings.Contains(literal, `"`) {
		return nil, "literal contains a double quote"
	}
	if n := len(literal); n < rulingsMinLiteral || n > rulingsMaxLiteral {
		return nil, fmt.Sprintf("literal is %d bytes (want %d-%d)", n, rulingsMinLiteral, rulingsMaxLiteral)
	}
	scope := head[2]
	switch scope {
	case "registry", "skills", "charters":
	default:
		return nil, fmt.Sprintf("unknown scope %q (want registry, skills or charters)", scope)
	}
	return &forbidDetector{ID: id, Scope: scope, Literal: literal}, ""
}

func parseSupersedes(fields []string, by string, byNum int, rows map[string]rulingRow) (*supersession, string) {
	if len(fields) != 3 {
		return nil, "malformed detector line (want <ID> supersedes <ID>)"
	}
	target := fields[2]
	tNum, ok := rulingNum(target)
	if !ok {
		return nil, fmt.Sprintf("bad ID %q (want D-FR-<n>)", target)
	}
	if tNum >= byNum {
		return nil, fmt.Sprintf("supersedes must name a LOWER-numbered ruling (%s supersedes %s)", by, target)
	}
	if _, isRow := rows[target]; !isRow {
		return nil, fmt.Sprintf("unknown ID %q", target)
	}
	return &supersession{By: by, Target: target}, ""
}

// active returns the IDs of RESOLVED rulings that no RESOLVED ruling supersedes.
func (l *rulingsLedger) active() map[string]bool {
	superseded := map[string]bool{}
	for _, s := range l.Supersedes {
		if l.Rows[s.By].Status == "RESOLVED" {
			superseded[s.Target] = true
		}
	}
	out := map[string]bool{}
	for id, row := range l.Rows {
		if row.Status == "RESOLVED" && !superseded[id] {
			out[id] = true
		}
	}
	return out
}

// activeForbids returns the forbid detectors whose ruling is active, in ledger order.
func (l *rulingsLedger) activeForbids() []forbidDetector {
	act := l.active()
	var out []forbidDetector
	for _, d := range l.Forbids {
		if act[d.ID] {
			out = append(out, d)
		}
	}
	return out
}

// formatRulingProblems renders problems sorted by line (stable), joined by "; ".
func formatRulingProblems(ps []rulingProblem) string {
	sorted := append([]rulingProblem(nil), ps...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Line < sorted[j].Line })
	parts := make([]string, 0, len(sorted))
	for _, p := range sorted {
		if p.Line == 0 {
			parts = append(parts, "file: "+p.Why)
		} else {
			parts = append(parts, fmt.Sprintf("line %d: %s", p.Line, p.Why))
		}
	}
	return strings.Join(parts, "; ")
}
