# Sprint Plan: M-STDLIB-FREEZE-GATE — Retire the Rotted Freeze Duplicate, Alias the Accepted Name to the Live Gate

**Design doc**: [m-stdlib-freeze-gate-path-mismatch.md](m-stdlib-freeze-gate-path-mismatch.md)
**Sprint ID**: `M-STDLIB-FREEZE-GATE`
**Target**: v0.33.4
**Estimated**: 0.5 day (3 milestones, 1 commit each)
**Base**: `da96b98a5` (detached pin), tree clean apart from `design_docs/planned/v0_33_4/`
**Quorum**: design doc cleared 2 rounds, routed under the narrow-refinement carve-out (Round-3 controller measurements V22–V35).

---

## Summary

`make test-stdlib-freeze` — the stdlib **interface** freeze gate accepted in v0.2.0 — has been
unrunnable for the whole life of the repository. It aborts at **rc=2 during prerequisite
resolution** (`No rule to make target 'goldens/stdlib/option.sha256'`), so its recipe body has
never executed. `goldens/stdlib/` has never existed in any commit on any ref. The live gate is
`verify-stdlib` (`tools/verify-stdlib.sh` over `.stdlib-golden/`, 45 modules, CI-wired at
`ci.yml:142` and `:145`).

This sprint **deduplicates**: the historical name becomes a delegation alias of the live gate, the
fossil recipe and its three orphan variables are deleted, and two permanent arms are added to
`verify-stdlib-selftest` so both the newly-reachable branch (missing golden) and the alias itself
stay red-provable in CI.

### THIS IS NOT THE `fmt` GATE

Decision **D-39** (`design_docs/v1-mission.md:106`) rules: *"do NOT wire or freeze the `fmt` gate
until [the width limit] lands and the second `fmt --write` pass has run, or the collapsed form gets
frozen as canonical by the gate rather than by a ruling."* That is `fmt-check-ail`, which stays in
`notWiredIntoCIVerify` untouched. This sprint touches the stdlib **interface** freeze gate only.
**Do not wire, unwire, freeze, or edit anything `fmt`-related in any milestone.** If a milestone
appears to require an `fmt` change, stop and report instead.

---

## Executor Protocol (READ FIRST)

1. **NO GIT WRITE OPERATIONS.** The executor runs under a sandbox and must NOT run `git add`,
   `git commit`, `git checkout`, `git restore`, `git stash`, `git reset`, `git clean`, `git mv`,
   `git rm`, `git branch`, or `git push`. **The controller commits.** Read-only git
   (`git status`, `git diff`, `git log`, `git ls-files`, `git grep`) is fine and is used by the
   acceptance criteria.
2. **Because git cannot restore anything, every mutation drill backs up with `cp` first and
   restores with `cp`/`mv`.** A drill that relies on `git checkout -- <file>` is not executable
   here and is a plan violation.
3. **Cumulative snapshots.** After each milestone `k` completes and its acceptance criteria are
   green, copy the current state of every file the sprint has touched *so far* into `.snap/M<k>/`,
   preserving relative paths:
   ```
   .snap/M1/{Makefile, make/test.mk}
   .snap/M2/{Makefile, make/test.mk, make/examples.mk}
   .snap/M3/{Makefile, make/test.mk, make/examples.mk, changelogs/v0.32-current.md,
             .claude/skills/sprint-executor/resources/developer_tools.md,
             .agents/skills/sprint-executor/resources/developer_tools.md}
   ```
   `.snap/` is **not** in `.gitignore`, so it will appear in `git status --porcelain`. Every
   acceptance criterion below that inspects `git status` is therefore **path-scoped**
   (`git status --porcelain -- <paths>`) or **before/after-differenced** — never "must be empty"
   against the whole tree. Tell the controller to delete `.snap/` before committing.
4. **One commit per milestone**, in order. Milestones are independently committable: M1 alone leaves
   the tree green; M2 alone leaves it green; M3 is docs.
5. **Never run `make freeze-stdlib`.** It rewrites all 45 goldens. Nothing in this sprint changes a
   stdlib interface, so a re-freeze can only launder a real regression. V34 measured all 5 legacy
   STDLIB modules already hash-match their goldens — **this fix makes the gate GREEN, not red.**

