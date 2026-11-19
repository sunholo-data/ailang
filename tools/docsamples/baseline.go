package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Entry is one line of a ratchet list: a block keyed by page and content hash (never
// by line number, so an edit elsewhere on the page does not churn it). In the baseline,
// Code is the block's first error code; in the unchecked list it is sketch or repl.
// Note is a trailing `# ...` comment (the unchecked list's `reviewed:` marker).
type Entry struct {
	Page, Key, Code, Note string
}

func (e Entry) id() string { return e.Page + " " + e.Key }

// List is a ratchet file: the baseline of known-failing blocks, or the list of blocks
// tagged sketch/repl (not checked). Both only shrink against the base ref.
type List struct {
	Entries map[string]Entry // by id
	Order   []string
}

// LoadList reads a ratchet file. A missing file is empty only when allowMissing is set
// (seeding); otherwise it is an error — a typo in the path must not read as "nothing
// is grandfathered".
func LoadList(path string, allowMissing bool) (*List, error) {
	l := &List{Entries: map[string]Entry{}}
	f, err := os.Open(path)
	if os.IsNotExist(err) && allowMissing {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return l, parseList(bufio.NewScanner(f), l, path)
}

func parseList(sc *bufio.Scanner, l *List, name string) error {
	n := 0
	for sc.Scan() {
		n++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		note := ""
		if i := strings.Index(line, "#"); i >= 0 {
			note, line = strings.TrimSpace(line[i+1:]), strings.TrimSpace(line[:i])
		}
		f := strings.Fields(line)
		if len(f) != 3 || len(f[1]) != 12 {
			return fmt.Errorf("%s:%d: want `<page> <sha1-12> <code>`, got %q", name, n, line)
		}
		e := Entry{Page: f[0], Key: f[1], Code: f[2], Note: note}
		if strings.HasPrefix(e.Code, "DOC_") {
			return fmt.Errorf("%s:%d: %s is a structural defect (a fence or tag the page cannot render as written) and cannot be baselined: fix the page", name, n, e.Code)
		}
		if _, dup := l.Entries[e.id()]; dup {
			return fmt.Errorf("%s:%d: duplicate entry %s", name, n, e.id())
		}
		l.Entries[e.id()] = e
		l.Order = append(l.Order, e.id())
	}
	return sc.Err()
}

// ListAtRef reads a ratchet file as it was at a git ref. found=false means the file did
// not exist there (first seeding). An unresolvable ref is an error: the ratchet fails
// closed rather than comparing against nothing.
func ListAtRef(repo, ref, path string) (l *List, found bool, err error) {
	if out, err := exec.Command("git", "-C", repo, "rev-parse", "--verify", "--quiet", ref+"^{commit}").CombinedOutput(); err != nil {
		return nil, false, fmt.Errorf("git ref %q not found (%s): fetch it, e.g. git fetch origin dev", ref, strings.TrimSpace(string(out)))
	}
	if exec.Command("git", "-C", repo, "cat-file", "-e", ref+":"+path).Run() != nil {
		return nil, false, nil
	}
	out, err := exec.Command("git", "-C", repo, "show", ref+":"+path).Output()
	if err != nil {
		return nil, false, fmt.Errorf("git show %s:%s: %w", ref, path, err)
	}
	l = &List{Entries: map[string]Entry{}}
	return l, true, parseList(bufio.NewScanner(strings.NewReader(string(out))), l, ref+":"+path)
}

// Added returns the ids in cur that base does not have — the ratchet violations. The
// rule is a SET rule, not a count: swapping one entry for another keeps the count and is
// still growth. allow exempts an entry (the unchecked list's reviewed marker).
func Added(cur, base *List, allow func(Entry) bool) []string {
	var out []string
	for _, id := range cur.Order {
		if _, ok := base.Entries[id]; !ok && (allow == nil || !allow(cur.Entries[id])) {
			out = append(out, id)
		}
	}
	return out
}

// reviewed is the unchecked list's explicit marker for a NEW sketch/repl block:
// `# reviewed: <why this block cannot be checked>`. It is visible in the diff a
// reviewer reads; it is a speed bump, not an authentication.
func reviewed(e Entry) bool {
	return strings.HasPrefix(e.Note, "reviewed:") && len(strings.TrimSpace(strings.TrimPrefix(e.Note, "reviewed:"))) > 0
}

// Verdict is the gate's judgement of one block, or of one orphaned list entry.
type Verdict string

const (
	VOK        Verdict = "ok"        // ran and met its class's contract, not listed
	VSkipped   Verdict = "skipped"   // sketch / repl, listed in the unchecked list
	VBaselined Verdict = "baselined" // fails with its listed code
	VExempt    Verdict = "EXEMPT"    // a named one-off structural exemption (structural.go)
	VFail      Verdict = "FAIL"      // fails, not listed; or structural; or a different code than listed
	VStale     Verdict = "STALE"     // listed, but passes now or is no longer that class
	VOrphan    Verdict = "ORPHAN"    // listed, but no such block exists any more
)

// Failing reports whether the verdict fails the gate.
func (v Verdict) Failing() bool { return v == VFail || v == VStale || v == VOrphan }

// Judged pairs a block (nil for an orphan) with its outcome and verdict.
type Judged struct {
	Block   *Block
	Outcome *Outcome
	Entry   *Entry // the list entry it matched, if any
	List    string // "baseline" or "unchecked", for orphans
	Verdict Verdict
	Why     string // for FAIL/STALE/EXEMPT: why, when it is not the block's own error
}

func isStructural(code string) bool { return strings.HasPrefix(code, "DOC_") }

// Judge applies the baseline, the unchecked list and the exemptions to the outcomes.
// Identical blocks on one page share one key and therefore one entry.
func Judge(blocks []*Block, outcomes map[*Block]*Outcome, bl, unchecked *List) []Judged {
	used := map[string]bool{}
	usedU := map[string]bool{}
	var out []Judged
	for _, b := range blocks {
		o := outcomes[b]
		if o == nil || o.Dropped {
			continue
		}
		id := b.Page + " " + b.Key
		j := Judged{Block: b, Outcome: o}
		e, listed := bl.Entries[id]
		u, listedU := unchecked.Entries[id]
		if listed {
			used[id] = true
			ec := e
			j.Entry = &ec
		}
		if listedU {
			usedU[id] = true
		}
		switch {
		case !o.Ran: // sketch / repl
			switch {
			case listed:
				j.Verdict, j.Why = VStale, "listed in the baseline but is tagged "+string(b.Class)+" and no longer checked — delete its baseline line"
			case !listedU:
				j.Verdict, j.Why = VFail, fmt.Sprintf("tagged %s but not in the unchecked list: add `%s %s %s # reviewed: <why it cannot be checked>`", b.Class, b.Page, b.Key, b.Class)
			case u.Code != string(b.Class):
				j.Verdict, j.Why = VFail, fmt.Sprintf("unchecked list says %s, the block is tagged %s", u.Code, b.Class)
			default:
				j.Verdict = VSkipped
			}
		case isStructural(o.Code):
			if why, ok := exemptFor(b); ok {
				j.Verdict, j.Why = VExempt, why
			} else {
				j.Verdict = VFail
			}
		case listedU:
			j.Verdict, j.Why = VStale, "listed in the unchecked list but is no longer tagged sketch/repl — delete its line"
		case o.Pass && listed:
			j.Verdict, j.Why = VStale, "listed in the baseline but PASSES now — delete its line: "+id
			if b.Class == ClassError {
				j.Why = "listed in the baseline but now fails exactly as tagged (error=" + b.WantCode + ") — delete its line: " + id
			}
		case o.Pass:
			j.Verdict = VOK
		case listed && e.Code != o.Code:
			j.Verdict = VFail
			j.Why = fmt.Sprintf("baselined as %s but now fails with %s — a different failure is a new failure: fix it, or update the code on its line if the block still fails for the same reason", e.Code, o.Code)
		case listed:
			j.Verdict = VBaselined
		default:
			j.Verdict = VFail
		}
		out = append(out, j)
	}
	for _, id := range bl.Order {
		if !used[id] {
			e := bl.Entries[id]
			out = append(out, Judged{Entry: &e, List: "baseline", Verdict: VOrphan})
		}
	}
	for _, id := range unchecked.Order {
		if !usedU[id] {
			e := unchecked.Entries[id]
			out = append(out, Judged{Entry: &e, List: "unchecked", Verdict: VOrphan})
		}
	}
	return out
}

// WriteLists records every currently failing non-structural block in the baseline and
// every sketch/repl block in the unchecked list, keeping existing `reviewed:` notes.
func WriteLists(blPath, unPath string, judged []Judged, oldUnchecked *List) (nb, nu int, err error) {
	var bl, un []string
	seen := map[string]bool{}
	for _, j := range judged {
		if j.Block == nil || j.Outcome == nil {
			continue
		}
		b, o := j.Block, j.Outcome
		id := b.Page + " " + b.Key
		if seen[id] {
			continue
		}
		switch {
		case !o.Ran:
			seen[id] = true
			line := fmt.Sprintf("%s %s %s", b.Page, b.Key, b.Class)
			if e, ok := oldUnchecked.Entries[id]; ok && e.Note != "" {
				line += " # " + e.Note
			}
			un = append(un, line)
		case !o.Pass && !isStructural(o.Code):
			seen[id] = true
			bl = append(bl, fmt.Sprintf("%s %s %s", b.Page, b.Key, o.Code))
		}
	}
	sort.Strings(bl)
	sort.Strings(un)
	for _, p := range []string{blPath, unPath} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return 0, 0, err
		}
	}
	if err := os.WriteFile(blPath, []byte(baselineHeader+joinLines(bl)), 0o644); err != nil {
		return 0, 0, err
	}
	return len(bl), len(un), os.WriteFile(unPath, []byte(uncheckedHeader+joinLines(un)), 0o644)
}

