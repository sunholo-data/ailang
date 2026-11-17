# Sprint Plan: M-FLEET-RULINGS-LEDGER — fleet-wide rulings ledger + `mission doctor` ruling notes

**Design doc**: [m-fleet-rulings-ledger.md](m-fleet-rulings-ledger.md) (R4.1, `78bbb7515`)
**Ruling implemented**: D-FLEET-25 A (`design_docs/fleet-mission.md:173`, RESOLVED, Mark attended 2026-11-16)
**Sprint JSON**: `.ailang/state/sprints/sprint_M-FLEET-RULINGS-LEDGER.json` (tracked: `.gitignore:105` whitelists it, and the fleet pre-push guard allows `.ailang/state/sprints/*`, `tools/launchd/githooks/pre-push:31`)
**Branch**: `fleet/i118-rulings-ledger` (worktree `.fleet-i118-rulings`, base origin/dev `950a0cc54`)
**Planned**: 2026-11-17, fleet iteration 118 (planner, unattended)

## Placement and Human-Decision Rules

This plan sits in the planned root beside its design doc, not in `ailang-core-triage/`.

**There are no human-decision gates.** D-FLEET-25 is RESOLVED ("A (as recommended)"). It is copied here as a
ruling, not a gate. Nothing in this sprint resolves, activates or edits a ruling on Mark's behalf. The two
seed rows are **transcriptions** of rulings already RESOLVED elsewhere (D-FLEET-25, V1 D-62/D-63), and each one
cites its source row (design §5).

**Controller-only precondition (not an executor step):** the design's merge-order condition (quorum R4, oc-glm-5-3,
verbatim): *"this PR rebases onto post-#2396 dev and re-runs V23 on that base before merge."* #2396 is still
OPEN (P12). The executor neither waits for it nor merges anything.

## Summary

Add one fleet-wide rulings ledger (`design_docs/fleet-rulings.md`), link it from the five in-repo charters
and from Gate 0 of the mission-control skill (both mirrors), and add a **report-only** check to
`ailang mission doctor`. The check prints `note` findings for charter, skill or registry text that contains
a literal a RESOLVED, unsuperseded ruling retired. Notes never change the exit code.

