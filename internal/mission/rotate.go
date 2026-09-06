package mission

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Log rotation, and the index that makes it safe.
//
// The v1 log reached 2.86 MB / ~715k tokens / 335 entries with NO rotation rule anywhere
// in Gate 4 — iteration 1 was still in the file iteration 335 appended to. Bounded greps
// keep it out of context today, but one careless Read costs ~715k tokens on every
// remaining turn of that iteration, and it grows ~2,250 lines a week forever.
//
// Rotation alone would be a regression: the log is the mission's memory, and a loop that
// cannot see what it already tried WILL repeat work. So rotation always writes three
// tiers:
//
//	live log   last N full entries          — the working memory
//	archive    every older full entry       — retrievable, never loaded
//	INDEX      one line per iteration, ALL  — complete history, small enough to read whole
//
// The index is generated from the entries' own headings, so it cannot drift from them and
// can be regenerated from scratch at any time.

// entryHeadingRe matches a log entry heading.
//
// THE FLEET USES THREE FORMATS, and a parser that knows only one silently rotates nothing
// (or, worse, mis-parses). Measured 2026-09-06 across the four live logs:
//
//	v1      ## 337 — 2026-09-06 — Title [TAG]
//	docs    ## ITERATION 11 — 2026-09-06T06:00Z (optional note)
//	world   ## Iteration 0 — 2026-07-23 — title
//
// Plus the awkward real cases a strict pattern dropped from v1's index: a date RANGE
// ("2026-07-10/11") and a combined entry ("321 & 322"). Deliberately tolerant, because
// under-matching here does not fail loudly — it produces a short index, and an index that
// quietly omits iterations is exactly what lets the loop repeat work.
var entryHeadingRe = regexp.MustCompile(
	`^## (?i:iteration\s+)?(\d+)(?:\s*&\s*\d+)?\s+—\s+(\d{4}-\d{2}-\d{2})(?:T[0-9:]+Z?)?(?:/\d{1,2})?\s*(?:—\s*)?(.*)$`)

// noteHeadingRe matches a non-iteration record that still belongs in the index, e.g. an
// attended note. These carry no iteration number, so they sort by date.
var noteHeadingRe = regexp.MustCompile(`^## ([A-Z][A-Z ]+)\s+—\s+(\d{4}-\d{2}-\d{2})\s*(.*)$`)

// LogEntry is one iteration record.
type LogEntry struct {
	Num   int
	Date  string
	Title string
	Body  string // the full entry, heading included
}

// parseLog splits a mission log into its preamble and entries.
func parseLog(body string) (preamble string, entries []LogEntry) {
	lines := strings.Split(body, "\n")
	var cur *LogEntry
	var pre []string
	var buf []string
	flush := func() {
		if cur != nil {
			cur.Body = strings.Join(buf, "\n")
			entries = append(entries, *cur)
		}
		buf = nil
	}
	for _, l := range lines {
		if m := entryHeadingRe.FindStringSubmatch(l); m != nil {
			flush()
			n, _ := strconv.Atoi(m[1])
			title := strings.TrimSpace(m[3])
			// A combined "321 & 322" entry keeps the first number for ordering, but the
			// index must say it covers both or the second looks unrecorded.
			//
			// No exclusion for "NO ENTRY" titles: v1's real combined record IS
			// "## 321 & 322 — ... — NO ENTRY: both slots died mid-flight", and a dead
			// slot is the case where "has 322 been attempted?" most needs an answer.
			if strings.Contains(strings.SplitN(l, "—", 2)[0], "&") {
				title = strings.TrimSpace(strings.SplitN(l, "—", 2)[0][3:]) + ": " + title
			}
			cur = &LogEntry{Num: n, Date: m[2], Title: title}
			buf = []string{l}
			continue
		}
		if m := noteHeadingRe.FindStringSubmatch(l); m != nil && cur != nil {
			// A note is its own record, not part of the entry above it.
			flush()
			cur = &LogEntry{Num: -1, Date: m[2], Title: strings.TrimSpace(m[1] + " " + m[3])}
			buf = []string{l}
			continue
		}
		if cur == nil {
			pre = append(pre, l)
			continue
		}
		buf = append(buf, l)
	}
	flush()
	return strings.Join(pre, "\n"), entries
}

// RotateResult reports what a rotation did, so the caller can print it rather than guess.
type RotateResult struct {
	Total, Kept, Archived int
	LogPath, ArchivePath  string
	IndexPath             string
	IndexEntries          int
	LogBefore, LogAfter   int
}

// indexLine renders one iteration as a single index row. Deliberately fixed-shape and
// short: the index only earns its place if the whole thing can be read at once.
func indexLine(e LogEntry) string {
	t := e.Title
	// Titles carry a trailing [TAG]; keep it, it is the cheapest filter the index has.
	//
	// TRUNCATE BY RUNES, NOT BYTES. `t[:147]` slices bytes, and these titles are full of
	// em-dashes (0xe2 0x80 0x94) — the first real run produced an index that was not valid
	// UTF-8 because a dash was cut in half. Every reader of the index then fails on it,
	// which is a worse outcome than a long line.
	if r := []rune(t); len(r) > 150 {
		t = string(r[:147]) + "..."
	}
	num := strconv.Itoa(e.Num)
	if e.Num < 0 {
		num = "—"
	}
	return fmt.Sprintf("| %s | %s | %s |", num, e.Date, strings.ReplaceAll(t, "|", "\\|"))
}