---

## Baselines (controller-measured on a pristine tree at `da96b98a5`, re-confirmed by the planner)

| Command | rc at base | Use |
|---|---|---|
| `make test-stdlib-freeze` | **2** (`No rule to make target 'goldens/stdlib/option.sha256'`) | the defect |
| `make verify-stdlib` | 0 (`✓ All 45 stdlib interfaces stable`) | acceptance gate |
| `make verify-stdlib-selftest` | 0 | acceptance gate |
| `make check-file-sizes` | 0 | acceptance gate |
| `make check-changelog` | 0 | acceptance gate (M3) |
| `make check-skills` | 0 | acceptance gate (M3) |
| `go build ./internal/...` | 0 | acceptance gate |
| `go test ./internal/cihygiene/` | 0 (`ok`) | acceptance gate |
| `go build ./...` | **1** — `cmd/wasm` and `gen/main` have no native `main` | **ALREADY RED AT BASE. MUST NOT be used as an acceptance gate.** |

Instrument shapes confirmed at base, used by the criteria below:

| Query | Value at base | Meaning |
|---|---|---|
| `make -pn 2>/dev/null \| grep -c 'goldens/stdlib'` | **7** | make's own view still carries the dead path |
| `make -pn 2>/dev/null \| grep -cE '^(STDLIB\|FREEZE_DIR\|TOOLS) '` | **3** | the three orphan variables |
| `make -pn 2>/dev/null \| grep -cE '^BINARY '` | **1** | **positive control** for the line above |
| `make -pn 2>/dev/null \| grep -E '^test-stdlib-freeze:'` | `test-stdlib-freeze: goldens/stdlib/option.sha256 …` (5 prereqs) | the fossil's prerequisites |
| `make -pn 2>/dev/null \| grep -E '^verify-stdlib:'` | `verify-stdlib: build` | the alias inherits fresh-binary resolution |
| `bash -c 'source tools/stdlib-iface-lib.sh; stdlib_modules \| wc -l'` | **45** | filesystem-derived module count |
| `ls .stdlib-golden/*.sha256 \| wc -l` | **45** | goldens present and committed |

`make -pn` prints **recipe text as well as the rule database** (confirmed: `selftestCanary`
appears in it twice). Two consequences the executor must respect:

* **M1's replacement comment must not contain the string `goldens/stdlib`**, or acceptance
  criterion M1-AC3 (`grep -c 'goldens/stdlib'` → 0) will fail on the comment. Refer to it as
  "the never-existing golden directory" if you need to.
* This is also why the mutation drills can assert "the mutation landed" against make's own view
  rather than against file bytes.

---

## Proposed Milestones

### Milestone 1 — Delegate the name, delete the fossil

**Files**: `make/test.mk`, `Makefile`. **Est. LOC**: −22 / +5.

**Changes**

1. `make/test.mk:183-201` — delete the `# Stdlib freeze` comment, the 5 golden prerequisites and
   the entire recipe body; replace with a bare delegation alias:
   ```make
   # Stdlib freeze — historical name, kept because v0.2.0 acceptance docs and the
   # sprint-executor skill reference it. The live gate is verify-stdlib
   # (tools/verify-stdlib.sh over .stdlib-golden/); do not grow a second implementation.
   test-stdlib-freeze: verify-stdlib ## Verify std/ interfaces haven't changed (alias of verify-stdlib)
   ```
   No recipe body. `verify-stdlib: build` (V28), so the alias inherits fresh-binary resolution.
2. `make/test.mk:9` — `test-stdlib-freeze` stays in `.PHONY`. **Do not remove it.**
3. `Makefile:58,59,60` — delete `STDLIB`, `FREEZE_DIR`, `TOOLS`.
4. `Makefile:66` — delete the whole line `export STDLIB FREEZE_DIR TOOLS`.

Deletion safety is measured, not linter-driven (`.claude/rules/coding-standards.md`): 4 `$(FREEZE_DIR)`
consumers, 1 `$(STDLIB)`, 1 `$(TOOLS)` — **all inside the recipe being deleted**; zero Go `Getenv`
consumers (V23, control V24 fired); zero bare-shell consumers outside makefiles (V25/V26, control
V27 fired — the 2 apparent `$STDLIB` hits are `$STDLIB_DIR`, a *different* variable defined locally
at `tools/stdlib-iface-lib.sh:8`).

