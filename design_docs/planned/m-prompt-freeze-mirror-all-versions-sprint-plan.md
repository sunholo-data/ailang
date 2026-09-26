# Sprint Plan — M-PROMPT-FREEZE-MIRROR-ALL-VERSIONS

**Design doc**: [`design_docs/planned/m-prompt-freeze-mirror-all-versions.md`](m-prompt-freeze-mirror-all-versions.md) (r3, 467 lines)
**Sprint ID**: `M-PROMPT-FREEZE-MIRROR-ALL-VERSIONS`
**Planner**: sprint-planner role, V1 mission iteration 297, lane `opus` (Agent-tool planner)
**Planned**: 2026-08-28
**Worktree**: `/Users/voightkampff/dev/sunholo-data/.wt-iter297` @ `3d80bf91005a3c57042e0d6358373c4b4f005a4a` (design doc committed; tree clean)
**Base**: origin/dev `0917693792d07c75ae0c4d233d2cc7e592fa9272`
**Platform of all measurements below**: darwin/arm64, `go version go1.26.6 darwin/arm64`, module `go 1.26.6`
**Executor lane**: `codex:gpt-5.6-sol` under `--sandbox workspace-write`
**Milestones**: 1 · **Estimate**: 1 day (≤ 6h focused)

---

## 0. What this sprint changes, in one paragraph

`make check-prompt-freeze` → `go run ./cmd/ailang prompt freeze --check` has three file-level
arms, all gated on `Frozen != nil`. The registry holds 59 entries, 58 frozen and exactly one
mutable — `v0.16.6`, which is also `active`, i.e. the prompt every agent reads. That one entry is
therefore unchecked at the file level. This sprint widens **mirror byte-agreement**,
**hash-vs-file-bytes**, **file existence** (as a named rc=1 violation, replacing an rc=2 abort) and
**hash-enforceability** to ALL entries, adds a mirror-only-registry-entry violation and a
`checked N registry entries` count observable, and widens the `--migrate` *validation* loop to
match. **Merge-base IMMUTABILITY deliberately stays frozen-only** — mutable means editable, and the
edit+hash-regen-in-one-commit workflow must stay green.

---

## 1. Measurement provenance — read this before trusting any number here

Three tiers appear in this plan and are labelled inline:

| Tier | Meaning |
|------|---------|
| **[P]** | **Planner-measured this session**, first-party, in the pristine worktree above. Command and rc recorded. |
| **[C]** | Controller-measured and handed to the planner; the planner independently re-derived it (result agrees). |
| **[D]** | **Designer-asserted only** (Verification Log V1–V22). Carried forward, NOT re-measured by the planner. Anything a milestone or AC rests on has been promoted to [P]. |

**Rule honored: a value transcribed is not a measurement.** Every acceptance criterion in §6 rests
on a [P] row. The [D] rows below are context, and no gate depends on one.

### 1.1 Gate baseline — the six commands, re-measured [P]

Run by the planner in `/Users/voightkampff/dev/sunholo-data/.wt-iter297` at `3d80bf910`, clean tree:

| # | Command | rc at base |
|---|---------|-----------|
| G1 | `go build ./cmd/ailang/...` | **0** |
| G2 | `go vet ./cmd/ailang/...` | **0** |
| G3 | `test -z "$(gofmt -l cmd/ailang)"` | **0** |
| G4 | `go test ./cmd/ailang/ -run 'Freeze\|Prompt' -count=1` | **0** (`ok … 2.367s`) |
| G5 | `go run ./cmd/ailang prompt freeze --check` | **0**, **stdout empty** (only the `Observatory: 429MB` stderr log line) |
| G6 | `make check-prompt-freeze` | **0** |

**`go build ./...` is DISQUALIFIED as a gate** — it is rc=1 at base (`cmd/wasm` has no native
`main`). It must not appear in any acceptance criterion, task, or CI-equivalence claim.

**`-skip` support probed [P]**: `go test ./cmd/ailang/ -run 'Freeze|Prompt' -skip
'^TestFreezeCheck_UnmodifiedTreeIsGreen$' -count=1` → rc=0. The inverse-arm criterion of §7 is
therefore expressible on this toolchain.

### 1.2 Structural facts re-derived by the planner [P]

- Registry shape: source `versions.json` **59** versions, **58** frozen, **1** mutable =
  `v0.16.6`; `.active` = `v0.16.6`; mirror registry also **59**; `prompts/*.md` = 62,
  `cmd/ailang/prompts/*.md` = 62. → the pristine count observable must read **`checked 59`**.
- `v0.16.6` hash is **fresh in both trees and both registries**:
  source registry = mirror registry = `2a868106…f804b67` = `shasum -a 256 prompts/v0.16.6.md` =
  `shasum -a 256 cmd/ailang/prompts/v0.16.6.md`. → AC1 is green *non-vacuously*; the widened arms
  do not turn the real repo red.
- `checkRegistries` call graph: **1 definition + 1 comment + 4 callers**, all `package main` in
  `cmd/ailang` — `prompt_freeze.go:69`, `prompt_freeze_check_git_test.go:134`,
  `prompt_freeze_test.go:125`, `prompt_freeze_test.go:138`. Agrees with [C] and with V20.
  **Both test files are in scope**; Go rejects a 2-variable assignment from 3 results, so the
  compiler enforces the update — a caller cannot silently take a stale zero.
