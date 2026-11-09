# Sprint plan — `rotate-log:status-archive-structural-header` (fleet iteration 46)

Ticket: `ailang mission rotate-log docs --stream status` refuses (rc 1) because
`design_docs/docs-mission-status-archive.md` has a structural `## ` heading between its entries.
The refusal (`internal/mission/rotate.go` `strayHeadings`, called from `RotateLog`) is correct and
stays unchanged. This sprint fixes the data and adds a CI ratchet. **No change to `rotate.go`.**

Base: `0384450a6`. Worktree: `/Users/voightkampff/.ailang-driver-pin/fleet-i46-docs-status-structural`.
Run every command from the worktree root. Do not run any git write command.

## Measured facts (planner, at base)

| # | Claim | Status |
|---|-------|--------|
| F1 | The tombstone is lines 463-467 of `design_docs/docs-mission-status-archive.md`; line 468 is blank; line 469 is `## STATUS 2026-08-28 — ITERATION 0: RUN: …` | VERIFIED `nl -ba … \| sed -n 460,469p` |
| F2 | The file has **no preamble**: line 1 is already `## STATUS 2026-09-07 — ITERATION 14: …` | VERIFIED `head -1` |
| F3 | Base smoke: rc 1, `First: "## (Charter sections — moved back 2026-10-01)"` | VERIFIED temp copy, binary built from base |
| F4 | Stray counts (`parseLog` + `strayHeadings`) on the files rotation treats as LIVE (`*-mission-log.md`, `*-mission-status-archive.md`): docs-log 1, docs-status 1, motoko-status 11, all others 0. Total 13; after the fix it is **12** | VERIFIED temp probe test (deleted afterwards) |
| F5 | `missionDocsInRepo` ALSO returns archives (`*-log-archive.md`, `*-status-archive-old.md`). `v1-mission-status-archive-old.md` has 1 stray (`## STATUS 2026-07-10/11 (night) …`). Counting through it unfiltered gives 13 after the fix, not 12. Rotation never treats an archive as live, so the test filters them out | VERIFIED same probe |
| F6 | The tombstone heading is not a record (`looksLikeRecord`: no "iteration", "## status" or leading digit), so moving it leaves `nonCanonicalHeadings` at 1 for this file and `knownNonCanonical = 11` holds | VERIFIED probe: noncanon 1 before and after |
| F7 | Trial fix (the exact edit in M1) → smoke rc 0, 16 of 16 entries kept, tombstone at line 6 after rotation, body below the preamble byte-identical to the original minus lines 463-468 | VERIFIED temp copy + `diff` |
| F8 | `git show 5d39a5c75:design_docs/docs-mission-status-archive.md` resolves | VERIFIED `git cat-file -e` |
| F9 | `design_docs/docs-mission.md:167` tells the loop to "move the now-4th stamp to the TOP of the archive file". If it is followed literally, the next stamp lands above the new preamble's `## ` heading and the stray comes back | VERIFIED file read; the recurrence itself is UNVERIFIED (the docs mission is disabled) |
| F10 | Baseline `go test ./internal/mission/ -run TestMissionDocHeadingsStayCanonical -count=1` → PASS | VERIFIED |
| F11 | Windows CI checks out `design_docs/*.md` with CRLF (`.gitattributes` is `* text=auto`) | VERIFIED `.gitattributes`; the effect on the regexes is UNVERIFIED, so the test strips `\r` |

## M1 — Data fix (archive preamble)

Edit `design_docs/docs-mission-status-archive.md`:
1. **Delete lines 463-468**: the `## (Charter sections — moved back 2026-10-01)` heading, its blank
   line, the 3-line paragraph, and the blank line after it. Line 462 (blank) stays, so the
   `## STATUS 2026-08-28 — ITERATION 0: RUN` entry still follows a single blank line.
2. **Insert at the very top of the file** (above the current line 1) these lines, then one blank line:
   ```
   # Docs Mission — STATUS archive

   Older STATUS stamps rotated out of [docs-mission.md](docs-mission.md) (newest 3 stay there). Append-only.
   New stamps go directly BELOW this preamble, above the newest `## STATUS` stamp — never above the title.

   <the 5 deleted lines 463-467, VERBATIM, `## ` heading kept>
   ```
   The heading stays `## ` on purpose. If it is ever buried again, `strayHeadings` and the M2
   ratchet report it. As `### ` it would be silently absorbed into an entry's body.
3. Mechanical recipe (exact; it reproduces F7):
   ```bash
   f=design_docs/docs-mission-status-archive.md; tmp=$(mktemp)
   { printf '# Docs Mission — STATUS archive\n\nOlder STATUS stamps rotated out of [docs-mission.md](docs-mission.md) (newest 3 stay there). Append-only.\nNew stamps go directly BELOW this preamble, above the newest `## STATUS` stamp — never above the title.\n\n'
     sed -n 463,467p "$f"; printf '\n'; sed '463,468d' "$f"; } > "$tmp" && mv "$tmp" "$f"
   ```
4. Edit `design_docs/docs-mission.md:167`. In that line, replace the words
   `to the TOP of the archive file.` with
   `directly below the archive's preamble (above its newest STATUS stamp).`
   Change nothing else in the charter (F9).

## M2 — Regression ratchet (`internal/mission/canonical_test.go`)

Append to the file (149 lines now). Reuse `missionDocs(t)`, `parseLog` and `strayHeadings`.
The name contains `Canonical`, so the Gate-4 check `go test ./internal/mission/ -run Canonical`
(in `.claude/skills/mission-control/resources/gate-4-record.md:137`) also runs it.