**Acceptance criteria** (each is a command that can fail; run each with rc captured *without* a pipe
where the rc is the assertion):

| # | Command | Expected |
|---|---|---|
| M1-AC1 | `make test-stdlib-freeze; echo rc=$?` | **rc=0** (was rc=2) and output ends `✓ All 45 stdlib interfaces stable` |
| M1-AC2 | `make -pn 2>/dev/null \| grep -qE '^test-stdlib-freeze:[[:space:]]*verify-stdlib[[:space:]]*$'` | **rc=0** — make's own view: the sole prerequisite is `verify-stdlib` |
| M1-AC3 | `test "$(make -pn 2>/dev/null \| grep -c 'goldens/stdlib')" -eq 0` | **rc=0** (was 7) |
| M1-AC4 | `test "$(make -pn 2>/dev/null \| grep -cE '^(STDLIB\|FREEZE_DIR\|TOOLS) ')" -eq 0` | **rc=0** (was 3) |
| M1-AC5 | `test "$(make -pn 2>/dev/null \| grep -cE '^BINARY ')" -eq 1` | **rc=0** — positive control proving M1-AC4's instrument still sees variables |
| M1-AC6 | `make -n test-stdlib-freeze 2>&1 \| grep -q 'tools/verify-stdlib.sh'` | **rc=0** — the trace reaches the ONE implementation |
| M1-AC7 | `test "$(make -n test-stdlib-freeze 2>&1 \| grep -c 'shasum\|iface --module')" -eq 0` | **rc=0** — no second implementation survived |
| M1-AC8 | `make verify-stdlib; echo rc=$?` | **rc=0** |
| M1-AC9 | `make verify-stdlib-selftest; echo rc=$?` | **rc=0** |
| M1-AC10 | `go test ./internal/cihygiene/; echo rc=$?` | **rc=0** — no new exemption entry needed (V10: `test-*` is outside both asserted prefix classes) |
| M1-AC11 | `make check-file-sizes; echo rc=$?` and `go build ./internal/...; echo rc=$?` | **rc=0** both |
| M1-AC12 | `git status --porcelain -- .stdlib-golden std; echo "[$(git status --porcelain -- .stdlib-golden std)]"` | **empty** — M1 must not touch a golden or a stdlib source |

**Mutation drill M1** — doc rows 1–5 and 8. Backup/restore with `cp`, never git.

> Setup once: `BK=$(mktemp -d)`; before each row, `cp` the file(s) you are about to mutate into
> `$BK`. After each row, restore by `cp` and re-run row 8 (the control) to prove the tree is back.