- Imports: `prompt_freeze_core.go` already imports `crypto/sha256`, `encoding/hex`, `bytes`,
  `encoding/json`, `fmt`, `os`, `os/exec`, `path/filepath`, `strings`, `eval_harness`.
  It does **not** import `errors` or `io/fs`. `prompt_freeze_check_git.go` imports `bytes`,
  `encoding/json`, `fmt`, `os`, `os/exec`, `path/filepath`, `strings` — also **no** `errors`/`io/fs`.
  Both new imports are required in both files. (Relevant to mutation discipline: the `if false &&
  errors.Is(...)` form keeps them used.)
- `orderedRegistry` exposes `VersionKeys []string` and `Versions map[string]*registryEntry`
  (a **pointer** map, line 29) — the `source.Versions[id] == nil` sweep in Solution Design §1
  compiles; the identical deref already ships at `prompt_freeze_core.go:351`.
- Test harness: `newFreezeCheckRepo(t)` builds a synthetic git repo in `t.TempDir()` with **one**
  entry `v1.0.0` (`active` = `v1.0.0`), `git init` → commit → `update-ref refs/remotes/origin/dev`
  → `switch -c feature`. `writeFreezeCheckRegistries(t, root, sourceHash, mirrorHash, frozen bool)`
  — the `frozen bool` parameter exists, so mutable fixtures need no new scaffolding.
  `runFreezeCheck(t, root)` stubs `corpusScanner`, calls `checkRegistries` **in-process** and
  `t.Fatal`s on a non-nil error, then re-execs `os.Args[0] -test.run=^TestFreezeCheckHelperProcess$`
  with `AILANG_FREEZE_CHECK_HELPER=1` for the rc. `hasViolation(violations, parts...)` does
  substring matching after `filepath.ToSlash`.
  → **M-ADD arithmetic on this fixture: 1 source entry → `checked 1`; + one mirror-only entry →
  `checked 2`.**
- `migrateRegistries` validation loop (`prompt_freeze_core.go:251-266`) skips
  `id == r.Active || e.Frozen != nil`; the freeze-writing loop at `:267` has the same skip. Only
  the **validation** loop is widened (Solution Design §4).
- `freezeFixture` (`prompt_freeze_test.go:14-44`) writes 8 entries with fresh
  `sha256(content)` hashes and matching `.md` in **both** trees, and writes both registries from the
  same in-memory `r` — so widening cannot make `TestPromptFreezeMigrate_*` or
  `TestPromptFreezeCheck_GreenOnMigratedFixture` red.
- `make/code-health.mk:184` help text is `## Check frozen prompt immutability and mirror agreement (CI gate)`.

### 1.3 Carried forward from the designer without re-measurement [D]

V2 (CI wiring: `make/ci.mk:11`, `.github/workflows/ci.yml:205`), V8 (`//go:embed all:prompts` is a
directory glob), V9/V16 (loader behavior), V10 (`create_prompt_version.sh` path and content),
V11 (3 unregistered `.md`), V12 (`eval_suite_manifest.go`), V21 (stdout consumers), V22
(`freezeVersion` pre-write validation). None of these gates an AC; each is context.

---

## 2. Velocity and estimate

- **Recent throughput [P]**: `git log --since=14.days --oneline | wc -l` → **412** commits in 14
  days on this checkout's history. The mission runs a nightly design→plan→execute→evaluate loop,
  so per-sprint scope, not commit count, is the binding constraint.
- **Diff size (designer estimate, [D])**: ~+46/-16 production LOC over three files, ~+174 test LOC
  over two files, 1 comment line in `make/code-health.mk`, 1 CHANGELOG entry.
- **Estimate: 1 day / ≤ 6h.** Basis: all five touched files are ≤ 400 lines and were read
  end-to-end this session; the fixture harness already has the `frozen bool` switch; the violation
  strings are normative and pre-specified (S1–S11); and the six gates are already green at base, so
  every red the executor sees is caused by the executor.
- **Not a two-day sprint because**: no new package, no new dependency, no CI wiring change, no
  schema change, no data migration.

---

## 3. Milestone structure — ONE milestone, and why

The doc specifies a single milestone ("M1 — the whole change (≤1 day)"). The plan keeps it, for a
reason that is a property of the change, not of its size:

**The signature change `checkRegistries(repoRoot string) ([]string, error)` →
`([]string, int, error)` is a compile-breaking edit with 4 callers in one package [P].** Any split
that lands the core widening without simultaneously updating `prompt_freeze.go:69`,
`prompt_freeze_check_git_test.go:134`, `prompt_freeze_test.go:125` and `:138` produces a
**non-compiling intermediate commit** — i.e. a non-bisectable boundary. Splitting *around* that
constraint (e.g. "signature + callers" as M1a) would create a milestone whose only content is a
mechanical arity change with no behavior and no test — a boundary that proves nothing.

So: one milestone, one commit, `go test ./cmd/ailang/ -run 'Freeze|Prompt' -count=1` green at its
boundary.

---

## 4. Doc-vs-plan acceptance-criteria reconciliation (required section)

**Doc's `## Milestones` section says**: "Solution Design §1-§5 + all tests in the mutation table +
**AC1-AC8** + CHANGELOG."

**Doc's `## Acceptance Criteria` table contains**: **AC1 … AC10** (AC9 and AC10 were added in the
r2 revision in response to the `gpt5-6-sol` arm-independence objection).

**Plan's milestone task list closes**: **AC1 … AC10**.

