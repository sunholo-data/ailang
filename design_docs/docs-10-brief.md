# docs-10 sprint brief — verify-examples is vacuous on two independent axes

**Not a design doc.** Per `docs-mission.md`'s Guardrails ("most items here need no design doc at
all — prefer a Gate-2 reality-check straight into a sprint"), this is a routing declaration only —
it exists so `tools/launchd/derive-planner-lane.sh` can route the planner instead of failing closed
to opus for a missing `Planner-Lane` field. It carries no design claims and needs no quorum.

**Planner-Lane**: opus-required

(Both target files live at the repo-root `scripts/`, which matches none of this mission's
`MISSION_PLANNER_ALLOWLIST` patterns — `tools/*`, `.claude/skills/mission-control/SKILL.md`,
`.claude/skills/design-doc-creator/*`, `docs/*`, `examples/*`, `README.md`, `CHANGELOG.md`,
`.claude/skills/docs-sync/scripts/*` — that last one is a different directory tree from repo-root
`scripts/`. A `codex-ok` declaration would fail closed to opus anyway via
`path-not-in-codex-allowlist`, so this declares the honest outcome directly.)

## Task

Fix two independent vacuous-pass defects in the docs mission's example-verification tooling.
Confirmed still present at current HEAD by reading both files and both GitHub issues in full
(`gh issue view 670 --repo sunholo-data/ailang`, `gh issue view 654 --repo sunholo-data/ailang`).

### 1. `scripts/verify_examples.go` never compares `expected.stdout` (issue #670)

`runExample` (currently ~line 120–292) decides pass/fail purely from `err == nil` plus a stderr
scan for `Error:`/`error:`. It never reads `examples/manifest.json`'s per-entry `expected.stdout`
field at all, so an example can print anything and still pass. Confirmed empirically in the issue
by corrupting one entry's `expected.stdout` to a deliberately wrong literal and observing
`make verify-examples` still exit 0.

Fix: load `examples/manifest.json` (the same `internal/manifest` package `scripts/validate_manifest.go`
already uses) and, for each example whose manifest entry has a **non-empty** `expected.stdout`,
compare it against the captured `stdout.String()` in `runExample`. On a mismatch, set
`result.Status = "failed"` with an error message showing both values (or a diff), the same way a
non-zero exit is reported today. Entries with empty/absent `expected.stdout` are unaffected — do
not require a stdout match where none is recorded (a check of the live manifest shows only 20 of
199 entries currently carry a non-empty `expected.stdout`; forcing all 199 to match an empty
string is out of scope and would falsely fail the other 179).

Add the anti-vacuity floor the issue itself proposes: if the count of entries with a non-empty
`expected.stdout` that actually got checked is zero, fail loudly (non-zero exit, clear stderr
message) rather than silently reporting a clean run — this is the same shape as the #654 floor
below and prevents this new check from becoming its own vacuous no-op (e.g. if the manifest
schema ever changes shape).

Acceptance behavior: corrupting one manifest entry's `expected.stdout` to a value that does not
match that example's actual stdout must make `make verify-examples` report that example as
**failed** (currently it reports rc=0 / all passed). Restoring the manifest byte-identically must
return the suite to green. An example with no recorded `expected.stdout` must be unaffected by
this change.

### 2. `scripts/validate_manifest.go` prints `checked` but never asserts it (issue #654)

The `checked` counter (incremented per successfully-parsed example, currently ~line 83) is printed
in the summary line (~line 93) but compared to nothing. The only non-zero exit path is
`driftCount > 0` (~lines 103–108). If the enumeration in `m.Examples` ever returns effectively
nothing resolvable (a glob change, a path move, an empty/broken manifest), the tool still prints
`0 modules checked, 0 drift, 0 missing-on-disk` followed by the green
`✓ manifest \`modules\` field is in sync with actual imports` line, and exits 0.

Fix: before printing the success line, assert `checked > 0`. If `checked == 0`, print a loud
failure to stderr (e.g. `INSTRUMENT FAILURE: validate_manifest enumerated 0 modules`) and exit 1
instead of printing the green line — this must apply in both `--ci` and non-`--ci` mode, since a
zero-enumeration is a broken instrument regardless of CI mode, not a matter of drift tolerance.
Use the honest minimum floor (`checked > 0`), not a fixed larger threshold — the issue explicitly
flags a fixed floor like `>= 150` as needing an ongoing maintenance story that is out of scope here.

Acceptance behavior: with the enumeration forced to return zero resolvable examples (e.g. by
pointing `--dir`/`--manifest` at an empty/nonexistent set, or temporarily faking zero matches),
`validate_manifest` must exit non-zero and print the loud failure message instead of the green
"✓ ... in sync" line. Normal runs against the real manifest must be unaffected (still exit 0 when
`driftCount == 0` and `checked > 0`).

### Non-goals

- Do not unify or restage the two files' unrelated responsibilities (stdout truth belongs to
  `verify_examples.go`; `modules`-drift truth belongs to `validate_manifest.go` — the header comment
  in `validate_manifest.go` already documents this split; preserve it).
- Do not attempt to backfill/re-capture the 179 manifest entries with empty `expected.stdout` — that
  is a separate, larger corpus-quality task, not this fix.
- Do not change `missingCount` handling in `validate_manifest.go` (already deliberately a warning,
  not a hard failure, per the file's own header comment) — only the unasserted `checked` floor is in
  scope.
- Do not add the staged flag/rollout machinery issue #670 sketches as one *possible* sequencing
  (flag-gated report → repair stale entries → full gate) — with only 20 entries actually carrying a
  pinned value, a direct fix is small enough not to need that staging.

## Files

- `scripts/verify_examples.go`
- `scripts/validate_manifest.go`
