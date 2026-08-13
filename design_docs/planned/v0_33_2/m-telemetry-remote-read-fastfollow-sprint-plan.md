# Sprint Plan: M-TELEMETRY-REMOTE-READ-FASTFOLLOW

**Design doc**: [m-mission-loop-unified-telemetry.md](m-mission-loop-unified-telemetry.md) (ratified, 2/3 executed)
**Prior plan**: [m-mission-loop-unified-telemetry-sprint-plan.md](m-mission-loop-unified-telemetry-sprint-plan.md)
**Issue**: [#698](https://github.com/sunholo-data/ailang/issues/698)
**Sprint ID**: `M-TELEMETRY-REMOTE-READ-FASTFOLLOW`
**Created**: 2026-08-13 (mission iteration 195)
**Baseline commit**: `644cf178ac3c6054f8039938abf122b0695f9891` (origin/dev)
**Duration (buildable part)**: ~0.5 day (~3.75 hours)
**Risk**: Low for M1–M3 (tests + one state file, no production code changed). M4 is **NOT EXECUTABLE** — it is decision-gated.
**Total LOC estimate (buildable part)**: ~175 (implementation 0, tests ~175)

---

## Summary

**Goal**: close the two gaps in #698 that are buildable with zero design risk, and put a *number* on
the third so a human can decide it in one word.

#698 has three parts. Exactly two of them are engineering:

| #698 part | This plan | Status |
|-----------|-----------|--------|
| 2 — pinned-ID guard survives mutation | **M1** | Buildable now |
| 3 — five unpinned error branches | **M2** | Buildable now (one of the five is unreachable — see M2) |
| — sprint JSON absent from `dev` | **M3** | Buildable now |
| 1 — opt-in remote READ | **M4** | **BLOCKED on Mark. One word.** |

**The decision M4 needs, in one word:**

> **How far should `--remote` reach — `view` or `eval`?**

Recommendation, reasoning and costs are in [M4](#m4_opt_in_remote_read--decision-gated-do-not-execute).

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

## Mutation discipline (applies to M1 and M2; non-negotiable)

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

### M4_OPT_IN_REMOTE_READ — DECISION-GATED, DO NOT EXECUTE

**#698 part 1**, and the ratified Design Freeze item that the prior plan's acceptance criteria lost.
**This milestone is not executable and carries no acceptance criteria yet, by design.** Writing them
before the scope is chosen is how the last one went missing in the other direction.

#### The constraint that makes this a decision and not a task

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

#### The three options, costed from the code

| | **`view`** — option (c) | **`eval`** — option (b) | option (a) |
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

#### Recommendation: **`view`**

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

#### The one-word question for Mark

> **How far should `--remote` reach — `view` or `eval`?**
>
> `view` (**recommended**, ~4h): remote read for `ailang chains view/list/tree/find/diagnose/stats`.
> `eval-*` stays local and says so out loud when you pass `--remote`.
>
> `eval` (~9h): the above, plus `eval-*` and `internal/eval_analysis` read remotely too — at the cost
> of a full-collection scan on the Firestore side, which a later schema sprint would replace.

Either answer is executable the same day it arrives; M4's acceptance criteria get written against the
chosen scope, under the same "every task has a criterion that can fail" rule as M1–M3.

---

## Out of Scope

- The `schema` variant of remote eval read (denormalized fields + composite indexes + cloud backfill).
- Remote read for the `observatory_*` command family — 12 raw `DB()` escapes; a separate concern.
- Remote read for `ailang chains live` — the live tail runs raw SQL against `Store().DB()`.
- Anything M1/M2/M3 would tempt an executor into: **no production-code changes in M1–M3.** If a test
  cannot be written without changing production code, that is a finding to report, not a licence.
- Re-running or re-evaluating #697's landed M2+M3 work.

## Findings for the issue and the doc

To be reported by the executor and filed by the controller, not silently dropped:

1. **#698 §2's mechanism is wrong in a way that helps.** The retry branch is reachable deterministically
   through the `id TEXT PRIMARY KEY` collision, not only through a `stage_number` race. The guard is
   still unprotected — the gap is real — but it is cheap to protect.
2. **#698 §3 enumerates five branches; four are testable.** `failed to marshal eval assessment` is
   unreachable by construction because `EvalAssessment` contains only marshalable field types. The
   honest options are: leave it with the type-guard test, or delete it. **Deleting it is a production
   change and therefore out of scope for this fast-follow** — recommend raising it separately.

## Doc amendments owned by this sprint

- `design_docs/planned/v0_33_2/m-mission-loop-unified-telemetry.md` — Design Freeze item 3's
  *"every `ailang chains` / `eval-*` consumer inherits the option"* is refuted for the `eval-*` half
  (Discoveries #6, #7). Amend with the measurement and with whichever scope Mark picks. **Do this in
  the same change as M4, not before** — the doc should record a decision, not a pending one.

## Success Metrics

- [ ] `go test ./internal/observatory/... -count=1` rc=0 · `./cmd/ailang/... -count=1` rc=0 ·
      `./internal/storage/firestore/... -count=1` rc=0 (all baseline rc=0; none may regress)
- [ ] `go build ./internal/observatory/... ./cmd/ailang/... ./internal/storage/...` rc=0
      (**never** `go build ./...` — baseline rc=1 on `cmd/wasm`)
- [ ] `make lint` clean
- [ ] **4 of 4 planned mutants die**, each asserted LANDED (sha256) and BUILDING (rc=0) before its
      result was read, and each restored byte-identically
- [ ] `jq -e .` passes on both sprint JSON files, **and both are staged with `git add -f`** —
      verify after committing with
      `git cat-file -e HEAD:.ailang/state/sprints/sprint_M-TELEMETRY-REMOTE-READ-FASTFOLLOW.json`
      (rc=0). Without the `-f`, `.gitignore:77` drops them silently and the sprint's own record
      repeats the orphan it was written to fix.
- [ ] Every task above has at least one criterion that would fail if the task were skipped — the rule
      this sprint exists to enforce

## Dependencies

- M1, M2, M3 are mutually independent and independently landable in any order.
- M4 depends on one word from Mark and on nothing else. No M1–M3 work is wasted under either answer.

## Risks

| Risk | Impact | Mitigation |
|------|--------|-----------|
| An executor "fixes" M2's unreachable branch by editing production code to make it reachable | Med | Explicit out-of-scope line; branch 5 is a **finding**, and the type-guard test is labelled as a type guard, not as coverage |
| A global `sed` mutates firestore line 244 instead of 375 and the test passes for the wrong reason | Med | Every mutation is line-scoped and sha-asserted; called out in the mutation-discipline section |
| `go build ./...` rc=1 is misread as this sprint's regression | Med | Baselined and stated in three places; all criteria are package-scoped |
| M4 gets executed on an assumed answer | **High** | M4 carries **no** acceptance criteria and is marked DO NOT EXECUTE. Guessing here is the identical failure to the one this plan documents |
| The nil-client Firestore positive control is weakened to a no-op to avoid a panic | Med | Explicit instruction to recover the panic rather than soften the assertion |
