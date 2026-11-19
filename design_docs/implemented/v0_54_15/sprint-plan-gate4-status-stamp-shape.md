# Sprint plan: M-FLEET-GATE4-STATUS-STAMP-SHAPE

Ticket: `skill:gate4-status-stamp-shape-unprescribed` (motoko iteration 40, 2026-10-01).
Fleet iteration: 44. Status: planned. Verified HEAD: `bd30497a0f5bd8a89a54483643cdc55f5ff69991`.
Design doc: none (`design_doc: null`). This is a mechanical skill documentation repair;
`design_docs/fleet-mission.md:73-74` requires design only when warranted. This plan authorizes
no implementation by the planner. One milestone, M1, as explicitly requested; the skill's
usual two-milestone minimum does not apply to this ticket. Controller owns approval and dispatch.

## Goal, scope and estimate

Make Gate 4 prescribe the code's canonical STATUS shape for every mission before the stamp
is written, preventing a later rotation from introducing a non-canonical archived heading.
Change only these three execution deliverables:

- `.claude/skills/mission-control/resources/gate-4-record.md`
- `.agents/skills/mission-control/resources/gate-4-record.md`
- `changelogs/unreleased/2026-10-16-gate4-status-stamp-shape.md`

The two resources must remain byte-identical. Add about 8–20 lines per copy (proposed insertion:
15 nonblank lines plus spacing), and one short historical qualifier. Estimate: 18 documentation
LOC per copy + 5 fragment LOC = **41 LOC**, zero persistent test LOC; **0.125 working days
(60 minutes)** including focused validation and evaluator drills. Risk: low. No `.ail` code,
showcase modules, new packages, or new test files; the existing normalizer and canonical test
are the working examples and verification instruments.

No routing, lane order, quota/ration thresholds, billing guards, CI workflows, SKILL.md,
Go code, tests, charters, or other resource changes. No git add/commit/stash/checkout/branch
by the planner; the controller commits. Planning artifacts are the two requested files only;
execution and controller record evidence are distinct from these planning outputs.

## Verified premises at HEAD

| Evidence | Finding |
| --- | --- |
| `internal/mission/normalize.go:27,34,37` | Example STATUS headline; `canonicalEntryRe` is at **34**, not 30 (30 is commentary); `canonicalStatusRe` at 37 is `^## STATUS \d{4}-\d{2}-\d{2} — ITERATION \d+: \S`. Literal U+2014, single spaces, upper-case ITERATION, number immediately followed by colon and a non-space headline. |
| `internal/mission/canonical_test.go:19,21-44,68-89` | Ratchet is 11; record-stream globs cover logs, log archives, status archives and status archive old files, deliberately excluding charters. At present the count must equal 11; a reduction also asks for lowering the constant. Never change that constant in this sprint. |
| `internal/mission/normalize.go:48-62,89-111,140-168` | Normalization converts COMPLETE and parenthesized iteration variants, preserves their information in the headline, and reports unconvertible headings. |
| `cmd/ailang/mission_cmd.go:138-144,419-427,430-488` | `ailang mission normalize` defaults to dry run; `--apply` writes. It repairs record streams, not charter prose. No live repair is part of this documentation sprint. |
| `.claude/skills/mission-control/resources/gate-4-record.md:109-112` | The queue/STATUS update step gives no heading grammar; insert the shared rule immediately here. |
| Same resource `:241-261` | Historical casing/grep lesson includes World's parenthesized lower-case stamp at 247. Retain that story and its `-ci` advice, marking that example as a non-canonical legacy form with a short pointer to the new rule. The new rule governs writing; tolerant searches still find history. |
| Same resource `:308-316,339-347` | Existing rotation arithmetic and archive-presence safeguards remain intact. Later descriptive per-mission casing discussion concerns legacy searches, not permission to write new variants. |
| Both resource copies (same line numbers) | `cmp` rc 0; 28,540 bytes each, 347 lines each; distinct inodes 187449562 (.claude) and 187448836 (.agents). Mirror both, consistent with D-FLEET-8 (`design_docs/fleet-mission.md:156`). No companion size split is needed. |
| `design_docs/v1-mission.md:219,221,223`; `docs-mission.md:172,188,232`; `motoko-mission.md:179,181,183`; `fleet-mission.md:46,52,58`; `triage-mission.md:44,46` | All actual STATUS stamps in the five named live charters match the production regex. Structural `## STATUS (rotation rule)` headings are not stamps. |
| `missions/world.toml:9-10` | World is a separate repo; its current charter/rotation cannot be verified from this checkout. Shared wording covers every mission, including World; cross-repo evidence belongs to the controller when available. |