**Duration:** 2 days (7 executor runs of 45–90 min each, about 8 h of executor time; rounded up from the design's 1.5 days at 2x)
**Dependencies:** none blocking. Sibling #2396 only changes D-FR-1's live count (1 → 0).
**Estimated total:** ~1,630 LOC across M1–M7 (about 700 lines of Go and Markdown, 930 of tests and testdata).
**Risk Level:** Low. Read-only Go code, all findings are `Note`, one changed line in `DoctorWith`, and one output change in the CLI.

## Rulings carried into the plan (not re-opened)

| Ruling | Source | Effect on this plan |
|---|---|---|
| D-FLEET-25 A | `fleet-mission.md:173` | Build: one ledger, a charter link, and a doctor check that REPORTS |
| Every ruling finding is `Note` | design §3 | No milestone may emit `Drift` from ruling code (T6 kills that mutant) |
| Forbid-only grammar, no `require`, no `except` | design §1 | Parser rejects any other verb |
| Whole ledger fails on any malformed line | design §3 (quorum R4 kimi, verbatim) | Malformed → exactly one finding, zero detectors run |
| `skills` scope enumerated with `filepath.WalkDir` over each `mission-*` dir, then sorted | quorum R4 gemini-4-argon (non-blocking) | M2 `rulingSkillFiles`. `filepath.Glob` has no `**` |
| `doctor_test.go`, `missions/*.toml`, `scripts/mission_*.sh`, `tools/launchd/**` not touched | design "Files touched" | No milestone edits them |

## Planner verification (re-checked at this worktree's HEAD `78bbb7515`, 2026-11-17)

Every claim below was re-run, not inherited. Each empty result is paired with a control that returns a positive.

| # | Design claim | Command | Observed | Verdict |
|---|---|---|---|---|
| P1 | `doctor.go:136` is a constant; `:151` `Doctor`; `:154` opens `DoctorWith`; loop closes `:294`, `:295` sort, `:296` return | `grep -n 'const sharedDriverRepo\|^func Doctor\|^func DoctorWith\|sort.Slice(rep.Rows\|return rep$' internal/mission/doctor.go`; `sed -n '290,297p'` | `136: const sharedDriverRepo = "sunholo-data/ailang"`, `151: func Doctor(…) { return DoctorWith(reg, p, nil) }`, `154: func DoctorWith(reg *Registry, p Paths, lc LaunchCtl) *Report {`, `294: }` (loop), `295: sort.Slice(rep.Rows…)`, `296: return rep` | ✓ |
| P2 | `doctor.go` has no charter / `m.Doc` logic | `grep -n 'm\.Doc' internal/mission/doctor.go; echo rc=$?`; control `grep -n 'm\.Doc' internal/mission/*.go` | no output, `rc=1`; control finds `registry.go:132`, `render.go:103` (+ tests) | ✓ |
| P3 | `Finding` / `Note` / `HasDrift` | `sed -n '78,133p' internal/mission/doctor.go` | `Finding{Mission, Kind, Severity, Detail}`; `String()` = `[%s] %-8s %-22s %s`; `Note Severity = "note"`; `HasDrift` (`:118`) counts only `Drift` | ✓ |
| P4 | `Root()` at `registry.go:90`, set in `LoadFile` at `:251`, `Load`→`LoadFile` at `:271` | `grep -n 'm.root = abs\|m, err := LoadFile\|func (m \*Mission) Root' internal/mission/registry.go` | `90`, `251` (`filepath.Abs(filepath.Join(filepath.Dir(path), ".."))`), `271` | ✓. Fixture note: `Validate` (`:118-187`) requires an absolute `workdir`, a non-empty `doc` and `repo`, and a valid `[schedule]`, so test TOML must carry them (M3 helper does) |
| P5 | CLI: per-mission registry `mission_cmd.go:249`, guard `:255`, print `:256` | `grep -n 'reg = &mission.Registry\|if len(rep.Findings) == 0\|os.Exit(rep.ExitCode())\|exit 0 clean, 1 drift' cmd/ailang/mission_cmd.go` | `249`, `255`, `258` exit, help `126`; `missionList` (`:220`) reads `rep.Rows` only | ✓ |
| P6 | All `Doctor`/`DoctorWith` callers | `grep -rn 'DoctorWith(\|mission\.Doctor(\|[^a-zA-Z]Doctor(' --include='*.go' .` | `mission_cmd.go:220,251`; `golden_live_test.go:330`; `doctor_test.go:75,307,332,356,370,394` | ✓ |
| P7 | No test asserts a total finding count | `grep -rn 'Findings' --include='*_test.go' internal/mission cmd/ailang \| grep -v findingsOfKind` | only `range` loops and failure-message joins (`doctor_test.go:79,139,163,274,310,340`, `golden_live_test.go:331`). Control: the same grep on `doctor.go` finds `:114,119,159` | ✓ |
| P8 | `mission-lane-check.sh:49-50` keys on exit code, `tail -1` of filtered output | `sed -n '45,52p' tools/launchd/mission-lane-check.sh` | exact match | ✓ for exit 0 |
| **P8b** | *(design gap: never analysed)* What lands in the **`config dead`** cell when the doctor exits 1 and ruling notes are **appended** | two stubs through the exact `:49-50` expression under `/bin/bash` (`/tmp/fleet_i118_plan/stubA` = DRIFT then note, `stubB` = note then DRIFT, both exit 1) | stubA (design's append order) → `world config dead [note] world ruling-link-missing …`. The drift reason is **replaced** by a ruling note. stubB (prepend) → `world config dead [DRIFT] world env-source-drift X differs` | **✗ — adjusted.** M4 **prepends** ruling findings: `rep.Findings = append(CheckRulings(reg.Missions), rep.Findings...)`. Same insertion point, still one line. Arming is unaffected either way (exit code unchanged). Only the human-readable cell changes. V27 shows 6 live `env-source-drift` rows, so this path is the live case today |
| P9 | `mission-arm.sh:30-36` refuses on non-zero lane-check | `sed -n '28,38p' tools/launchd/mission-arm.sh` | `rc -ne 0` without `--force` → "NOT arming" exit 1 | ✓ |
| P10 | `mission_decisions.sh` / `mission_answer.sh` work unchanged on a non-charter D-FR ledger that has a detector block | scratch `/tmp/fleet_i118_plan/scratch-fleet-rulings.md` (D-FR-1 RESOLVED, D-FR-9 OPEN, 1 detector line); `--check --file`, `--open --file`; `mission_answer.sh --dry-run` with a scratch identity; `cmp` against the pre-run copy | `decision ledger valid: 2 rows`; `--open` → only `D-FR-9`; dry-run diff rewrites only the D-FR-9 row (detector line untouched); `--id D-FR-1` → "row D-FR-1 is RESOLVED, not OPEN — refusing"; `cmp` identical. Header (`\| ID \|`) and separator rows are **not** counted by the script (it matches `^\|[[:space:]]*D-` only), so the Go parser must skip them too (M1) | ✓ |
| P11 | Gate 0 mirrors exist and are byte-identical | `ls -l {.claude,.agents}/skills/mission-control/resources/gate-0-preflight.md; cmp …` | 27779 bytes each; `cmp` rc 0 | ✓ |
| P12 | Sibling #2396 state; post-#2396 `world.toml` | `gh pr view 2396 --json state,isDraft,headRefOid,mergedAt`; `git cat-file -t c34c409c3`; `git show c34c409c3:missions/world.toml \| grep -cF 'WORLD IS A FORK'` | `OPEN`, not draft, head `c34c409c3881…`, unmerged; object **present locally** (`commit`); count **0** (control: today's `missions/world.toml` = 1) | ✓ |
| P13 | Seed counts (V11/V12) | under `/bin/bash` (zsh does not word-split `$SK`: the first attempt under zsh printed `File name too long`, an instrument failure, not a zero): `cat $SK \| grep -cF …` over 32 skill files, `missions/*.toml`, 7 charters; awk inside/outside ledger | `WORLD IS A FORK`: registry 1 (`world.toml:3`), skills 0, charters 1 = `fleet-mission.md` inside=1 outside=0, the other 6 charters 0/0. `generator≠judge guard (HARD`: 0/0/0; control: 1 in each `349e311b9^` mirror blob (line 242). `generator≠judge — a PREFERENCE` = 2 in skills (`gate-3-route.md:257` ×2) | ✓ |
| P14 | Skills corpus | `ls -d .claude/skills/mission-* .agents/skills/mission-*`; `find … -name '*.md' \| wc -l`; `find … -type f ! -name '*.md'` | 3 dirs × 2 mirrors; **32** `.md`; no non-md files; `.md` files sit at depth 2 (`resources/*.md`), so a one-level glob misses 26 of them | ✓ |
| P15 | `.agents` mirror sync tool | `grep -rn 'agents skills in sync' tools/ make/ Makefile` | `tools/launchd/sync-agents-skills.sh:34`. Copy = `bash tools/launchd/sync-agents-skills.sh`; check = `bash tools/launchd/sync-agents-skills.sh --check` → today `agents skills in sync (mission-control mission-brief mission-loop-change sprint-planner sprint-executor sprint-evaluator release-manager)` rc 0. Enforced by `tools/launchd/test_agents_skills_sync.sh` inside `make test-launchd-drivers` (`make/test.mk:105`) | ✓ |
| P16 | Charter locations | `grep -nE '^(name\|repo\|workdir\|doc) *=' missions/*.toml`; `ls` both cross-repo charters | 5 × `repo = "sunholo-data/ailang"` (v1, fleet, docs, motoko, triage); world → `ailang-world`, stapledon → `stapledons-godot`, both charters exist at `<workdir>/design_docs/…`; all `doc` relative | ✓ |
| P17 | Baselines green before any change | `go test ./internal/mission/ ./cmd/ailang/ -run 'Mission\|Doctor\|Ruling' -count=1`; `make test-launchd-drivers` (unpiped, log to file) | `ok internal/mission 2.017s`, `ok cmd/ailang 39.900s`; `passed=36 failed=0`, `RC=0` | ✓ |
| P18 | File sizes | `wc -l` | `doctor.go` 407, `mission_cmd.go` 427, `registry.go` 282. All far from 800. New files are planned at ≤ 300 lines each | ✓ |
| P19 | Registry reuse | `ailang pkg search ledger`; `ailang pkg search lint` | one unrelated hit (`sunholo/relativity`); "No packages found". This is Go code inside `internal/mission`, so an AILANG package cannot supply it | `none` for every milestone |
| P20 | Write scope | `sed -n '13,36p' tools/launchd/githooks/pre-push` | harness `internal/mission/*`, `cmd/ailang/mission*`, `.claude/skills/mission-*`, `.agents/skills/mission-*`; support `design_docs/*`, `changelogs/*`, `docs/*`, `.ailang/state/sprints/*` | ✓. Every file in this plan is inside it |

**Design claims found false or incomplete:** one, **P8b**. The design's lane-check analysis (V26) covers only the
exit-0 path. On the exit-1 path, appending ruling notes after the per-mission findings puts a ruling note
into the `config dead` cell instead of the drift reason. **Adjustment:** prepend instead of append. M4 also
corrects the design doc's §3 call-site sentence, so the doc and the code agree. Two minor stale items, not
load-bearing: the `Root()` doc comment cites "LoadFile, :245–246" while the assignment is at `:251`; and the
brief's scope list omits `.ailang/state/sprints/*`, which the enforcing guard does allow (P20).

## Current Status Analysis

- Nothing of this design is built: `internal/mission/rulings*.go` and `design_docs/fleet-rulings.md` do not exist, and no charter links them (V14: 0/0/0).
- Velocity, last 7 days (`analyze_velocity.sh 7`): the dev line carried the Go-emitter removal and the `extern` fixes. The fleet's own recent sprints (one or two milestones per executor run, 100–400 LOC) set the milestone size used here.

## Registry Reuse Audit

| Milestone | Searches | Decision | Reason |
|---|---|---|---|
| M1–M7 | `ailang pkg search ledger`, `ailang pkg search lint` | `none` | Go code in `internal/mission` (doctor internals) plus Markdown. AILANG packages cannot be imported into the Go doctor |

## Conventions for every milestone (read once)

- **Work only in** `/Users/voightkampff/.ailang-driver-pin/.fleet-i118-rulings`. Commit at the end of each milestone. Do not push.
- **Test names** must match `-run 'Mission|Doctor|Ruling'` (the fleet done-gate regex). Use `TestRulings…`, `TestDoctor_…` or `TestMissionDoctor…`.
- **Lint per Go milestone:** `gofmt -l internal/mission cmd/ailang` (no output); `go vet ./internal/mission/` (and `./cmd/ailang/` in M5); `golangci-lint run ./internal/mission/...`. Every `os.ReadFile(path)` on a computed path carries `//nolint:gosec // <why the path is trusted>`, the same as `registry.go:239`.
- **Mutation procedure** (memory rule: scratchpad copy, never `git checkout <file>`):
  `mkdir -p /tmp/i118-mut && cp <file> /tmp/i118-mut/<file-basename>.orig` → apply the mutation by hand →
  `go test ./internal/mission/ -run '<TestName>' -count=1` must FAIL → `cp /tmp/i118-mut/<file-basename>.orig <file>` →
  the same test must PASS again → `git diff --stat` matches the pre-mutation state. Record each mutant and its FAIL line in the commit body.
- **Run gates unpiped** (`make … ; echo rc=$?`, never `make … | tail`). A pipe reports the pipe's exit code, not the gate's.
- **Never quote a seed literal** (`WORLD IS A FORK`, `generator≠judge guard (HARD`) in a charter outside its decision-ledger block, in a `missions/*.toml`, or in any `mission-*` skill file. Doing so creates a true `ruling-contradiction` note. Paraphrase instead. Test fixtures under `internal/mission/testdata/` and this plan are outside every scope.
- **Never `cp` over `~/go/bin/ailang`.** Build scratch binaries to `/tmp/i118-build/ailang` (M7).

## Proposed Milestones

### M1: Ledger parser — rows, detectors, bounds, supersession (~380 LOC)

**Goal:** a pure parser for `design_docs/fleet-rulings.md` that returns the ledger or the complete list of problems.
**Files (new):** `internal/mission/rulings_parse.go` (~160 lines), `internal/mission/rulings_parse_test.go` (~220 lines)
**Dependencies:** none
**Estimated:** 160 impl + 220 tests. One executor run (~90 min).

**Constants and types (exact):**

```go
const (
	rulingsLedgerRel        = "design_docs/fleet-rulings.md"
	rulingsMission          = "rulings" // Finding.Mission for fleet-wide findings (not "fleet": that is a mission name)
	rulingsMaxDetectors     = 64
	rulingsMinLiteral       = 8   // bytes
	rulingsMaxLiteral       = 200 // bytes
	rulingsMaxFileBytes     = 4 << 20
	rulingsMaxHitsPerDetector = 20
	ledgerStartMarker       = "<!-- decision-ledger:start -->"
	ledgerEndMarker         = "<!-- decision-ledger:end -->"
	detectorsStartMarker    = "<!-- fleet-rulings-detectors:start -->"
	detectorsEndMarker      = "<!-- fleet-rulings-detectors:end -->"
	rulingsLinkText         = "fleet-rulings.md"
)

type rulingRow struct { ID string; Num int; Status string; Line int }           // Status "OPEN"|"RESOLVED"
type forbidDetector struct { ID, Scope, Literal string; Line int }              // Scope "registry"|"skills"|"charters"
type supersession struct { By, Target string; Line int }
type rulingsLedger struct {
	Rows       map[string]rulingRow
	Forbids    []forbidDetector // ledger order
	Supersedes []supersession
}
type rulingProblem struct { Line int; Why string } // Line is 1-based; 0 = whole-file problem

func parseRulingsLedger(data []byte) (*rulingsLedger, []rulingProblem)
func (l *rulingsLedger) active() map[string]bool            // RESOLVED and not superseded by a RESOLVED ruling
func (l *rulingsLedger) activeForbids() []forbidDetector     // Forbids whose ID is active, ledger order
func formatRulingProblems(ps []rulingProblem) string         // sorted by Line (stable); "line N: why" / "file: why"; joined "; "
```

**Parsing rules (implement exactly; each rule has a test):**
1. Split on `"\n"`. Line numbers are 1-based. A marker line is any line that **contains** the marker string (the same match `mission_decisions.sh` uses).
2. Block structure: each of the 4 markers appears exactly once, each start comes before its end, and the two blocks do not overlap. Otherwise return **only** block problems (`Line` = the offending marker line, or 0 if a marker is absent): `expected exactly one <marker> (found N)`, `<start> must come before <end>`, `the decision-ledger and detector blocks overlap`.
3. Ledger block lines (between its markers): blank → skip. Lines not starting with `|` → `non-table line inside the decision-ledger block`. Protect `\|` (replace with `"\x01"`), then split on `|`. Header (trimmed field 1 == `ID`) → skip. Separator (every field trimmed ∈ `""` or matches `^:?-+:?$`) → skip. Otherwise: field count ≠ 6 → `row has N fields, want a 4-column row`. ID must match `^D-FR-([1-9][0-9]*)$` → else `bad ID "X" (want D-FR-<n>)`. Status ∉ {OPEN, RESOLVED} → `status "X" (want OPEN or RESOLVED)`. Empty ruling or evidence → `empty ruling or evidence cell`. Repeated ID → `duplicate ID "X"`. Zero rows → `{0, "decision ledger has no rows"}`.
4. Detector block lines: `strings.TrimSpace`. Blank or `#`-prefixed → skip (and **not counted**). Count the rest in order. Every detector line from the 65th on gets a problem `detector line N exceeds the 64-line cap` (N = ordinal) **in addition to** any grammar problem.
5. `forbid`: first `"` index `a`, last `"` index `b`. `a<0 || a==b` → `missing quoted literal`. Head = `strings.Fields(line[:a])` must be exactly `[ID, "forbid", scope]`, and `strings.TrimSpace(line[b+1:]) == ""`. Literal = `line[a+1:b]`. It contains `"` → `literal contains a double quote`. `len(literal)` (**bytes**) outside 8–200 → `literal is N bytes (want 8-200)`. Scope ∉ {registry, skills, charters} → `unknown scope "X" (want registry, skills or charters)`.
6. `supersedes`: `strings.Fields(line)` must be exactly `[ID, "supersedes", T]`. Target number must be **lower** than ID's → else `supersedes must name a LOWER-numbered ruling (ID supersedes T)` (covers self and forward).
7. Any other second token → `unknown verb "X" (want forbid or supersedes)`. Any ID or target that is well-formed but not a row → `unknown ID "X"`. Malformed IDs → `bad ID "X" (want D-FR-<n>)`.
8. Return `(nil, problems)` whenever `len(problems) > 0`. A partial ledger is never returned.

**Tests (`rulings_parse_test.go`) → the mutant each kills:**

| Test | Asserts | Kills |
|---|---|---|
| `TestRulingsParse_SeedShapeIsValid` | the design §1 seed text (2 rows, 3 detectors, verbatim) → 0 problems, `len(Rows)==2`, `len(Forbids)==3`, Forbids[2].Literal == `generator≠judge guard (HARD` | parser rejects a valid ledger; header/separator counted as rows |
| `TestRulingsParse_MalformedDetectorsListEveryLine` (T4a) | rows D-FR-1/D-FR-2 RESOLVED; detector lines: 1 valid; `D-FR-x forbid skills "abcdefgh"`; `D-FR-1 forbid skill "abcdefgh"`; `D-FR-1 forbid skills "abcdefg"`; `D-FR-1 supersedes D-FR-2`; `D-FR-9 forbid skills "abcdefgh"`; `D-FR-1 require skills "abcdefgh"` → ledger nil; problem line set == exactly the 6 bad lines | stop at first error; silent skip of a bad line |
| `TestRulingsParse_MalformedRowFailsWholeLedger` (T4b-row) | (a) row `\| D-FR-3 \| RESOLVED \|` (3 cols) → problem at that line, ledger nil; (b) status `PARKED` → problem; (c) non-table text inside the block → problem | malformed row read as silently non-RESOLVED |
| `TestRulingsParse_DetectorCap` (T4b) | 64 valid detector lines with 3 `#` and 2 blank lines interleaved → 0 problems; 65 → exactly one problem, at the **file line** of the 65th detector line, `Why` contains `64-line cap`, and the formatted detail contains `line <thatLine>:` | cap not enforced; `>= 64` off-by-one; comments/blank lines counted |
| `TestRulingsParse_LiteralBytes` | `"≠abcde"` (8 bytes, 6 runes) accepted; `"abcdefg"` (7) rejected; 200 accepted; 201 rejected; `"ab"cdefghij"` rejected (quote) | rune-count instead of byte-count; bounds off-by-one |
| `TestRulingsParse_BlockStructure` | missing detector block → problem `expected exactly one <!-- fleet-rulings-detectors:start -->`; duplicated ledger start → problem; end before start → problem; ledger nil in each | block check missing |
| `TestRulingsActive_Supersession` (T2 parse) | D-FR-1 R, D-FR-2 R + `D-FR-2 supersedes D-FR-1` → active = {D-FR-2}; same with D-FR-2 OPEN → {D-FR-1}; D-FR-1 OPEN alone → {} | supersession ignored; inverted; OPEN treated active; OPEN superseder still suppresses |
| `TestRulingsParse_SupersedesOrderAndDuplicates` | `D-FR-1 supersedes D-FR-1` (self) and `D-FR-1 supersedes D-FR-2` (forward) → problems; duplicate row ID → problem | cycle possible |

**Acceptance (run all, record outputs):**
- [ ] `go test ./internal/mission/ -run 'TestRulings' -count=1 -v` → all 8 tests PASS
- [ ] `gofmt -l internal/mission` → no output; `go vet ./internal/mission/` rc 0; `golangci-lint run ./internal/mission/...` rc 0
- [ ] `wc -l internal/mission/rulings_parse.go` ≤ 300
- [ ] Mutations (each FAILs its test, then is restored): (1) in rule 4 change `> rulingsMaxDetectors` to `>= rulingsMaxDetectors` → `TestRulingsParse_DetectorCap` FAIL; (2) replace `len(lit)` with `utf8.RuneCountInString(lit)` → `TestRulingsParse_LiteralBytes` FAIL; (3) `return` after the first detector problem → `…MalformedDetectorsListEveryLine` FAIL; (4) make `active()` ignore `Supersedes` → `TestRulingsActive_Supersession` FAIL; (5) accept any status → `…MalformedRowFailsWholeLedger` FAIL
- [ ] Commit: `feat(mission): fleet rulings ledger parser (M-FLEET-RULINGS-LEDGER M1)`

### M2: Corpus — root derivation, charter location, skills walk, line matcher (~280 LOC)

**Goal:** the helpers that decide *which files* are checked and *which lines* match.
**Files (new):** `internal/mission/rulings_corpus.go` (~110 lines), `internal/mission/rulings_corpus_test.go` (~170 lines)
**Dependencies:** M1 (constants)
**Estimated:** 110 impl + 170 tests. One executor run (~60 min).

**Functions (exact signatures):**

```go
// rulingsRoot derives the checkout whose ledger is checked (design §2). Exactly one of the results is set.
func rulingsRoot(missions []*Mission) (root string, skipped *Finding)
// charterPath: absolute m.Doc as-is; m.Repo == sharedDriverRepo → filepath.Join(root, m.Doc); else filepath.Join(m.Workdir, m.Doc).
func charterPath(root string, m *Mission) string
// rulingSkillFiles: for prefix in {".claude/skills", ".agents/skills"}: os.ReadDir(root/prefix) (IsNotExist → skip that prefix;
// other error → return it); for each DIR entry whose name has prefix "mission-": filepath.WalkDir collecting regular files
// ending ".md"; finally sort.Strings. Never filepath.Glob ("**" is unsupported).
func rulingSkillFiles(root string) ([]string, error)
// outsideLedger[i] reports whether lines[i] is OUTSIDE every decision-ledger block; marker lines count as inside.
func outsideLedger(lines []string) []bool
// literalHits returns 1-based numbers of lines containing lit (byte-exact, case-sensitive; one hit per line);
// when charter is true, lines inside decision-ledger blocks are skipped.
func literalHits(lines []string, lit string, charter bool) []int
// displayPath: absolute p under root → filepath.Rel(root, p); otherwise p unchanged.
func displayPath(root, p string) string
```

`rulingsRoot` skipped findings, all `Finding{Mission: rulingsMission, Kind: "rulings-skipped", Severity: Note}`, with Detail exactly:
`registry has no missions` / `no mission carries a registry root` / `missions disagree on registry root: <a>, <b>` (distinct roots, sorted).

**Tests (`rulings_corpus_test.go`; first add the `rulingsFixture` helper specified under M3, since `TestRulingsRoot_Derivation` needs `LoadFile`d missions):**

| Test | Asserts | Kills |
|---|---|---|
| `TestRulingsRoot_Derivation` (T10 unit) | `nil` → skipped "registry has no missions"; `[]*Mission{{Name:"a"}}` (hand-built) → "no mission carries a registry root"; two `LoadFile`d missions from two `t.TempDir()` registries → skipped whose Detail contains **both** roots; one `LoadFile`d mission → root == its `Root()`, skipped nil | silent return; first-root-wins |
| `TestRulingsCharterPath_Location` (T11 unit) | in-repo (`Repo: sharedDriverRepo`) → under root; foreign repo → under Workdir; absolute `Doc` → unchanged | workdir used for in-repo; root used for foreign |
| `TestRulingSkillFiles_WalkDirDepthAndFilter` | tree: `.claude/skills/mission-a/SKILL.md`, `.claude/skills/mission-a/resources/g.md`, `.claude/skills/mission-a/resources/deep/x.md`, `.claude/skills/mission-a/notes.txt`, `.claude/skills/other/SKILL.md`, `.agents/skills/mission-a/resources/g.md` → exactly the 4 `.md` mission files, sorted; a root without `.agents/` → no error | one-level `filepath.Glob`; non-mission dir included; non-md included |
| `TestRulingsLiteralHits_LedgerExclusion` (T3 unit) | literal on line 2 (outside) and line 5 (inside a ledger block) → charter=true gives `[2]`, charter=false gives `[2 5]` | exclusion missing; exclusion over-broad (applied to non-charters) |
| `TestRulingsLiteralHits_ByteExactCaseSensitive` | `WORLD IS A FORK` does not match `World is a fork`; `generator≠judge guard (HARD` matches its exact bytes; a line holding the literal twice → one hit | case-folding matcher; per-occurrence counting |

**Acceptance:**
- [ ] `go test ./internal/mission/ -run 'TestRuling' -count=1 -v` → M1 + M2 tests PASS
- [ ] `gofmt -l internal/mission` empty; `go vet ./internal/mission/` rc 0; `golangci-lint run ./internal/mission/...` rc 0
- [ ] Mutations: (1) replace the WalkDir with `filepath.Glob(filepath.Join(root, prefix, "mission-*", "*.md"))` → `…WalkDirDepthAndFilter` FAIL; (2) in `rulingsRoot` return the first non-empty root without the disagreement check → `TestRulingsRoot_Derivation` FAIL; (3) in `charterPath` always use `m.Workdir` → `…CharterPath_Location` FAIL; (4) `literalHits` ignore `charter` → `…LedgerExclusion` FAIL
- [ ] Commit: `feat(mission): rulings corpus — root, charter location, skills walk (M-FLEET-RULINGS-LEDGER M2)`

### M3: `CheckRulings` — the report (~430 LOC)

**Goal:** the exported check that turns ledger + corpus into sorted `Note` findings.
**Files (new):** `internal/mission/rulings.go` (~130 lines), `internal/mission/rulings_test.go` (~300 lines)
**Dependencies:** M1, M2
**Estimated:** 130 impl + 300 tests. One executor run (~90 min).

```go
// CheckRulings reports — as Notes only, never Drift — charter, skill and registry text that contains a literal
// retired by an active (RESOLVED, unsuperseded) fleet-wide ruling. It derives the root itself (design §2).
func CheckRulings(missions []*Mission) []Finding
```

**Algorithm (exact order):**
1. `root, skipped := rulingsRoot(missions)`. If skipped → `return []Finding{*skipped}`.
2. Ledger `filepath.Join(root, rulingsLedgerRel)`: `os.Stat` IsNotExist → return one `rulings-ledger-absent` (Mission `rulings`, Detail `<path> does not exist; no fleet-wide ruling is checked`). Any other stat/read error or size > `rulingsMaxFileBytes` → return one `ruling-file-unreadable` (Mission `rulings`).
3. `parseRulingsLedger`. If there are problems → return one `rulings-malformed` (Mission `rulings`, Detail `design_docs/fleet-rulings.md: N problem(s) — ` + `formatRulingProblems`). **No detector runs.**
4. A per-call read cache: `map[path]*struct{lines []string; err error}`. Size check by `os.Stat` before `os.ReadFile`. A missing file → err `does not exist`. A file > 4 MiB → err `is N bytes (> 4 MiB)`. The first failing read of a path appends one `ruling-file-unreadable` (owner = mission or `rulings`, Detail `<displayPath>: <err>`). Later reads of that path add nothing.
5. **Link check**, for each mission in `missions`: read `charterPath(root, m)`; on success, if no line with `outsideLedger==true` contains `rulingsLinkText` → `ruling-link-missing` (Mission `m.Name`, Detail `<displayPath> has no link to design_docs/fleet-rulings.md outside its decision-ledger block`).
6. **Detectors**, for each `activeForbids()` in ledger order, over its scope: `registry` → each `m.Path` (owner `m.Name`, charter=false); `charters` → each `charterPath` (owner `m.Name`, charter=true); `skills` → `rulingSkillFiles(root)` (owner `rulings`, charter=false; a walk error → one `ruling-file-unreadable` for the skills dir). The first 20 hits per detector → `ruling-contradiction`, Detail `D-FR-n <displayPath>:<line> contains retired text "<lit>"` (use `%q`). If more than 20 → one more `ruling-contradiction`, Detail `D-FR-n … N more line(s) contain "<lit>" (only the first 20 are listed)`.
7. Every finding is `Severity: Note`. `sort.SliceStable` by `Mission`, then `Kind`, then return.

**Test helper** (M2 creates it at the top of `rulings_corpus_test.go`, because M2's root tests already need `LoadFile`d missions; M3 and M4 reuse it):

```go
type rulingsFixture struct{ t *testing.T; root string; n int }
func newRulingsFixture(t *testing.T) *rulingsFixture { t.Helper(); return &rulingsFixture{t: t, root: t.TempDir()} }
func (f *rulingsFixture) write(rel, body string) string // MkdirAll + WriteFile under root; returns the absolute path
// mission writes missions/<name>.toml (name, repo, workdir, doc, extra lines, [schedule] mode="keepalive"
// throttle_seconds=60 boot_offset=f.n*60 with f.n incremented per call, so Load() of the dir sees no collision) and LoadFile()s it.
func (f *rulingsFixture) mission(name, repo, workdir, doc, extra string) *Mission
// ledger writes design_docs/fleet-rulings.md: title, ledger block with header+separator+rows, then the detector block.
func (f *rulingsFixture) ledger(rows, detectors string)
const linkedCharter = "# C\n\n> **Fleet-wide rulings:** [fleet-rulings.md](fleet-rulings.md).\n\n<!-- decision-ledger:start -->\n| ID | Status | Ruling | Evidence |\n|---|---|---|---|\n<!-- decision-ledger:end -->\n"
```

Workdirs in fixtures: `filepath.Join(f.root, "wd", name)`, which is absolute, so `Validate` accepts it.

**Tests (`rulings_test.go`):**

| Test | Asserts | Kills |
|---|---|---|
| `TestRulings_SkillHitNamesRulingFileAndLine` (T1) | D-FR-1 RESOLVED + `D-FR-1 forbid skills "retired-phrase-x"`; `.claude/skills/mission-a/resources/g.md` line 3 holds it; one in-repo mission with a linked charter → exactly one finding: Kind `ruling-contradiction`, Mission `rulings`, Detail contains `D-FR-1` and `.claude/skills/mission-a/resources/g.md:3` | matcher always false; 0-based line numbers |
| `TestRulings_OpenAndSupersededAreQuiet` (T2) | same literal: D-FR-1 OPEN → 0 contradictions; D-FR-1 R superseded by D-FR-2 R → 0 for D-FR-1's detector; D-FR-2 OPEN → the hit returns | supersession ignored / inverted |
| `TestRulings_CharterLedgerBlockExcluded` (T3) | charters-scope literal inside the charter's ledger block → 0; the same line moved outside → 1 (Mission = owning mission) | exclusion missing / over-broad |
| `TestRulings_MalformedLedgerRunsNothing` (T4) | ledger with one valid detector that **would** hit plus one 7-byte literal → exactly 1 finding, `rulings-malformed`, detail names the bad line; 0 contradictions, 0 link findings | partial evaluation |
| `TestRulings_DetectorCapEndToEnd` (T4b) | 65 valid detector lines, one of which would hit → exactly 1 `rulings-malformed`, detail contains the 65th line's file line number; 64 lines → 0 `rulings-malformed`, ≥1 contradiction | cap not wired into the check |
| `TestRulings_AbsentLedgerIsOneNote` (T5) | no ledger file → exactly 1 finding `rulings-ledger-absent`, `(&Report{Findings: got}).ExitCode() == 0` | absent ledger as Drift (judge B4) |
| `TestRulings_EveryFindingIsNote` (T6) | a fixture producing contradiction + link-missing + unreadable (missing foreign charter) → every `Severity == Note`; `ExitCode()==0` | any kind regressed to Drift |
| `TestRulings_HitCapTwentyThenSummary` | a skill file with 23 matching lines → 21 contradictions; the 21st Detail contains `3 more` | cap removed; off-by-one |
| `TestRulings_OutputIsSorted` | two missions `b`, `a` passed in that order, each producing a link-missing → findings ordered `a` before `b`; two calls return `reflect.DeepEqual` results | unsorted / nondeterministic output |

**Acceptance:**
- [ ] `go test ./internal/mission/ -run 'TestRuling' -count=1 -v` → M1–M3 tests PASS
- [ ] `go test ./internal/mission/ -count=1` → `ok` (whole package, not wired yet)
- [ ] `gofmt -l internal/mission` empty; `go vet` rc 0; `golangci-lint run ./internal/mission/...` rc 0; `wc -l internal/mission/rulings.go` ≤ 300
- [ ] Mutations: (1) `Severity: Drift` on `ruling-link-missing` → `TestRulings_EveryFindingIsNote` FAIL; (2) on malformed, fall through and run the detectors → `TestRulings_MalformedLedgerRunsNothing` FAIL; (3) remove the 20-cap → `TestRulings_HitCapTwentyThenSummary` FAIL; (4) iterate `l.Forbids` instead of `activeForbids()` → `TestRulings_OpenAndSupersededAreQuiet` FAIL; (5) `charter=false` for the charters scope → `TestRulings_CharterLedgerBlockExcluded` FAIL
- [ ] Commit: `feat(mission): CheckRulings — report-only fleet ruling notes (M-FLEET-RULINGS-LEDGER M3)`

### M4: Wire into `DoctorWith` (prepend) + scope tests (~200 LOC)

**Goal:** every doctor caller gets ruling notes **ahead of** per-mission findings (P8b), with scope behaviour proven end to end.
**Files:** `internal/mission/doctor.go` (+1 line), `internal/mission/rulings_test.go` (+~190 lines), `design_docs/planned/m-fleet-rulings-ledger.md` (§3 call-site sentence)
**Dependencies:** M3
**Estimated:** 1 impl + ~190 tests + doc sentence. One executor run (~75 min).

**The one line**, inserted between `doctor.go:294` (`}` closing the per-mission loop) and `:295` (`sort.Slice(rep.Rows…)`):

```go
	rep.Findings = append(CheckRulings(reg.Missions), rep.Findings...) // prepended: lane-check shows the LAST line (P8b)
```

**Design doc correction (same commit):** in §3 "Call site, exactly", replace the quoted
`rep.Findings = append(rep.Findings, CheckRulings(reg.Missions)...)` with the prepend form, and add one sentence:
"Prepended, not appended: `mission-lane-check.sh:49-50` puts the last output line into the `config dead` cell, and
an appended note would replace the drift reason there (sprint plan P8b)." Make the same edit to the Implementation-plan row 2.

**Tests (append to `rulings_test.go`):**

| Test | Asserts | Kills |
|---|---|---|
| `TestRulings_LinkMissing` (T8) | charter with no link → one `ruling-link-missing` for that mission; link only **inside** the ledger block → still one; link outside → none | link check vacuous; ledger-block mention counted |
| `TestRulings_PerMissionScope` (T9) | missions `a`, `b` (in-repo), each with the charters-scope literal outside its ledger block and the registry-scope literal in a TOML comment (`extra`) → `CheckRulings([]*Mission{a})` reports only `a`'s charter and `missions/a.toml` hits; skills hits are still reported; `CheckRulings({a,b})` reports both | scope leak |
| `TestRulings_RootDerivationEndToEnd` (T10) | `CheckRulings(nil)` → exactly 1 `rulings-skipped`; hand-built missions only → 1 `rulings-skipped`; missions from two fixtures → exactly 1 finding total, `rulings-skipped`, naming both roots; a one-mission slice from a loaded mission → the check runs (a hit is returned) | silent return; first-root-wins |
| `TestRulings_CharterLocationEndToEnd` (T11) | in-repo mission, **different** text at `root/doc` (has literal) and `workdir/doc` (clean) → 1 hit, path under root; swapped → 0; foreign mission, workdir copy has literal, root copy clean → 1 hit; foreign mission with no charter file → `ruling-file-unreadable`, Mission = that mission | workdir used for in-repo; missing charter skipped |
| `TestDoctor_RulingFindingsPrecedeMissionFindings` | reuse `newFleet` + the env-drift setup from `TestDoctor_GATE_V4_V5_EnvDriftIsFoundAndNamed` (`doctor_test.go:119`) → `rep.Findings[0].Kind == "rulings-skipped"` (hand-built mission, no root) and `rep.Findings[len-1].Severity == Drift`; `rep.ExitCode()==1` | append instead of prepend (P8b) |
| `TestDoctor_RulingCheckRunsForLoadedMission` | `newFleet(t)`, then append a `LoadFile`d fixture mission (no ledger in its root) to `f.reg.Missions`; `DoctorWith(f.reg, f.p, nil)` → exactly 1 `rulings-ledger-absent`; `findingsOfKind(rep, "rulings-skipped")` empty | wiring line missing |

**Acceptance:**
- [ ] `go test ./internal/mission/ -count=1` → `ok` (whole package, incl. every existing `TestDoctor_*` **unchanged**: `git diff --stat internal/mission/doctor_test.go` empty)
- [ ] `go test ./internal/mission/ -run 'TestRuling|TestDoctor_Ruling' -count=1 -v 2>&1 | grep -c -- '--- PASS'` = (the same count recorded at the end of M3) + 6
- [ ] `git diff HEAD~1 -- internal/mission/doctor.go | grep -c '^+[^+]'` = 1
- [ ] Mutations: (1) change the line to `append(rep.Findings, CheckRulings(reg.Missions)...)` → `TestDoctor_RulingFindingsPrecedeMissionFindings` FAIL; (2) delete the line → `TestDoctor_RulingCheckRunsForLoadedMission` FAIL; (3) scope leak: in the `charters` branch of `CheckRulings` replace `for _, m := range missions` with `all, _ := Load(filepath.Join(root, "missions")); for _, m := range all.Missions` → `TestRulings_PerMissionScope` FAIL (the fixture's distinct `boot_offset`s let `Load` succeed); (4) in `charterPath` swap the branches → `TestRulings_CharterLocationEndToEnd` FAIL
- [ ] Commit: `feat(mission): doctor reports fleet ruling notes, prepended (M-FLEET-RULINGS-LEDGER M4)`

### M5: CLI — summary whenever `!HasDrift()`, help text (~80 LOC)

**Goal:** `ailang mission doctor` prints notes and then `N mission(s): no drift` whenever nothing drifted. Zero-finding output stays byte-identical.
**Files:** `cmd/ailang/mission_cmd.go` (~+15/−5), new `cmd/ailang/mission_doctor_print_test.go` (~60 lines)
**Dependencies:** M4
**Estimated:** 20 impl + 60 tests. One executor run (~45 min; `cmd/ailang` tests compile in ~40 s).

1. Extract the print loop at `mission_cmd.go:252-257` into:

```go
// printDoctorReport writes every finding, then the summary line whenever nothing DRIFTED (notes alone keep it).
// Ruling notes come first (DoctorWith prepends them), so the LAST line is the summary on exit 0 and a
// per-mission finding on exit 1 — the line mission-lane-check.sh:49-50 puts in its config row.
func printDoctorReport(w io.Writer, rep *mission.Report) {
	for _, f := range rep.Findings {
		fmt.Fprintln(w, f)
	}
	if !rep.HasDrift() {
		fmt.Fprintf(w, "%d mission(s): no drift\n", len(rep.Rows))
	}
}
```

   `missionDoctor` calls `printDoctorReport(os.Stdout, rep)` and then `os.Exit(rep.ExitCode())`, unchanged.
2. Help text: under `mission_cmd.go:126` (`exit 0 clean, 1 drift, 2 registry error`) add one aligned line:
   `                                   also notes text contradicting design_docs/fleet-rulings.md (never fails)`.

**Tests (`mission_doctor_print_test.go`, package `main`):**

| Test | Asserts | Kills |
|---|---|---|
| `TestMissionDoctorPrint_ZeroFindingsByteIdentical` | `Report{Rows: 2 rows}` → output **exactly** `"2 mission(s): no drift\n"` | any change to the zero-finding output |
| `TestMissionDoctorPrint_NotesThenSummary` | one `Note` → 2 lines; the last is `1 mission(s): no drift` | the `len(rep.Findings)==0` guard kept; summary printed before findings |
| `TestMissionDoctorPrint_DriftHasNoSummary` | `[Note, Drift]` → the last line contains `[DRIFT]`; no line contains `no drift` | guard removed entirely |

**Acceptance:**
- [ ] `go test ./cmd/ailang/ -run 'TestMissionDoctorPrint' -count=1 -v` → 3 PASS
- [ ] `go test ./cmd/ailang/ -run 'TestCLIReference' -count=1` → ok (`make check-cli-docs` equivalent; help-line change only)
- [ ] `gofmt -l cmd/ailang` empty; `go vet ./cmd/ailang/` rc 0
- [ ] `bash tools/launchd/test_mission_lane_check.sh; echo rc=$?` → rc 0 (it stubs `ailang` and asserts `config ok 1 mission(s): no drift`, `:80`)
- [ ] Mutations: (1) restore `if len(rep.Findings) == 0` → `…NotesThenSummary` FAIL; (2) drop the condition → `…DriftHasNoSummary` FAIL
- [ ] Commit: `feat(mission): doctor prints its summary whenever nothing drifted (M-FLEET-RULINGS-LEDGER M5)`

### M6: The ledger, charter links, Gate 0, live-ledger tests, changelog (~200 LOC, mostly text)

**Goal:** ship the content. Prove the real ledger parses and the seed detectors are quiet on activation.
**Files:** new `design_docs/fleet-rulings.md`; `design_docs/{v1,fleet,docs,motoko,triage}-mission.md` (+2 lines each);
`design_docs/mission-charter-TEMPLATE.md` (+2 lines, planner addition: new charters inherit the link);
`.claude/skills/mission-control/resources/gate-0-preflight.md` (+2 lines) and its `.agents` mirror (regenerated);
new `internal/mission/rulings_live_test.go` (~70 lines); new `internal/mission/testdata/rulings/{gate-3-route.pre-349e311b9.md, gate-3-route.current.md, world.pre-2396.toml, world.post-2396.toml}`;
new `changelogs/unreleased/<YYYY-MM-DD>-fleet-rulings-ledger.md`
**Dependencies:** M1 (the parser, used by the live test). It can run after M5 in sequence.
**Estimated:** ~60 doc lines + ~70 test + ~25 testdata + ~15 changelog. One executor run (~60 min).

1. **`design_docs/fleet-rulings.md`** with exactly this content:

````markdown
# Fleet-wide rulings

Rulings in this file bind **every** mission. Each charter links here, and nothing in a charter or a mission
skill overrides a RESOLVED, unsuperseded row. Design: `m-fleet-rulings-ledger` (D-FLEET-25 A, Mark attended 2026-11-16).

- **Rows** use the decision-ledger table: `scripts/mission_decisions.sh --check --file design_docs/fleet-rulings.md`.
  A new ruling arrives OPEN and is resolved only by Mark, attended:
  `scripts/mission_answer.sh --id D-FR-<n> --file design_docs/fleet-rulings.md`. A resolved row is never re-answered.
  To revise one, add a new ID plus a `supersedes` line, in one reviewed PR. A transcription row cites its source row.
- **Detectors** (the block after the table) are what `ailang mission doctor` checks. One per line:
  `<ID> forbid <registry|skills|charters> "<literal, 8-200 bytes>"` or `<ID> supersedes <lower ID>`; at most 64.
  A hit is a `note`, never drift: the doctor reports and does not block.
- Do not quote a forbidden literal outside a decision-ledger block in any charter, registry file or mission skill.
  Paraphrase it instead.

<!-- decision-ledger:start -->
| ID | Status | Ruling | Evidence |
|---|---|---|---|
| D-FR-1 | RESOLVED | World runs the shared driver; it is not a fork. Text calling it a fork is stale. | Transcribed from D-FLEET-25 (fleet-mission.md, Mark attended 2026-11-16). |
| D-FR-2 | RESOLVED | generator≠judge is a PREFERENCE at vendor level, never a blocker (model-level self-review stays refused). | Transcribed from V1 D-62/D-63 (v1-mission.md, Mark attended 2026-09-08); skill fixed in 349e311b9. |
<!-- decision-ledger:end -->

<!-- fleet-rulings-detectors:start -->
D-FR-1 forbid registry "WORLD IS A FORK"
D-FR-1 forbid charters "WORLD IS A FORK"
D-FR-2 forbid skills "generator≠judge guard (HARD"
<!-- fleet-rulings-detectors:end -->
````

2. **Charter link.** In each of the five in-repo charters and the template, insert directly after line 1 (the `# …` title)
   one blank line and this line (the design §6 text, verbatim):
   `> **Fleet-wide rulings:** [fleet-rulings.md](fleet-rulings.md). Every RESOLVED, unsuperseded ruling there binds this mission, and nothing in this charter overrides one. If this charter disagrees with it, the ledger wins and the disagreement is a bug to report.`
   so the file reads: title, blank, link line, blank, then the original line 2 onward.
3. **Gate 0** (`.claude/skills/mission-control/resources/gate-0-preflight.md`): after line 4 (`Attended setup: …`), insert a blank line and:
   `**Fleet-wide rulings:** \`design_docs/fleet-rulings.md\` binds every mission. Every RESOLVED, unsuperseded ruling there binds this mission, and nothing in its charter or in this skill overrides one. If either disagrees with it, the ledger wins and the disagreement is a bug to report (\`ailang mission doctor\` lists known contradictions as notes).`
   Then regenerate the mirror: `bash tools/launchd/sync-agents-skills.sh` followed by `bash tools/launchd/sync-agents-skills.sh --check`.
4. **Testdata snapshots** (exact commands):
   `mkdir -p internal/mission/testdata/rulings`;
   `git show 349e311b9^:.claude/skills/mission-control/resources/gate-3-route.md | sed -n '240,245p' > internal/mission/testdata/rulings/gate-3-route.pre-349e311b9.md`;
   `sed -n '255,260p' .claude/skills/mission-control/resources/gate-3-route.md > internal/mission/testdata/rulings/gate-3-route.current.md`;
   `git show 950a0cc54:missions/world.toml > internal/mission/testdata/rulings/world.pre-2396.toml` (judge F1, i118: #2396 merged as `ad1fdac57`, so the working-tree `world.toml` no longer carries the literal; a `cp` would yield a 0-hit "pre" snapshot);
   `git show ad1fdac57:missions/world.toml > internal/mission/testdata/rulings/world.post-2396.toml` (the merged post-#2396 file; byte-identical content to `c34c409c3`'s). Assert both: `grep -cF 'WORLD IS A FORK'` = 1 on pre, 0 on post.
   If `c34c409c3` is missing: `git fetch origin pull/2396/head:refs/i118/pr2396` (a read), then use `refs/i118/pr2396`.
5. **`rulings_live_test.go`:**

| Test | Asserts | Kills |
|---|---|---|
| `TestRulingsLiveLedgerParses` (T7a) | `os.ReadFile("../../design_docs/fleet-rulings.md")` → `parseRulingsLedger` returns **0 problems**, ≥ 2 rows, ≥ 3 forbids, and every forbid ID is a row (guaranteed by the parser; asserted anyway). **No live-corpus hit count** is asserted (design T7a) | a malformed ledger merged |
| `TestRulingsSeedQuietOnActivation` (T7b) | take the seed detectors from the **live** ledger; `literalHits` of D-FR-2's skills literal: `gate-3-route.pre-349e311b9.md` → 1, `gate-3-route.current.md` → 0 **and** that file contains `generator≠judge` (control: the bytes are there); D-FR-1's registry literal: `world.pre-2396.toml` → 1, `world.post-2396.toml` → 0 | a seed detector that self-fails or never fires |

6. **Changelog fragment** `changelogs/unreleased/$(date +%Y-%m-%d)-fleet-rulings-ledger.md`:

```markdown
### Added — fleet-wide rulings ledger and `ailang mission doctor` ruling notes (D-FLEET-25 A)

`design_docs/fleet-rulings.md` holds rulings that bind every mission. The five in-repo charters and Gate 0 of the
mission-control skill link to it. `ailang mission doctor` now prints `note` findings for registry, charter or
mission-skill text that contains a literal retired by an active ruling (`ruling-contradiction`), plus
`ruling-link-missing`, `rulings-malformed`, `rulings-ledger-absent`, `ruling-file-unreadable` and `rulings-skipped`.
Notes never change the exit code. The doctor now prints `N mission(s): no drift` whenever nothing drifted, after any notes.
Design: `design_docs/planned/m-fleet-rulings-ledger.md`.
```

**Acceptance:**
- [ ] `bash scripts/mission_decisions.sh --check --file design_docs/fleet-rulings.md` → `decision ledger valid: 2 rows`, rc 0
- [ ] `for f in design_docs/{v1,fleet,docs,motoko,triage}-mission.md; do bash scripts/mission_decisions.sh --check --file "$f" || echo FAIL "$f"; done` → no `FAIL` (charters still valid)
- [ ] `/bin/bash -c 'for f in design_docs/{v1,fleet,docs,motoko,triage}-mission.md; do awk "/<!-- decision-ledger:start -->/{i=1;next}/<!-- decision-ledger:end -->/{i=0;next} !i && index(\$0,\"fleet-rulings.md\"){n++} END{print FILENAME\": \"n+0}" "$f"; done'` → each `1`
- [ ] `bash tools/launchd/sync-agents-skills.sh --check; echo rc=$?` → `agents skills in sync (…)`, rc 0; `cmp .claude/skills/mission-control/resources/gate-0-preflight.md .agents/skills/mission-control/resources/gate-0-preflight.md` rc 0; `git status --short .agents/` lists only the gate-0 mirror
- [ ] Seed literals were not introduced: `/bin/bash -c 'SK=$(find .claude/skills/mission-* .agents/skills/mission-* -name "*.md"); cat $SK | grep -cF "generator≠judge guard (HARD"'` → 0; control: the same command with `generator≠judge — a PREFERENCE` → 2
- [ ] `go test ./internal/mission/ -run 'TestRulingsLive|TestRulingsSeed' -count=1 -v` → 2 PASS
- [ ] `make check-changelog; echo rc=$?` → rc 0
- [ ] Mutations: (1) scratchpad-copy `design_docs/fleet-rulings.md`, change `forbid skills` to `forbid skill` → `TestRulingsLiveLedgerParses` FAIL, restore from the copy; (2) temporarily replace the T7b pre-fix snapshot with the current one (`cp` the current snapshot over it) → `TestRulingsSeedQuietOnActivation` FAIL, then restore from the scratchpad copy
- [ ] Commit: `docs(fleet): fleet-wide rulings ledger + charter/Gate-0 links (M-FLEET-RULINGS-LEDGER M6)`

### M7: Fleet done-gate, live-run gate, design-doc record and move (~60 LOC of doc)

**Goal:** prove the whole change with the fleet's done-gate and one recorded live run. Then file the design as implemented, or correct it.
**Files:** `design_docs/planned/m-fleet-rulings-ledger.md` (Status, V34 row, live output), then `git mv` of it and this plan
**Dependencies:** M1–M6
**Estimated:** ~60 doc lines. One executor run (~60 min, mostly waiting on gates).

1. **Fleet done-gate (all three must pass; run unpiped, log to `/tmp/i118-gate/`):**
   - `go test ./internal/mission/ ./cmd/ailang/ -run 'Mission|Doctor|Ruling' -count=1 > /tmp/i118-gate/go.log 2>&1; echo rc=$?` → rc 0
   - `make test-launchd-drivers > /tmp/i118-gate/ltd.log 2>&1; echo rc=$?` → rc 0, `passed=36 failed=0` or more passed (baseline P17)
   - **Dry-run, healthy and degraded**, at two layers:
     a. *Doctor layer (exercises this change).* `mkdir -p /tmp/i118-build && go build -o /tmp/i118-build/ailang ./cmd/ailang`.
        Healthy: from the worktree root, `ailang mission doctor world > /tmp/i118-gate/doc-base.txt 2>&1; echo rc=$?` (installed binary = baseline), then
        `/tmp/i118-build/ailang mission doctor world > /tmp/i118-gate/doc-new.txt 2>&1; echo rc=$?`. Expect: the same rc, the same per-mission lines, plus ruling notes
        (#2396 merged as `ad1fdac57` 2026-11-17, so 0 `ruling-contradiction`; `ruling-link-missing` for world), and a last line equal to the baseline's last line when rc ≠ 0.
        Degraded: `D=$(mktemp -d) && cp -R missions "$D/missions" && mkdir -p "$D/design_docs" && sed 's/forbid skills/forbid skill/' design_docs/fleet-rulings.md > "$D/design_docs/fleet-rulings.md" && AILANG_MISSION_REGISTRY="$D/missions" /tmp/i118-build/ailang mission doctor world; echo rc=$?`
        → exactly one `rulings-malformed` note naming the bad line, **zero** `ruling-contradiction`, and the same rc as the healthy run.
     b. *Driver layer (the charter's mission-loop-change pre-flight, `design_docs/fleet-mission.md:105-124`).* The driver is untouched, so this only proves no regression.
        Pick an idle, armed sibling (no `~/.ailang/state/mission-<m>.disabled`, no live pidfile). Run
        `env -i HOME=$HOME PATH=$PATH USER=$USER TMPDIR=$TMPDIR AILANG_WORKTREE_SWEEP=0 AILANG_DRIVER_PINNED=$(git rev-parse HEAD) MISSION_PROFILE=<m> MISSION_DRY_RUN=1 /bin/bash tools/launchd/mission-control.sh` → `DRY RUN ok … lanes=ok` (healthy),
        then the same with `MISSION_EXECUTOR_MODEL=codex:bogus MISSION_PROBE_TIMEOUT=10` → `lanes=DEGRADED(` (degraded). If no sibling is idle and armed, use the synthetic-HOME recipe at `fleet-mission.md:116-124`. Record which profile ran.
2. **Focused gates:** `make check-file-sizes; echo rc=$?`, `make check-changelog; echo rc=$?`, `make test-changed; echo rc=$?` (touched packages + `test-core`; **never** `make test`), all rc 0.
3. **Live-run gate (design Success criteria).** Immediately before, record the baseline: `ailang mission doctor > /tmp/i118-gate/live-base.txt 2>&1; echo rc=$?`.
   Then `/tmp/i118-build/ailang mission doctor > /tmp/i118-gate/live-new.txt 2>&1; echo rc=$?`.
   Count ruling kinds: `grep -oE '(ruling-contradiction|ruling-link-missing|rulings-malformed|rulings-ledger-absent|ruling-file-unreadable|rulings-skipped)' /tmp/i118-gate/live-new.txt | sort | uniq -c`.
   **Predicted** (design §Success): rc equal to the baseline rc; 0 `ruling-contradiction` (#2396 merged as `ad1fdac57`, judge F2); `2 ruling-link-missing` (world, stapledon); no other ruling kind.
   Add **V34** to the design doc's Verification Log: binary (`/tmp/i118-build/ailang`, `git rev-parse --short HEAD`), both rcs, the kind counts, and the full `live-new.txt` in a fenced block.
   **If the counts differ from the prediction**, correct the design doc's Success-criteria line to what was measured, citing V34. A predicted count never ships as verified.
4. **Move:** set the design doc Status to `Implemented — <date>, sprint M-FLEET-RULINGS-LEDGER (V34 live run)`. Compute the version directory:
   `v=$(git describe --tags --abbrev=0); d="design_docs/implemented/$(echo "$v" | awk -F. '{printf "v%s_%s_%d", substr($1,2), $2, $3+1}')"` (today `v0.60.5` → `design_docs/implemented/v0_60_6`).
   `mkdir -p "$d" && git mv design_docs/planned/m-fleet-rulings-ledger.md "$d/" && git mv design_docs/planned/m-fleet-rulings-ledger-sprint-plan.md "$d/"`.
   Fix references to the old path: `grep -rn 'planned/m-fleet-rulings-ledger' design_docs changelogs .ailang/state/sprints` → update each hit (changelog fragment, sprint JSON `design_doc`/`sprint_plan`).
5. Commit: `docs(fleet): M-FLEET-RULINGS-LEDGER done-gate + live run V34; move design to implemented`.

**Acceptance:**
- [ ] go done-gate rc 0; `make test-launchd-drivers` rc 0; the doctor-layer healthy and degraded runs match the expectations in 1a; driver-layer `lanes=ok` and `lanes=DEGRADED(` both recorded with their profile
- [ ] `make check-file-sizes`, `make check-changelog`, `make test-changed` rc 0
- [ ] V34 present with the full output; counts match the prediction, or the doc was corrected to the measurement
- [ ] `ls design_docs/implemented/v0_60_6/m-fleet-rulings-ledger*.md` lists 2 files; `grep -rn 'planned/m-fleet-rulings-ledger' design_docs changelogs .ailang/state/sprints` → no output (control: `grep -rn 'implemented/v0_60_6/m-fleet-rulings-ledger' …` ≥ 1)
- [ ] Mutation: none new. This milestone runs gates. If any gate is red, fix it in the milestone that owns the file and re-run. Never weaken a gate.

## Success Metrics

- T1–T11 (incl. T4b, T7a/T7b) plus the P8b ordering test and the 3 CLI tests: 33 new tests in total (M1 8, M2 5, M3 9, M4 6, M5 3, M6 2), each with a recorded mutant that it kills
- `go test ./internal/mission/ ./cmd/ailang/ -run 'Mission|Doctor|Ruling' -count=1` rc 0; `make test-launchd-drivers` rc 0
- Zero-finding doctor output byte-identical (`TestMissionDoctorPrint_ZeroFindingsByteIdentical`)
- Five charters + template linked; Gate 0 mirrors byte-identical; `.agents` sync `--check` rc 0
- Live run recorded (V34). Exit code equals the same-rig baseline
- Docs: changelog fragment; design doc moved to `implemented/` with V34. No `examples/*.ail` (no language feature; the AILANG syntax gate does not apply, since no milestone writes `.ail`)

## Files (complete; all inside the fleet write scope, P20)

New: `internal/mission/rulings_parse.go`, `rulings_parse_test.go`, `rulings_corpus.go`, `rulings_corpus_test.go`, `rulings.go`, `rulings_test.go`, `rulings_live_test.go`, `internal/mission/testdata/rulings/*` (4 files), `cmd/ailang/mission_doctor_print_test.go`, `design_docs/fleet-rulings.md`, `changelogs/unreleased/<date>-fleet-rulings-ledger.md`.
Edited: `internal/mission/doctor.go` (1 line), `cmd/ailang/mission_cmd.go`, `design_docs/{v1,fleet,docs,motoko,triage}-mission.md`, `design_docs/mission-charter-TEMPLATE.md`, `.claude/skills/mission-control/resources/gate-0-preflight.md`, `.agents/skills/mission-control/resources/gate-0-preflight.md` (regenerated), `design_docs/planned/m-fleet-rulings-ledger.md` (→ `implemented/`), this plan (→ `implemented/`), `.ailang/state/sprints/sprint_M-FLEET-RULINGS-LEDGER.json`.
**Not touched:** `internal/mission/doctor_test.go`, `missions/*.toml`, `scripts/mission_*.sh`, `tools/launchd/**`, world/stapledon repos.

## Dependencies

- None blocking. #2396 (OPEN) changes only D-FR-1's live count. The rebase-onto-post-#2396 condition is the controller's, before merge.
- Attended follow-up (out of scope, as the design says): link lines in the world and stapledon charters (other repos). Until then the doctor reports `ruling-link-missing` for them, which is true.

## Open Questions

None. D-FLEET-25 is RESOLVED, and no still-OPEN decision gates this sprint.

## Notes / Risks

- **Fixture realism:** `LoadFile` needs a valid TOML (P4). The M3 helper writes one. Hand-built missions have no root and only exercise the `rulings-skipped` path.
- **Lane-check cells change:** on exit 0 the `config ok` cell keeps showing the summary (V26). On exit 1 it keeps showing a per-mission finding, because of P8b's prepend. Arming is unaffected in both cases.
- **Per-mission runs** report skills hits for every mission's lane check. The cells still show the last line, so this is visible only in the full doctor output.
- **Cost:** `mission list` also runs the check (it calls `Doctor`): about 40 small file reads, which is negligible.
- **A mutant that cannot be written is not a pass.** If any listed mutant cannot be applied as described, say so in the commit body and record the closest one that can be applied, together with its FAIL line.
