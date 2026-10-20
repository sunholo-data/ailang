package mission

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// indexPaths retains the distinct log and status naming conventions in both modes.
func indexPaths(logPath string) (stem, archivePath, indexPath string) {
	dir := filepath.Dir(logPath)
	stem = strings.TrimSuffix(filepath.Base(logPath), ".md")
	archivePath = filepath.Join(dir, stem+"-archive.md")
	base := strings.TrimSuffix(stem, "-log")
	if strings.HasSuffix(stem, "-status-archive") {
		base = strings.TrimSuffix(stem, "-archive")
		archivePath = filepath.Join(dir, stem+"-old.md")
	}
	return stem, archivePath, filepath.Join(dir, base+"-index.md")
}

// Only absent optional history is ordinary first-use state. Never hide other I/O errors.
func readOptionalIndexInput(path string) ([]byte, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // derived mission path
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	return raw, nil
}

// RegenerateIndex rebuilds only the index, without moving or rewriting either source.
// Structural headings are safe to ignore here because no entry bodies move.
func RegenerateIndex(logPath string) (*RotateResult, error) {
	raw, err := os.ReadFile(logPath) //nolint:gosec // caller-supplied mission path
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", logPath, err)
	}
	stem, archivePath, indexPath := indexPaths(logPath)
	archiveRaw, err := readOptionalIndexInput(archivePath)
	if err != nil {
		return nil, err
	}
	_, live := parseLog(string(raw))
	_, archived := parseLog(string(archiveRaw))
	all := append(archived, live...)
	if len(all) == 0 {
		return nil, fmt.Errorf("%s and %s: no iteration entries found; add numbered headings or recognized notes before regenerating the index", logPath, archivePath)
	}
	prepared, err := prepareIndex(stem, indexPath, all)
	if err != nil {
		return nil, err
	}
	res := &RotateResult{LogPath: logPath, ArchivePath: archivePath, IndexPath: indexPath,
		IgnoredHeadings: ignoredIndexHeadings(string(raw)) + ignoredIndexHeadings(string(archiveRaw))}
	if err := writeIndex(indexPath, prepared, res); err != nil {
		return nil, err
	}
	return res, nil
}

func ignoredIndexHeadings(body string) int {
	count := 0
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "## ") && !entryHeadingRe.MatchString(line) &&
			!statusParenIterRe.MatchString(line) &&
			!statusHeadingRe.MatchString(line) && !statusNoteRe.MatchString(line) && !noteHeadingRe.MatchString(line) &&
			!legacyEntryRe.MatchString(line) && !legacyNoteRe.MatchString(line) {
			count++
		}
	}
	return count
}

type indexRow struct {
	num       int
	date, raw string
}
type preparedIndex struct {
	body    []byte
	rows    int
	orphans []int
}

// prepareIndex merges source-derived rows and raw numbered orphans before any writes.
func prepareIndex(stem, indexPath string, all []LogEntry) (*preparedIndex, error) {
	old, err := readOptionalIndexInput(indexPath)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Num != all[j].Num {
			return all[i].Num > all[j].Num
		}
		return all[i].Date > all[j].Date
	})
	seen := map[string]bool{}
	numbered := map[int]bool{}
	var rows []indexRow
	for _, e := range all {
		key := strconv.Itoa(e.Num)
		if e.Num < 0 {
			key = e.Date + "|" + e.Title
		} else {
			numbered[e.Num] = true
		}
		if !seen[key] {
			seen[key] = true
			rows = append(rows, indexRow{e.Num, e.Date, indexLine(e)})
		}
	}
	orphanSet := map[int]bool{}
	for _, line := range strings.Split(string(old), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		cells := strings.SplitN(line, "|", 3)
		if len(cells) < 3 {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(cells[1]))
		if err != nil || n < 0 || numbered[n] {
			continue
		}
		rows = append(rows, indexRow{num: n, raw: line})
		orphanSet[n] = true
	}
	// Stable ordering keeps duplicate orphan rows in their original relative order.
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].num != rows[j].num {
			return rows[i].num > rows[j].num
		}
		return rows[i].date > rows[j].date
	})
	var orphans []int
	for n := range orphanSet {
		orphans = append(orphans, n)
	}
	sort.Ints(orphans)
	var b strings.Builder
	b.WriteString(indexPreamble(stem))
	b.WriteString(indexQuotes(string(old)))
	b.WriteString("| # | date | what happened |\n|---|---|---|\n")
	for _, row := range rows {
		b.WriteString(row.raw + "\n")
	}
	return &preparedIndex{[]byte(b.String()), len(rows), orphans}, nil
}

// indexQuotes keeps only pre-table blockquote blocks, including quoted blank lines.
// Blank separation between adjacent blocks is retained; old prose and tables are not.
func indexQuotes(old string) string {
	var out, pending strings.Builder
	for _, line := range strings.Split(old, "\n") {
		cells := strings.SplitN(line, "|", 3)
		if len(cells) == 3 && strings.TrimSpace(cells[0]) == "" && strings.TrimSpace(cells[1]) == "#" {
			break
		}
		if strings.HasPrefix(line, ">") {
			out.WriteString(pending.String())
			pending.Reset()
			out.WriteString(line + "\n")
		} else if strings.TrimSpace(line) == "" && out.Len() > 0 {
			pending.WriteString(line + "\n")
		} else {
			pending.Reset()
		}
	}
	if out.Len() == 0 {
		return ""
	}
	return out.String() + "\n"
}

func indexPreamble(stem string) string {
	return "# " + strings.TrimSuffix(stem, "-log") + " — ITERATION INDEX\n\n" +
		"One line per iteration, newest first, covering the live log AND the archive.\n\n" +
		"**GREP THIS BEFORE PICKING WORK.** It is the cheapest way to find out whether\n" +
		"something has already been tried, and it is small enough to read in full — which the\n" +
		"log (2.8 MB, ~715k tokens) has not been for a long time.\n\n" +
		"Regenerated wholesale by `ailang mission rotate-log`, never appended to: an\n" +
		"append-only index drifts the moment an entry is edited, and an index that answers\n" +
		"confidently and wrongly is worse than none.\n\n"
}

// writeIndex is the single writer used by rotation and regeneration.
func writeIndex(path string, prepared *preparedIndex, res *RotateResult) error {
	if err := writeAtomic(path, prepared.body, 0o644); err != nil {
		return err
	}
	res.IndexEntries = prepared.rows
	res.OrphanNumbers = prepared.orphans
	return nil
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
