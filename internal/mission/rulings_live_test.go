package mission

import (
	"os"
	"strings"
	"testing"
)

// M-FLEET-RULINGS-LEDGER M6: the real ledger parses, and the seed detectors are quiet on activation.
// These tests read the live design_docs/fleet-rulings.md and the snapshots under testdata/rulings/.

const (
	liveLedgerPath      = "../../design_docs/fleet-rulings.md"
	seedSkillLiteral    = "generator≠judge guard (HARD" // D-FR-2, scope skills
	seedRegistryLiteral = "WORLD IS A FORK"             // D-FR-1, scope registry
)

// readRulingsTestdata returns the lines of a snapshot under testdata/rulings/.
func readRulingsTestdata(t *testing.T, name string) []string {
	t.Helper()
	data, err := os.ReadFile("testdata/rulings/" + name) //nolint:gosec // fixed test fixture path under the package testdata
	if err != nil {
		t.Fatalf("read snapshot %s: %v", name, err)
	}
	return strings.Split(string(data), "\n")
}

// liveForbidLiteral returns the literal of the forbid detector with the given ID and scope in the live ledger.
func liveForbidLiteral(t *testing.T, l *rulingsLedger, id, scope string) string {
	t.Helper()
	for _, d := range l.Forbids {
		if d.ID == id && d.Scope == scope {
			return d.Literal
		}
	}
	t.Fatalf("live ledger has no %s forbid %s detector", id, scope)
	return ""
}

func TestRulingsLiveLedgerParses(t *testing.T) {
	data, err := os.ReadFile(liveLedgerPath) //nolint:gosec // fixed repo-relative path to the committed ledger
	if err != nil {
		t.Fatalf("read live ledger: %v", err)
	}
	l, problems := parseRulingsLedger(data)
	if len(problems) != 0 {
		t.Fatalf("live ledger has %d problem(s): %s", len(problems), formatRulingProblems(problems))
	}
	if l == nil {
		t.Fatal("live ledger parsed to nil with no problems")
	}
	if len(l.Rows) < 2 {
		t.Errorf("live ledger rows = %d, want >= 2", len(l.Rows))
	}
	if len(l.Forbids) < 3 {
		t.Errorf("live ledger forbids = %d, want >= 3", len(l.Forbids))
	}
	for _, d := range l.Forbids {
		if _, ok := l.Rows[d.ID]; !ok {
			t.Errorf("forbid %s (line %d) names no ledger row", d.ID, d.Line)
		}
	}
}

func TestRulingsSeedQuietOnActivation(t *testing.T) {
	data, err := os.ReadFile(liveLedgerPath) //nolint:gosec // fixed repo-relative path to the committed ledger
	if err != nil {
		t.Fatalf("read live ledger: %v", err)
	}
	l, problems := parseRulingsLedger(data)
	if len(problems) != 0 {
		t.Fatalf("live ledger has %d problem(s): %s", len(problems), formatRulingProblems(problems))
	}

	// D-FR-2 skills literal: the retired text fires on the pre-fix skill and is quiet on the fixed one.
	skillLit := liveForbidLiteral(t, l, "D-FR-2", "skills")
	if skillLit != seedSkillLiteral {
		t.Fatalf("live D-FR-2 skills literal = %q, want %q", skillLit, seedSkillLiteral)
	}
	pre := readRulingsTestdata(t, "gate-3-route.pre-349e311b9.md")
	if hits := literalHits(pre, skillLit, false); len(hits) != 1 {
		t.Errorf("pre-fix gate-3-route hits = %v, want exactly 1 line", hits)
	}
	cur := readRulingsTestdata(t, "gate-3-route.current.md")
	if hits := literalHits(cur, skillLit, false); len(hits) != 0 {
		t.Errorf("current gate-3-route hits = %v, want 0", hits)
	}
	// Control: the fixed bytes are really there, so the 0 above is not a snapshot that lost its text.
	if !strings.Contains(strings.Join(cur, "\n"), "generator≠judge") {
		t.Error("current gate-3-route snapshot lacks the text 'generator≠judge' (control failed)")
	}

	// D-FR-1 registry literal: the retired text fires on the pre-#2396 world.toml and is quiet on the post one.
	regLit := liveForbidLiteral(t, l, "D-FR-1", "registry")
	if regLit != seedRegistryLiteral {
		t.Fatalf("live D-FR-1 registry literal = %q, want %q", regLit, seedRegistryLiteral)
	}
	worldPre := readRulingsTestdata(t, "world.pre-2396.toml")
	if hits := literalHits(worldPre, regLit, false); len(hits) != 1 {
		t.Errorf("pre-2396 world.toml hits = %v, want exactly 1 line", hits)
	}
	worldPost := readRulingsTestdata(t, "world.post-2396.toml")
	if hits := literalHits(worldPost, regLit, false); len(hits) != 0 {
		t.Errorf("post-2396 world.toml hits = %v, want 0", hits)
	}
}