| Row | Mutation | Assert the mutation LANDED (effect, not bytes) | Mutant still builds/parses | Named arm that MUST go red | Expected |
|---|---|---|---|---|---|
| 1 | `printf '%064d\n' 0 > .stdlib-golden/option.sha256` | `test "$(./bin/ailang iface std/option \| shasum -a 256 \| awk '{print $1}')" != "$(cat .stdlib-golden/option.sha256)"` → rc=0. **Control**: the same comparison for `result` with `=` → rc=0 (unmutated golden still matches) | `make -n test-stdlib-freeze >/dev/null 2>&1` rc=0 | `make test-stdlib-freeze` (floor 6, reached via the alias) | **rc=1**, output contains `option: interface changed!` and a `Diff (golden → current):` block |
| 2 | `rm .stdlib-golden/option.sha256` | `test -e .stdlib-golden/option.sha256` → **rc=1**. **Control**: `test -e .stdlib-golden/result.sha256` → rc=0 | `make -n test-stdlib-freeze >/dev/null 2>&1` rc=0 | `make test-stdlib-freeze` (floor 2) | **rc=1**, output contains `option: no golden file — module is UNCOVERED` |
| 3 | `mv .stdlib-golden "$BK/gold"` | `test -d .stdlib-golden` → **rc=1** | `make -n test-stdlib-freeze >/dev/null 2>&1` rc=0 | `make test-stdlib-freeze` (floor 1) | **rc=1**, and `grep -c 'no golden file' <out>` equals **45** — enumerated, not a single abort |
| 4 | `cp tools/stdlib-iface-lib.sh "$BK/"; sed -i '' 's/^STDLIB_DIR="std"$/STDLIB_DIR="std-nonexistent"/' tools/stdlib-iface-lib.sh` | `bash -c 'source tools/stdlib-iface-lib.sh; stdlib_modules 2>/dev/null \| wc -l'` prints **0**. **Control**: the same command at base printed **45** | `bash -n tools/stdlib-iface-lib.sh` rc=0 | `tools/verify-stdlib.sh` (floor 3 — the loop-runs-zero-times trap) | **rc=1**, stderr contains `no modules found under std-nonexistent/ — refusing to report success` |
| 5 | `cp tools/stdlib-iface-lib.sh "$BK/"; sed -i '' 's#^    AILANG="./bin/ailang"$#    AILANG="/usr/bin/true"#' tools/stdlib-iface-lib.sh` | `bash -c 'source tools/stdlib-iface-lib.sh; resolve_ailang; echo "$AILANG"'` prints `/usr/bin/true` | `bash -n tools/stdlib-iface-lib.sh` rc=0 | `tools/verify-stdlib.sh` (floor 4 — rc=0 with empty stdout) | **rc=1**, and `grep -c 'produced no output' <out>` equals **45** |
| 8 | **CONTROL, must stay GREEN** — clean tree, no mutation | `git status --porcelain -- .stdlib-golden std tools` is **empty** | n/a | `make test-stdlib-freeze` **and** `make verify-stdlib-selftest` | **rc=0 both**; `✓ All 45 stdlib interfaces stable` |

Row 8 is the known-positive control. **Run it before row 1 and again after every restore.** If row 8
is ever red, rows 1–5 prove nothing and the drill must stop and be reported.

---

### Milestone 2 — Two permanent selftest arms (missing-golden + alias-integrity)

**File**: `make/examples.mk` (target `verify-stdlib-selftest`, currently ending at `:169`).
**Est. LOC**: +22.

**Changes** — append two arms to the *same* target, each as its **own `@(...)` recipe line, i.e. its
own shell**. The existing canary arm and its EXIT trap are **not edited**.

**Why isolated shells is a hard requirement, not a style preference** (V21, quorum round 1): the
existing trap restores `std/option.ail` ONLY and knows nothing about `.stdlib-golden/`; and
`trap ... EXIT` is per-shell, so a second `trap` in the same shell **replaces** the first. An arm
that leans on the existing trap would permanently delete the developer's
`.stdlib-golden/option.sha256`.

1. **Missing-golden arm** — use the exact recipe text in the design doc (§M2, lines 133–146): its own
   `mktemp` backup of `.stdlib-golden/option.sha256`, its own EXIT trap restoring exactly that file,
   `rm` the golden, require `tools/verify-stdlib.sh` to exit non-zero, and require the output to
   contain `option: no golden file` (stable ASCII prefix of `option: no golden file — module is
   UNCOVERED`, `verify-stdlib.sh:33`). Every `exit 1` inside the arm fires that arm's own trap.
2. **Alias-integrity arm** — read-only, no backup/trap needed. The design doc proposes
   `make -n test-stdlib-freeze | grep -q 'verify-stdlib'`. **Strengthen it**: `-n` output would also
   match `verify-stdlib-selftest` or any other `verify-stdlib*` target, so assert make's rule
   database instead, then additionally assert the trace:
   ```make
   	@echo "Gate self-test (alias-integrity arm): the historical name must reach the live gate..."
   	@$(MAKE) -pn 2>/dev/null | grep -qE '^test-stdlib-freeze:[[:space:]]*verify-stdlib[[:space:]]*$$' || { \
   		echo "❌ SELF-TEST FAIL: test-stdlib-freeze no longer delegates to verify-stdlib"; exit 1; }
   	@$(MAKE) -n test-stdlib-freeze 2>&1 | grep -q 'tools/verify-stdlib.sh' || { \
   		echo "❌ SELF-TEST FAIL: test-stdlib-freeze does not reach tools/verify-stdlib.sh"; exit 1; }
   	@echo "✓ alias intact: test-stdlib-freeze -> verify-stdlib -> tools/verify-stdlib.sh"
   ```
   (Recursive `$(MAKE)` inside a recipe is fine here — both invocations are `-n`, so nothing is
   built. Escape `$` as `$$` in the make recipe.)