func joinLines(ls []string) string {
	if len(ls) == 0 {
		return ""
	}
	return strings.Join(ls, "\n") + "\n"
}

const baselineHeader = `# Known-failing documentation code blocks, grandfathered by tools/docsamples
# (make check-doc-samples). Format: <page> <sha1-12 of the normalised block> <first error code>
#
# This file only SHRINKS, as a set: CI refuses any line whose page+hash the base commit's
# copy does not have (a swap that keeps the count is still growth). The gate also fails on
#   - a failing block that is not listed (fix it, or tag it: parse / error=CODE / file=path.ail,
#     or sketch / repl via doc_samples_unchecked.txt — see .claude/rules/docs-review.md);
#   - a listed block that now passes, no longer exists, or fails with a DIFFERENT code;
#   - any structural defect (DOC_*: mis-fenced AILANG, bad tag, broken fence) — those
#     cannot be listed here at all.
# Editing a listed block changes its hash, so touching a block means fixing or tagging it.
# After a burn-down: make check-doc-samples DOC_SAMPLES_ARGS=-write-baseline
`

const uncheckedHeader = `# Documentation code blocks tagged sketch or repl, which tools/docsamples does NOT check.
# Format: <page> <sha1-12 of the normalised block> <sketch|repl> [# reviewed: <why>]
#
# A ratchet like doc_samples_baseline.txt: the set may only shrink against the base
# commit, except that a NEW line is accepted when it carries "# reviewed: <reason>" —
# an explicit marker a reviewer sees in the diff (sketch is otherwise an escape hatch:
# tagging a broken program sketch would turn the gate green). Every sketch/repl block
# must be listed, and every line must still be a sketch/repl block.
`
