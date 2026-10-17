package mission

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func indexWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
func indexRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func fixturePaths(t *testing.T, stream string) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	if stream == "status" {
		return filepath.Join(dir, "test-mission-status-archive.md"), filepath.Join(dir, "test-mission-status-archive-old.md"), filepath.Join(dir, "test-mission-status-index.md")
	}
	return filepath.Join(dir, "test-mission-log.md"), filepath.Join(dir, "test-mission-log-archive.md"), filepath.Join(dir, "test-mission-index.md")
}
func runIndexMode(path, mode string) (*RotateResult, error) {
	if mode == "rotate" {
		return RotateLog(path, 1)
	}
	return RegenerateIndex(path)
}
func numbered(n int, title string) string {
	return fmt.Sprintf("## %d — 2026-10-17 — %s\n\nfull body %d\n", n, title, n)
}

const handQuotes = "> **Reset date.** Keep this exact.\n> more text\n>\n> after quoted blank\n\n> Second block — 手書き.\n"
const tableHeader = "| # | date | what happened |\n|---|---|---|\n"
const orphan182 = "|  182  | old date | 手書き \\| unusual   spacing |"

func TestMissionIndexPreservation(t *testing.T) {
	for _, arm := range []string{"PreserveOrphanRow", "PreserveHandNote", "SourceTitlesWinAndNotesDedupe", "PreservationIdempotent"} {
		t.Run(arm, func(t *testing.T) {
			for _, mode := range []string{"rotate", "index-only"} {
				t.Run(mode, func(t *testing.T) {
					live, archive, index := fixturePaths(t, "log")
					title := strings.Repeat("界—", 100) + " | heading"
					indexWrite(t, live, "# Preamble\n\n"+numbered(181, "source wins")+numbered(183, "newer")+numbered(184, title)+"## STATUS 2026-10-17 — note A\nbody\n## STATUS 2026-10-17 — note B\nbody\n")
					if arm == "SourceTitlesWinAndNotesDedupe" {
						indexWrite(t, archive, "# Archive\n"+numbered(181, "source wins")+"## STATUS 2026-10-17 — note A\nbody\n")
					}
					// Other arms start without an archive, exercising first-use and idempotence.
					old := "# Old prose\n\n" + handQuotes + "\n" + tableHeader + "| 181 | stale | hand edited |\n" + orphan182 + "\n| 182 | duplicate | second |\n| 0 | ancient | zero |\n| — | old | orphan note |\n> record quote must not be preserved\n"
					indexWrite(t, index, old)
					res, err := runIndexMode(live, mode)
					if err != nil {
						t.Fatal(err)
					}
					got := indexRead(t, index)
					switch arm {
					case "PreserveOrphanRow":
						if !strings.Contains(got, orphan182+"\n| 182 | duplicate | second |\n") {
							t.Fatalf("raw orphans lost: %s", got)
						}
						if !(strings.Index(got, "| 183 |") < strings.Index(got, orphan182) && strings.Index(got, orphan182) < strings.Index(got, "| 181 |")) {
							t.Fatal("orphan numeric order")
						}
						if !reflect.DeepEqual(res.OrphanNumbers, []int{0, 182}) || res.IndexEntries != 8 {
							t.Fatalf("metadata: %+v", res)
						}
					case "PreserveHandNote":
						if strings.Count(got, handQuotes) != 1 || strings.Index(got, handQuotes) > strings.Index(got, tableHeader) || strings.Contains(got, "record quote must not be preserved") || strings.Contains(got, "Old prose") {
							t.Fatalf("quotes lost/reordered: %s", got)
						}
					case "SourceTitlesWinAndNotesDedupe":
						if strings.Contains(got, "hand edited") || strings.Contains(got, "orphan note") || strings.Count(got, "| 181 |") != 1 || strings.Count(got, "| note A |") != 1 || strings.Count(got, "| note B |") != 1 {
							t.Fatalf("wrong source precedence/dedupe: %s", got)
						}
						if !strings.Contains(got, indexLine(LogEntry{Num: 184, Date: "2026-10-17", Title: title})) || !utf8.ValidString(got) {
							t.Fatal("title rendering changed")
						}
					case "PreservationIdempotent":
						_, err = runIndexMode(live, mode)
						if err != nil {
							t.Fatal(err)
						}
						if got != indexRead(t, index) {
							t.Fatal("second regeneration changed index")
						}
					}
				})
			}
		})
	}
}