**Ordering**: put the missing-golden arm *after* the canary arm (so a canary failure still aborts
before any golden is touched) and the alias arm last.

**Acceptance criteria**

| # | Command | Expected |
|---|---|---|
| M2-AC1 | `make verify-stdlib-selftest; echo rc=$?` | **rc=0** |
| M2-AC2 | `make verify-stdlib-selftest 2>&1 \| grep -c '^✓\|^✅'` | **≥ 6** — three canary `✓`s + the canary `✅` + the missing-golden `✓` + the alias `✓` |
| M2-AC3 | `make verify-stdlib-selftest 2>&1 \| grep -q 'missing golden -> non-zero + module reported UNCOVERED'` | **rc=0** |
| M2-AC4 | `make verify-stdlib-selftest 2>&1 \| grep -q 'alias intact'` | **rc=0** |
| M2-AC5 | run `make verify-stdlib-selftest`, then `git status --porcelain -- .stdlib-golden std` | **empty** — restore-safety on the SUCCESS path |
| M2-AC6 | `shasum -a 256 .stdlib-golden/option.sha256 > /tmp/g.pre; make verify-stdlib-selftest; shasum -a 256 .stdlib-golden/option.sha256 > /tmp/g.post; diff /tmp/g.pre /tmp/g.post` | **rc=0** |
| M2-AC7 | `test "$(ls .stdlib-golden/*.sha256 \| wc -l \| tr -d ' ')" -eq 45` | **rc=0** — no golden was consumed |
| M2-AC8 | `make check-file-sizes; echo rc=$?` · `go test ./internal/cihygiene/; echo rc=$?` · `make verify-stdlib; echo rc=$?` · `go build ./internal/...; echo rc=$?` | **rc=0** all four |
| M2-AC9 | `ls "${TMPDIR:-/tmp}"/ailang-selftest-* 2>/dev/null \| wc -l` after a full run | **0** — no leaked temp files (`make check-tmpfile-hygiene` also stays rc=0) |

**Mutation drill M2** — doc rows 6, 7, 9 plus one planner-added mutant for the canary arm.