**Divergence found, and it is internal to the doc**: the `## Milestones` string "AC1-AC8" is stale
r1 text that the r2 revision did not update. It is not a scope ruling — the same milestone sentence
says "**all tests in the mutation table**", and the mutation table's M8 / M9 rows name
`TestFreezeCheck_InvalidHashAndMissingSourceEmitsBoth` and
`TestFreezeCheck_BothTreesDeletedEmitsBoth`, whose fixtures **are** AC9 and AC10 verbatim. The
milestone therefore already includes AC9/AC10 by way of the mutation table it incorporates.

**Ruling**: the doc wins, and the doc's own AC table + mutation table are the doc. The milestone
closes **AC1–AC10**. This is a stale-string reconciliation, not an override of a reviewed decision.
The plan makes no other change to any AC. Second-order evidence: AC9/AC10 rest on V17/V18, which
are r2 Verification-Log rows, and the r2 Quorum revision log records both as "pinned by new AC9/AC10
plus independence mutants M8/M9".

**Planner disagreements with the doc's ACs**: none on substance. One presentational repair, in §6:
AC8 as written bundles a **regression** half (frozen cells 2–3 unchanged; existing tests pass) that
is at its target state at base **by construction** with a **progress** half (cell 4's mirror-deletion
rename) that is not. Bundled, AC8 reads as a criterion that half-cannot-fail. Split into
**AC8-REG** and **AC8-Δ** with the base measurement stated for each; no criterion is weakened, and
the at-target-at-base half is explicitly labelled as a regression guard rather than passed off as a
progress gate.

---

## 5. Milestone M1 — task list

Ordered. Every task is executable inside the worktree by a `workspace-write` sandbox. **No task
performs a git write operation** — the controller builds the commit.