```go
// knownStrayHeadings is a RATCHET over the files rotate-log treats as LIVE (*-mission-log.md,
// *-mission-status-archive.md): "## " headings interleaved with entries, which make RotateLog
// refuse. 11 are motoko's status-archive structure (deliberately unrotated), 1 is docs-log's
// irregular "## ITERATION 3 — died mid-flight…" record. Archives are excluded: never rotated as live.
// May FALL (lower it), never rise: a new one should fail CI where it is added, not at a later rotation.
const knownStrayHeadings = 12

func TestCanonicalRecordStreamsStrayHeadingsRatchet(t *testing.T) {
	var live []string
	for _, d := range missionDocs(t) {
		b := filepath.Base(d)
		if strings.HasSuffix(b, "-mission-log.md") || strings.HasSuffix(b, "-mission-status-archive.md") {
			live = append(live, d)
		}
	}
	if len(live) == 0 {
		t.Skip("no mission record streams on this machine")
	}
	var stray []string
	for _, d := range live {
		raw, err := os.ReadFile(d) //nolint:gosec // repo-relative doc
		if err != nil {
			t.Fatal(err)
		}
		body := strings.ReplaceAll(string(raw), "\r\n", "\n") // Windows checkout (F11)
		_, entries := parseLog(body)
		for _, h := range strayHeadings(body, entries) {
			stray = append(stray, filepath.Base(d)+": "+h)
		}
	}
	switch {
	case len(stray) > knownStrayHeadings:
		t.Errorf("interleaved structural headings ROSE to %d (limit %d) — rotate-log will refuse these files. "+
			"Move the section above the first entry; do not raise the constant.\n  %s",
			len(stray), knownStrayHeadings, strings.Join(stray, "\n  "))
	case len(stray) < knownStrayHeadings:
		t.Errorf("interleaved structural headings FELL to %d — good. Lower knownStrayHeadings to %d.\n  %s",
			len(stray), len(stray), strings.Join(stray, "\n  "))
	}
}
```
No new imports are needed (`os`, `path/filepath`, `strings`, `testing` are already imported).

## Test plan — each arm must FAIL under its mutation

Mutate from a scratchpad copy only: `cp <file> /tmp/i46-orig`, mutate, run, `cp /tmp/i46-orig <file>`,
then check that `git diff --stat` shows only the sprint's intended edits. **Never `git checkout <file>`.**

| Arm | Mutation | Command | Must show |
|-----|----------|---------|-----------|
| a | Put the 5 tombstone lines back mid-stream (before `## STATUS 2026-08-28 — ITERATION 0: RUN`) | `go test ./internal/mission/ -run StrayHeadingsRatchet -count=1` | FAIL "ROSE to 13" |
| b | Delete the `## Skill` line from `design_docs/motoko-mission-status-archive.md` | same | FAIL "FELL to 11" |
| c | Restore the pre-M1 archive (`git show HEAD:design_docs/docs-mission-status-archive.md > design_docs/docs-mission-status-archive.md`) | rotation smoke below + ratchet | smoke rc 1 naming the tombstone; ratchet "ROSE to 13" |
| d | none (green) | ratchet + `-run TestMissionDocHeadingsStayCanonical` | both PASS |

## Acceptance (the controller re-runs these outside the sandbox)

```bash
go test ./internal/mission/ -count=1                      # all PASS, including both ratchets
go test ./internal/mission/ -run TestMissionDocHeadingsStayCanonical -count=1 -v   # PASS (F6)
go vet ./internal/mission/ && test -z "$(gofmt -l internal/mission)"
scripts/changelog_fold.sh --check                          # fragment accepted
# rotation smoke on a temp copy; expect rc 0 and the tombstone still in the preamble
T=$(mktemp -d); mkdir -p $T/missions $T/design_docs $T/bin
cp missions/*.toml $T/missions/; cp design_docs/*-mission*.md $T/design_docs/
go build -o $T/bin/ailang ./cmd/ailang
(cd $T && AILANG_MISSION_REGISTRY=$T/missions $T/bin/ailang mission rotate-log docs --stream status --keep 1000); echo rc=$?
f=$T/design_docs/docs-mission-status-archive.md
awk '/^## STATUS/{exit} /Charter sections — moved back/{print "IN-PREAMBLE"}' $f   # prints IN-PREAMBLE
grep -c '^## STATUS' $f                                    # 16
```

## M3 — Changelog fragment `changelogs/unreleased/2026-10-17-docs-status-archive-preamble.md`

```markdown
### Fixed — `ailang mission rotate-log docs --stream status` refused the docs STATUS archive (2026-10-17)

A tombstone section, `## (Charter sections — moved back 2026-10-01)`, sat between the entries of
`design_docs/docs-mission-status-archive.md`, so rotation correctly refused to archive it as part of
an entry. It now sits in a new preamble above the first stamp, with its text unchanged, and the
docs charter tells the loop to file new stamps below that preamble rather than at the file's top.
The refusal is unchanged. A new ratchet test, `TestCanonicalRecordStreamsStrayHeadingsRatchet`,
counts interleaved structural headings in every live record stream (12: motoko's status
structure and one irregular docs-log record) and fails if the count changes, so the next one
fails CI in the commit that adds it.
```

## Out of scope (observed, not fixed — do not touch)
- docs-log's `## ITERATION 3 — died mid-flight…` (counted in the constant; fixing it needs `ailang mission normalize`).
- motoko's 11 structural sections; v1's status archive repeating its rotation banner, with one stamp filed above it.