| Row | Mutation | Assert the mutation LANDED (effect, not bytes) | Mutant still builds/parses | Named arm that MUST go red | Expected |
|---|---|---|---|---|---|
| 6 | none — the canary arm injects and reverts its own `selftestCanary` export | `make -pn 2>/dev/null \| grep -c 'selftestCanary'` equals **2** (the recipe carries the injection and the assertion) | n/a | canary arm of `make verify-stdlib-selftest` | **rc=0 for the selftest**, output contains `gate exited non-zero on the injected export`, `gate named the changed module`, and `gate showed the actual diff` |
| **6b** (planner addition — proves the canary arm's assertion has teeth) | `cp make/examples.mk "$BK/"` then change the canary arm's injected payload from the `export pure func selftestCanary…` line to a comment-only payload (`-- selftestCanary`), so the interface does **not** change | `make -pn 2>/dev/null \| grep -c 'export pure func selftestCanary'` goes **1 → 0** (make's own view of the recipe) | `make -n verify-stdlib >/dev/null 2>&1` rc=0 | canary arm | **rc=1**, output contains `SELF-TEST FAIL: gate exited 0 despite a new export in std/option` |
| 7 | `cp make/test.mk "$BK/"` then replace the alias line with `test-stdlib-freeze: ;` | `make -pn 2>/dev/null \| grep -cE '^test-stdlib-freeze:[[:space:]]*verify-stdlib[[:space:]]*$'` goes **1 → 0** — make's own view, not file bytes | `make -n verify-stdlib >/dev/null 2>&1` rc=0 (makefile still parses) | **alias-integrity arm** of `make verify-stdlib-selftest` | **rc=1**, output contains `test-stdlib-freeze no longer delegates to verify-stdlib` |
| 9 | `cp make/examples.mk "$BK/"` then change the missing-golden arm's expected pattern from `option: no golden file` to `ZZZ-NEVER-PRINTED`, forcing that arm to fail **while the golden is removed** | `make -pn 2>/dev/null \| grep -c 'ZZZ-NEVER-PRINTED'` goes **0 → 1** — make's own view of the recipe | `make -n verify-stdlib-selftest >/dev/null 2>&1` rc=0 | **missing-golden arm** | see the sequence below |

**Row 9 is the restore-on-FAILURE proof** demanded by quorum round 1. Run it exactly as:

```bash
shasum -a 256 .stdlib-golden/option.sha256 > /tmp/r9.pre
git status --porcelain -- .stdlib-golden std make tools Makefile > /tmp/r9.st.pre
make verify-stdlib-selftest; echo "selftest rc=$?"          # MUST be non-zero
shasum -a 256 .stdlib-golden/option.sha256 > /tmp/r9.post
git status --porcelain -- .stdlib-golden std make tools Makefile > /tmp/r9.st.post
diff /tmp/r9.pre /tmp/r9.post; echo "golden-sha diff rc=$?"  # MUST be 0
diff /tmp/r9.st.pre /tmp/r9.st.post; echo "status diff rc=$?" # MUST be 0
git status --porcelain -- .stdlib-golden std; echo "[$(git status --porcelain -- .stdlib-golden std)]"
```

**Required:** selftest **rc≠0**, golden-sha diff **rc=0**, status diff **rc=0**, and
`git status --porcelain -- .stdlib-golden std` **empty** — i.e. the arm's own EXIT trap restored
`.stdlib-golden/option.sha256` on its **failure** path. (The status comparison is path-scoped and
before/after-differenced because `.snap/` and the sprint's own in-progress edits are legitimately
present; a bare "must be empty over the whole tree" is unsatisfiable mid-sprint and would make this
criterion vacuous or permanently red.) Then restore `make/examples.mk` from `$BK` and re-run row 8
from the M1 drill — it must be green again.

---

### Milestone 3 — Changelog + skill-doc correction

**Files**: `changelogs/v0.32-current.md`, `.claude/skills/sprint-executor/resources/developer_tools.md`,
`.agents/skills/sprint-executor/resources/developer_tools.md`. **Est. LOC**: +18.

**Changes**

1. `changelogs/v0.32-current.md` under `## [Unreleased]` → `### Fixed`: an entry naming
   `make test-stdlib-freeze`, the rc=2 prerequisite abort, `goldens/stdlib` never having existed on
   any ref, the delegation to `verify-stdlib`/`.stdlib-golden/`, the deletion of `STDLIB`/`FREEZE_DIR`/
   `TOOLS`, and the two new selftest arms. **Do NOT touch root `CHANGELOG.md`** — `check-changelog`
   asserts it stays an index.
2. Both `developer_tools.md` copies, lines 67 and 142: annotate the target as an alias, e.g.
   `make test-stdlib-freeze    # Verify stdlib interfaces haven't changed (alias of verify-stdlib)`.
   The two files are **byte-identical today** — keep them so.
3. `design_docs/implemented/v0_2_0/*` are immutable historical records. **Do not edit them.**

**Acceptance criteria**

| # | Command | Expected |
|---|---|---|
| M3-AC1 | `make check-changelog; echo rc=$?` | **rc=0** |
| M3-AC2 | `make check-skills; echo rc=$?` | **rc=0** |
| M3-AC3 | `awk '/^## \[Unreleased\]/,/^## \[v/' changelogs/v0.32-current.md \| grep -q 'test-stdlib-freeze'` | **rc=0** — the entry is in the Unreleased section, not appended anywhere |
| M3-AC4 | `git diff --name-only -- CHANGELOG.md \| wc -l` → `0`; `git diff --name-only -- design_docs/implemented \| wc -l` → `0` | **rc=0** both — index and historical records untouched |
| M3-AC5 | `diff .claude/skills/sprint-executor/resources/developer_tools.md .agents/skills/sprint-executor/resources/developer_tools.md` | **rc=0** — the two copies stay identical |
| M3-AC6 | `grep -c 'alias of verify-stdlib' .claude/skills/sprint-executor/resources/developer_tools.md` | **≥ 1** (and the same for the `.agents/` copy) |
| M3-AC7 | **The doc's claim must be TRUE, not merely present**: for each target named on the stdlib lines of `developer_tools.md`, `make -pn 2>/dev/null \| grep -qE "^<target>:"` | **rc=0** for `test-stdlib-freeze` and `verify-stdlib` |
| M3-AC8 | `make -pn 2>/dev/null \| grep -qE '^test-stdlib-freeze:[[:space:]]*verify-stdlib[[:space:]]*$'` | **rc=0** — the alias the docs now describe still holds |
| M3-AC9 | `make verify-stdlib-selftest; echo rc=$?` and `make test-stdlib-freeze; echo rc=$?` | **rc=0** both — M3 did not regress M1/M2 |

**Mutation drill M3** — M3-AC7 is the drill: it is a *doc-truth* check, not a doc-presence check.
Mutant: temporarily add `make test-stdlib-frozen  # typo` to a scratch copy of the stdlib lines and
run M3-AC7's loop over it — it must report the missing target and exit non-zero. Restore the scratch
copy. (Run this against a copy under `$BK`, not the tracked file.)

---

## Explicitly Out of Scope

- **Anything `fmt`.** D-39 forbids wiring/freezing the `fmt` gate until the width limit lands.
  `fmt-check-ail` stays exempt in `internal/cihygiene/gate_wiring_test.go`. Do not touch it.
- **Adding a `test-stdlib-freeze` step to `.github/workflows/ci.yml`.** After M1 it is the same
  script as `verify-stdlib`; a second step would run it twice per job. Same reasoning as the
  recorded `verify-lowering` exemption (`gate_wiring_test.go:39`). The alias-integrity arm is what
  makes "the name reaches the gated path" CI-enforced.
- **Adding an exemption entry to `notWiredIntoCI`/`notWiredIntoCIVerify`.** `test-stdlib-freeze`
  matches neither asserted prefix class (`check-*`/`test-check-*` at `:130`, `verify-*` +
  `fmt-check-ail` at `:134`), so it trips nothing (V10). If `go test ./internal/cihygiene/` goes red,
  something else broke — do **not** answer it with an exemption.
- **Renaming or moving `.stdlib-golden/`** (90 committed files, CI-proven).
- **A `STDLIB_DIR` env override** — deferred with reason in the design doc (§M2): `freeze-stdlib.sh`
  sources the same lib, so an override becomes a footgun that silently re-freezes the wrong tree.
- **A permanent empty-enumeration arm** — proven by one-time mutation (M1 drill row 4) instead.
- **`go build ./...` as a gate** — red at base (`cmd/wasm`, `gen/main`).
- **`make ci` end-to-end as a gate** — `verify-examples-toplevel` is red at HEAD on
  `examples/ai_modes.ail` (recorded in `notWiredIntoCIVerify`), unrelated to this sprint.

## Sprint Success Criteria

- [ ] `make test-stdlib-freeze` rc=0 and its `-n` trace names `tools/verify-stdlib.sh` (M1-AC1, M1-AC6)
- [ ] `goldens/stdlib` and `STDLIB`/`FREEZE_DIR`/`TOOLS` gone from make's own view, with the `BINARY` control still firing (M1-AC3, M1-AC4, M1-AC5)
- [ ] M1 drill rows 1–5 all red with the named message; row 8 control green before and after every row
- [ ] `make verify-stdlib-selftest` rc=0 with both new arms reporting, and M2 drill rows 6b, 7, 9 all red on the correct named arm
- [ ] Restore-on-failure proven (row 9): golden sha unchanged and path-scoped `git status --porcelain` unchanged after a FAILING selftest
- [ ] `go test ./internal/cihygiene/` rc=0 with **no** new exemption entry
- [ ] `make verify-stdlib`, `make check-file-sizes`, `make check-changelog`, `make check-skills`, `go build ./internal/...` all rc=0
- [ ] CHANGELOG entry under `## [Unreleased]` in `changelogs/v0.32-current.md`; root `CHANGELOG.md` and `design_docs/implemented/` untouched
- [ ] Nothing `fmt`-related changed (D-39)
- [ ] `.snap/M1`, `.snap/M2`, `.snap/M3` present and cumulative; `.snap/` removed before the controller commits