The defect is delayed: a wrong stamp in a charter passes silently because the charter is not
linted. Gate 4 later moves that line into `*-mission-status-archive.md`; the same line then
counts against `TestMissionDocHeadingsStayCanonical`'s ratchet and reds it, usually in a
**later iteration** than the authoring iteration. The fix belongs at the shared writing step,
rather than in each mission's local note or by relaxing the regex/test.

## Registry reuse and velocity

M1 decision: **none**. There is no package-like capability to search or depend on. Reuse
`canonicalStatusRe`, `canonicalEntryRe`, the existing normalizer and canonical test by reference;
no registry network query or new helper is warranted.

Comparable landed repair `a8042fa3f` (2026-10-16) added 25/deleted 3 lines per mirrored Gate-3b
resource plus a 5-line fragment. This sprint is smaller. The velocity helper was inspected and
run read-only; its `git log | head` pipeline exited 141 under pipefail, so it supplies no reliable
aggregate throughput. Reported, not fixed. Use the measured comparable patch size and a
conservative 60-minute estimate, not a claimed historical LOC/day. JSON target 328 LOC/day
is the planning quotient 41 / 0.125, not measured throughput.

## M1 — Prescribe and prove canonical record headings

Dependencies: none. Deliverables: the three execution paths above. Estimate: 41 documentation
LOC, zero persistent tests. Day 1 (one bounded session):

1. **0–10 min:** recheck HEAD, the regexes and insertion point; record any baseline changes.
2. **10–25 min:** insert the following compact rule immediately after the STATUS step in both
   copies; keep total additions around 8–20 lines per copy. Preserve existing surrounding text.
3. **25–30 min:** qualify World's example at the historical casing passage, e.g. append
   `(non-canonical legacy form; see the shared STATUS rule above)` after the example.
   Preserve the war story, measured counts, and case-insensitive search advice.
4. **30–40 min:** add the 5-line fragment beginning `### Fixed — `, explaining the delayed
   rotation failure and linking this ticket/plan, per `changelogs/unreleased/README.md:3-24`.
5. **40–60 min:** run the focused gates, positive/negative shape check, mutation drills,
   mirror comparison and scope review; supply controller-only evidence requests to the evaluator.

Proposed insertion (15 nonblank lines; long command lines keep this proportionate):

```markdown
**Shared canonical record headings (all missions).**
For every mission, write `## STATUS <YYYY-MM-DD> — ITERATION <n>: <headline>`.
Source of truth: `internal/mission/normalize.go` `canonicalStatusRe`; the code regex wins over prose.
Use EM DASH (U+2014), single spaces, upper-case `ITERATION`, a colon directly after <n>,
and a headline starting with a non-space. No `COMPLETE`/`DONE` between number and colon,
no parenthetical on the date, and no lower-case `iteration` (e.g. `## STATUS 2026-10-16 — ITERATION 44 COMPLETE: x` is forbidden).
Log entry companion (`canonicalEntryRe`): `## <n> — <YYYY-MM-DD> — <title>`.
WHY: charters are not linted; a wrong stamp passes silently until rotation moves it into
`*-mission-status-archive.md`, where `TestMissionDocHeadingsStayCanonical`'s ratchet
reds it, usually in a LATER iteration than the one that wrote it.
Before committing the record: `go test ./internal/mission/ -run Canonical -count=1` and
`printf '%s\n' "$new_stamp" | grep -Eq '^## STATUS [0-9]{4}-[0-9]{2}-[0-9]{2} — ITERATION [0-9]+: [^[:space:]]'` (new_stamp = actual new heading).
Repair: `ailang mission normalize` (dry run), inspect rewrites/unconvertibles, then `ailang mission normalize --apply`.
The normalizer repairs record streams, not charter prose; correct a charter stamp before rotation.
Mission charters may keep a local note; this is the shared rule and governs new stamps over legacy examples below.
```

No automated invocation of `--apply` is authorized by this plan. The shared text documents
repair for future affected records; a baseline defect is reported separately. A mission may
retain its own local note, but the shared rule and production regex take precedence.

## Acceptance and validation

M1 passes only when all criteria below have actual evidence. Unavailable controller evidence
stays pending; it is not inferred from sandbox output or from baseline results.

1. Only the three execution deliverables change (beyond the already supplied planning artifacts).
   Both copies contain the rule at the STATUS step, stay byte-identical, add about 8–20 lines
   each, preserve the historical casing story with its legacy qualifier, and leave all policy and
   other files untouched. Fragment begins `### Fixed — `.
