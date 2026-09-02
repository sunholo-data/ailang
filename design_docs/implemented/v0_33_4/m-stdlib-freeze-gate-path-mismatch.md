# M-STDLIB-FREEZE-GATE-PATH-MISMATCH: Retire the Rotted Freeze Duplicate, Alias the Accepted Name to the Live Gate

**Status**: Planned
**Target**: v0.33.4
**Priority**: P1 (a gate accepted in v0.2.0 is unrunnable; it was the purpose-built independent check on iteration 282's 30-file `std/` reformat and was dead when needed)
**Estimated**: 0.5 day
**Dependencies**: None
**Queue row**: `m-stdlib-freeze-gate-path-mismatch` (filed iteration 282, [v1-mission.md](../../v1-mission.md) queue)
**Quorum**: authored unattended (mission loop) → design-quorum REQUIRED before planning, per the design-doc-creator skill. No freeze items — every decision below is agent-resolvable.

---

## Problem Statement

`make test-stdlib-freeze` — the stdlib interface-stability gate whose v0.2.0 acceptance
checkbox reads "`make test-stdlib-freeze` target works (SHA256 matching)"
([M-S1-B.md:100](../../implemented/v0_2_0/M-S1-B.md)) — exits **rc=2 during prerequisite
resolution**, before its recipe body runs:

```
make: *** No rule to make target `goldens/stdlib/option.sha256', needed by `test-stdlib-freeze'.  Stop.
```

`Makefile:59` sets `FREEZE_DIR := goldens/stdlib`; that directory does not exist and has
**never existed in any commit on any ref of this repository** — verified exhaustively over
all history, with in-scope positive controls proving the instrument fires (V18–V20). The
v0.2.0 recipe in M-S1-B.md (`mkdir -p goldens/stdlib`) could only ever have created it in a
developer's local working tree; no commit ever carried it. The real goldens live in
`.stdlib-golden/` (90 committed files: 45 modules x {json, sha256}), written by
`tools/freeze-stdlib.sh` via the shared `GOLDEN_DIR=".stdlib-golden"` in
`tools/stdlib-iface-lib.sh:9`. The target is wired into **zero** workflows, so dev stays
green over the corpse. Sprint-executor skill docs still recommend the broken target by name
(`developer_tools.md:67,142`, in both the `.claude/` and `.agents/` skill trees).

**NOTE on the queue row**: it attributes the golden-writing to `tools/freeze-stdlib.sh`
directly. Verified relationship: `freeze-stdlib.sh` is the executable *entry point*; it
`source`s `tools/stdlib-iface-lib.sh`, which defines `GOLDEN_DIR=".stdlib-golden"` plus the
shared module enumeration and JSON generation. `tools/verify-stdlib.sh` sources the same lib
— by design, so freezer and verifier cannot disagree on binary, module list, or JSON shape.

## Root Cause: the recipe is not one bug, it is a fossil (three independent rots)

The `make/test.mk:184-201` recipe would fail even if the path were patched:

1. **Wrong path** — `$(FREEZE_DIR)` = `goldens/stdlib` vs. the real `.stdlib-golden/`.
   The five `$(FREEZE_DIR)/*.sha256` prerequisites have no rule, so make aborts at
   rc=2 and the recipe body — including its own missing-golden floor at test.mk:193
   (`if [ ! -f $$golden ]; then echo "MISSING $$golden"; ok=1;`) — is unreachable.
2. **Invalid CLI invocation** — the body runs `$(TOOLS) iface --module "$name" --json`.
   The current binary rejects it: `flag provided but not defined: -module` (rc=2,
   verified — see log V9). The v0.2.0-era flag form no longer exists; the live scripts
   invoke `ailang iface "std/$module"`.
3. **Wrong binary + wrong coverage** — `TOOLS := ailang` resolves from PATH (the
   stale-system-binary trap `resolve_ailang` in the lib was written to close), and
   `STDLIB` hardcodes 5 of 45 modules. The lib's own comment (stdlib-iface-lib.sh:29-31)
   calls the 5-module hardcode "worse than none, because it reads as coverage."

**History**: the gate family was accepted in v0.2.0 (M-S1-B). Commit `80019d4e0`
("revive verify-stdlib — dead since v0.0.12") rebuilt the *tools-side* gate properly —
all-45-module enumeration, real error surfacing, `.stdlib-golden/` goldens, a CI-run
red-on-change selftest — but left the make-side duplicate in `test.mk` untouched. Since
then the repo has had one live gate and one corpse with the historically accepted name.

**Systemic check (audit before patching)**: this is not a one-off path typo to fix in
place. The modern gate already exists, is CI-wired, and already carries the anti-vacuity
floors this design must provide. The systemic fix is *deduplication*: one gate, one name
reaching it, zero parallel implementations that can drift again. Drift between duplicate
implementations of this exact gate is precisely what already happened once.

## Decision (a): canonical golden path is `.stdlib-golden/`

Justification, not assertion:

| Criterion | `.stdlib-golden/` | `goldens/stdlib/` |
|---|---|---|
| Exists / populated | YES — 90 files committed to git (V4) | NO — `test -d` fails (V1); never in any commit on any ref (V18–V19) |
| Written by | `tools/freeze-stdlib.sh` (live, `make freeze-stdlib` at examples.mk:133) | nothing (V5: no writer anywhere in repo) |
| Read by | `tools/verify-stdlib.sh` (live, CI at ci.yml:142) | only the dead recipe (V5) |
| CI-verified today | YES — `verify-stdlib` + `verify-stdlib-selftest` (V7) | never (V6) |
| Coverage | all 45 `std/` modules, filesystem-derived | 5 hardcoded modules |

Repointing the tools at `goldens/stdlib` would rewrite 90 committed files, 3 scripts, and
CI-proven behavior to satisfy one dead recipe. Repointing the recipe at `.stdlib-golden`
would still leave rots #2 and #3 and a second implementation to drift. Canonical =
`.stdlib-golden/`; the recipe is retired, not repaired.

## Solution Design

### M1 — Delegate the name, delete the fossil (core fix, one commit)

In `make/test.mk`, replace lines 184-201 with a delegation alias:

```make
# Stdlib freeze — historical name, kept because v0.2.0 acceptance docs and the
# sprint-executor skill reference it. The live gate is verify-stdlib
# (tools/verify-stdlib.sh over .stdlib-golden/); do not grow a second implementation.
test-stdlib-freeze: verify-stdlib ## Verify std/ interfaces haven't changed (alias of verify-stdlib)
```

`test-stdlib-freeze` stays in `.PHONY` (test.mk:9). `verify-stdlib` already depends on
`build` (examples.mk:137), so the alias inherits fresh-binary resolution.

In `Makefile`, delete `STDLIB` (:58), `FREEZE_DIR` (:59), `TOOLS` (:60) and remove them
from the `export` at :66. This is a measured deletion, not a linter-driven one
(per `.claude/rules/coding-standards.md`): `make -pn` shows `FREEZE_DIR` consumed only by
the dead recipe (V8), and a repo-wide grep for consumers of the three exported names over
`examples/ scripts/ tools/ .github/` returns zero hits with a positive control (V8).

Precedent for alias-shape: `verify-lowering` is an accepted alias of the wired
`verify-no-shim` and is exempted from gate-wiring with the recorded reason "wiring this
adds an alias, not coverage" (`internal/cihygiene/gate_wiring_test.go:39`).

### M2 — Permanent selftest arms (one commit)

`make verify-stdlib-selftest` (examples.mk:147, CI at ci.yml:145) today proves ONE red
branch: canary export → "interface changed" + diff shown. Add two cheap arms to the same
target.

**Restore-safety constraint (quorum round 1, confirmed by reading the recipe, V21):** the
existing EXIT trap restores `std/option.ail` ONLY, from a backup of `std/option.ail` — it
has no knowledge of `.stdlib-golden/`. A missing-golden arm that leans on it would
permanently delete `.stdlib-golden/option.sha256` from the developer's tree. Additionally,
`trap ... EXIT` is per-shell and a second trap in the same shell **replaces** the first, so
the arms must not share a shell. Therefore: **each arm runs as its own `@(...)` recipe line
(its own shell), takes its OWN mktemp backup, and installs its OWN EXIT trap that restores
exactly what that arm moved. The canary arm's trap and the golden arm's trap are fully
independent; neither is edited.** Every `exit 1` inside an arm still fires that arm's trap,
so restore holds on all failure paths, not just success.

1. **Missing-golden arm** — exact recipe text (same mktemp/cp/trap idiom as the canary
   arm at examples.mk:149-153, but backing up the GOLDEN, in an isolated shell):

   ```make
   	@echo "Gate self-test (missing-golden arm): removing option's golden..."
   	@( GOLDEN_BK=$$(mktemp "$${TMPDIR:-/tmp}/ailang-selftest-golden.XXXXXX") || exit 1; \
   	ARM_OUT=$$(mktemp "$${TMPDIR:-/tmp}/ailang-selftest-arm-out.XXXXXX") || { rm -f "$$GOLDEN_BK"; exit 1; }; \
   	cp .stdlib-golden/option.sha256 "$$GOLDEN_BK" || { rm -f "$$GOLDEN_BK" "$$ARM_OUT"; exit 1; }; \
   	trap 'cp "$$GOLDEN_BK" .stdlib-golden/option.sha256; rm -f "$$GOLDEN_BK" "$$ARM_OUT"' EXIT; \
   	rm -f .stdlib-golden/option.sha256; \
   	if tools/verify-stdlib.sh >"$$ARM_OUT" 2>&1; then \
   		echo "❌ SELF-TEST FAIL: gate exited 0 with option's golden missing"; exit 1; \
   	fi; \
   	grep -q "option: no golden file" "$$ARM_OUT" || { \
   		echo "❌ SELF-TEST FAIL: gate failed but never reported option as UNCOVERED"; \
   		cat "$$ARM_OUT"; exit 1; }; \
   	echo "✓ missing golden -> non-zero + module reported UNCOVERED" )
   ```

   Failure-path audit: mktemp failures exit before anything is moved (nothing to
   restore); the `cp`-backup failure cleans up temps with the golden untouched; from the
   `trap` line onward, every exit path — gate-green failure, wrong-message failure,
   success — fires the trap, which restores the golden from the arm's own backup. The
   expected full message is `option: no golden file — module is UNCOVERED`
   (verify-stdlib.sh:33, V13); the grep matches the stable ASCII prefix.
2. **Alias-integrity arm**: `make -n test-stdlib-freeze | grep -q 'verify-stdlib'` —
   the historical name must still reach the live gate. This is the guard against the
   alias itself rotting the way the recipe did. (`test-*` is outside the gate-wiring
   test's asserted prefix classes — see Conflict Surface — so this arm is the only
   automated check on the alias.) Read-only: no backup/trap needed.

**Considered and deferred**: a permanent empty-enumeration arm. It would require making
`STDLIB_DIR` in `stdlib-iface-lib.sh` env-overridable so the selftest can point it at an
empty directory — but `freeze-stdlib.sh` sources the same lib, so an override variable
becomes a footgun where a stray exported `STDLIB_DIR` silently re-freezes the wrong tree.
The empty-enumeration floor already exists and fails loudly (verify-stdlib.sh:23-26,
`refusing to report success`, exit 1); it is proven by one-time reviewer mutation
(section (d), row 4) instead of a permanent arm. Revisit only if `std/` layout changes.

### M3 — Docs + changelog (one commit)

- CHANGELOG entry under v0.33.4 (fixed category).
- One-line correction in the two `developer_tools.md` skill copies: note the target is an
  alias of `verify-stdlib`. (Both files currently recommend the broken name — after M1
  they become *correct again* with no edit, so this is clarity, not repair.)
- Historical docs in `design_docs/implemented/v0_2_0/` are immutable records — untouched.

## CI Wiring (c)

After M1 the accepted name and the CI gate are the same script, so CI wiring is already
in place and stays where it is:

- `verify-stdlib` runs in `.github/workflows/ci.yml:142` and in the `ci` aggregate
  (`make/ci.mk:11`).
- `verify-stdlib-selftest` (red-on-change proof, extended by M2) runs at `ci.yml:145`
  and in `make/ci.mk:11`.

Adding a separate `test-stdlib-freeze` CI step would run the identical script twice per
job; we deliberately do NOT do that (same reasoning as the recorded `verify-lowering`
exemption). The alias-integrity arm in M2 is what makes "the name reaches the gated path"
itself CI-enforced. The gate-wiring assertion (`TestGateTargetsAreWiredIntoAWorkflow`)
requires `verify-stdlib` to be wired (gate_wiring_test.go:191-192 fails the *instrument*
if verify-stdlib is not found), so the underlying gate cannot silently fall out of CI.

## Anti-Vacuity Floors (b)

All vacuity paths land in the ONE delegated implementation. Floors are inherited, cited,
and (for the branches the selftest can safely exercise) permanently re-proven in CI:

| # | Vacuity path | Guard after this design | Where |
|---|---|---|---|
| 1 | Golden **dir** missing entirely | Every module hits the no-golden branch → `UNCOVERED` per module → exit 1. Loud, enumerated, non-abort | verify-stdlib.sh:31-37 |
| 2 | **One** golden file missing | Same branch, names the module; M2 arm 1 keeps this red-provable in CI | verify-stdlib.sh:31-37 |
| 3 | **Empty module enumeration** (the loop-zero-times-then-`exit 0` trap) | Pre-loop check: `no modules found under std/ — refusing to report success`, exit 1 — the loop is never entered on empty input | verify-stdlib.sh:22-26 |
| 4 | `ailang iface` **emits nothing** (rc=0, empty stdout) | `[ ! -s "$outfile" ]` → `produced no output` → module fails | stdlib-iface-lib.sh:52-56 |
| 5 | `ailang iface` **fails** | rc captured, stderr surfaced (not `2>/dev/null`-discarded), module fails | stdlib-iface-lib.sh:46-51 |
| 6 | Hash **mismatch** | `interface changed!` + expected/got + unified diff; CI selftest proves this branch goes red on a real export change | verify-stdlib.sh:53-62 |
| 7 | The **alias itself** rots (this bug's own shape) | M2 arm 2: `make -n test-stdlib-freeze` must name verify-stdlib | new, examples.mk |

Note the old recipe's floor (test.mk:193) guarded only path 2, and unreachably — make's
prerequisite abort preempted it. Prerequisite-based freshness is exactly the mechanism
that turned a stale path into an un-runnable gate; the delegated design has no
golden-file prerequisites, so a missing golden is a *loud in-recipe failure*, never a
missing-rule abort.

## (d) Reviewer Proof of Non-VacUITY — one mutation per guarded branch

Run from a clean tree; every mutation must be reverted afterwards (`git status` clean).
Each row names the arm that MUST go red; a green run on any row rejects the implementation.

| # | Mutation | Command | MUST go red, with |
|---|---|---|---|
| 1 | Perturb one golden sha: `printf '%064d\n' 0 > .stdlib-golden/option.sha256` | `make test-stdlib-freeze` | rc!=0; `option: interface changed!` + diff (floor 6, via the alias) |
| 2 | Remove one golden: `rm .stdlib-golden/option.sha256` | `make test-stdlib-freeze` | rc!=0; `option: no golden file — module is UNCOVERED` (floor 2) |
| 3 | Remove the whole dir: `mv .stdlib-golden /tmp/g.bak` | `make test-stdlib-freeze` | rc!=0; 45 x `UNCOVERED` lines, one per module (floor 1) |
| 4 | Empty enumeration: in a scratch worktree, `git mv std std.away` (or temp-edit `STDLIB_DIR="std-nonexistent"` in the lib) | `tools/verify-stdlib.sh` | rc!=0; `no modules found under ... — refusing to report success` (floor 3) — proves loop-zero-times is NOT green |
| 5 | Silent-empty iface: temp-edit `iface_json` to `: > "$outfile"; return 0` before the size check... or simpler, stub `AILANG` to `/usr/bin/true` via temp-edit of `resolve_ailang` | `tools/verify-stdlib.sh` | rc!=0; `produced no output` per module (floor 4) |
| 6 | Interface change: `make verify-stdlib-selftest` (no manual mutation — the target injects/reverts a canary export itself) | `make verify-stdlib-selftest` | rc=0 *for the selftest* while asserting the GATE went red, named `option`, and showed `selftestCanary` in the diff |
| 7 | Alias rot: temp-edit test.mk alias to `test-stdlib-freeze: ;` | `make verify-stdlib-selftest` (with M2 arm 2) | rc!=0; alias-integrity arm reports the name no longer reaches verify-stdlib |
| 8 | Control (must stay GREEN): clean tree | `make test-stdlib-freeze && make verify-stdlib-selftest` | rc=0 both; `All 45 stdlib interfaces stable` |
| 9 | **Restore-on-FAILURE** (not just on success): temp-edit the missing-golden arm's expected `grep` pattern to a string the gate never prints, forcing that arm to fail mid-flight with the golden removed | `shasum -a 256 .stdlib-golden/option.sha256; make verify-stdlib-selftest; echo rc=$$?; git status --porcelain; shasum -a 256 .stdlib-golden/option.sha256` | selftest rc!=0 AND afterwards `git status --porcelain` is EMPTY AND the golden's sha256 is identical to before — the arm's own trap restored `.stdlib-golden/option.sha256` on its failure path. (Revert the temp-edit; row 8 must then be green again) |

Row 8 is the known-positive control: if it is red at HEAD, the instrument is broken and
rows 1-7 prove nothing. Row 9 is the restore-on-failure proof demanded by quorum round 1:
a trap that only demonstrably runs on success is not evidence the tree survives a red run.

## Conflict Surface

Derived by command (V5, V8, V10-V12), not memory. This design touches `Makefile`,
`make/test.mk`, `make/examples.mk` — no parser/typechecker/codegen files, so the
language-conflict enumeration is N/A; the build-surface enumeration is:

| Surface | Relationship | Decision |
|---|---|---|
| `FREEZE_DIR`, `STDLIB`, `TOOLS` (Makefile:58-60, exported :66) | Sole consumer is the dead recipe (make -pn + repo grep, V8). Exported, but zero env consumers in `examples/ scripts/ tools/ .github/` (V8) | DELETE all three + export entries |
| `verify-stdlib` (examples.mk:137), `freeze-stdlib` (:133), `verify-stdlib-selftest` (:147), `tools/*.sh` | The live gate. M1 REUSES (alias prerequisite). M2 EXTENDS verify-stdlib-selftest — additive arms only, each in its own recipe-line shell with its own backup + EXIT trap (the existing canary trap restores `std/option.ail` ONLY, V21); the canary arm's assertions and trap are untouched | reuse / additive-extend |
| `.github/workflows/ci.yml:142,145` + `make/ci.mk:11` | Already invoke the live gate | unchanged |
| `TestGateTargetsAreWiredIntoAWorkflow` (internal/cihygiene/gate_wiring_test.go) | Asserted prefix classes are `check-*`/`test-check-*` (:130) and `verify-*` + `fmt-check-ail` (:134). `test-stdlib-freeze` is in NEITHER class, so the alias needs no exemption entry and trips nothing (V10). The test *requires* verify-stdlib to be wired (:191) | no change needed; cite in commit |
| `check-golden-drift` (make/test.mk:242, in `make ci`) | Despite the name, reads ONLY `internal/parser/testdata/parser/` (V11) — it does not see `.stdlib-golden/` and is unaffected | no overlap; named here to preempt confusion |
| Sprint-executor skill docs (`.claude/skills/.../developer_tools.md:67,142` + `.agents/` copy) | Recommend `make test-stdlib-freeze` by name (V12) | become live again via alias; M3 adds clarifying line |
| Historical docs `implemented/v0_2_0/{M-S1-B,M-S1,M-S1_STDLIB_STATUS}.md` | Name the target (V12) | immutable records, untouched |
| `test-stdlib-freeze` in `.PHONY` (test.mk:9) | Stays — alias target is phony | keep |

## Non-Goals

- No renaming/moving of `.stdlib-golden/` (90 committed files; CI-proven).
- No new freeze framework, no Go code, no new make variables.
- No expansion of the gate-wiring test's prefix classes to cover `test-*` (deliberately
  narrow per its own comment, gate_wiring_test.go:22).
- No `STDLIB_DIR` env override (deferred with reason, M2).

## Axiom / Language-Surface Note

No language semantics, syntax, stdlib *content*, or `ailang` binary behavior changes —
build tooling only. The 12-axiom scoring matrix does not apply; no `ailang check`-able
language claims are made in this doc (the only CLI claims are about flag parsing, V9).

## Related Documents

- [M-S1-B.md](../../implemented/v0_2_0/M-S1-B.md) — v0.2.0 acceptance of the original target (historical).
- Commit `80019d4e0` — "fix(stdlib-gate): revive verify-stdlib — dead since v0.0.12" (the tools-side revival this design completes on the make side).
- `make/examples.mk:141-147` comment block — the prior art on "a green gate is not evidence; a gate that goes red on a real change is."
- [v1-mission.md](../../v1-mission.md) queue row `m-stdlib-freeze-gate-path-mismatch` (iteration 282 measurements).
- No planned/ doc overlaps this topic (duplicate gate: closest matches are the implemented v0_2_0 M-S1 family above — historical, not duplicative).

## Success Criteria

- [ ] `make test-stdlib-freeze` exits 0 on a clean tree and its trace shows `tools/verify-stdlib.sh` (not a second implementation)
- [ ] `STDLIB`/`FREEZE_DIR`/`TOOLS` absent from `Makefile` and from `make -pn` output
- [ ] Mutation rows 1-3 red via `make test-stdlib-freeze`; row 8 control green
- [ ] `make verify-stdlib-selftest` passes with the two new arms, and mutation row 7 reds it
- [ ] **Restore-on-failure proven** (mutation row 9): after a FAILING selftest run the worktree is byte-identical — `git status --porcelain` empty AND `.stdlib-golden/option.sha256`'s sha256 unchanged
- [ ] `go test ./internal/cihygiene/` green (no new exemption entries needed)
- [ ] Full `make ci` unaffected except the strengthened selftest
- [ ] CHANGELOG updated

## Verification Log

All commands run 2026-08-26 in `/Users/voightkampff/.ailang-driver-pin/v1` at HEAD
`da96b98a5` (detached), clean tree. Outputs elided to the load-bearing lines.

| # | Claim | Command | Observed |
|---|---|---|---|
| V1 | `goldens/stdlib` ABSENT; `.stdlib-golden` EXISTS, populated (control in same call) | `test -d goldens/stdlib && echo EXISTS \|\| echo ABSENT; ls .stdlib-golden/ \| head` | `goldens/stdlib ABSENT`; listing shows `ai.json ai.sha256 ... option.sha256 ... zip.sha256` (92 entries incl. `.`/`..`) |
| V2 | `test-stdlib-freeze` aborts rc=2 in prerequisite resolution | `make test-stdlib-freeze; echo rc=$?` | ``make: *** No rule to make target `goldens/stdlib/option.sha256', needed by `test-stdlib-freeze'.  Stop.`` then `rc=2` (matches controller's two-arm measurement: rc=2 on both stale-PATH and fresh ldflags-stamped binaries) |
| V3 | Control: surrounding stdlib tooling works — `verify-stdlib` green over all 45 modules | `make verify-stdlib >/tmp/v.out 2>&1; echo rc=$?; tail -1 /tmp/v.out` | `rc=0`; `✓ All 45 stdlib interfaces stable` |
| V4 | Goldens are COMMITTED (CI can see them) | `git ls-files .stdlib-golden \| wc -l` | `90` (45 x json+sha256; first entries `ai.json`, `ai.sha256`) |
| V5 | `goldens/stdlib` has no writer/reader outside Makefile:59 + dead recipe; the live gate wiring is examples.mk + tools + ci.yml | `grep -rn 'FREEZE_DIR\|test-stdlib-freeze\|verify-stdlib\|freeze-stdlib\|stdlib-golden' Makefile make/ tools/ .github/ scripts/` | Hits ONLY: Makefile:59,:66; make/test.mk:9,:184-193; make/examples.mk:133-154; make/ci.mk:11; tools/{freeze,verify}-stdlib.sh + lib:9; ci.yml:142,:145. No other `goldens/stdlib` reference exists |
| V6 | `test-stdlib-freeze` wired into NO workflow (control: verify-stdlib IS, same file) | `grep -n 'test-stdlib-freeze\|verify-stdlib' .github/workflows/ci.yml` | Zero `test-stdlib-freeze` hits; `142: run: make verify-stdlib`, `145: run: make verify-stdlib-selftest` (controller's independent control: `verify-examples` appears 5x in ci.yml) |
| V7 | Live gate + red-on-change selftest run in CI and `make ci` | `grep -n 'verify-stdlib' make/ci.mk .github/workflows/ci.yml` | `ci.mk:11` lists both in `ci:`; ci.yml:142,:145 as above |
| V8 | Sole consumer of `FREEZE_DIR`/`STDLIB`/`TOOLS` is the dead recipe; no env consumers of the exports (with positive control) | `make -pn \| grep -n 'goldens/stdlib'` ; `grep -rn '\$(STDLIB)\|\$(TOOLS)' Makefile make/` ; `grep -rn 'FREEZE_DIR\|\$STDLIB\b\|\${STDLIB}\|\$TOOLS\b\|\${TOOLS}' examples/ scripts/ tools/ .github/` ; control `grep -c STDLIB Makefile` | make -pn: FREEZE_DIR + the 5 prereq stubs + test-stdlib-freeze only. `$(STDLIB)` only test.mk:188; `$(TOOLS)` only test.mk:191. Env grep: `NO HITS`; control: `2` |
| V9 | Dead recipe's CLI form is invalid on the current binary (rot #2) | `./bin/ailang iface --module "std/option" --json; echo rc=$?` | `flag provided but not defined: -module` + usage; `rc=2`. (Binary present: `bin/ailang`, built Aug 26) |
| V10 | Gate-wiring test's prefix classes EXCLUDE `test-stdlib-freeze`; alias precedent recorded; verify-stdlib wiring is itself asserted | read `internal/cihygiene/gate_wiring_test.go` | `:130` `check-`/`test-check-` prefixes; `:134` `verify-` + `fmt-check-ail`; `:39` verify-lowering alias exemption ("adds an alias, not coverage"); `:191` instrument-fails unless `verify-stdlib` found among wired targets |
| V11 | `check-golden-drift` does NOT read `.stdlib-golden` | `sed -n '/^check-golden-drift/,/^$/p' make/test.mk` | recipe diffs only `internal/parser/testdata/parser/` |
| V12 | Broken name still recommended by skill docs; historical docs reference it | `grep -rn 'test-stdlib-freeze' . \| grep -v .git \| grep -v ^./make/` | `.claude/skills/sprint-executor/resources/developer_tools.md:67,142` + `.agents/` copy; `implemented/v0_2_0/{M-S1-B.md:100 (via :90-110 read), M-S1.md:107, M-S1_STDLIB_STATUS.md:48}`; mission docs |
| V13 | Anti-vacuity floors exist at cited lines (positive-existence) | `grep -n 'refusing to report success\|no golden file\|produced no output\|interface changed' tools/verify-stdlib.sh tools/stdlib-iface-lib.sh` | verify-stdlib.sh:24 (empty enumeration), :33 (UNCOVERED), :54 (mismatch); stdlib-iface-lib.sh:54 (empty output). Module list is filesystem-derived (`stdlib_modules`, lib:32-34) |
| V14 | `freeze-stdlib.sh` sources the lib (queue-row attribution corrected); lib defines `GOLDEN_DIR=".stdlib-golden"` at :9 | `cat tools/freeze-stdlib.sh tools/stdlib-iface-lib.sh` | `source "$SCRIPT_DIR/stdlib-iface-lib.sh"` in freeze-stdlib.sh (~:13); `GOLDEN_DIR=".stdlib-golden"` at lib:9; freezer writes `$GOLDEN_DIR/$module.{json,sha256}` |
| V15 | Tools-side revival history | `git log --oneline -5 -- tools/stdlib-iface-lib.sh tools/verify-stdlib.sh` | `80019d4e0 fix(stdlib-gate): revive verify-stdlib — dead since v0.0.12, and the loader bug behind it` |
| V16 | `verify-stdlib-selftest` currently proves only the mismatch branch (basis for M2) | `sed -n '147,169p' make/examples.mk` | single canary-export arm: asserts non-zero exit, module named, diff shows `selftestCanary`. No missing-golden or alias arm |
| V17 | No planned design doc already covers this (duplicate gate; control: search DOES find the implemented v0_2_0 family) | `grep -rln 'stdlib-golden\|verify-stdlib' design_docs/planned/ design_docs/implemented/` | planned/: no topic-overlapping doc (only mission-log/status files reference the queue row); implemented/: v0_2_0 M-S1 family + unrelated sprint plans |
| V18 | No commit on ANY ref ever ADDED or touched a `goldens/stdlib` path | `git log --all --oneline --diff-filter=A -- 'goldens/stdlib/*'` ; `git log --all --oneline -- 'goldens/stdlib'` | 0 commits from both (measured this session by the quorum controller, same HEAD) |
| V19 | Exhaustive: no tree object in the entire history contains a `goldens/stdlib/` path | scan of `git rev-list --all` (400 commits), `git ls-tree -r --name-only` per commit, grepped for `^goldens/stdlib/` | NO hit in any tree object (measured this session by the quorum controller, same HEAD) |
| V20 | Controls — the SAME instrument fires on in-scope positives | `git log --all --oneline --diff-filter=A -- '.stdlib-golden/*'` ; log query for any `goldens` path | `.stdlib-golden/*` adds → 2 commits; any `goldens` path → 5 log entries. The empty V18/V19 results are measurements, not blind spots |
| V21 | Existing selftest EXIT trap restores `std/option.ail` ONLY — it has no knowledge of `.stdlib-golden/` (basis for M2's independent-backup requirement) | `sed -n '149,153p' make/examples.mk` | `cp std/option.ail "$SELFTEST_BK"; trap 'cp "$SELFTEST_BK" std/option.ail; rm -f "$SELFTEST_BK" "$SELFTEST_OUT"' EXIT` — backs up and restores std/option.ail, nothing else |

V18–V20 settle the history question: `goldens/stdlib` has **never existed in this
repository's history, on any ref**. The v0.2.0 M-S1-B.md recipe (`mkdir -p goldens/stdlib`,
:90-93) would have created it only in a developer's local working tree — never in a commit.

## Quorum Verification Log

**Round 1 (design-quorum, iteration 283): BLOCKED**, two objections. Direction (delegate
to verify-stdlib, delete the fossil) was not disputed by either reviewer.

| Objection | Reviewer | Resolution |
|---|---|---|
| Problem Statement asserted "goldens/stdlib never held the goldens" while the V-log flagged exactly that as unverified | gpt5-6-sol | **REFUTED BY MEASUREMENT** — the assumption was correct. Controller ran the full-history scan at HEAD `da96b98a5`: zero adds, zero log entries, zero hits across all 400 tree objects on all refs, with two in-scope positive controls proving the instrument fires (V18–V20). Caveat deleted; claim promoted to VERIFIED with cited rows. |
| M2's missing-golden arm relied on "the existing EXIT trap" to restore the moved golden — but that trap restores `std/option.ail` only (V21); as written, M2 would permanently delete `.stdlib-golden/option.sha256` | gemini-3-1-pro | **ACCEPTED — M2 REDESIGNED.** The arm now runs in its own recipe-line shell with its OWN mktemp backup and OWN EXIT trap restoring exactly what it moved, covering every failure path (exact recipe text shown in M2). The two arms have INDEPENDENT backup/restore; no shared trap (a second `trap EXIT` in one shell would replace the first). New acceptance criterion + mutation row 9 prove restore-on-FAILURE: after a failing selftest, `git status --porcelain` empty and the golden's sha256 unchanged. |

## Round-3 controller measurements (narrow-refinement carve-out)

Round 2 returned BLOCKED on two objections. **Neither disputed the design DIRECTION** (delegation to
`verify-stdlib` + deleting the fossil was uncontested in both rounds); both asked for premise
verification. Per the mission skill's rule 3f the controller MEASURED them rather than forwarding
them, and per the narrow-refinement carve-out applies the reviewers' own requested fix — the
verification rows below — instead of spending a third authoring run.

**Surface log (rule from iteration 257).** R1: history premise (`gpt5-6-sol`) + selftest trap scope
(`gemini-3-1-pro`). R2: exported-variable consumer scope (`gpt5-6-sol`) + `verify-stdlib`
prerequisites and V-log completeness (`gemini-3-1-pro`). The surfaces differ per round and do **not**
localise onto one consumer, so this is not a decomposition signal; all four objections are
evidence-completeness asks against a design neither reviewer has challenged.

| Row | Claim under test | Command | Observed | Verdict |
|---|---|---|---|---|
| V22 | `STDLIB`/`FREEZE_DIR`/`TOOLS` are exported (Makefile:66), so an env consumer could exist outside make syntax | `sed -n '66p' Makefile` | `export STDLIB FREEZE_DIR TOOLS` | Objection's premise TRUE — audit widened below |
| V23 | No Go code reads them from the environment | `grep -rIn 'Getenv("<V>")' --include='*.go' .` | `FREEZE_DIR` **0**, `STDLIB` **0**, `TOOLS` **0** | REFUTED (no consumers) |
| V24 | Control for V23 — the instrument can see a real `Getenv` | `grep -rIn 'Getenv("AILANG_SEED")' --include='*.go' .` | **1** | Control FIRED |
| V25 | No shell consumer outside makefiles | `git grep -InE '\$\{?<V>\}?([^A-Za-z0-9_]\|$)' -- ':!*.mk' ':!Makefile'` | bare `$STDLIB` **0**, `$FREEZE_DIR` **0**, `$TOOLS` **0** | REFUTED (no consumers) |
| V26 | The 2 apparent `$STDLIB` hits are a DIFFERENT, locally-defined variable | `git grep -In '\$STDLIB' -- ':!*.mk' ':!Makefile'` | `tools/stdlib-iface-lib.sh:33` and `tools/verify-stdlib.sh:24`, both `$STDLIB_DIR`; defined locally at `tools/stdlib-iface-lib.sh:8` `STDLIB_DIR="std"` | Substring false positive — not a consumer |
| V27 | Control for V25/V26 — the `git grep` instrument fires | `git grep -In '$GOCACHE'` → **5**; `STDLIB_DIR` refs → **2** | non-zero both | Control FIRED |
| V28 | `verify-stdlib` really depends on `build` (make's own view, not a file grep) | `make -pn \| grep -E '^verify-stdlib:'` | `verify-stdlib: build` | CONFIRMED |
| V29 | `verify-stdlib-selftest` likewise | `make -pn \| grep -E '^verify-stdlib-selftest:'` | `verify-stdlib-selftest: build` | CONFIRMED |
| V30 | The dead target's prerequisites are the absent goldens | `make -pn \| grep -E '^test-stdlib-freeze:'` | `test-stdlib-freeze: goldens/stdlib/option.sha256 …` (5 entries) | CONFIRMED |
| V31 | `STDLIB` hardcodes 5 modules while the live gate derives 45 | `make -pn \| grep '^STDLIB :='` ; `ls .stdlib-golden/*.sha256 \| wc -l` | `std/option.ail std/result.ail std/list.ail std/string.ail std/io.ail` (5); goldens **45** | CONFIRMED |
| V32 | The repo already documents that the 5-module gate is worse than none | `sed -n '25,35p' tools/stdlib-iface-lib.sh` | *"The old scripts hardcoded MODULES=… 5 of 45. A freeze gate covering 11% of the surface is worse than none, because it reads as coverage."* | CONFIRMED — corroborates the design |
| V33 | The delegated script already has an empty-enumeration floor | `sed -n '22,24p' tools/verify-stdlib.sh` | `if [ -z "$MODULES" ]; then … "✗ no modules found under $STDLIB_DIR/ — refusing to report success"` | CONFIRMED |
| V34 | The goldens are CURRENT — the fix makes the gate green, not red | for each of the 5 STDLIB modules: `ailang iface std/<m> \| shasum -a 256` vs `.stdlib-golden/<m>.sha256` | **5 of 5 MATCH**; negative control (`io` vs `option`'s golden) DIFFERS | CONFIRMED |
| V35 | Baselines for the acceptance gates on pristine `origin/dev` | each run with rc captured without a pipe | `verify-stdlib` **0**, `verify-stdlib-selftest` **0**, `check-file-sizes` **0**, `go build ./internal/...` **0**, `go build ./...` **1 (already red at base — excluded from the gate list)** | CONFIRMED |

**Disposition:** both round-2 objections REFUTED by measurement; deletion safety established
repo-wide including the environment channel. Routed to sprint-planner under the carve-out.
