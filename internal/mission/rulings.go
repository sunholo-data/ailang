package mission

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// M-FLEET-RULINGS-LEDGER M3: CheckRulings reports, as Notes only and never Drift, charter,
// skill and registry text that contains a literal retired by an active (RESOLVED,
// unsuperseded) fleet-wide ruling. Every finding is advisory; nothing here fails a run.

// rulingFile is one cached read: the lines of a file, or the error that stopped the read.
type rulingFile struct {
	lines []string
	err   error
}

// rulingScope is one file a detector reads: owner names the finding's Mission, charter
// selects the decision-ledger exclusion of literalHits.
type rulingScope struct {
	owner   string
	path    string
	charter bool
}

// rulingsReader collects findings and reads each path at most once per check. The first
// failing read of a path reports it; later reads of that path are silent.
type rulingsReader struct {
	root     string
	cache    map[string]*rulingFile
	findings []Finding
}

func (rd *rulingsReader) note(owner, kind, detail string) {
	rd.findings = append(rd.findings, Finding{Mission: owner, Kind: kind, Severity: Note, Detail: detail})
}

func (rd *rulingsReader) read(owner, path string) *rulingFile {
	if rf, ok := rd.cache[path]; ok {
		return rf
	}
	lines, err := readRulingLines(path)
	rf := &rulingFile{lines: lines, err: err}
	rd.cache[path] = rf
	if err != nil {
		rd.note(owner, "ruling-file-unreadable", displayPath(rd.root, path)+": "+err.Error())
	}
	return rf
}

// readRulingLines stats before reading so an oversized file is never loaded.
func readRulingLines(path string) ([]string, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, errors.New("does not exist")
	}
	if err != nil {
		return nil, err
	}
	if info.Size() > rulingsMaxFileBytes {
		return nil, fmt.Errorf("is %d bytes (> 4 MiB)", info.Size())
	}
	data, err := os.ReadFile(path) //nolint:gosec // path is under the registry root or a mission charter, both from the registry
	if err != nil {
		return nil, err
	}
	return strings.Split(string(data), "\n"), nil
}

// CheckRulings reports — as Notes only, never Drift — charter, skill and registry text that contains a
// literal retired by an active (RESOLVED, unsuperseded) fleet-wide ruling. It derives the root itself (design §2).
func CheckRulings(missions []*Mission) []Finding {
	root, skipped := rulingsRoot(missions)
	if skipped != nil {
		return []Finding{*skipped}
	}
	rd := &rulingsReader{root: root, cache: map[string]*rulingFile{}}

	ledgerPath := filepath.Join(root, rulingsLedgerRel)
	if _, err := os.Stat(ledgerPath); os.IsNotExist(err) {
		return []Finding{{Mission: rulingsMission, Kind: "rulings-ledger-absent", Severity: Note,
			Detail: rulingsLedgerRel + " does not exist; no fleet-wide ruling is checked"}}
	}
	ledgerFile := rd.read(rulingsMission, ledgerPath)
	if ledgerFile.err != nil {
		return rd.findings
	}
	ledger, problems := parseRulingsLedger([]byte(strings.Join(ledgerFile.lines, "\n")))
	if len(problems) > 0 {
		return []Finding{{Mission: rulingsMission, Kind: "rulings-malformed", Severity: Note,
			Detail: fmt.Sprintf("%s: %d problem(s) — %s", rulingsLedgerRel, len(problems), formatRulingProblems(problems))}}
	}

	// Link check: a charter that cannot be read is already reported as unreadable, so it is skipped here.
	for _, m := range missions {
		cp := charterPath(root, m)
		cf := rd.read(m.Name, cp)
		if cf.err != nil {
			continue
		}
		outside := outsideLedger(cf.lines)
		linked := false
		for i, ln := range cf.lines {
			if outside[i] && strings.Contains(ln, rulingsLinkText) {
				linked = true
				break
			}
		}
		if !linked {
			rd.note(m.Name, "ruling-link-missing", displayPath(root, cp)+" has no link to design_docs/fleet-rulings.md outside its decision-ledger block")
		}
	}

	var skillFiles []string
	skillsWalked := false
	for _, d := range ledger.activeForbids() {
		var scope []rulingScope
		switch d.Scope {
		case "registry":
			for _, m := range missions {
				scope = append(scope, rulingScope{owner: m.Name, path: m.Path})
			}
		case "charters":
			for _, m := range missions {
				scope = append(scope, rulingScope{owner: m.Name, path: charterPath(root, m), charter: true})
			}
		case "skills":
			if !skillsWalked {
				skillsWalked = true
				files, err := rulingSkillFiles(root)
				if err != nil {
					rd.note(rulingsMission, "ruling-file-unreadable", ".claude/skills and .agents/skills: "+err.Error())
				} else {
					skillFiles = files
				}
			}
			for _, p := range skillFiles {
				scope = append(scope, rulingScope{owner: rulingsMission, path: p})
			}
		}
		rd.detect(d, scope)
	}

	sort.SliceStable(rd.findings, func(i, j int) bool {
		a, b := rd.findings[i], rd.findings[j]
		if a.Mission != b.Mission {
			return a.Mission < b.Mission
		}
		return a.Kind < b.Kind
	})
	return rd.findings
}

// detect reports the hits of one forbid detector: the first rulingsMaxHitsPerDetector as
// ruling-contradiction findings, then one summary finding when more lines matched.
func (rd *rulingsReader) detect(d forbidDetector, scope []rulingScope) {
	var hits []Finding
	for _, s := range scope {
		f := rd.read(s.owner, s.path)
		if f.err != nil {
			continue
		}
		for _, line := range literalHits(f.lines, d.Literal, s.charter) {
			hits = append(hits, Finding{Mission: s.owner, Kind: "ruling-contradiction", Severity: Note,
				Detail: fmt.Sprintf("%s %s:%d contains retired text %q", d.ID, displayPath(rd.root, s.path), line, d.Literal)})
		}
	}
	shown := hits
	if len(hits) > rulingsMaxHitsPerDetector {
		shown = hits[:rulingsMaxHitsPerDetector]
	}
	rd.findings = append(rd.findings, shown...)
	if extra := len(hits) - len(shown); extra > 0 {
		rd.note(rulingsMission, "ruling-contradiction", fmt.Sprintf("%s … %d more line(s) contain %q (only the first %d are listed)",
			d.ID, extra, d.Literal, rulingsMaxHitsPerDetector))
	}
}