2. The literal shape, all forbidden variations, log companion, code-over-prose authority,
   delayed rotation/ratchet WHY, canonical test command, actual-new-stamp one-line grep,
   dry-run-first repair and optional local-note rule are present as specified above.
3. The behavioural check below derives its positive heading from the **edited document**, not
   a separately hard-coded good sample; it matches the production `canonicalStatusRe` extracted
   from Go source. The documented COMPLETE counterexample does not match. Evaluator re-runs it.
4. All eight static gates, mirror comparison, and canonical tests return rc 0 on edited contents;
   full `go test ./internal/mission/ -count=1` must have real controller evidence outside this
   sandbox because its quota tests bind sockets. No full-repository `make test` is needed.
5. All four mutations below are caught, with original count/rc and mutant count/rc recorded by
   the independent evaluator using disposable text only.
6. Controller demonstrates one charter-to-status-archive rotation for each of the five named
   live missions on disposable copies, with canonical stamps and green full mission tests as
   described below. World needs its own checkout if included in the controller's live inventory.
7. CONTROLLER ONLY, outside sandbox:
   `env -i HOME=$HOME PATH=$PATH USER=$USER TMPDIR=$TMPDIR make test-launchd-drivers` returns rc 0.
8. The controller's actual iteration-44 Gate-4 record uses the prescribed shape, e.g.
   `## STATUS 2026-10-16 — ITERATION 44: <actual headline>`, and its post-record
   `go test ./internal/mission/ -run Canonical -count=1` returns rc 0. Controller record changes
   are outside the executor's three-file scope. Independent sprint-evaluator judges the result.

Run each required gate; if one is red, retain its result and run the other independent gates.
A red baseline is a finding, never a reason to change unrelated files or loosen a gate.
Any network/socket/gh-dependent result in the sandbox is **UNINFORMATIVE UNDER SANDBOX**,
never a pass or fail. `test-launchd-drivers` is explicitly controller-only; do not run it here.

```bash
make check-skills
make check-context-docs
make check-referenced-paths
make check-changelog
make check-file-sizes
make check-doc-review
make check-no-personal-email
make check-tmpfile-hygiene
go test ./internal/mission/ -run Canonical -count=1
go test ./internal/mission/ -count=1
cmp .claude/skills/mission-control/resources/gate-4-record.md .agents/skills/mission-control/resources/gate-4-record.md
git diff --stat
git diff --name-only
```

Planner baseline: all eight static gates rc 0 (doc-review warns about two existing missing
reviewBy fields, with zero malformed dates); `cmp` rc 0; focused canonical test rc 0.
The full mission package encounters `httptest: failed to listen ... bind: operation not permitted`
in `TestObserveAnthropicQuota_OverRationBlocksByDefault`: **UNINFORMATIVE UNDER SANDBOX**.
The controller's supplied full-test and driver-test baseline rc 0 is attributed evidence,
not independently reproduced here. No informative baseline gate was red.

### Behavioural shape check (throwaway Perl, no persistent test/script)

Run from repository root against each resource copy. Perl reads the regex from production code;
for these concrete ASCII fields and literal em dash, it exercises the same pattern as Go.
Expected: `shape MATCH; forbidden NO MATCH`, rc 0. Missing/mutated prescription fails loudly.