func TestMissionIndexOnlySourcesUnchanged(t *testing.T) {
	for _, stream := range []string{"log", "status"} {
		t.Run(stream, func(t *testing.T) {
			for _, withArchive := range []bool{false, true} {
				t.Run(fmt.Sprintf("archive=%t", withArchive), func(t *testing.T) {
					live, archive, index := fixturePaths(t, stream)
					source := "# preamble\n## Routing\nkeep routing\n" + numbered(2, "live") + "## Backlog\nstructural body\n"
					indexWrite(t, live, source)
					if withArchive {
						indexWrite(t, archive, numbered(1, "archived")+"## Archive structure\nkeep me\n")
					}
					before := sha256.Sum256([]byte(source))
					var ab string
					if withArchive {
						ab = indexRead(t, archive)
					}
					res, err := RegenerateIndex(live)
					if err != nil {
						t.Fatal(err)
					}
					if got := indexRead(t, live); got != source || sha256.Sum256([]byte(got)) != before {
						t.Fatal("live modified")
					}
					if withArchive {
						if got := indexRead(t, archive); got != ab || sha256.Sum256([]byte(got)) != sha256.Sum256([]byte(ab)) {
							t.Fatal("archive modified")
						}
					} else if _, err := os.Stat(archive); !os.IsNotExist(err) {
						t.Fatalf("archive created: %v", err)
					}
					if res.IndexPath != index {
						t.Fatalf("index path: %s", res.IndexPath)
					}
				})
			}
		})
	}
}

func TestMissionIndexOnlyEmptyLive(t *testing.T) {
	live, archive, index := fixturePaths(t, "log")
	indexWrite(t, live, "# Empty live\n## Routing\ntext\n")
	indexWrite(t, archive, numbered(0, "history"))
	before := indexRead(t, live)
	res, err := RegenerateIndex(live)
	if err != nil {
		t.Fatal(err)
	}
	if res.IndexEntries != 1 || !strings.Contains(indexRead(t, index), "| 0 |") || indexRead(t, live) != before {
		t.Fatalf("empty-live regeneration: %+v", res)
	}
}
func TestMissionIndexOnlyNotesAreRecords(t *testing.T) {
	live, archive, index := fixturePaths(t, "status")
	indexWrite(t, live, "# Preamble only\n")
	indexWrite(t, archive, "## STATUS 2026-10-17 — scope decision\nbody\n")
	res, err := RegenerateIndex(live)
	if err != nil {
		t.Fatal(err)
	}
	if res.IndexEntries != 1 || !strings.Contains(indexRead(t, index), "| — | 2026-10-17 | scope decision |") {
		t.Fatal("note-only archive was not indexed")
	}
}

