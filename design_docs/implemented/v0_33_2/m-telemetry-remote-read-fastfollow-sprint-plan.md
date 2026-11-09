# Sprint Plan: M-TELEMETRY-REMOTE-READ-FASTFOLLOW

**Design doc**: [m-mission-loop-unified-telemetry.md](m-mission-loop-unified-telemetry.md) (ratified, 2/3 executed)
**Prior plan**: [m-mission-loop-unified-telemetry-sprint-plan.md](m-mission-loop-unified-telemetry-sprint-plan.md)
**Issue**: [#698](https://github.com/sunholo-data/ailang/issues/698)
**Sprint ID**: `M-TELEMETRY-REMOTE-READ-FASTFOLLOW`
**Created**: 2026-08-13 (mission iteration 195)
**Baseline commit**: `644cf178ac3c6054f8039938abf122b0695f9891` (origin/dev) — M1–M3
**M4 baseline commit**: `bad8f3647` (origin/dev) — every M4 number below re-measured there on 2026-08-14
**Duration**: ~0.5 day for M1–M3 (~3.75h, LANDED in [#699](https://github.com/sunholo-data/ailang/pull/699)) · **~1 day for M4 (~7h)**
**Risk**: Low for M1–M3 (tests + one state file, no production code changed). **M4 is now EXECUTABLE** — `D-15` was ratified `view` by Mark, attended, 2026-08-14. M4 is Medium risk: it is the first production-code change in this sprint.
**Total LOC estimate**: ~175 for M1–M3 (implementation 0, tests ~175) · **~400 for M4 (~205 production, ~195 tests)**

---

## Summary

**Goal**: close the two gaps in #698 that are buildable with zero design risk, and put a *number* on
the third so a human can decide it in one word.

#698 has three parts. Exactly two of them are engineering:

| #698 part | This plan | Status |
|-----------|-----------|--------|
| 2 — pinned-ID guard survives mutation | **M1** | **LANDED** (#699) |
| 3 — five unpinned error branches | **M2** | **LANDED** (#699, one of the five is unreachable — see M2) |
| — sprint JSON absent from `dev` | **M3** | **LANDED** (#699) |
| 1 — opt-in remote READ | **M4** | **UNBLOCKED 2026-08-14 — `D-15` = `view`. Executable.** |

**The decision M4 needed, and the answer:**

> **How far should `--remote` reach — `view` or `eval`?**
> → **`view`.** Ratified by Mark in person, 2026-08-14, recorded in
> [`design_docs/v1-mission.md`](../../v1-mission.md) under *"RATIFIED (attended, 2026-08-14
> morning)"*, item 3.

The costing, the options table and the Firestore constraint that made this a decision are preserved
in [M4](#m4) as the **recorded rationale** — they are
the evidence for why `view` was chosen, and they are marked RESOLVED-BY-RULING rather than deleted.

---

## Why the ratified item vanished — and the rule this plan adopts because of it

The design doc's Design Freeze item 3 is ratified. The prior sprint plan's **M3 task list** carries it
verbatim ("Opt-in remote read for analysis; local stays the default"). The prior plan's **M3
acceptance-criteria section** carries **five** criteria and **zero** of them mention read or remote.

Measured on the pristine tree (controls in the same call, same file, same scope):

```bash
awk '/^### M3_NODE_GENERIC_CLOUD_ROUTING/,/^## Out of Scope/' \
    design_docs/planned/v0_33_2/m-mission-loop-unified-telemetry-sprint-plan.md > /tmp/m3.txt
awk '/\*\*Acceptance criteria:\*\*/,0' /tmp/m3.txt > /tmp/m3ac.txt
grep -ciE 'read|remote' /tmp/m3ac.txt   # -> 0      the omission
grep -c  '^- '          /tmp/m3ac.txt   # -> 5      CONTROL: the AC section is non-empty
awk '/\*\*Tasks:\*\*/,/\*\*Acceptance criteria:\*\*/' /tmp/m3.txt | grep -ciE 'remote read'  # -> 1  CONTROL: the task existed
```

So M3 passed every one of its acceptance criteria while a third of its task list was never written.
**A task with no acceptance criterion is invisible to the gate.** It is not "less tested" — it is
structurally unobservable, and a PASS on the milestone is evidence about the other two thirds only.

**Rule adopted by this plan, without exception:** *every* task below carries an acceptance criterion
that can FAIL, and each criterion names the command whose output would be **different** if the task
were not done. Where the criterion is "a test exists and passes", it is paired with the **mutation**
that must make it fail — because a test that passes with the property removed is the same class of
non-observation as an absent criterion.

---

## Mutation discipline (applies to M1, M2 and M4; non-negotiable)

A mutation result may only be read after the mutation is proven **landed** and **building**. In this
order, in one shell block, with the file's baseline sha256 recorded first:

```bash
# 0. baselines measured on 644cf178a
#    internal/observatory/store_chains.go               54432ddb531082a0160eca2ffe86bb45df8b06563490d7dc5e3bae8dca31b624
#    cmd/ailang/chains_post.go                          04ba78d9eec6a0ac207565f7c16f16069bef0ae944da9b3b264e88d98fdb3a31
#    internal/storage/firestore/observatory_chains.go   73b7267458975e8379c854ca52c97e89e1b4057851f58fe02a8ac852b4995af8

cp <file> /tmp/mut-backup                      # restore source; NEVER `git checkout`
shasum -a 256 <file>                           # before
sed -i '' '<LINE>s/<exact>/<mutant>/' <file>   # LINE-SCOPED, always
shasum -a 256 <file>                           # after — MUST differ, else the sed silently no-op'd
go build ./internal/observatory/... ./cmd/ailang/... ./internal/storage/... ; echo "rc=$?"   # MUST be 0
go test <scoped -run> -count=1 ; echo "rc=$?"  # only NOW is this result meaningful
cp /tmp/mut-backup <file>
shasum -a 256 <file>                           # MUST equal the "before" sha, byte-identical
```

Three environment facts that will otherwise cost an hour:

- **`go build ./...` is rc=1 on the untouched tree.** `cmd/wasm` reports
  `function main is undeclared in the main package`. Measured on `644cf178a`. This is **not** this
  sprint's regression. Every build assertion in this plan is package-scoped (`rc=0`, measured).
- **The shell is zsh.** Quote glob-shaped flag values (`--include='*.go'`, `-run 'TestX'`), brace
  `"${var}:path"`, never `echo` raw bytes, and remember `cmd | tail; echo $?` reports **tail's**
  status — use `cmd > /tmp/out 2>&1; echo "rc=$?"` instead.
- **`if stageID == "" {` occurs TWICE** in `internal/storage/firestore/observatory_chains.go`
  (lines 244 and 375). A global `sed` mutates the wrong branch and the test still passes for the
  wrong reason. Line-scope every mutation.

### Mutating code this sprint WRITES (M4 only)

M1 and M2 mutate lines that already exist, so their `sed` addresses are literal line numbers. Most
of M4's mutants target code M4 itself creates, which has no line number until it is written. **Line
scoping is still mandatory** — the address is resolved in the same shell block, never globalised:

```bash
F=cmd/ailang/chains_read_backend.go
cp "$F" /tmp/mut-backup
shasum -a 256 "$F"                                     # before
L=$(grep -n '<THE EXACT MANDATED LITERAL>' "$F" | cut -d: -f1)
test "$(printf '%s\n' "$L" | wc -l)" -eq 1 || echo "AMBIGUOUS — refuse to mutate"   # MUST be exactly one
sed -i '' "${L}s/<exact>/<mutant>/" "$F"               # still line-scoped
shasum -a 256 "$F"                                     # after — MUST differ
go build ./internal/observatory/... ./cmd/ailang/... ./internal/storage/... ; echo "rc=$?"   # MUST be 0
go test <scoped -run> -count=1 ; echo "rc=$?"          # only NOW meaningful
cp /tmp/mut-backup "$F"; shasum -a 256 "$F"            # MUST equal "before"
```

Each M4 task below therefore **mandates the exact source literal** its mutation targets. If the
executor writes the logic with different text, it must write the literal too, or record in the
sprint JSON that the mutation address could not be resolved — an unresolvable mutation address is a
FAILED criterion, not a waived one.

**M4 baseline sha256, measured on `bad8f3647` (2026-08-14):**

```
294103708b20b0a5e5f4199fb8c80951c53e59ed720d07218f996f02c678854e  cmd/ailang/chains.go
63e9d2b27e46397e5a47894dfb21365dddc793e19bd3f83d909382b85a1d0915  cmd/ailang/chains_util.go
baa479cecf73b9e550ec452dd1bcbfffcb4b820ea6c683e498bb4629d33249df  cmd/ailang/chains_tree.go
04e6393e07e6f04cf2a7ac9be610324a2d5cd04433b09036a0bc1138b3f4f7ba  cmd/ailang/chains_diagnostics.go
3b973dba8f68566b443f84cb8ffdaf5ee1a3cfb3fde031bc557d4ea35678c395  cmd/ailang/chains_find.go
2262fb9c69dd6b4bfa94100d4b7bc53cb8bcac7fc903eaec6151eb50c175c3b7  cmd/ailang/chains_stats_cvs.go
31d731a1a3f6c9d8ac7bfde47da06204983dbe76142b3f55704187c7f29c1370  cmd/ailang/chains_stats_mission.go
8a47722e5259baf77c0a2c6c92a97eefa9c3867d73c95e85df3acb2152713b6d  cmd/ailang/chains_live.go
9335f4999e1f6f39f7f3ce24ee610a11e6ee7241ceab0d307f5fbf7acd9e7d12  cmd/ailang/eval_chains.go
73b7267458975e8379c854ca52c97e89e1b4057851f58fe02a8ac852b4995af8  internal/storage/firestore/observatory_chains.go
2b53059128153e9809cec5cb0d92cff6d4a69db9db20257d4b37bfeacff59edf  design_docs/planned/v0_33_2/m-mission-loop-unified-telemetry.md
```

**M4 pristine baselines, re-measured on `bad8f3647`, not inherited:**
`go build ./internal/observatory/... ./cmd/ailang/... ./internal/storage/...` → **rc=0** ·
`go test ./cmd/ailang/... -count=1` → **rc=0, 21.695s** (the plan's inherited "48.4s" is a
`644cf178a` figure and does **not** reproduce; time is not a criterion, rc is) ·
`go build ./...` → **rc=1** (`cmd/wasm`, BASE failure).

---

## Planning Discovery — what moved an estimate

Everything here was measured on the pristine worktree at `644cf178a` during planning. Items marked
**REFUTES** contradict a claim in #698 or in one of the two prior documents.

| # | Measurement | Effect |
|---|-------------|--------|
| 1 | **REFUTES #698 §2.** `chain_stages` has BOTH `id TEXT PRIMARY KEY` *and* `UNIQUE(chain_id, stage_number)` (`schema_chains.sql:37,76`). #698 says the retry path "is only reached on a real `stage_number` collision". Probed with the sqlite3 CLI on the same DDL: inserting a **duplicate pinned id** yields `UNIQUE constraint failed: t.id (19)` — which contains `"UNIQUE constraint"` and therefore enters the retry branch. Positive control in the same call: a non-duplicate insert returned rc=0 and landed. | M1 drops from a **flaky concurrency test** to a 55-LOC deterministic one. **−3h** |
| 2 | **Scope correction to #698 §3.** `observatory.EvalAssessment` (`models_chains.go:139-177`) is 26 fields, all `string`/`bool`/`int`/`int64`. `json.Marshal` on it **cannot fail**, so the `failed to marshal eval assessment` branch is unreachable by construction. | 1 of 5 branches cannot be red-tested. M2 ships 4 real arms + 1 guard test + a documented finding, not 5 arms. |
| 3 | **REFUTES the framing of #698 §1.** The `chains` command family already uses `observatory.Backend` via `NewSQLiteBackendFromPath`, **not** `*Store`: 35 of 41 non-test call sites are in `cmd/ailang/`, and of the 18 `chains*.go` files only `chains_import.go` (a WRITE) touches `OpenDefaultStore`. | The design doc's own worked example, `ailang chains view <iter> --remote`, is far closer to reachable than the `OpenDefaultStore`-centric framing suggests. Makes M4 option **view** viable at ~4h. |
| 4 | …but 15 `cmd/ailang` helpers take the **concrete** `*observatory.SQLiteBackend`, including `resolveChainID` (`chains_util.go:314`). 16 mentions across 5 files. | Option **view** is not pure wiring: a 16-signature widening pass. **+1.5h** |
| 5 | Every method those helpers call on their backend param is on the `Backend` interface **except `DB()`** — and `DB()` appears nowhere in `chains_util.go` / `chains_tree.go` / `chains_diagnostics.go`. The 12 raw-`DB()` escapes live in the `observatory_*` family plus `chains_live.go:203`. | A clean, defensible remote/local boundary exists. **De-risks option `view`.** |
| 6 | **This is the load-bearing one.** Firestore stores `eval_assessment` as a JSON **string**: `observatory_chains.go:386` → `{Path: "eval_assessment", Value: string(data)}`. Firestore cannot query inside a string. All six of `QueryEvalResults`' assessment filters (`model`, `language`, `benchmark_id`, `condition`, `eval_mode`, `stdout_ok`) are therefore **unqueryable server-side**. | Option **eval** must filter client-side after unmarshalling. **+4h and a permanent perf caveat** until the schema work is done. |
| 7 | `ListEvalChains` (`store_chains_eval.go:236`) is a **5-line wrapper** over `ListChains(SourceType: "eval_suite")`. `ListChains` IS on `Backend` (control: 1 hit) and IS implemented by Firestore. | Of the two `*Store`-only methods, only **`QueryEvalResults`** is genuinely missing. **−2h off option `eval`.** |
| 8 | No `firestore.indexes.json` anywhere in the repo — `find . -name '*.indexes.json'` → 0, control `find . -name 'package.json'` → 2 (so `find` does see JSON config in this tree). | The efficient variant of remote eval read needs out-of-band composite-index deploys **and** a backfill of existing cloud docs. That is a separate sprint, not a fast-follow. |
| 9 | `os.UserHomeDir()` with `HOME=""` returns `err = $HOME is not defined` (probed; control `HOME=/tmp` → `err=<nil>`). `DefaultDatabasePath()` (`models.go:14`) uses `$HOME`, **not** `AILANG_STATE_DIR`. | Both `chains_post.go` error branches are deterministically reachable from a hermetic `t.Setenv`. M2 needs no fakes. |
| 10 | Pristine baselines: `go test ./internal/observatory/...` rc=0 (2.6s) · `./cmd/ailang/...` rc=0 (48.4s) · `./internal/storage/firestore/...` rc=0 (0.4s) · package-scoped `go build` rc=0 · `go build ./...` **rc=1** (`cmd/wasm`). Named-test probe for M1 returns `[no tests to run]` today. | Every AC below is stated against a measured baseline. |
| 11 | `sprint_M-MISSION-LOOP-UNIFIED-TELEMETRY.json` is absent from `origin/dev` (control: `sprint_M-STD-YAML.json` **is** present on `origin/dev`); `origin/coordinator/task-d98bb271` exists on the remote. | #698's tail note confirmed. M3, ~0.25h. |
| 11b | **Root cause of #11, and it applies to this sprint's own JSON.** `.gitignore:77` ignores `.ailang/` with no negation. `git check-ignore --no-index` is rc=0 even for the already-tracked `sprint_M-STD-YAML.json`, and `git status --porcelain` after writing this sprint's JSON showed **only** the `.md`. New sprint JSONs are silently skipped by `git add`. | **Controller must use `git add -f`** for both sprint JSONs, or this sprint reproduces the exact defect M3 repairs. |
| 12 | `internal/observatory/store_chains_test.go` and `setupTestStore(t) *Store` (`store_test.go:11`) already exist; `internal/storage/firestore/*_test.go` use `package firestore` (not `firestore_test`), so `&ObservatoryStore{}` with a nil client is constructible in-package. | M1 and M2 need no new harness. |

**V1–V3 as handed to planning: all three reproduce.** The only correction is Discovery #1 and #3 above,
which narrow #698's *explanation* of the gaps without touching the gaps themselves.

---

## Which document wins where they disagree

The old plan, the design doc and the code disagree in three places. The precedence used throughout
this plan:

1. **On policy and intent → the DESIGN DOC wins.** Its Design Freeze item 3 ("Local analysis reads
   cloud — RATIFIED: yes, OPT-IN … `OpenDefaultStore()` keeps its local default") is Mark-ratified,
   2026-08-13. The prior sprint plan's M3 acceptance-criteria section silently dropped it. **An
   omission carries no authority.** The remote-read obligation stands.
2. **On facts about the code → the CODE wins over both documents.** The design doc's Architecture
   section asserts *"'rig data reaches the cloud' is a routing/wiring problem, not a porting one"*,
   and the freeze asserts *"every `ailang chains` / `eval-*` consumer inherits the option."* Measured
   (Discovery #3, #6, #7): true for `Backend`-shaped consumers, **false** for `ListEvalChains` and
   `QueryEvalResults`, which are `*Store`-only and which the remote backend physically cannot answer.
   Where a document asserts a code fact that a control-backed measurement refutes, the measurement
   wins and the document gets amended — see [Doc amendments](#doc-amendments-owned-by-this-sprint).
3. **On questions the design doc left explicitly open → the design doc's Deferred Decisions win.**
   *"Whether `--remote` is a flag, an env var, or a config key — agent may choose."* That remains the
   executor's call and no criterion below constrains it.

---

## Milestones

### M1_PINNED_ID_COLLISION_REDS_UNDER_MUTATION (~55 LOC, ~1.0h)

**#698 part 2.** The pinned-stage-id guard at `internal/observatory/store_chains.go:334` is what makes
M3's cross-store identity hold, and nothing tests it. `TestPostIteration_PinnedIDIsNotSilentlyReplaced`
(`iteration_post_test.go:329`) is named for the property and **passes with the property removed** — it
posts four stages with distinct pinned ids onto a fresh chain, so no `UNIQUE constraint` is ever
raised and the retry branch is never entered.

**Tasks:**

- [ ] Add `TestCreateStage_PinnedIDSurvivesUniqueConstraintRetry` to
      `internal/observatory/store_chains_test.go`, driving `(*Store).CreateStage` **directly** (the
      mutation lives in `CreateStage`; going through `PostIteration` adds a layer that can mask it).
      Shape, per Discovery #1:
      1. `setupTestStore(t)`; create a chain; `CreateStage` with `ID: "pinned-stage-1"` → succeeds.
      2. `CreateStage` **again** with the same `ID: "pinned-stage-1"` → the INSERT violates
         `id TEXT PRIMARY KEY`, SQLite reports `UNIQUE constraint failed: chain_stages.id`, and the
         retry branch at :333 is entered on every one of the 5 attempts.
      3. Assert `err != nil` **and** that the store contains exactly one stage carrying that id and
         no extra stage under a regenerated id.
- [ ] Add the non-pinned counterpart in the same test (table-driven or a sibling): an **empty** `ID`
      under the same duplicate condition still regenerates and succeeds. Without this arm the guard
      could be "fixed" by disabling regeneration entirely and M1 would not notice.
- [ ] Comment the test with *why* the PK route is used rather than a `stage_number` race: the
      `stage_number` collision needs two interleaving writers and would be flaky, whereas both paths
      enter the identical branch. Record that #698's "only reached on a real `stage_number` collision"
      is refuted.

**Acceptance criteria** *(each names the command whose output changes if the task is not done)*:

- **Baseline, measured today:**
  `go test ./internal/observatory/... -run 'TestCreateStage_PinnedIDSurvivesUniqueConstraintRetry' -count=1`
  → `ok … [no tests to run]`. After M1 the same command runs the test and is rc=0 **without** the
  `[no tests to run]` marker. *Not done ⇒ the marker is still there.*
- **MUTATION — the criterion that makes M1 real.** With
  `sed -i '' '334s/if req.ID == "" {/if true {/' internal/observatory/store_chains.go`
  asserted **LANDED** (sha256 differs from
  `54432ddb531082a0160eca2ffe86bb45df8b06563490d7dc5e3bae8dca31b624`) and **BUILDING**
  (`go build ./internal/observatory/... ./cmd/ailang/...` rc=0), the named test **FAILS** (rc≠0).
  *Not done ⇒ the mutant survives, exactly as it does today.*
- **Restore proof:** after `cp` restore, `shasum -a 256 internal/observatory/store_chains.go` equals
  the baseline sha byte-for-byte.
- **No collateral:** `go test ./internal/observatory/... -count=1` rc=0 (baseline: rc=0, 2.6s), and
  `TestPostIteration_PinnedIDIsNotSilentlyReplaced` still passes — M1 **adds** coverage, it does not
  replace a test that is merely narrow.
- **No production code changed**: `git diff --stat` touches `internal/observatory/store_chains_test.go`
  only. *(Reported by the executor; the controller makes the commit.)*

**Risks:** the second `CreateStage` returns after 5 identical failures, so the test is ~5 round-trips
on an in-memory/temp SQLite DB — sub-millisecond, no sleep, no goroutine. If `setupTestStore` returns
a shared-cache DB, use a fresh chain id per subtest to stay hermetic.

---

### M2_ERROR_BRANCH_NEGATIVE_ARMS (~120 LOC, ~2.5h)

**#698 part 3.** Five error branches shipped in `769d920a0` with nothing pinning them. Four are
reachable and get one neutering mutation each; the fifth is unreachable and gets a finding instead of
a fake test.

Neutering form is always `if false && <cond> {` — **never** deleting the branch — so the identifiers
inside the body stay referenced and the package still compiles (Go's unused check is
reference-based). A build that fails is a mutation that was never tested.

**Tasks:**

- [ ] `cmd/ailang/chains_post_dualwrite_test.go` — two arms against `checkRemoteIsElsewhere`
      **directly** (it is a package-`main` function; calling it directly avoids opening the real
      `~/.ailang/state/observatory.db` that `openPostTargets` would touch):
      - `TestCheckRemoteIsElsewhere_UnresolvableHomeIsAnError`: `t.Setenv("AILANG_STATE_DIR", "")` +
        `t.Setenv("HOME", "")` → `os.UserHomeDir()` errors (Discovery #9) → assert the returned error
        is non-nil and mentions `cannot resolve`.
      - `TestCheckRemoteIsElsewhere_SelfTargetIsRejected`: `t.Setenv("HOME", t.TempDir())` +
        `t.Setenv("AILANG_STATE_DIR", filepath.Join(home, ".ailang", "state"))`, mode `local` →
        assert non-nil and mentions `resolves to this node's own observatory`.
      - Plus a **positive control arm** in the same file: mode `gcp` returns `nil` (the function
        short-circuits for non-local modes) and mode `local` with a genuinely different
        `AILANG_STATE_DIR` returns `nil`. Without these, a mutation to `return fmt.Errorf(...)` on
        *every* path would still be green.
- [ ] `internal/storage/firestore/observatory_chains_test.go` (**new file**, `package firestore`) —
      two arms against `UpdateStageEvalAssessment` using `&ObservatoryStore{}` with a **nil** client.
      Both guards return before touching `s.client` (verified at :375-380), so no emulator, no
      credentials, no network:
      - `TestUpdateStageEvalAssessment_RequiresStageID`: `stageID: ""` → error mentions
        `stage_id is required`.
      - `TestUpdateStageEvalAssessment_RequiresAssessment`: valid stageID, `assessment: nil` → error
        mentions `assessment is required`.
      - **Positive control, mandatory**: with both a valid stageID and a non-nil assessment, the call
        must get *past* both guards. With a nil client it will then panic or error on the Firestore
        call — assert it does **not** return either guard message (e.g. via `recover()` or by
        asserting the error text differs). Without this arm, `return fmt.Errorf("stage_id is
        required")` as the function's *first unconditional* line would pass both negative arms.
- [ ] `TestEvalAssessment_IsAlwaysMarshalable` in the same new file: `json.Marshal` a fully-populated
      `obs.EvalAssessment` and assert no error. This is the only meaningful guard for branch 5 (see
      finding below): it fails the day someone adds a `chan`, `func`, or `map[SomeType]…` field, which
      is the only way that branch could ever become live.
- [ ] Record the branch-5 finding in the test file's header comment and in this plan's
      [Findings](#findings-for-the-issue-and-the-doc) section.

**Acceptance criteria:**

- **Baseline:** `go test ./cmd/ailang/... -count=1` rc=0 (48.4s measured) and
  `go test ./internal/storage/firestore/... -count=1` rc=0 (0.4s measured). Both still rc=0 after M2.
- **`internal/storage/firestore/observatory_chains_test.go` did not exist before this sprint.**
  Verifiable: `git log --oneline -1 -- internal/storage/firestore/observatory_chains_test.go` is empty
  on `644cf178a`; control: the same command on `internal/storage/firestore/cache_test.go` is non-empty.
- **MUTATION 1** — `cmd/ailang/chains_post.go` line 203,
  `if err != nil {` → `if false && err != nil {`. LANDED (sha ≠
  `04ba78d9eec6a0ac207565f7c16f16069bef0ae944da9b3b264e88d98fdb3a31`) + BUILDING (rc=0) ⇒
  `go test ./cmd/ailang/... -run 'TestCheckRemoteIsElsewhere' -count=1` **FAILS**.
- **MUTATION 2** — `cmd/ailang/chains_post.go` line 209,
  `if filepath.Clean(remoteDir) == filepath.Clean(localDir) {` →
  `if false && filepath.Clean(remoteDir) == filepath.Clean(localDir) {`. LANDED + BUILDING ⇒ the same
  `-run` **FAILS**.
- **MUTATION 3** — `internal/storage/firestore/observatory_chains.go` **line 375** (NOT 244 — the
  identical text also occurs there), `if stageID == "" {` → `if false && stageID == "" {`.
  LANDED (sha ≠ `73b7267458975e8379c854ca52c97e89e1b4057851f58fe02a8ac852b4995af8`) + BUILDING ⇒
  `go test ./internal/storage/firestore/... -run 'TestUpdateStageEvalAssessment' -count=1` **FAILS**.
- **MUTATION 4** — same file, line 378, `if assessment == nil {` → `if false && assessment == nil {`.
  LANDED + BUILDING ⇒ the same `-run` **FAILS**.
- **All four restores are byte-identical**: post-restore `shasum -a 256` equals each baseline sha.
- **Branch 5 is reported, not faked.** No test in this sprint claims to cover
  `failed to marshal eval assessment`. The finding is written into the issue and the plan, and
  `TestEvalAssessment_IsAlwaysMarshalable` is labelled as a *type guard*, not as coverage of that line.
- **No production code changed** by M2.

**Risks:** the nil-client positive-control arm may panic rather than error inside the Firestore SDK.
That is fine and expected — assert on "did not return a guard message", recovering the panic, and say
so in a comment. Do **not** weaken the arm to a no-op; a positive control that cannot fail is the exact
failure mode this whole sprint exists to correct.

---

### M3_RECONCILE_ORPHANED_SPRINT_JSON (~1 file, ~0.25h)

**#698 tail note**, confirmed (Discovery #11): the sprint's own progress artifact never reached `dev`.

**Tasks:**

- [ ] Read `.ailang/state/sprints/sprint_M-MISSION-LOOP-UNIFIED-TELEMETRY.json` out of
      `origin/coordinator/task-d98bb271` with `git show` (a **read**), and write it into the worktree
      at the same path. No branch switch, no checkout, no fetch of that branch into a working tree.
- [ ] If the file's milestone records claim milestones that #697 did not land, correct the `passes`
      / `notes` fields to match what actually landed, and say so in `notes`. Do **not** invent history.

**Acceptance criteria:**

- `jq -e . .ailang/state/sprints/sprint_M-MISSION-LOOP-UNIFIED-TELEMETRY.json` rc=0. *Not done ⇒
  the file does not exist and `jq` rc≠0.*
- `git cat-file -e origin/dev:.ailang/state/sprints/sprint_M-MISSION-LOOP-UNIFIED-TELEMETRY.json`
  is rc≠0 **today** (measured; control: the same command for `sprint_M-STD-YAML.json` is rc=0), and
  after the controller commits, rc=0.
- The restored file carries no unfilled scaffold tokens: every `.features[].id` is a real milestone
  name, and the `create_sprint_json.sh` placeholder id does not appear.
- No other file under `.ailang/state/sprints/` is modified.

**Root cause of the orphan — found during planning, and it will bite this sprint too.**
`.gitignore:77` ignores `.ailang/` outright, with **no negation rule**:

```bash
git check-ignore -v --no-index .ailang/state/sprints/sprint_M-STD-YAML.json
# -> .gitignore:77:.ailang/   ...   rc=0     an ALREADY-TRACKED sprint JSON is still matched by the rule
git check-ignore -v .ailang/state/sprints/sprint_M-TELEMETRY-REMOTE-READ-FASTFOLLOW.json
# -> .gitignore:77:.ailang/   ...   rc=0     and a NEW one is ignored outright
git status --porcelain
# -> shows ONLY the .md — the new JSON is invisible
```

Existing sprint JSONs survive only because they are already tracked. A **new** one never appears in
`git status` and is silently skipped by `git add`. That is almost certainly how
`sprint_M-MISSION-LOOP-UNIFIED-TELEMETRY.json` ended up on `coordinator/task-d98bb271` and nowhere
else — and it applies verbatim to this sprint's own JSON.

> **CONTROLLER ACTION, both sprint JSONs:** stage with **`git add -f`**. A plain `git add` will
> reproduce the exact defect this milestone exists to repair.

The durable fix (a `!.ailang/state/sprints/` negation in `.gitignore`) is a production change and is
**out of scope for this fast-follow** — recommend raising it as its own issue.

**Risks:** the `git add -f` step is the whole milestone. The executor performs **no git write
operations** — `git show` is a read; the controller builds the commit.

---

<a id="m4"></a>

### M4_OPT_IN_REMOTE_READ — RULING RECEIVED (`D-15` = `view`) (~400 LOC, ~7h)

**#698 part 1**, and the ratified Design Freeze item that the prior plan's acceptance criteria lost.
**This milestone was decision-gated and is now EXECUTABLE.** The ruling, verbatim, from
[`design_docs/v1-mission.md`](../../v1-mission.md), *"RATIFIED (attended, 2026-08-14 morning)"*,
item 3:

> **`D-15`: `view`** — opt-in remote read reaches the `Backend`-shaped `chains` view commands
> only; `eval-*` errors loudly on `--remote` (turning assumed demand into a dated signal). The
> ratified freeze item is knowingly NARROWED by this ruling.

Two clauses, both load-bearing, and the second is not decoration:

1. **`Backend`-shaped** is the scope test, applied per **command**, not per family. A `chains`
   subcommand whose read path calls a method that is not on `observatory.Backend` is **not**
   `Backend`-shaped and does **not** get remote read — it joins the loud-refusal set. Re-derivation
   below found **three such commands the option table did not name**.
2. **`eval-*` errors loudly** is the *measurement instrument* for the narrowing. It is why the
   narrowing is a cheap experiment rather than a deletion, and it therefore gets its own task
   (**T5**) and its own failing-capable criterion. Silently falling back to local would destroy the
   signal the ruling was chosen for, and violates Critical Principle 2.

#### The constraint that made this a decision — RESOLVED BY RULING, retained as evidence

The freeze says *"every `ailang chains` / `eval-*` consumer inherits the option."* That sentence is
**not satisfiable by wiring**, and the reason is one line of code:

```go
// internal/storage/firestore/observatory_chains.go:386
{Path: "eval_assessment", Value: string(data)},
```

Firestore stores the whole eval assessment as an **opaque JSON string**. `(*Store).QueryEvalResults`
(`store_chains_eval.go:68`) filters on six fields *inside* that blob with `json_extract(...)`, joins
to `execution_chains` for the cohort window, and prefix-matches `chain_id` with `LIKE`. Firestore can
do none of the three against a string. The method is also not on `Backend` at all
(`Backend.QueryEvalResults` → 0 hits; controls `Backend.GetChain` → 1, `Backend.GetChainStages` → 1).

So the eval consumers are a **porting** problem, while the chains consumers are the wiring problem the
design doc described. They were merged into one sentence in the freeze and have to be separated now.

*(Re-verified at `bad8f3647`: `grep -rn 'QueryEvalResults' internal/observatory/backend*.go | wc -l`
→ **0**; same-path control `grep -rn 'GetChain(' internal/observatory/backend*.go | wc -l` → **7**.
And `internal/storage/firestore/observatory_chains.go:386` still reads
`{Path: "eval_assessment", Value: string(data)}` verbatim. The constraint holds.)*

#### The three options, costed from the code — RESOLVED BY RULING (`view` chosen)

> **Status: this table is no longer a question.** It is retained because it is the evidence Mark
> ruled on, and because the `eval` column is the pre-registered cost of the revisit trigger in
> M4 task **T5**. Costs are as
> priced at `644cf178a`; the `view` column's LOC/hours are **superseded** by the re-derivation
> below, which found more scope than it priced.

| | **`view`** — option (c) — **CHOSEN** | **`eval`** — option (b) | option (a) |
|---|---|---|---|
| Scope | Remote read for the `Backend`-shaped `chains` commands only. `eval-*` documented as local-only and **erroring** on `--remote` | `view` **plus** the three eval read methods behind a narrow `ChainReader` | Same functional scope as `eval`, but by promoting the 2 methods onto the 90-method `Backend` |
| New helper `openChainsReadBackend` (env `AILANG_CHAINS_READ` + `--remote`, resolving through the same `storage.NewBackendsForMode` the write half already uses at `chains_post.go:181`) | ~70 LOC | ~70 LOC | ~70 LOC |
| Widen 15 helpers from `*observatory.SQLiteBackend` → `observatory.Backend` (Discovery #4; every method they call is on `Backend` except `DB()`, which is absent from these files — Discovery #5) | ~16 edits | ~16 edits | ~16 edits |
| Swap the ~20 chains-family `NewSQLiteBackendFromPath` sites to the helper | ~20 LOC | ~20 LOC | ~20 LOC |
| Loud refusal for the local-only surface — `chains_live` (raw SQL tail), the `observatory_*` family (12 `DB()` escapes), and (under `view`) every `eval-*` command. **Must error, not silently read local** (Critical Principle 2) | ~30 LOC | ~30 LOC | ~30 LOC |
| `ChainReader` interface; `*Store` already satisfies all three methods, so zero adapter | — | ~12 LOC | — |
| Firestore `ListEvalChains` — a delegate to its own `ListChains(SourceType:"eval_suite")`, because that is literally all the SQLite one is (Discovery #7) | — | ~8 LOC | ~8 LOC |
| Firestore `QueryEvalResults` — client-side filtering after unmarshal (Discovery #6), 2-pass cohort join with `in`-chunking at Firestore's 30-value cap, prefix range trick for short `chain_id` | — | ~180 LOC | ~180 LOC |
| Switch 7 consumer sites (`loader_chains.go` ×3, `eval_chains.go` ×4) | — | ~40 LOC | ~40 LOC |
| Boilerplate in the 4 other `Backend` impls — none of them embed, each spells out every method; Jaeger and GCPTrace already carry 33 `not supported` stub lines apiece | — | **0** | ~45 LOC |
| Tests | ~110 LOC | ~200 LOC | ~200 LOC |
| **Total** | **~230 LOC · ~4h** | **~550 LOC · ~9h** | **~595 LOC · ~10h** |

There is a fourth shape, **`schema`** — denormalize the six filter fields onto the stage document,
add composite indexes, backfill existing cloud docs — which is the only version that is *efficient*
at scale. It is **not costed as an option here** because there is no `firestore.indexes.json` in the
repo at all (Discovery #8, with a positive `find` control), so index management today is out-of-band
console/gcloud work needing a human, and the backfill is its own command. **~2–3 days. Not a
fast-follow.** It is named only so that "`eval` ships a full-collection scan" is understood as a
deliberate, temporary posture rather than an oversight.

#### Recommendation: **`view`** — RESOLVED BY RULING, this is the recorded rationale

> **Status: accepted.** Mark ruled `view` on 2026-08-14. Reasoning retained verbatim because point 4
> is not merely a justification — it is the **specification of T5**, and point 5 is the reason the
> ruling had to be human.

Reasoning, in the order it should be challenged:

1. **It is what was actually asked for.** Mark's authorization sentence is
   *"I'm looking to be able to follow a whole mission loop across providers in our data"*, and the
   design doc's own worked example is `ailang chains view <iter-191> --remote`. Option `view` delivers
   that example literally, and with it the design doc's Primary Goal for the mission-loop case.
2. **The freeze's `eval-*` clause was a wiring assumption, and the code refutes it.** It was written
   before anyone had read that `QueryEvalResults` is not on `Backend` and that Firestore keeps
   `eval_assessment` as a string. Discoveries #6 and #7 are new information, and the freeze deserves
   to be re-decided against them rather than executed against them.
3. **`eval` now buys code that `schema` would replace.** The client-side-filter implementation is
   correct but scans; when the volume argument arrives it gets rewritten. Building it now is paying
   ~5h for something with a known replacement date and no known user.
4. **`view` makes the demand for `eval` measurable instead of assumed.** Under `view`, an `eval-*`
   command given `--remote` **errors loudly** — so the first person who wants it produces a dated,
   attributable signal. Under `eval` we would never learn whether anyone needed it. Pre-registered
   trigger to revisit: *the first `--remote` / `AILANG_CHAINS_READ` invocation against an `eval-*`
   command.* That is the cheap experiment, and it is the one that respects
   "if `--remote` turns out to be always-passed, flipping the default later is one line **with evidence
   behind it**" — the exact reasoning Mark used to ratify opt-in in the first place.
5. **Choosing `view` narrows a ratified freeze item, which is precisely why it is not my call.**

**If the answer is `eval`, the mechanism is option (b) `ChainReader`, not option (a) `Backend`.** That
part is the agent's call and needs no human: (b) is ~45 LOC cheaper, requires zero changes to
`JaegerBackend` / `GCPTraceBackend` / `CompositeBackend` / `SQLiteBackend`, and avoids adding a 91st
and 92nd method to an interface that already forces two of its five implementations to stub 33 lines
apiece. (a) buys nothing over (b) and grows the god-interface.

#### The one-word question for Mark — **ANSWERED: `view`** (attended, 2026-08-14)

> ~~**How far should `--remote` reach — `view` or `eval`?**~~
>
> **`view`.** Remote read for the `Backend`-shaped `chains` view commands. `eval-*` stays local and
> **errors** when you pass `--remote`. The `eval` option (~9h, full-collection Firestore scan a
> later `schema` sprint would replace) is **not** taken, and its revisit trigger is pre-registered
> in T5.

---

#### Re-derivation at `bad8f3647` — six measurements, and four of them changed the milestone

Everything the ruling's scope depends on was re-measured on the pristine worktree at `bad8f3647`
during this planning pass. Items marked **REFUTES** contradict a claim in the option table above,
in this plan's Discovery section, or in the `M4` sprint-JSON notes. Each has a same-path control,
because an empty search is a claim and not a fact.

| # | Measurement (command + control) | Effect on M4 |
|---|---|---|
| **D1** | **REFUTES the "~20 chains-family `NewSQLiteBackendFromPath` sites" row.** *Mentions* ≠ *call sites* ≠ *chains-family read sites*. Measured: `grep -rn 'NewSQLiteBackendFromPath' cmd/ailang/ \| wc -l` → **38** *mentions*; `grep -rn 'NewSQLiteBackendFromPath(' cmd/ailang/ --include='*.go' \| grep -v '_test.go' \| wc -l` → **35** call sites across all of `cmd/ailang`; and in `chains*.go` non-test specifically → **17**, enumerated function-by-function in the table below (control: the same grep for `NewSQLiteBackendFromPathZZZ` → **0**, so the instrument does discriminate). Of the 17: **13 swap**, **3 refuse-only**, **1 is the write half and is not touched**. | The swap task is **13** sites, not ~20 — and 3 of the remaining 4 are a *new refusal obligation*, not a no-op. |
| **D2** | **REFUTES "15 helpers / ~16 edits".** `grep -rn '\*observatory\.SQLiteBackend' cmd/ailang/ --include='*.go' \| wc -l` → **16** (same-path control `grep -rn 'observatory\.Backend' cmd/ailang/ \| wc -l` → **13**). Per file: `chains_util.go` **5**, `chains_tree.go` **5**, `chains_diagnostics.go` **2**, `observatory_metrics.go` **3**, `chains_live.go` **1**. But `observatory_metrics.go`'s 3 are in the `observatory_*` family, which is **explicitly Out of Scope**, and `chains_live.go`'s 1 is a *type assertion* (`backend.(*observatory.SQLiteBackend)`, line 199), not a signature — it is the local-only escape and must **stay** concrete. | The widening pass is **12** signatures in **3** files, not 16 in 5. **−0.4h**, and it removes the temptation to widen `observatory_metrics.go` out of scope. |
| **D3** | **REFUTES Discovery #5's "a clean, defensible remote/local boundary exists"** — as stated for the whole chains family. Discovery #5 only grepped `DB()`. `Store()` is a *second* escape hatch with the same effect and it was never grepped: `grep -rn '\.Store()' cmd/ailang/ --include='*.go' \| grep -v '_test.go'` → **4** hits — `chains_stats_cvs.go:62`, `chains_live.go:203`, `trace_local.go:46`, `trace_local.go:189` (control: `grep -c '\.Store()' cmd/ailang/chains_util.go` → **0**, so the grep is not matching everything). `Store` is **not** on `Backend` (`grep -cE '^[[:space:]]+Store\(' <Backend iface block>` → **0**; control `GetChain` → **1**). | **`ailang chains stats --cost-per-verified-success` cannot go remote.** It is a chains *view* command, and it is not `Backend`-shaped. New refusal. |
| **D4** | **Two more chains-family methods are on `*SQLiteBackend` but NOT on `Backend`.** Extracting the 90-method `Backend` block and checking every `backend.<M>(` call in the 11 chains read files: `chains.go` → **`GetChainJourney`** NOT on `Backend` (`chains.go:533`, inside `chainsJourneyCommand`); `chains_find.go` → **`GetTaskSpanSummary`** NOT on `Backend` (`chains_find.go:86`, the `--task` span-summary fallback). Both are defined only on `*SQLiteBackend` (`backend_sqlite.go:457` and `:403`) and on `*Store`. Controls in the same run: `chains_chat.go`, `chains_data.go`, `chains_diagnostics.go`, `chains_diff.go`, `chains_stats.go`, `chains_stats_mission.go`, `chains_tree.go`, `chains_util.go` → **all methods on `Backend`**. | **`ailang chains journey` cannot go remote**, and **`chains find --task`'s fallback branch cannot**. Two more refusals the option table did not name. |
| **D5** | **A Critical-Principle-2 violation sits directly in the `view` path, and M4 would be the thing that exposes it.** Scanning all 94 `ObservatoryStore` methods for a silent stub body: exactly **one** — `internal/storage/firestore/observatory_chains.go:544` `GetMissionRollups` → `return nil, nil`, commented *"not supported on the Firestore backend … returns empty"*. Its **only** consumer is `cmd/ailang/chains_stats_mission.go:66`. Today nothing can reach it remotely, so the stub is inert; the moment `chains stats --by-mission` accepts `--remote gcp` it prints *"no missions"* for a store that was never asked. | New task **T6**. The stub must become a loud error before the path that reaches it exists. ~8 LOC + a mutation. |
| **D6** | **The rest of the `view` surface genuinely is servable — verified method-by-method, not assumed.** All 16 methods the swappable commands call are implemented by `firestore.ObservatoryStore`: `GetChain`, `GetChainStages`, `ListChains`, `GetSessionTools`, `ListSpans`, `GetChatMessagesBySession`, `GetChatMessagesByTaskID`, `GetChainStatsByAgent`, `GetChainStatusCounts`, `GetCostRollup`, `GetChainByGitHubIssue`, `GetChainByMessageID`, `GetChainByTaskID`, `GetSession`, `Close` — each located at a named `func (s *ObservatoryStore) …` (control: `GetMissionRollups` located too, which is how D5 was found). And `*SQLiteBackend` satisfies `Backend` (`backend_sqlite.go:526 var _ Backend = (*SQLiteBackend)(nil)`), so widening a parameter never breaks an existing local caller. | The `view` scope is real, and **12 of 17** chains sites can be swapped with no porting at all. Confirms the ruling was priced on something true. |

**Per-site disposition of all 17 chains-family `NewSQLiteBackendFromPath` call sites** (`awk` over
`cmd/ailang/chains*.go`, non-test, printing the enclosing `func`):

| Site | Function | Disposition |
|------|----------|-------------|
| `chains.go:112` | `chainsListCommand` | **SWAP** |
| `chains.go:204` | `chainsViewCommand` | **SWAP** |
| `chains.go:353` | `chainsActiveCommand` | **SWAP** |
| `chains.go:519` | `chainsJourneyCommand` | **REFUSE** — `GetChainJourney` not on `Backend` (D4) |
| `chains_chat.go:44` | `chainsChatCommand` | **SWAP** |
| `chains_data.go:190` | `getObservatoryBackend` | **SWAP** — already returns `observatory.Backend`; the cleanest seam in the file set |
| `chains_diagnostics.go:35` | `chainsDiagnoseCommand` | **SWAP** |
| `chains_diagnostics.go:294` | `chainsHealthCommand` | **SWAP** |
| `chains_diff.go:33` | `chainsDiffCommand` | **SWAP** |
| `chains_find.go:54` | `chainsFindCommand` | **SWAP**, and the `--task` fallback at `:86` **REFUSES** under remote (D4) |
| `chains_live.go:66` | `chainsLiveCommand` | **REFUSE** — raw SQL tail, Out of Scope |
| `chains_post.go:157` | `openPostTargets` | **UNTOUCHED** — this is the write half, and it already routes `--cloud`/`AILANG_CHAINS_CLOUD` through `storage.NewBackendsForMode` |
| `chains_stats.go:83` | `chainsStatsCommand` | **SWAP** |
| `chains_stats_cvs.go:47` | `chainsStatsCostPerVerifiedSuccess` | **REFUSE** — `backend.Store()` escape at `:62` (D3) |
| `chains_stats_mission.go:49` | `chainsStatsByMission` | **SWAP**, gated on T6 landing first (D5) |
| `chains_tree.go:58` | `chainsTreeCommand` | **SWAP** |
| `chains_util.go:19` | `runChainsInteractive` | **SWAP** |

#### Surface form — constrained only by the write-half precedent

The design doc's Deferred Decisions leave *"whether `--remote` is a flag, an env var, or a config
key"* to the executor, and this plan does **not** overturn that. The one binding constraint is
**consistency with the write half**, which is already shipped and already reviewed:

- flag `--cloud <mode>` on the subcommand's own `FlagSet` (`chains_post.go:88`),
- defaulting to env `AILANG_CHAINS_CLOUD` when the flag is empty (`chains_post.go:167`),
- resolved through `storage.NewBackendsForMode(ctx, storage.Mode(mode))` (`chains_post.go:181`) —
  *"the same local/gcp/hybrid resolution `AILANG_STORAGE` goes through … deliberately NOT a second
  selector, since two selectors drift."*

The read half must use the **same three-part shape** and the **same** `NewBackendsForMode` entry
point. Measured at `bad8f3647`: `--remote` does not exist (`grep -rn '"remote"' cmd/ailang/` → **1**
hit, and it is a `git remote get-url` argument at `coordinator_cloud_github.go:88`; same-path control
`--cloud` → the flag at `chains_post.go:88`), and `AILANG_CHAINS_READ` → **0** hits repo-wide
(control `AILANG_CHAINS_CLOUD` → **4**). Both names are free. Naming them `--remote` /
`AILANG_CHAINS_READ` is the executor's call; **routing them anywhere other than
`NewBackendsForMode` is not.**

---

#### Tasks

**T1 — the read-backend resolver.** New file `cmd/ailang/chains_read_backend.go`. A single
`openChainsReadBackend(ctx, remoteFlag string) (observatory.Backend, func(), error)` that mirrors
`openPostTargets`' resolution exactly: flag → `AILANG_CHAINS_READ` → **empty means local**, and any
non-empty mode goes through `storage.NewBackendsForMode`. Local default returns
`observatory.NewSQLiteBackendFromPath(observatory.DefaultDatabasePath())` unchanged, so an
invocation with no flag and no env is byte-for-byte the behaviour that ships today. **~55 LOC.**

**T2 — widen the 12 chains-family helper signatures** in `chains_util.go` (5), `chains_tree.go` (5)
and `chains_diagnostics.go` (2) from `*observatory.SQLiteBackend` to `observatory.Backend`. Do
**not** touch `observatory_metrics.go`'s 3 (Out of Scope) or `chains_live.go`'s type assertion (it
is the local-only escape). **~12 LOC.**

**T3 — swap the 13 SWAP sites** in the table above to `openChainsReadBackend`, and register the
surface-form flag on each of those subcommands' `FlagSet`s. **~40 LOC.**

**T4 — loud refusal for the five non-`Backend`-shaped chains surfaces**: `chains live`,
`chains journey`, `chains stats --cost-per-verified-success`, `chains find --task`'s span-summary
fallback, and — stated explicitly so a later reader does not mistake it for an oversight — the
`observatory_*` family, which stays Out of Scope and does not gain the flag at all. Each refusal
must **exit non-zero** and name (a) the command, (b) *why* it is local-only (the specific method
that is not on `Backend`), and (c) that this is the `D-15` narrowing. **~45 LOC.**

**T5 — `eval-*` errors loudly on `--remote`: the instrument the ruling was chosen for.** A shared
guard, applied before dispatch, over the **14** eval-family commands measured in
`cmd/ailang/main.go:275-315` (`eval`, `eval-analyze`, `eval-compare`, `eval-paired`, `eval-matrix`,
`eval-sweet-spot`, `eval-summary`, `eval-report`, `eval-suite`, `eval-elo`, `eval-trend`,
`eval-publish`, `eval-chains`, and `eval-chains`' subcommands). Two arms, and the asymmetry between
them is a deliberate judgment recorded here rather than buried:

- **Explicit `--remote` on an `eval-*` command → hard error, exit non-zero.** The user asked for
  something the tool will not do; answering locally would be a silent fallback (Critical Principle
  2) *and* would destroy the signal.
- **`AILANG_CHAINS_READ` set in the environment while an `eval-*` command runs → a loud one-line
  stderr WARNING naming the ignored variable and the command, and the command proceeds locally.**
  An exported env var is a node-wide default; hard-failing it would make the variable unexportable
  and would break the rig the moment anyone sets it globally. The warning is still dated and still
  attributable, which is all the instrument needs.

The message on **both** arms must be greppable and must carry the revisit trigger, so the first
real demand is a searchable event and not a shrug. **~45 LOC.**

**T6 — remove the silent-empty Firestore `GetMissionRollups` stub** (D5). Change
`internal/storage/firestore/observatory_chains.go:544-546` from `return nil, nil` to a typed,
non-nil error naming the backend and the method, and update the comment above it. Its only consumer
already prints the error and exits non-zero (`chains_stats_mission.go:67-70`), so the CLI behaviour
falls out. **~8 LOC.** *This is a production change and it is deliberately in M4's scope — M4 is
what makes the stub reachable.*

**T7 — the doc amendment, in this same change** (owned by this sprint, see
[Doc amendments](#doc-amendments-owned-by-this-sprint)). Amend Design Freeze item 3 of
`design_docs/planned/v0_33_2/m-mission-loop-unified-telemetry.md:115-117` to record the **decision**
`D-15` = `view`, dated and attributed, together with the **measurement that motivated it** (the
`eval_assessment`-as-string constraint and `QueryEvalResults` not being on `Backend`) and the
narrowing found during execution (D3/D4: three chains commands are not `Backend`-shaped either). The
doc must record a decision, not a pending one, and must not leave *"every `ailang chains` /
`eval-*` consumer inherits the option"* standing unqualified. **~15 lines of prose.**

**Tests** — `cmd/ailang/chains_read_backend_test.go` (new) and
`internal/storage/firestore/observatory_chains_test.go` (extend the file M2 created). Follow the M2
precedent: call the package-`main` functions **directly** rather than driving `os.Exit` paths, and
keep every arm hermetic (`t.Setenv`, no network — `NewGCPBackends` fails on an empty
`AILANG_CLOUD_PROJECT` *before* it constructs a Firestore client, which is what makes the remote arm
testable offline). **~195 LOC.**

#### Acceptance criteria

*Every criterion below names the command whose output would be **different** if its task were
skipped. Baselines are measured at `bad8f3647` and quoted with the command that produced them.*

**Global (all of M4):**

- **No regression:** `go test ./cmd/ailang/... -count=1` rc=0 (baseline rc=0, measured this pass) ·
  `go test ./internal/storage/firestore/... -count=1` rc=0 · `go test ./internal/observatory/... -count=1`
  rc=0 · `go build ./internal/observatory/... ./cmd/ailang/... ./internal/storage/...` rc=0
  (baseline rc=0). **Never** `go build ./...` — baseline rc=1 on `cmd/wasm`.
- **The default did not move.** With no flag and no `AILANG_CHAINS_READ`, `openChainsReadBackend`
  returns a `*observatory.SQLiteBackend` over `observatory.DefaultDatabasePath()`. *Not done ⇒ the
  ratified "local stays the default" is broken, which is the one thing the freeze was explicit about.*

**T1 — resolver:**

- `grep -c 'storage.NewBackendsForMode' cmd/ailang/chains_read_backend.go` ≥ **1**.
  *Baseline: the file does not exist, `grep` rc=2.* Same-path control that the constraint is real:
  `grep -c 'storage.NewBackendsForMode' cmd/ailang/chains_post.go` → **1** today.
  *Not done, or routed around `NewBackendsForMode` ⇒ 0, and the read half has grown the second
  selector `chains_post.go:181` exists to prevent.*
- `TestOpenChainsReadBackend_DefaultsToLocal`: no flag, `t.Setenv("AILANG_CHAINS_READ", "")` →
  returned backend is a `*observatory.SQLiteBackend` and `err == nil`.
- `TestOpenChainsReadBackend_RemoteRoutesThroughStorageMode`: flag `"gcp"`,
  `t.Setenv("AILANG_CLOUD_PROJECT", "")` → `err != nil` and the message contains
  `AILANG_CLOUD_PROJECT` (the string `NewGCPBackends` produces, `internal/storage/backend.go`).
  Hermetic, offline, no credentials. *This is the arm that proves the flag reaches
  `NewBackendsForMode` at all; a stub that returned local would pass every other criterion.*
- `TestOpenChainsReadBackend_EnvIsTheFallbackNotTheOverride`: flag `""` + env `"gcp"` → routes
  remote; flag `"local"` + env `"gcp"` → routes **local**. Mirrors `chains_post.go:165-168`
  precedence exactly. *Positive control against a resolver that only ever reads the env.*
- **MUTATION M4-1 (kills T1).** Mandated literal, which the executor MUST write verbatim so the
  address resolves: `mode := remoteFlag`. Resolve `L` with
  `grep -n 'mode := remoteFlag' cmd/ailang/chains_read_backend.go` (assert exactly one hit), then
  `sed -i '' "${L}s/mode := remoteFlag/mode := \"\"/"`. LANDED (sha differs) + BUILDING
  (package-scoped rc=0) ⇒
  `go test ./cmd/ailang/... -run 'TestOpenChainsReadBackend' -count=1` **FAILS**.
  *Not done ⇒ the mutant survives and `--remote` is decorative.*
- **Restore proof:** post-`cp`-restore `shasum -a 256` is byte-identical. Never `git checkout --`.

**T2 — widening:**

- `grep -c '\*observatory\.SQLiteBackend' cmd/ailang/chains_util.go cmd/ailang/chains_tree.go cmd/ailang/chains_diagnostics.go`
  → **`5 / 5 / 2`** today (measured); after T2 → **`0 / 0 / 0`**.
- **Same-command controls that must NOT change**, so the criterion cannot be satisfied by a blanket
  sed: `grep -c '\*observatory\.SQLiteBackend' cmd/ailang/observatory_metrics.go` → **3** before
  **and** after (Out of Scope), and `cmd/ailang/chains_live.go` → **1** before **and** after (the
  type assertion at `:199` is the local-only escape and must survive). *A global replace ⇒ these go
  to 0 and the criterion FAILS.*
- `go build ./cmd/ailang/...` rc=0 after the widening (baseline rc=0). *A widened signature with a
  `DB()`/`Store()` call left inside would not compile — which is exactly the check that the
  `Backend`-shaped claim is true rather than asserted.*

**T3 — swap:**

- `grep -c 'openChainsReadBackend(' cmd/ailang/chains*.go` summed over non-test files → **13**
  (baseline: **0**, the identifier does not exist).
- `grep -n 'NewSQLiteBackendFromPath(' cmd/ailang/chains*.go` over non-test files → **17** today
  (enumerated above); after T3 → exactly **4**, and the surviving four are exactly
  `chains_post.go` (write half), `chains_live.go`, `chains.go` (journey), `chains_stats_cvs.go`.
  *Any other survivor is a missed swap; any missing one is a refusal accidentally converted.*
- **`TestChainsReadCommands_UseTheResolver` (a source guard, in `cmd/ailang`).** Reads the
  `chains*.go` sources and asserts the direct-`NewSQLiteBackendFromPath` allowlist is **exactly**
  that four-file set. *This is the criterion that survives the sprint: it fails the day a new
  `chains` read command is added with a hard-wired local backend, which is the same class of silent
  omission this whole sprint exists to close.*
- **MUTATION M4-2 (kills T3).** Line-scoped revert of **one** swapped site — resolve `L` with
  `grep -n 'openChainsReadBackend(' cmd/ailang/chains_tree.go` (assert exactly one hit), then
  `sed -i '' "${L}s/openChainsReadBackend(/observatory.NewSQLiteBackendFromPath(observatory.DefaultDatabasePath()) \/\/ MUTANT: /"`
  — or any line-scoped edit that restores a direct local open at that site. LANDED (sha ≠
  `baa479cecf73b9e550ec452dd1bcbfffcb4b820ea6c683e498bb4629d33249df`) + BUILDING ⇒
  `go test ./cmd/ailang/... -run 'TestChainsReadCommands_UseTheResolver' -count=1` **FAILS**.
  *If it does not fail, the guard is enumerating something other than the sources.*

**T4 — loud refusal:**

- `TestRemoteReadRefusal_LocalOnlySurfaces`, table-driven over the **five** entries, each asserting
  a **non-nil error** whose text contains the command name **and** the reason token. Baseline: the
  refusal function does not exist; `go test -run 'TestRemoteReadRefusal'` → `[no tests to run]`
  (measured pattern: the same probe for M1's test returned `[no tests to run]` at `644cf178a`).
- **Positive control, mandatory:** the same table includes at least three **swappable** commands
  (`chains view`, `chains list`, `chains tree`) asserting the refusal does **NOT** fire. *Without
  this, a refusal function that returned an error for every input would pass all five negative arms
   — the exact failure mode M2's Firestore arms were written to avoid.*
- **MUTATION M4-3 (kills T4).** Delete **one** entry from the refusal set: resolve `L` with
  `grep -n 'chains stats --cost-per-verified-success' cmd/ailang/<refusal file>` (assert exactly one
  hit) and comment that line out with a line-scoped `sed`. LANDED + BUILDING ⇒
  `go test ./cmd/ailang/... -run 'TestRemoteReadRefusal' -count=1` **FAILS** on that arm
  specifically. *Not done ⇒ the CVS command silently reads local under `--remote`, which is both a
  wrong answer and a silent fallback.*

**T5 — `eval-*` loud error (the ruling's instrument — this criterion is the milestone's reason for
existing):**

- `TestEvalRemoteReadIsRefused`, table-driven over **all 14** eval-family command names, each
  asserting the explicit-flag arm returns a **non-nil error** and that the message contains the
  revisit-trigger token. Baseline for the command list, re-derivable:
  `grep -cE '^\tcase "eval' cmd/ailang/main.go` → **13** top-level cases (measured; `eval-chains`'
  own subcommands make 14 surfaces). *Not done ⇒ zero of 14 arms exist and the ruling's measurement
  instrument was never built — which is precisely how the previous plan lost this obligation.*
- **Env arm:** `TestEvalRemoteReadEnvWarnsAndProceeds` — with `AILANG_CHAINS_READ=gcp` set, the
  guard returns **no** error but **does** emit the warning line to the captured writer. *Two
  distinguishable outcomes; a guard that hard-failed here would break every eval on a rig with the
  env exported, and a guard that stayed silent would destroy the signal. Both are caught.*
- **Positive control, mandatory:** the same test asserts the guard does **NOT** fire for
  `chains`, `chains view` or `messages`. *Without it, `return errors.New(...)` unconditionally would
  pass all 14 arms.*
- **MUTATION M4-4 (kills T5).** Shrink the guarded set by exactly one: resolve `L` with
  `grep -n '"eval-paired"' cmd/ailang/<guard file>` (assert exactly one hit), line-scoped `sed` to
  `"eval-paired-DISABLED"`. LANDED + BUILDING ⇒
  `go test ./cmd/ailang/... -run 'TestEvalRemoteReadIsRefused' -count=1` **FAILS** on the
  `eval-paired` arm. *A table that iterates a list it does not actually consult would survive this.*
- **The message is the signal, so it is graded as such.** `ailang eval-paired --remote gcp` (or the
  chosen surface form) writes to **stderr** and exits non-zero, and the text names the command, says
  remote read is `view`-scoped per `D-15`, and points at `#698` part 1 as the place to register
  demand. *Not done ⇒ "assumed demand becomes a dated signal" is false and the ruling's stated
  reason for choosing `view` is unfulfilled.*

**T6 — the Firestore silent stub:**

- `grep -c 'return nil, nil' internal/storage/firestore/observatory_chains.go` → **1** today
  (measured; the hit is `GetMissionRollups` at `:545`); after T6 → **0**. Same-file control that the
  grep discriminates: `grep -c 'func (s \*ObservatoryStore)' internal/storage/firestore/observatory_chains.go`
  is unchanged before and after.
- `TestGetMissionRollups_RefusesLoudlyOnFirestore` in
  `internal/storage/firestore/observatory_chains_test.go` (the file M2 created, `package firestore`):
  `(&ObservatoryStore{}).GetMissionRollups(ctx, nil, "", 5)` with a **nil** client returns
  `err != nil` and `rollups == nil`. Nil-client-safe because the guard returns before touching
  `s.client` — the same construction M2's arms already use.
- **MUTATION M4-5 (kills T6).** Restore the stub: resolve `L` with
  `grep -n 'func (s \*ObservatoryStore) GetMissionRollups' internal/storage/firestore/observatory_chains.go`,
  then line-scope the `sed` to `L+1` replacing the new `return` with `return nil, nil`. LANDED
  (sha ≠ `73b7267458975e8379c854ca52c97e89e1b4057851f58fe02a8ac852b4995af8`) + BUILDING ⇒
  `go test ./internal/storage/firestore/... -run 'TestGetMissionRollups' -count=1` **FAILS**.
  *Not done ⇒ `chains stats --by-mission --remote gcp` prints "no missions" for a store that was
  never queried.*

**T7 — doc amendment:**

- `grep -c 'D-15' design_docs/planned/v0_33_2/m-mission-loop-unified-telemetry.md` → **0** today
  (measured); after T7 → **≥ 1**. Same-file control that the grep sees this document at all:
  `grep -c 'Design Freeze'` → **3**, before and after.
- The amended item must contain **all three** of: the word `view`, the date `2026-08-14`, and the
  string `eval_assessment` (the measurement that motivated the narrowing). *Three greps, three
  failure modes: a decision with no scope, a decision with no date, a decision with no evidence.*
- `grep -n 'every .ailang chains. / .eval-\*. consumer inherits the option' <doc>` must **either**
  return 0 hits **or** every surviving hit must be on a line the amendment marks as superseded.
  *Not done ⇒ the doc still asserts, in the present tense, a code fact that D3/D4/#6/#7 refute — and
  the next planner reads the doc, not this plan.*
- **This amendment ships in the SAME commit range as T1–T6.** `git diff --name-only` for M4 includes
  the doc. *Not done ⇒ the doc records a pending decision after the decision was made and executed,
  which is the failure the "Doc amendments" section was written to prevent.*

#### Estimate, and why it is not the option table's ~230 LOC / ~4h

| Task | LOC | Note |
|------|-----|------|
| T1 resolver | ~55 | table's ~70, narrowed — it delegates to `NewBackendsForMode` rather than re-resolving |
| T2 widen | ~12 | table said ~16; **D2** measured 12 |
| T3 swap | ~40 | table said ~20 LOC for "~20 sites"; 13 sites × (call + flag registration) |
| T4 refusal | ~45 | table said ~30 for one family; **D3/D4** added three commands it did not know about |
| T5 `eval-*` guard | ~45 | 14 surfaces × 2 arms, plus the message text that IS the instrument |
| T6 Firestore stub | ~8 | **entirely new — D5**, not in the table at all |
| Tests | ~195 | table said ~110; six tasks each need a mutation-killable arm plus a positive control |
| **Total** | **~400 LOC · ~7h (~1 day)** | |

**The `view` column's ~230 LOC / ~4h is REFUTED — it under-scoped by ~1.7×.** Not because the
ruling was priced dishonestly, but because the option table costed the *`Backend`-shaped* set by
grepping one escape hatch (`DB()`) and missing a second (`Store()`), and because it never scanned
the Firestore implementations for silent stubs. Both omissions are the same shape as the one this
sprint exists to correct: a property that was asserted rather than measured. **The ruling itself is
unaffected** — every discovery makes `view` *narrower* and therefore *more* clearly the cheaper
option; none of them makes `eval` cheaper.

<a id="m4-risks"></a>

#### Risks

| Risk | Impact | Mitigation |
|------|--------|-----------|
| The executor widens `observatory_metrics.go`'s 3 signatures "for consistency" | Med | T2's criterion asserts that count is **3 before and after**; a blanket sed FAILS the criterion |
| The refusal set is implemented as a message but not an exit code | **High** | T4/T5 criteria assert **non-nil error / non-zero exit**, not text presence alone — a warning where an error was ruled is a silent fallback |
| `AILANG_CHAINS_READ` exported on the rig hard-fails every nightly eval | **High** | T5's deliberate asymmetry: flag ⇒ error, env ⇒ warn-and-proceed. Both arms have their own criterion so neither can be quietly dropped |
| T6 changes Firestore behaviour for some other caller | Low | Measured: `GetMissionRollups` has exactly **one** non-interface consumer, `chains_stats_mission.go:66`, which already error-exits |
| M4's mutants target code that does not exist yet, so a `sed` silently no-ops | Med | Every M4 mutation mandates an exact source literal, resolves its line with `grep -n`, asserts the hit is unique, and asserts sha CHANGED before the test result is read |
| `journey` / `find --task` / `stats --cvs` are quietly given the flag anyway because they "look like view commands" | Med | They are named individually in T4's table and in the per-site disposition table; each has its own arm |

---

## Out of Scope

- The `schema` variant of remote eval read (denormalized fields + composite indexes + cloud backfill).
- Remote read for the `observatory_*` command family — 12 raw `DB()` escapes; a separate concern.
  **M4 does not widen `observatory_metrics.go`'s 3 `*observatory.SQLiteBackend` signatures**, and T2
  carries a control criterion asserting that count is unchanged.
- Remote read for `ailang chains live` — the live tail runs raw SQL against `Store().DB()`.
- **Remote read for `ailang chains journey`** — `GetChainJourney` is not on `Backend` (D4).
- **Remote read for `ailang chains stats --cost-per-verified-success`** — it escapes through
  `backend.Store()` (D3).
- **Remote read for `ailang chains find --task`'s span-summary fallback** — `GetTaskSpanSummary`
  is not on `Backend` (D4).
  *(These three are new to Out of Scope as of the `bad8f3647` re-derivation. They are **not**
  silently dropped: T4 makes each of them refuse loudly, with its own arm.)*
- **Remote read for `eval-*`** — the `D-15` narrowing. Not merely unimplemented: T5 makes it an
  explicit, dated, greppable refusal.
- Anything M1/M2/M3 would tempt an executor into: **no production-code changes in M1–M3.** If a test
  cannot be written without changing production code, that is a finding to report, not a licence.
  **This constraint does NOT extend to M4** — M4 is production code by construction (T1–T6), and its
  no-collateral criteria are stated per task instead.
- Re-running or re-evaluating #697's landed M1+M2+M3 work.

## Findings for the issue and the doc

To be reported by the executor and filed by the controller, not silently dropped:

1. **#698 §2's mechanism is wrong in a way that helps.** The retry branch is reachable deterministically
   through the `id TEXT PRIMARY KEY` collision, not only through a `stage_number` race. The guard is
   still unprotected — the gap is real — but it is cheap to protect.
2. **#698 §3 enumerates five branches; four are testable.** `failed to marshal eval assessment` is
   unreachable by construction because `EvalAssessment` contains only marshalable field types. The
   honest options are: leave it with the type-guard test, or delete it. **Deleting it is a production
   change and therefore out of scope for this fast-follow** — recommend raising it separately.
3. **`Store()` is a second escape hatch and nobody had grepped it (D3).** This plan's own Discovery
   #5 concluded "a clean, defensible remote/local boundary exists" from a `DB()` grep alone.
   `.Store()` has the identical effect and appears at `chains_stats_cvs.go:62` — inside the chains
   family, in a command the option table listed as remotable. **An audit that greps one escape hatch
   measures one escape hatch.** Whoever next asks "is this surface interface-shaped?" should extract
   the interface's method set and check every call, which is what D3/D4 did and what found two more.
4. **A silent-empty Firestore stub was one milestone away from becoming a wrong answer (D5).**
   `GetMissionRollups` returns `nil, nil` with a comment saying it is "not supported". It is inert
   only because nothing can reach it remotely today. **A fallback is not safe because it is
   unreachable; it is unreached.** T6 fixes it in the same change that makes it reachable.

## Doc amendments owned by this sprint

- `design_docs/planned/v0_33_2/m-mission-loop-unified-telemetry.md:115-117` — Design Freeze item 3's
  *"every `ailang chains` / `eval-*` consumer inherits the option"* is refuted for the `eval-*` half
  (Discoveries #6, #7) **and, as of the `bad8f3647` re-derivation, for three `chains` commands too**
  (D3, D4). **Owned by M4 task T7, shipping in the same change as T1–T6** — the doc records the
  `D-15` decision, its date and attribution, and the measurement that motivated it. T7 carries four
  acceptance criteria, including one asserting the un-amended sentence no longer stands unqualified.
  Baseline measured 2026-08-14: `grep -c 'D-15' <doc>` → **0** (control `grep -c 'Design Freeze'`
  → **3**).

## Success Metrics

- [ ] `go test ./internal/observatory/... -count=1` rc=0 · `./cmd/ailang/... -count=1` rc=0 ·
      `./internal/storage/firestore/... -count=1` rc=0 (all baseline rc=0; none may regress)
- [ ] `go build ./internal/observatory/... ./cmd/ailang/... ./internal/storage/...` rc=0
      (**never** `go build ./...` — baseline rc=1 on `cmd/wasm`)
- [ ] `make lint` clean
- [ ] **4 of 4 planned M1–M3 mutants die**, each asserted LANDED (sha256) and BUILDING (rc=0) before
      its result was read, and each restored byte-identically
- [ ] **5 of 5 planned M4 mutants die** (`M4-1` resolver-forces-local · `M4-2` one swapped site
      reverted · `M4-3` one refusal entry removed · `M4-4` one `eval-*` name removed from the guard ·
      `M4-5` the Firestore stub restored), each with its `sed` address resolved by `grep -n` and
      **asserted unique**, each asserted LANDED (sha256 CHANGED) and BUILDING (package-scoped rc=0)
      before its result was read, and each restored byte-identically from a `cp` backup
- [ ] **The `D-15` narrowing is measurable, not just implemented.** `ailang eval-paired --remote gcp`
      (or the chosen surface form) exits non-zero with a message naming the command, `D-15`, and
      `#698` part 1 — the dated, attributable signal the ruling was chosen to produce
- [ ] **The default did not move.** `ailang chains view <id>` with no flag and no
      `AILANG_CHAINS_READ` opens `observatory.DefaultDatabasePath()`, exactly as at `bad8f3647`
- [ ] **The doc records a decision, not a pending one.**
      `grep -c 'D-15' design_docs/planned/v0_33_2/m-mission-loop-unified-telemetry.md` ≥ 1
      (baseline **0**; control `grep -c 'Design Freeze'` → **3**), and the amendment ships in the
      same change as T1–T6
- [ ] `jq -e .` passes on both sprint JSON files, **and both are staged with `git add -f`** —
      verify after committing with
      `git cat-file -e HEAD:.ailang/state/sprints/sprint_M-TELEMETRY-REMOTE-READ-FASTFOLLOW.json`
      (rc=0). Without the `-f`, `.gitignore:77` drops them silently and the sprint's own record
      repeats the orphan it was written to fix.
- [ ] Every task above has at least one criterion that would fail if the task were skipped — the rule
      this sprint exists to enforce

## Dependencies

- M1, M2, M3 are mutually independent and independently landable in any order. **All three LANDED**
  in [#699](https://github.com/sunholo-data/ailang/pull/699).
- M4 depended on one word from Mark. **It arrived 2026-08-14: `view`.** No M1–M3 work was wasted.
- **Inside M4**, the order is partially constrained:
  - **T1 before T3** (the swap needs the resolver).
  - **T2 before T3** for `chains_util.go` / `chains_tree.go` / `chains_diagnostics.go` — a swapped
    site hands an `observatory.Backend` to a helper that still declares `*observatory.SQLiteBackend`,
    and that does not compile. This is a **compile-enforced** ordering, not a convention.
  - **T6 before (or with) T3's `chains_stats_mission.go:49` swap** — swapping that site first makes
    `--by-mission --remote` answer "no missions" silently for as long as the two are apart.
  - **T4 and T5 are independent of everything else** and can land first; they are pure refusals and
    are the two tasks whose absence the previous plan's failure mode would reproduce.
  - **T7 ships in the same change** as T1–T6, by criterion.
- M4 extends `internal/storage/firestore/observatory_chains_test.go`, the file **M2 created**. M4 is
  therefore not landable on a tree without #699.

## Risks

| Risk | Impact | Mitigation |
|------|--------|-----------|
| An executor "fixes" M2's unreachable branch by editing production code to make it reachable | Med | Explicit out-of-scope line; branch 5 is a **finding**, and the type-guard test is labelled as a type guard, not as coverage |
| A global `sed` mutates firestore line 244 instead of 375 and the test passes for the wrong reason | Med | Every mutation is line-scoped and sha-asserted; called out in the mutation-discipline section |
| `go build ./...` rc=1 is misread as this sprint's regression | Med | Baselined and stated in three places; all criteria are package-scoped |
| ~~M4 gets executed on an assumed answer~~ **RESOLVED 2026-08-14** | ~~High~~ | The answer arrived from Mark, attended, and is quoted verbatim at the head of M4 with its charter location. M4's criteria are written against `view` and **only** `view` |
| M4's scope creeps back toward `eval` because a `chains` command turns out to need `QueryEvalResults` | Med | Exactly one does — `chains stats --cost-per-verified-success` — and it is in the **refusal** set (T4), not the port set. The revisit trigger stays pre-registered in T5, unexecuted |
| M4's per-task risks | — | See the [M4 risk table](#m4-risks); they are milestone-local and not repeated here |
| The nil-client Firestore positive control is weakened to a no-op to avoid a panic | Med | Explicit instruction to recover the panic rather than soften the assertion |