```bash
g=.claude/skills/mission-control/resources/gate-4-record.md
perl -e '
use strict; use warnings; use utf8;
sub read_utf8 {
    open my $fh, "<:encoding(UTF-8)", $_[0] or die "$!\n";
    local $/; return <$fh>;
}
my $src = read_utf8("internal/mission/normalize.go");
my ($rx) = $src =~ /canonicalStatusRe = regexp\.MustCompile\(`([^`]+)`\)/;
defined $rx or die "production regex not found\n";
my $doc = read_utf8($ARGV[0]);
my ($shape) = $doc =~ /^For every mission, write `([^`]+)`\.$/m;
defined $shape or die "prescribed shape not found\n";
$shape =~ s/<YYYY-MM-DD>/2026-10-16/;
$shape =~ s/<n>/44/;
$shape =~ s/<headline>/x/;
$shape =~ /$rx/ or die "prescribed shape does not match\n";
my ($bad) = $doc =~ /e\.g\. `([^`]+)` is forbidden/;
defined $bad or die "forbidden example not found\n";
$bad !~ /$rx/ or die "forbidden example matched\n";
print "shape MATCH; forbidden NO MATCH\n";
' "$g"
```
Repeat with `g=.agents/skills/mission-control/resources/gate-4-record.md`.

### Four evaluator mutation drills

Use the insertion's exact lines above so each assertion is reproducible. Set `g` to the
edited resource. Create mutants only in a unique scratch directory, never edit tracked files.
The same assertions must work against both resource copies. These are acceptance instruments,
not new committed tests. Mutations must not touch the historical example or source regex.

```bash
drill=$(mktemp -d "${TMPDIR:-/tmp}/gate4-shape.XXXXXX")
trap 'rm -rf "$drill"' EXIT
perl -Mutf8 -CSD -pe 's/ — / - / if /^For every mission, write /' "$g" > "$drill/hyphen.md"
perl -pe 's/<n>:/<n> COMPLETE:/ if /^For every mission, write /' "$g" > "$drill/complete.md"
perl -ne 'print unless /^Source of truth:/' "$g" > "$drill/no-source.md"
perl -0777 -pe 's/^WHY:.*\n[^\n]*\n[^\n]*\n//m' "$g" > "$drill/no-why.md"
```

For each row, set `candidate="$g"` for the original, then to the named mutant and run the
command verbatim. Require **original stdout 1 / rc 0; mutant stdout 0 / rc 1** in every row.
These counts target operative prose, so a copy of the shape or explanation elsewhere cannot
make a damaged rule pass. The source deletion removes both normalize.go and canonicalStatusRe.

| Mutation | Assertion command |
| --- | --- |
| Replace prescribed em dash by hyphen (`hyphen.md`) | ``grep -Fxc 'For every mission, write `## STATUS <YYYY-MM-DD> — ITERATION <n>: <headline>`.' "$candidate"`` |
| Insert COMPLETE before the colon (`complete.md`) | ``grep -Fxc 'For every mission, write `## STATUS <YYYY-MM-DD> — ITERATION <n>: <headline>`.' "$candidate"`` |
| Drop source-of-truth line (`no-source.md`) | `grep -Ec '^Source of truth: .*internal/mission/normalize[.]go.*canonicalStatusRe.*' "$candidate"` |
| Drop rotation/ratchet WHY (`no-why.md`) | `grep -A2 '^WHY:' "$candidate" \| tr '\n' ' ' \| grep -Ec 'charters are not linted;.*rotation.*mission-status-archive[.]md.*TestMissionDocHeadingsStayCanonical.*ratchet.*LATER iteration'` |

In the last table cell, `\|` is Markdown escaping: execute the two actual shell pipes `|`.
The behavioural check also rejects either mutated prescribed shape. Retain the grep evidence
for the missing source/WHY prose, which cannot be detected by a regex match alone.

### Controller rotation rehearsal and actual record

Outside the sandbox, create a unique disposable repository snapshot under TMPDIR using
read-only `git archive HEAD` (no git worktree/branch mutation). Overlay the edited guide copies
so the controller follows the new rule. Keep the delivery worktree and live charters unchanged.
For each `v1`, `docs`, `motoko`, `fleet`, `triage`:

- Read that snapshot's actual charter STATUS lines. Triage starts with two; seed a canonical
  fixture stamp to reach three before the drill. This is fixture setup, not a retention-policy edit.
- Write a new stamp derived from the edited prescription with the next fixture iteration number.
  Follow the existing Gate-4 **single heading line + blank** move: retain newest three, move the
  oldest heading and blank into that mission's status archive. This must actually move a stamp,
  not a no-op. Do not move blocks delimited by the next header or change the rotation algorithm.
- Verify the new and moved headings match canonicalStatusRe, require the exact moved heading
  present in the destination, and check `after == before + 2 - 2*len(moved)` plus preservation of
  non-STATUS charter content/known queue rows. Record before/after headings and archive path.
- From the snapshot root run `go test ./internal/mission/ -run Canonical -count=1` and
  `go test ./internal/mission/ -count=1`; require rc 0 after **each** mission's rotation. Do not
  normalize historical headings to conceal a red ratchet and do not change its constant.

`ailang mission rotate-log <name> --stream status` rotates the **archive stream**, not the
charter into the archive (`cmd/ailang/mission_cmd.go:397-404`); it is not a substitute for this
rehearsal. If the controller's live registry includes World, use its own disposable checkout
and record its rotation evidence too; access absent here remains explicitly unverified.
Finally, the controller writes its real fleet iteration-44 Gate-4 record with the prescribed
shape, runs the canonical test after record/rotation, and supplies the outside-sandbox driver
gate result. Do not mark M1 accepted while any required controller evidence is pending.