func TestMissionIndexOnlyEmptyBoth(t *testing.T) {
	live, archive, index := fixturePaths(t, "log")
	for _, p := range []string{live, archive} {
		indexWrite(t, p, "# Empty\n## Routing\ntext\n")
	}
	indexWrite(t, index, tableHeader+orphan182+"\n")
	before := map[string]string{}
	for _, p := range []string{live, archive, index} {
		before[p] = indexRead(t, p)
	}
	if _, err := RegenerateIndex(live); err == nil || !strings.Contains(err.Error(), "no iteration entries") {
		t.Fatalf("expected actionable empty error: %v", err)
	}
	for p, want := range before {
		if indexRead(t, p) != want {
			t.Fatalf("modified %s", p)
		}
	}
}
func TestMissionIndexOnlyIgnoresStructuralHeadings(t *testing.T) {
	for _, stream := range []string{"log", "status"} {
		t.Run(stream, func(t *testing.T) {
			live, archive, index := fixturePaths(t, stream)
			indexWrite(t, live, "## Routing\nsecret structural text\n"+numbered(3, "live")+"## Backlog\nsecret structural text\n## STATUS 2026-10-17 — source note\nbody\n## Skills\nsecret structural text\n")
			indexWrite(t, archive, "## Archive routing\nsecret structural text\n## STATUS 2026-10-16 — ITERATION 2: archived status\nbody\n## Archive backlog\nsecret structural text\n")
			res, err := RegenerateIndex(live)
			if err != nil {
				t.Fatal(err)
			}
			got := indexRead(t, index)
			if res.IgnoredHeadings != 5 {
				t.Fatalf("ignored %d, want 5", res.IgnoredHeadings)
			}
			for _, s := range []string{"Routing", "Backlog", "Skills", "secret structural text"} {
				if strings.Contains(got, s) {
					t.Fatalf("structure indexed: %s", s)
				}
			}
			if res.IndexEntries != 3 || !strings.Contains(got, "| source note |") || !strings.Contains(got, "| archived status |") {
				t.Fatalf("source records lost: %s", got)
			}
		})
	}
}
func TestMissionNormalRotationArchives(t *testing.T) {
	live, archive, _ := fixturePaths(t, "log")
	old := numbered(1, "old")
	newest := numbered(2, "new")
	indexWrite(t, live, "# preamble\n## Routing\nretained\n"+old+newest)
	res, err := RotateLog(live, 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.Archived != 1 || !strings.Contains(indexRead(t, archive), strings.TrimSpace(old)) || strings.Contains(indexRead(t, live), "full body 1") || !strings.Contains(indexRead(t, live), strings.TrimSpace(newest)) || !strings.Contains(indexRead(t, live), "## Routing\nretained") {
		t.Fatal("rotation did not move full bodies")
	}
}
func TestMissionNormalRotationRefusesStructure(t *testing.T) {
	live, archive, index := fixturePaths(t, "log")
	for _, p := range []string{live, archive, index} {
		indexWrite(t, p, numbered(1, "old")+"## Backlog\nstructural\n"+numbered(2, "new"))
	}
	before := map[string]string{}
	for _, p := range []string{live, archive, index} {
		before[p] = indexRead(t, p)
	}
	if _, err := RotateLog(live, 1); err == nil || !strings.Contains(err.Error(), "Backlog") {
		t.Fatalf("structure not refused: %v", err)
	}
	for p, want := range before {
		if indexRead(t, p) != want {
			t.Fatalf("modified %s", p)
		}
	}
}
func TestMissionIndexReadErrorsBeforeWrites(t *testing.T) {
	for _, mode := range []string{"rotate", "index-only"} {
		for _, bad := range []string{"archive", "index", "live"} {
			t.Run(mode+"/"+bad, func(t *testing.T) {
				live, archive, index := fixturePaths(t, "log")
				indexWrite(t, live, sampleLog(3))
				before := indexRead(t, live)
				target := map[string]string{"archive": archive, "index": index, "live": live}[bad]
				if bad == "live" {
					if err := os.Remove(live); err != nil {
						t.Fatal(err)
					}
				} else if err := os.Mkdir(target, 0o700); err != nil {
					t.Fatal(err)
				}
				if _, err := runIndexMode(live, mode); err == nil {
					t.Fatal("read error silently ignored")
				}
				if bad != "live" && indexRead(t, live) != before {
					t.Fatal("live changed before read error")
				}
			})
		}
	}
}
func indexNumbers(body string) []int {
	seen := map[int]bool{}
	for _, line := range strings.Split(body, "\n") {
		cells := strings.Split(line, "|")
		if len(cells) < 3 {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(cells[1]))
		if err == nil && n >= 0 {
			seen[n] = true
		}
	}
	var nums []int
	for n := range seen {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	return nums
}
func resetBlock(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, "> **Reset ")
	if start < 0 {
		t.Fatal("Reset block missing")
	}
	var lines []string
	for _, l := range strings.Split(body[start:], "\n") {
		if !strings.HasPrefix(l, ">") {
			break
		}
		lines = append(lines, l)
	}
	return strings.Join(lines, "\n") + "\n"
}
func TestMissionMotokoSnapshot(t *testing.T) {
	dir := t.TempDir()
	original := map[string][]byte{}
	for _, suffix := range []string{"log", "log-archive", "index"} {
		name := "motoko-mission-" + suffix + ".md"
		b, err := os.ReadFile(filepath.Join("..", "..", "design_docs", name))
		if err != nil {
			t.Fatal(err)
		}
		original[name] = b
		indexWrite(t, filepath.Join(dir, name), string(b))
	}
	live := filepath.Join(dir, "motoko-mission-log.md")
	index := filepath.Join(dir, "motoko-mission-index.md")
	wantNums := indexNumbers(string(original["motoko-mission-index.md"]))
	wantReset := resetBlock(t, string(original["motoko-mission-index.md"]))
	if _, err := RegenerateIndex(live); err != nil {
		t.Fatal(err)
	}
	got := indexRead(t, index)
	if !reflect.DeepEqual(indexNumbers(got), wantNums) {
		t.Fatalf("number sets differ: %v vs %v", indexNumbers(got), wantNums)
	}
	if resetBlock(t, got) != wantReset || strings.Count(got, wantReset) != 1 {
		t.Fatal("Reset changed")
	}
	if !utf8.ValidString(got) {
		t.Fatal("invalid UTF-8")
	}
	for _, suffix := range []string{"log", "log-archive"} {
		name := "motoko-mission-" + suffix + ".md"
		if sha256.Sum256([]byte(indexRead(t, filepath.Join(dir, name)))) != sha256.Sum256(original[name]) {
			t.Fatalf("source hash changed: %s", name)
		}
	}
	// Every source-backed heading title, including 38/39, follows production rendering.
	var entries []LogEntry
	for _, suffix := range []string{"log-archive", "log"} {
		_, es := parseLog(string(original["motoko-mission-"+suffix+".md"]))
		entries = append(entries, es...)
	}
	for _, e := range entries {
		if !strings.Contains(got, indexLine(e)) {
			t.Fatalf("heading-derived row missing: %d", e.Num)
		}
	}
	t.Logf("number sets equal: %v (%d numbers); Reset exact; source sha256 unchanged; UTF-8 valid", wantNums, len(wantNums))
}
