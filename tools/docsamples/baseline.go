package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// Entry is one baseline line: a known-failing block, keyed by page and content
// hash (never by line number, so an edit elsewhere on the page does not churn it).
type Entry struct {
	Page, Key, Code string
}

func (e Entry) id() string { return e.Page + " " + e.Key }

// Baseline is the shrink-only list of known-failing blocks.
type Baseline struct {
	Entries map[string]Entry // by id
	Order   []string
}

const baselineHeader = `# Known-failing documentation code blocks, grandfathered by tools/docsamples
# (make check-doc-samples). Format: <page> <sha1-12 of the normalised block> <first error code>
#
# This file only SHRINKS. The gate fails on:
#   - a failing block that is not listed here (fix it, or tag it: parse / sketch / repl /
#     error=CODE / file=path.ail — see .claude/rules/docs-review.md);
#   - a listed block that now passes, or that no longer exists (delete its line);
#   - more entries than origin/dev has (CI compares the counts).
# Editing a listed block changes its hash, so touching a block means fixing or tagging it.
# The error code is informational: a different first error is reported, not failed.
# Regenerate after burn-down with: make check-doc-samples DOC_SAMPLES_ARGS=--write-baseline
`

// LoadBaseline reads a baseline file. A missing file is an empty baseline only
// when allowMissing is set (seeding); otherwise it is an error — a typo in the
// path must not read as "nothing is grandfathered".
func LoadBaseline(path string, allowMissing bool) (*Baseline, error) {
	bl := &Baseline{Entries: map[string]Entry{}}
	f, err := os.Open(path)
	if os.IsNotExist(err) && allowMissing {
		return bl, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return bl, parseBaseline(bufio.NewScanner(f), bl, path)
}

func parseBaseline(sc *bufio.Scanner, bl *Baseline, name string) error {
	n := 0
	for sc.Scan() {
		n++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		if len(f) != 3 || len(f[1]) != 12 {
			return fmt.Errorf("%s:%d: want `<page> <sha1-12> <code>`, got %q", name, n, line)
		}
		e := Entry{Page: f[0], Key: f[1], Code: f[2]}
		if _, dup := bl.Entries[e.id()]; dup {
			return fmt.Errorf("%s:%d: duplicate entry %s", name, n, e.id())
		}
		bl.Entries[e.id()] = e
		bl.Order = append(bl.Order, e.id())
	}
	return sc.Err()
}

// CountAtRef counts the entries the baseline had at a git ref (the shrink-only
// ratchet). found=false means the file did not exist there (first seeding).
func CountAtRef(repo, ref, path string) (count int, found bool, err error) {
	if out, err := exec.Command("git", "-C", repo, "rev-parse", "--verify", "--quiet", ref+"^{commit}").CombinedOutput(); err != nil {
		return 0, false, fmt.Errorf("git ref %q not found (%s): fetch it, e.g. git fetch origin dev", ref, strings.TrimSpace(string(out)))
	}
	if exec.Command("git", "-C", repo, "cat-file", "-e", ref+":"+path).Run() != nil {
		return 0, false, nil
	}
	out, err := exec.Command("git", "-C", repo, "show", ref+":"+path).Output()
	if err != nil {
		return 0, false, fmt.Errorf("git show %s:%s: %w", ref, path, err)
	}
	bl := &Baseline{Entries: map[string]Entry{}}
	if err := parseBaseline(bufio.NewScanner(strings.NewReader(string(out))), bl, ref+":"+path); err != nil {
		return 0, true, err
	}
	return len(bl.Entries), true, nil
}

// Verdict is the gate's judgement of one block, or of one orphaned baseline entry.
type Verdict string

const (
	VOK        Verdict = "ok"        // ran and met its class's contract, not listed
	VSkipped   Verdict = "skipped"   // sketch / repl
	VBaselined Verdict = "baselined" // fails, and is listed
	VFail      Verdict = "FAIL"      // fails, not listed
	VStale     Verdict = "STALE"     // listed, but passes now or is no longer checked
	VOrphan    Verdict = "ORPHAN"    // listed, but no such block exists any more
)

// Failing reports whether the verdict fails the gate.
func (v Verdict) Failing() bool { return v == VFail || v == VStale || v == VOrphan }

// Judged pairs a block (nil for an orphan) with its outcome and verdict.
type Judged struct {
	Block    *Block
	Outcome  *Outcome
	Entry    *Entry // the baseline entry it matched, if any
	Verdict  Verdict
	CodeNote string // baselined with a different first code (informational)
}

// Judge applies the baseline to the outcomes. Identical blocks on one page share
// one key and therefore one entry.
func Judge(blocks []*Block, outcomes map[*Block]*Outcome, bl *Baseline) []Judged {
	used := map[string]bool{}
	var out []Judged
	for _, b := range blocks {
		o := outcomes[b]
		id := b.Page + " " + b.Key
		e, listed := bl.Entries[id]
		j := Judged{Block: b, Outcome: o}
		if listed {
			used[id] = true
			ec := e
			j.Entry = &ec
		}
		switch {
		case !o.Ran && listed:
			j.Verdict = VStale
		case !o.Ran:
			j.Verdict = VSkipped
		case o.Pass && listed:
			j.Verdict = VStale
		case o.Pass:
			j.Verdict = VOK
		case listed:
			j.Verdict = VBaselined
			if e.Code != o.Code {
				j.CodeNote = fmt.Sprintf("baselined as %s, now %s", e.Code, o.Code)
			}
		default:
			j.Verdict = VFail
		}
		out = append(out, j)
	}
	for _, id := range bl.Order {
		if !used[id] {
			e := bl.Entries[id]
			out = append(out, Judged{Entry: &e, Verdict: VOrphan})
		}
	}
	return out
}

// WriteBaseline records every currently failing block (one line per page+key).
func WriteBaseline(path string, judged []Judged) (int, error) {
	seen := map[string]bool{}
	var lines []string
	for _, j := range judged {
		if j.Block == nil || j.Outcome == nil || !j.Outcome.Ran || j.Outcome.Pass {
			continue
		}
		e := Entry{Page: j.Block.Page, Key: j.Block.Key, Code: j.Outcome.Code}
		if seen[e.id()] {
			continue
		}
		seen[e.id()] = true
		lines = append(lines, fmt.Sprintf("%s %s %s", e.Page, e.Key, e.Code))
	}
	sort.Strings(lines)
	body := baselineHeader
	if len(lines) > 0 {
		body += strings.Join(lines, "\n") + "\n"
	}
	return len(lines), os.WriteFile(path, []byte(body), 0o644)
}