// RotateLog trims the live log to the newest keep entries, appends the rest to the
// archive, and regenerates the complete index.
//
// Regenerating the index from BOTH files (rather than appending to it) is deliberate: an
// append-only index drifts the moment an entry is edited or a rotation is re-run, and a
// drifted "what have we already done" index is worse than none — it would answer
// confidently and wrongly.
func RotateLog(logPath string, keep int) (*RotateResult, error) {
	if keep < 1 {
		return nil, fmt.Errorf("keep must be >= 1, got %d", keep)
	}
	raw, err := os.ReadFile(logPath) //nolint:gosec // caller-supplied mission path
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", logPath, err)
	}
	pre, entries := parseLog(string(raw))
	if len(entries) == 0 {
		return nil, fmt.Errorf("%s: no iteration entries found (expected headings like '## 337 — 2026-09-06 — ...')", logPath)
	}

	dir, base := filepath.Dir(logPath), filepath.Base(logPath)
	stem := strings.TrimSuffix(base, ".md")
	archivePath := filepath.Join(dir, stem+"-archive.md")
	indexPath := filepath.Join(dir, strings.TrimSuffix(stem, "-log")+"-index.md")

	// Entries already archived stay in the index — the index is the COMPLETE history or
	// it does not do its job.
	var archived []LogEntry
	if ab, aerr := os.ReadFile(archivePath); aerr == nil { //nolint:gosec // derived path
		_, archived = parseLog(string(ab))
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Num < entries[j].Num })
	cut := len(entries) - keep
	if cut < 0 {
		cut = 0
	}
	toArchive, toKeep := entries[:cut], entries[cut:]

	res := &RotateResult{
		Total: len(entries), Kept: len(toKeep), Archived: len(toArchive),
		LogPath: logPath, ArchivePath: archivePath, IndexPath: indexPath,
		LogBefore: len(raw),
	}

	if len(toArchive) > 0 {
		var b strings.Builder
		if len(archived) == 0 {
			b.WriteString("# " + stem + " — ARCHIVE\n\n" +
				"Full iteration records rotated out of the live log. Nothing here is loaded by the\n" +
				"loop; it exists so a rotated entry can still be read in full when something needs it.\n" +
				"The one-line history of EVERY iteration, including these, is in the index beside it.\n")
		} else {
			ab, _ := os.ReadFile(archivePath) //nolint:gosec // derived path
			b.WriteString(strings.TrimRight(string(ab), "\n"))
		}
		for _, e := range toArchive {
			b.WriteString("\n\n" + strings.TrimRight(e.Body, "\n"))
		}
		b.WriteString("\n")
		if werr := writeAtomic(archivePath, []byte(b.String()), 0o644); werr != nil { //nolint:gosec // a doc
			return nil, werr
		}
	}

	// The live log: preamble + the newest entries.
	var lb strings.Builder
	lb.WriteString(strings.TrimRight(pre, "\n"))
	lb.WriteString("\n\n> **Older entries are ARCHIVED.** This file holds the newest " +
		strconv.Itoa(keep) + ". The full record of every\n" +
		"> iteration is in `" + filepath.Base(archivePath) + "`, and a one-line index of ALL of them —\n" +
		"> the thing to grep before picking work, so the loop never repeats itself — is in\n" +
		"> `" + filepath.Base(indexPath) + "`.\n")
	for _, e := range toKeep {
		lb.WriteString("\n" + strings.TrimRight(e.Body, "\n") + "\n")
	}
	if werr := writeAtomic(logPath, []byte(lb.String()), 0o644); werr != nil { //nolint:gosec // a doc
		return nil, werr
	}
	res.LogAfter = lb.Len()

	// The index: EVERY entry, archived and live, regenerated from scratch.
	all := append(append([]LogEntry{}, archived...), entries...)
	seen := map[string]bool{}
	var uniq []LogEntry
	sort.Slice(all, func(i, j int) bool {
		if all[i].Num != all[j].Num {
			return all[i].Num > all[j].Num
		}
		return all[i].Date > all[j].Date
	})
	for _, e := range all {
		// Numbered entries dedupe by number; notes (Num -1) by date+title, since several
		// can share the sentinel.
		k := strconv.Itoa(e.Num)
		if e.Num < 0 {
			k = e.Date + "|" + e.Title
		}
		if !seen[k] {
			seen[k] = true
			uniq = append(uniq, e)
		}
	}
	var ib strings.Builder
	ib.WriteString("# " + strings.TrimSuffix(stem, "-log") + " — ITERATION INDEX\n\n" +
		"One line per iteration, newest first, covering the live log AND the archive.\n\n" +
		"**GREP THIS BEFORE PICKING WORK.** It is the cheapest way to find out whether\n" +
		"something has already been tried, and it is small enough to read in full — which the\n" +
		"log (2.8 MB, ~715k tokens) has not been for a long time.\n\n" +
		"Regenerated wholesale by `ailang mission rotate-log`, never appended to: an\n" +
		"append-only index drifts the moment an entry is edited, and an index that answers\n" +
		"confidently and wrongly is worse than none.\n\n" +
		"| # | date | what happened |\n|---|---|---|\n")
	for _, e := range uniq {
		ib.WriteString(indexLine(e) + "\n")
	}
	res.IndexEntries = len(uniq)
	if werr := writeAtomic(indexPath, []byte(ib.String()), 0o644); werr != nil { //nolint:gosec // a doc
		return nil, werr
	}
	return res, nil
}