| # | Task | Files | Done when |
|---|------|-------|-----------|
| T0 | **Baseline smoke** — re-run G1–G6 (§1.1) *before touching anything* and record rc. All six must be 0. | — | six rc=0 recorded in the executor report. **If any is non-zero, STOP and escalate** — the environment differs from the plan's baseline and every subsequent rc is uninterpretable (see Risk R1). |
| T1 | Widen the per-entry loop in `checkRegistries` into the three **independent** arms of Solution Design §1 (hash-enforceability / source-existence / bytes-vs-hash), carrying the `state` label. Frozen strings S1/S3 stay **byte-identical** (the `(unenforceable freeze)` suffix is preserved by the `if e.Frozen != nil` suffix switch, not by a separate branch). | `cmd/ailang/prompt_freeze_core.go` | S1–S5 emitted per §1; arm dependencies exactly the §3 table and nothing more. |
| T2 | Missing-source adjudication: `errors.Is(srcErr, fs.ErrNotExist)` → **violation S5 (rc=1)**; any other read error → `return nil, 0, srcErr` (rc=2, environmental). Add `errors` + `io/fs` imports. | same | AC3/AC7 fixture yields rc=1 with the named string, not an abort. |
| T3 | Add the mirror-only sweep over `mirror.VersionKeys` emitting S10, and the `checked` counter — **incremented as the last statement of each visiting loop body**, never as a `len()` computed beside the work. Change the signature to `([]string, int, error)`. | same | `checked` is a product of the per-entry work (doc §1, r2 objection 2). |
| T4 | Update **all four** callers for the new arity: `prompt_freeze.go:69`, `prompt_freeze_check_git_test.go:134`, `prompt_freeze_test.go:125`, `prompt_freeze_test.go:138`. | `prompt_freeze.go`, both test files | `go build ./cmd/ailang/...` rc=0 **and** `go vet ./cmd/ailang/...` rc=0. |
| T5 | Print `checked %d registry entries\n` to **stdout** in `runPromptFreeze --check`, on every completed run, green or red. | `cmd/ailang/prompt_freeze.go` | AC1/AC6 stdout assertions satisfiable. Base stdout is empty [P], so this line is new. |
| T6 | Widen the mirror loop: replace `if entry.Frozen == nil { continue }` (`prompt_freeze_check_git.go:51`) with the `state` label; make **mirror existence unconditional** (a missing source must not hide a missing/diverged mirror); emit S8 on `fs.ErrNotExist`; keep S6/S7 for divergence only. Leave the frozen-only `sameFrozenRegistryEntry` dedupe recheck exactly as-is. | `cmd/ailang/prompt_freeze_check_git.go` | S6/S7/S8 per §2; AC10 emits **both** S5 and S8. |
| T7 | **Do NOT touch** the merge-base loop's `if baseEntry.Frozen == nil { continue }` at `prompt_freeze_check_git.go:28`. | — | `git diff` shows line 28 unchanged (controller verifies at commit time). |
| T8 | Widen the `migrateRegistries` **validation** loop only: drop `id == r.Active \|\| e.Frozen != nil` from the loop at `:251`. The **freeze-writing** loop at `:267` keeps both skips. | `cmd/ailang/prompt_freeze_core.go` | migrate still never freezes `active`; `TestPromptFreezeMigrate_*` stay green. |
| T9 | New tests, per the mutation table (§7), in `prompt_freeze_check_git_test.go` using `newFreezeCheckRepo` + `writeFreezeCheckRegistries(…, frozen=false)`: `_MutablePlaceholderIsRed`, `_MutableSourceDivergenceIsRed`, `_SourceMissingIsViolationNotError` (frozen + mutable sub-cases), `_MutableMirrorDivergenceIsRed`, `_MirrorMissingIsRed` (frozen + mutable), `_MirrorOnlyEntryIsRed`, `_InvalidHashAndMissingSourceEmitsBoth`, `_BothTreesDeletedEmitsBoth`, `_CheckedCountMovesOnAddition`. Each asserts the **in-process violation slice** (exact S-string via `hasViolation`) **and** the helper-subprocess rc. | `cmd/ailang/prompt_freeze_check_git_test.go` | `go test ./cmd/ailang/ -run 'Freeze\|Prompt' -count=1` rc=0. |
| T10 | New migrate test `TestPromptFreezeMigrate_RefusesStaleActiveHash` extending `freezeFixture` (corrupt the `active` entry's file bytes after the fixture writes, so its recorded hash is stale) — `migrateRegistries` must return a non-nil error. | `cmd/ailang/prompt_freeze_test.go` | asserts the returned error, which is the direct product of the widened validation loop. |
| T11 | **Run the mutation matrix** (§7) — all 10 mutants, each with its LANDED + BUILDS preconditions and byte-identical restore. Record per mutant: sha256 before/after/restored, build rc, the **enumerated** failing-test set, and the blast-radius verdict. | working copies only | §7's per-mutant criterion satisfied and recorded; tree byte-identical to pre-mutation state at the end (sha256 equality, per file). |
| T12 | Update the `make/code-health.mk:184` help text from `Check frozen prompt immutability and mirror agreement (CI gate)` to `Check prompt registry integrity (all entries) + frozen immutability (CI gate)`. Comment/help only — the recipe line at `:185` is unchanged. | `make/code-health.mk` | `make check-prompt-freeze` still rc=0; `make help` renders. |
| T13 | CHANGELOG entry under Unreleased → Fixed/Changed, naming the widened arms, the new S-strings, and the `checked N registry entries` stdout line as a behavior change. | `CHANGELOG.md` | present. |
| T14 | **Final gate sweep**: re-run G1–G6 and record each rc. | — | all six rc=0. |

**Explicitly out of scope** (Non-Goals — the executor must not do these): merge-base immutability
widening; either loader (`internal/prompt/loader.go`, `internal/eval_harness/prompt_loader.go`);
registering the 3 unregistered `.md` files; changing `freezeVersion`; any CI/Makefile **wiring**
change; `go build ./...`.

---

## 6. Acceptance criteria — with planner-measured base rcs

**Probe protocol used for every base measurement below [P]** (identical construction to the doc's,
re-run first-party by the planner this session):
`/tmp/i297p/repo` ← `prompts/`, `cmd/ailang/prompts/`, empty `eval_results/baselines/` copied from
the worktree; `git init` + one commit + `git update-ref refs/remotes/origin/dev HEAD`; tar snapshot
(`--exclude .git`) restored before **and** after every cell; probe binary
`go build -o /tmp/i297p/bin/ailang ./cmd/ailang` from the worktree; invocation
`/tmp/i297p/bin/ailang prompt freeze --check --repo /tmp/i297p/repo`. Post-restore control cell:
**rc=0** — the restore is byte-faithful, so each cell's rc is attributable to that cell.

**Positive controls in the same harness**: cells AC8a/AC8b/AC8c returned **rc=1 with the exact
frozen violation strings**. Four reds in the same instrument make the six rc=0 cells measurements
of a hole, not a broken probe.

| # | Criterion | Command | **rc at base (planner-measured)** | Required post-change |
|---|-----------|---------|-----------------------------------|----------------------|
| AC1 | Pristine green **and the count line appears** | repo-root `make check-prompt-freeze`; probe pristine `--check` | **0**, and **stdout empty** — no count line exists at base | rc=0 **and** stdout contains `checked 59 registry entries`. *Falsifiable at base via the stdout half*; the rc half is a regression guard. |
| AC2 | Mutable source divergence refused | `printf 'x' >> prompts/v0.16.6.md` → `--check` | **0** ← the hole | rc=1, contains `mutable version v0.16.6: file bytes do not match recorded hash` |
| AC3 | Mutable source deletion refused, **named** | `rm prompts/v0.16.6.md` → `--check` | **0** ← the hole | rc=1, contains `mutable version v0.16.6: prompt file missing at prompts/v0.16.6.md` |
| AC4 | Mutable mirror divergence refused | `printf 'x' >> cmd/ailang/prompts/v0.16.6.md` → `--check` | **0** ← the hole | rc=1, contains `mutable version v0.16.6: mirror bytes differ at cmd/ailang/prompts/v0.16.6.md` |
| AC5 | Mutable `PLACEHOLDER` refused | set `v0.16.6` hash to `PLACEHOLDER` in **both** registries → `--check` | **0** ← the hole | rc=1, contains `mutable version v0.16.6: recorded hash is not a 64-hex sha256 (unenforceable)` |
| AC6 | Mirror-only entry refused **and counted** | add `v9.9.9-extra` to the **mirror** registry only → `--check` | **0** ← the hole; and no count line exists at base | rc=1, contains `entry v9.9.9-extra missing from source registry`, **and** stdout reads `checked 60 registry entries` (AC1 pins pristine at 59, so the 59→60 move is the assertion, not the verdict alone) |
| AC7 | Missing **frozen** source is a named violation, not an abort | `rm prompts/v0.3.21.md` → `--check` | **2**, stderr exactly `open /tmp/i297p/repo/prompts/v0.3.21.md: no such file or directory` and nothing else | rc=1, contains `frozen version v0.3.21: prompt file missing at prompts/v0.3.21.md` |
| **AC8-REG** | **Regression guard — at target at base BY DESIGN.** Frozen source divergence and frozen mirror divergence stay byte-identical; existing tests keep passing | probe cells AC8a (`printf 'x' >> prompts/v0.3.21.md`) and AC8b (`printf 'x' >> cmd/ailang/prompts/v0.3.21.md`); repo-root `go test ./cmd/ailang/ -run 'Freeze\|Prompt' -count=1` | **AC8a rc=1** with exactly `frozen version v0.3.21: file bytes do not match recorded hash` + `frozen version v0.3.21: immutability violation in prompts/v0.3.21.md bytes` + `frozen version v0.3.21: mirror bytes differ at cmd/ailang/prompts/v0.3.21.md`; **AC8b rc=1** with `frozen version v0.3.21: mirror bytes differ at cmd/ailang/prompts/v0.3.21.md`; **test rc=0** | identical rc and identical strings; test rc=0. **This criterion cannot fail at base and is not claimed to — it is a no-drift guard, labelled as such.** |
| **AC8-Δ** | **Progress — the mirror-deletion rename.** Frozen mirror deletion is reported honestly | `rm cmd/ailang/prompts/v0.3.21.md` → `--check` | **1**, with the *misleading* string `frozen version v0.3.21: mirror bytes differ at cmd/ailang/prompts/v0.3.21.md` | rc=1, containing `frozen version v0.3.21: mirror file missing at cmd/ailang/prompts/v0.3.21.md` and **NOT** containing `mirror bytes differ at cmd/ailang/prompts/v0.3.21.md`. *The negative half is what makes this falsifiable at base.* |
| AC9 | **Compound**: invalid hash **and** missing source emit **both** named violations | set `v0.16.6` hash to `PLACEHOLDER` in both registries **and** `rm prompts/v0.16.6.md` → `--check` | **0** ← the compound hole | rc=1, stderr contains **both** `mutable version v0.16.6: recorded hash is not a 64-hex sha256 (unenforceable)` **and** `mutable version v0.16.6: prompt file missing at prompts/v0.16.6.md` |
| AC10 | **Compound**: source **and** mirror both deleted emit **both** named violations | `rm prompts/v0.3.21.md cmd/ailang/prompts/v0.3.21.md` → `--check` | **2**, stderr names only the FIRST missing file; the mirror deletion is invisible | rc=1, stderr contains **both** `frozen version v0.3.21: prompt file missing at prompts/v0.3.21.md` **and** `frozen version v0.3.21: mirror file missing at cmd/ailang/prompts/v0.3.21.md` |

**Audit of the "can this AC fail at base?" question, as required**: nine of eleven rows are
falsifiable at base outright (AC2–AC7, AC8-Δ, AC9, AC10 differ from their required post-state in rc,
string, or both). **AC1** is rc=0 at base and rc=0 after — falsifiable only through its stdout half,
which is why the stdout assertion is mandatory and not decorative; without it AC1 would be a gate
that cannot fail. **AC8-REG** is at its target state at base by construction and is labelled a
regression guard, not a progress gate; it is retained because string drift on the frozen path is a
real risk of this diff (T1 rewrites the block that produces S1/S3).

### 6.1 Executor vs CONTROLLER split (sandbox reality)

The executor lane is `codex:gpt-5.6-sol` under `--sandbox workspace-write`: **no loopback sockets,
no writes outside the worktree.**

| Gate | Owner | Why |
|------|-------|-----|
| G1 G2 G3 G4 G5 G6 (§1.1) | **Executor** | In-worktree; read-only git (`merge-base`, `show`); no sockets. G5/G6 shell out to `git` inside the worktree only. |
| T9/T10 Go tests | **Executor** | The harness writes to `t.TempDir()` (under `TMPDIR`) and runs `git init` **in that temp dir** — not in the repo. `go test` also writes `GOCACHE`. This is the standing assumption of every Go sprint in this mission, and T0 probes it before any edit (Risk R1). |
| T11 mutation matrix | **Executor** | Edits and restores files **inside the worktree** only; runs `go build` / `go test`. |
| **AC1–AC10 probe-matrix cells** | **CONTROLLER** | They construct a synthetic repo **outside the worktree** (`/tmp/i297p/repo`) and run `git init` / `git commit` / `git update-ref` in it. Outside-worktree writes are not an executor obligation. |
| **AC1 repo-root `make check-prompt-freeze` half** | **Executor** | In-worktree, read-only. |
| CI (`.github/workflows/ci.yml:205`) | **CONTROLLER** | Post-commit; needs network. |

**Consequence, stated plainly**: the executor's obligations are T0–T14 minus the probe matrix. The
probe matrix is the controller's post-hoc confirmation, and the Go tests of T9/T10 are the
executor-side encoding of the same coverage — which is why every AC row above has a corresponding
test row in §7. **If the executor cannot satisfy an AC because it is a controller gate, that is not
a failure; failing to have written the corresponding test IS.**

---

## 7. Mutation / refusal-branch test plan

The deliverable's headline verb is **refuse**, so every refusal branch gets a mutant.

### 7.1 Non-negotiable mutation discipline

1. **Neutering form only.** `if false && <cond>` (or, for M7/M8/M9, the condition-**wrapping** form
   given in the table). **Never delete a block, never delete a statement, never remove an import.**
   Reason: with the block intact every import stays used, so "the mutant does not compile" can never
   masquerade as "the guard fired".
2. **LANDED check, before reading any test result.** `shasum -a 256 <file>` before and after the
   edit **must differ**. A mutant that did not land makes the subsequent green meaningless.
3. **BUILDS check, before reading any test result.** `go build ./cmd/ailang/...` **rc=0** under the
   mutant. A compile failure is not a guard firing.
4. **Restore by `cp` from a pre-mutation backup, verified by sha256 equality.**
   **NEVER `git checkout -- <file>` and NEVER `git stash`** — the sprint's own uncommitted work
   lives in these exact files, and a checkout would delete it. Suggested shape:
   `cp cmd/ailang/prompt_freeze_core.go /tmp/i297mut/prompt_freeze_core.go.bak` before, then
   `cp /tmp/i297mut/prompt_freeze_core.go.bak cmd/ailang/prompt_freeze_core.go` after, then assert
   the sha256 equals the pre-mutation value. (Backups under `/tmp` are *reads* into the worktree on
   restore; if the sandbox refuses `/tmp` writes, keep backups at `<worktree>/.mutbak/` — still no
   git operation. Do not commit `.mutbak/`; delete it in T14.)
5. **Order**: backup → mutate → LANDED → BUILDS → run → record → restore → sha256-equality assert.
   All ten mutants restored before T12.

### 7.2 Blast-radius classification — determined by RUNNING, not predicting

Per mutant the executor must **run** `go test ./cmd/ailang/ -run 'Freeze|Prompt' -count=1` and
**enumerate the actual failing test names** from the output. Then apply, per mutant:

- **Criterion A — single-test (strongest).** The failing set is exactly `{target}`. The executor
  must additionally run the **inverse arm**:
  `go test ./cmd/ailang/ -run 'Freeze|Prompt' -skip '^<target>$' -count=1` → **rc must be 0**.
  This proves the mutant is visible to that test and to no other. `-skip` is supported on this
  toolchain [P].
- **Criterion B — broad blast.** The failing set has >1 member. Criterion A is unsatisfiable *by
  construction* here, so the required evidence is the **enumerated set itself**, plus: (i) the
  target test **is in the set**, and (ii) **every other member is explained** by one sentence naming
  the shared code path the mutant sits on. An unexplained member is a finding, not noise.

**Which criterion applies is a measurement, not a prediction.** The "prior" column below is the
planner's *unverified expectation* and carries no authority; the executor records the observed set
and picks A or B from it. If the observation contradicts the prior, the observation wins and the
executor notes it.

### 7.3 The matrix

| # | Refusal branch | Neutering mutation | Test that must go red | Downstream observable asserted | Criterion (planner prior — **unverified**) |
|---|----------------|--------------------|-----------------------|-------------------------------|--------------------------------------------|
| M1 | S2 mutable hash-enforceable | `if false && !eval_harness.IsHexSHA256(e.Hash)` | `TestFreezeCheck_MutablePlaceholderIsRed` | violation containing `unenforceable` **and** rc 1→0; only this branch emits S2 | A expected |
| M2 | S4 mutable bytes-vs-hash | `if false && (hex.EncodeToString(sum[:]) != e.Hash)` on a `frozen=false` fixture | `TestFreezeCheck_MutableSourceDivergenceIsRed` | S4 string + rc; the frozen branch cannot fire on a `frozen:false` fixture | A expected |
| M3 | S5 missing source file | `if false && errors.Is(srcErr, fs.ErrNotExist)` | `TestFreezeCheck_SourceMissingIsViolationNotError` (frozen + mutable sub-cases) | neutered → the read error propagates → `checkRegistries` returns `err` → in-process `t.Fatal` path **and** helper rc=**2**, not 1. The test asserts rc==1 **and** the S5 string; both are produced only by the branch | A expected |
| M4 | S7 mutable mirror divergence | `if false && !bytes.Equal(sourceBytes, mirrorBytes)` | `TestFreezeCheck_MutableMirrorDivergenceIsRed` | S7 string + rc; no other arm reads mirror `.md` bytes | **B plausible** — this is the shared S6/S7 comparison, so `TestFreezeCheck_MirrorMdDivergenceIsRed` (existing, frozen) may also go red. If so that is Criterion B with one explained member. **Measure it.** |
| M5 | S8 mirror file missing | `if false && errors.Is(mirrorErr, fs.ErrNotExist)` | `TestFreezeCheck_MirrorMissingIsRed` (frozen + mutable) | neutered → falls to `else if mirrorErr != nil → return nil, err` → rc=2, so the rc==1 assertion goes red for the reason it claims | A expected |
| M6 | S10 mirror-only entry | `if false && (source.Versions[id] == nil)` | `TestFreezeCheck_MirrorOnlyEntryIsRed` | S10 string + rc; no other loop visits mirror-only keys | **B plausible** — the same mutant also suppresses the `checked++` inside that sweep, so `TestFreezeCheck_CheckedCountMovesOnAddition` is expected to go red too. Both members are targets of the same branch; enumerate and explain. |
| M7 | migrate validation widening | restore the skip as `if false \|\| id == r.Active { continue }` inside the **validation** loop only | `TestPromptFreezeMigrate_RefusesStaleActiveHash` (new) | migrate's returned error is the direct product of the validation loop; with the skip restored, migrate succeeds and the error assertion fails | A expected |
| M8 | **arm independence**: source-existence must not depend on hash validity | wrap arm 2's emission as `if hashOK && !sourceExists` (reintroduces r1 masking; no block deleted, all imports stay used) | `TestFreezeCheck_InvalidHashAndMissingSourceEmitsBoth` (AC9 fixture: `frozen=false`, hash `PLACEHOLDER`, source absent) | asserts **both** S2 and S5; under the mutant only S2 is emitted → red **specifically on the S5 assertion** | A expected |
| M9 | **arm independence**: mirror-existence must not depend on source existence | wrap the S8 emission as `if sourceExists && errors.Is(mirrorErr, fs.ErrNotExist)` | `TestFreezeCheck_BothTreesDeletedEmitsBoth` (AC10 fixture: both `.md` removed) | asserts **both** S5 and S8; under the mutant only S5 is emitted → red **specifically on the S8 assertion** | A expected |
| **M-ADD** | enumerator completeness — an **addition**, not a removal. Every removal proves the check FIRES; only an addition proves it **LOOKS** | fixture-side: ADD `v9.9.9-extra` to the mirror registry of an otherwise-green fixture. **No production edit**, so the LANDED/BUILDS preconditions apply to the *test* file | `TestFreezeCheck_CheckedCountMovesOnAddition` | `checked` counts **visits** and is incremented inside the visiting body (§1). Fixture arithmetic **[P]**: 1 source entry alone → `checked 1`; with the mirror-only entry → `checked` MUST move to **2** AND S10 must fire. An enumerator that never visits mirror-only keys contributes neither → red **on the count**, not merely on the verdict | A expected |

**Honest scope of the two observables, restated so the executor does not conflate them**: the
`checked` count detects **enumeration blindness** (an entry no loop visits). It structurally
**cannot** detect a neutered arm inside a *visited* body — the entry is still visited and still
counted. Those defects are killed by M1–M9. The observables are complementary; both are asserted.

**Test-design constraint (carried from the doc, non-negotiable)**: every new test drives the REAL
check path — the in-process `checkRegistries` **and** the `TestFreezeCheckHelperProcess` subprocess
rc — against a synthetic git repo. **No test may seed a verdict by raw fixture**, which is how the
parent sprint's tests once stayed green while a writer did not exist.

---

## 8. Day-by-day breakdown

One day. Hour bands are guidance, not a contract; the ordering is the contract.

| Band | Tasks | Exit condition |
|------|-------|----------------|
| **H0 — 0:00-0:20** | T0 | Six rc=0 recorded. Any non-zero → STOP + escalate. |
| **H1 — 0:20-2:00** | T1, T2, T3, T4, T5 | `go build ./cmd/ailang/...` rc=0, `go vet ./cmd/ailang/...` rc=0, `gofmt -l cmd/ailang` empty. Existing `Freeze|Prompt` tests may be red here — expected, T9 has not run. |
| **H2 — 2:00-2:45** | T6, T7, T8 | Same three rc=0. Diff touches `prompt_freeze_check_git.go:49-72` and `prompt_freeze_core.go:251-266` but **not** `prompt_freeze_check_git.go:28`. |
| **H3 — 2:45-4:15** | T9, T10 | `go test ./cmd/ailang/ -run 'Freeze\|Prompt' -count=1` rc=0 with all 10 new tests present and passing. |
| **H4 — 4:15-5:30** | T11 | All 10 mutants run, each with LANDED + BUILDS recorded, an enumerated failing set, a Criterion A or B verdict, and a sha256-verified restore. |
| **H5 — 5:30-6:00** | T12, T13, T14 | G1–G6 all rc=0; `.mutbak/` removed; report written. |
| **Post — controller** | AC1–AC10 probe matrix; commit; CI | Every AC row in §6 at its required post-state. |

**Pause point**: after H3. If `go test` is not green at H3 the sprint does not proceed to mutation
testing — a mutation matrix run against a red suite measures nothing.

---

## 9. Test plan (summary)

- **Unit / integration**: `go test ./cmd/ailang/ -run 'Freeze|Prompt' -count=1` — 5 existing
  `TestFreezeCheck_*` + the migrate/check tests in `prompt_freeze_test.go` + 10 new tests.
  **rc=0 at base [P]**, so any red is caused by this sprint.
- **String normativity**: S1–S11 in the doc's Solution Design table are **normative**. Frozen-path
  strings S1/S3/S6/S9/S11 must stay byte-identical; S8 is the one deliberate rename. Assert with
  `hasViolation(violations, parts...)` (substring after `ToSlash`), which already exists.
- **rc semantics**: 0 green / 1 violations / 2 usage-or-IO error. The core behavioral change is that
  **`fs.ErrNotExist` moves from the 2 bucket to the 1 bucket**; every other error stays at 2. Two
  tests (`_SourceMissingIsViolationNotError`, `_MirrorMissingIsRed`) exist specifically to pin that
  boundary, and their neutering mutants (M3, M5) both manifest as rc 1→2.
- **Mutation**: §7, 10 mutants, LANDED+BUILDS gated, restore-by-cp.
- **Whole-gate**: `make check-prompt-freeze` rc=0 at repo root, stdout containing
  `checked 59 registry entries`.
- **Probe matrix** (controller): all 11 cells of §6.
- **Not run**: `go build ./...` — rc=1 at base for an unrelated reason [P].

---

## 10. Risks

| # | Risk | Likelihood | Impact | Mitigation / detection |
|---|------|-----------|--------|------------------------|
| **R1** | **Sandbox denies `TMPDIR` or `GOCACHE` writes**, so `go test` / `go build` fail for environmental reasons and the executor misreads it as a code failure. The whole test harness builds repos in `t.TempDir()`. | Low (every prior Go sprint in this mission ran in this lane) | **Sprint-fatal if misdiagnosed** | **T0 runs first and is the probe.** All six gates rc=0 at base before a single edit. A T0 failure is an environment escalation to the controller, never a code fix. |
| R2 | **Frozen-path string drift.** T1 rewrites the block that produces S1 and S3; the `(unenforceable freeze)` vs `(unenforceable)` suffix is one `if` away from silently changing 58 entries' messages. | Medium | High (breaks existing tests and any operator muscle memory) | AC8-REG pins cells AC8a/AC8b to byte-identical strings; `TestFreezeCheck_FrozenPlaceholderIsRed` and `_MirrorMdDivergenceIsRed` already assert them. |
| R3 | **Accidentally widening merge-base immutability** (the tempting "just delete the Frozen guards" refactor), which would make the legitimate mutable edit+hash-regen workflow red. | Medium | High — breaks `create_prompt_version.sh`'s documented flow | T7 is an explicit do-not-touch task naming `prompt_freeze_check_git.go:28`; controller verifies line 28 is absent from the diff. |
| R4 | **Signature change leaves a caller behind**, producing a non-compiling commit. | Low | Medium | 4 callers enumerated by name in T4 [P]; **Go is the enforcer** — 3 results cannot be assigned to 2 variables, so this fails loudly at `go build`, never silently as a stale zero. |
| R5 | **`checked` implemented as a `len()` beside the work** instead of an in-body increment — this is exactly the r2 quorum objection, and it produces a count that is green while the enumerator is blind. | Medium (it is the easier implementation) | High — it silently voids AC6 and M-ADD | T3 states the constraint; M-ADD's kill condition is **on the count**; a `len(source.VersionKeys)+len(mirrorOnly)` implementation still passes M-ADD, so the **code review must read the increment site**. Flagged for the evaluator. |
| R6 | **The `checked N` stdout line breaks a consumer.** | Low | Medium | Designer inventory [D, V21] finds exactly one invocation, `make/code-health.mk:185`, which neither pipes nor parses stdout. Planner confirms base stdout is currently **empty** [P], so this is a genuine behavior change — hence T13 records it in CHANGELOG as *Changed*, not merely *Fixed*. |
| R7 | **Mutation restore via `git checkout`** destroys the sprint's uncommitted work in the same files. | Low but catastrophic | Sprint-fatal | §7.1 rule 4 forbids it explicitly; restore is `cp` + sha256 equality assert. |
| R8 | **Committed `PLACEHOLDER` becomes a CI violation** (Solution Design §5) and breaks someone's flow. | Very low | Low | Cost today is **zero**: 0 of 59 entries use PLACEHOLDER — the one mutable entry is 64-hex [P], the 58 frozen were already banned. The loader's runtime hatch is untouched. |
| R9 | Existing `TestPromptFreezeCheck_RedOnMissingMarker` sets `b1.Frozen = nil`, making `b1` **mutable** and newly subject to all four widened arms; it asserts `len(v) == 1`. | Low | Medium | `freezeFixture` writes fresh hashes and matching `.md` in both trees from one in-memory `r` [P], so the widened arms emit nothing for `b1` and the count stays 1. Detected immediately by T9's test run if wrong. |

---

## 11. Rollback

The whole sprint is one commit touching six files, none of which is a schema, a migration, or a
wire format.

1. **Pre-merge**: the controller simply does not commit. The worktree is discarded; nothing else is
   affected. There is no released artifact and no state written anywhere by this change.
2. **Post-merge, gate too strict** (the realistic failure: some entry in a contributor's tree is red
   for a reason we did not anticipate): revert the single commit. `make check-prompt-freeze`
   returns to its current frozen-only behavior. **No data is rolled back** — the change reads files
   and returns an exit code; it writes nothing except one stdout line.
3. **Post-merge, partial rollback if only the count observable is unwanted**: remove the
   `fmt.Printf("checked %d registry entries\n", n)` in `prompt_freeze.go` and revert the signature to
   `([]string, error)`. The four widened arms are independent of the count and stay.
4. **What rollback cannot undo**: nothing. `--migrate` is not run by CI, `freezeVersion` is
   untouched, and no registry bytes are written by any code path this sprint modifies.

---

## 12. Definition of done

- [ ] T0 recorded: G1–G6 all rc=0 **before** any edit.
- [ ] T1–T8 landed; `go build ./cmd/ailang/...`, `go vet ./cmd/ailang/...`, `gofmt -l cmd/ailang`
      all clean.
- [ ] `prompt_freeze_check_git.go:28` (merge-base `Frozen == nil` skip) **unchanged**.
- [ ] 10 new tests present; `go test ./cmd/ailang/ -run 'Freeze|Prompt' -count=1` rc=0.
- [ ] All 10 mutants run with LANDED + BUILDS + enumerated failing set + Criterion A/B verdict +
      sha256-verified restore; tree byte-identical afterwards.
- [ ] `make/code-health.mk` help text updated; CHANGELOG entry written.
- [ ] T14: G1–G6 all rc=0.
- [ ] **Controller**: AC1–AC10 probe matrix at required post-state; commit; CI green.
