# Iteration 309 — SPRINT-PLANNER disposition plan
## `m-coordinator-route-authority-recovery` (M1 + M1r + M2a), post-quorum-round-2 BLOCKED

**Verdict in one line:** the controller's provisional call is **right in direction, wrong in
shape** — park the *code merge*, but do not park the iteration. The correct disposition is
**SPLIT into three rows** (the round-3 localisation rule fires now), land one real test and
the split docs on dev with no further approval, and return to Mark a **re-scoped** ask rather
than a spent authorization.

---

## §0 REFUTATIONS AND REFINEMENTS OF THE HANDED-DOWN MEASUREMENTS (read first)

### R-1 — Controller claim 4 is FALSE AS STATED. The reviewer is still right, for a sharper reason.

Claim: "`coordinatorConfigDiff` ... performs **no route evaluation whatsoever**."

```
$ cd /Users/voightkampff/.ailang-driver-pin/.wt-v1-iter309
$ sed -n '/^func coordinatorConfigDiff/,/^}/p' cmd/ailang/coordinator_config.go
```
Observed (tail of the function):
```go
	if string(local) == string(live) {
		fmt.Printf("identical to gs://%s/%s generation %d\n", store.bucket, store.object, gen)
		return nil                       // <-- EARLY RETURN, no validation
	}
	fmt.Printf("DIFFERS from ... (local %d bytes, live %d bytes)\n", ...)
	if verr := validateCoordinatorConfigBytes(local); verr != nil {
		fmt.Printf("and your local copy would be REFUSED: %v\n", verr)
	}
```
It **does** call `validateCoordinatorConfigBytes`. And post-M1 that function gains route
evaluation (`git show 3500db0a7 -- cmd/ailang/coordinator_config.go` adds
`coordinator.ApplyCoordinatorConfigDefaults(&file.Coordinator)` and
`coordinator.ValidateCloudExecutionRoutes(&file.Coordinator)`).

But the mitigation is vacuous **exactly where the design cites it**, and this is a stronger
finding than the one filed:

- it validates **`local`**, never `live`;
- it validates **only in the DIFFERS branch**.

The design's scenario is *deploy new code, config unchanged*. There `local == live` → the
"identical" branch → `return nil` → **zero route evaluation**. The one branch that runs is
the one the operator will never be in. `gpt5-6-sol`'s objection stands, and his arity point
is literally correct too: the doc's AC and Rollout §2 write `ailang coordinator config diff`
with **no file argument**, and the function's first statement is
`if len(args)==0 { return errors.New("usage: ailang coordinator config diff <file>") }` — the
AC step as written exits non-zero before touching GCS.

### R-2 — The 39/0 figure is CONFIRMED by a second, independent instrument.

Not the Python replica; direct analytic enumeration off the two real Go sources
(`sed -n '55,175p' internal/dispatch/cloudrun/dispatcher.go`,
`git show 3500db0a7:internal/coordinator/execution_route.go`):

- dev `checkVariantProviderAgreement` accepts **49/104** = 13 (`provider==""` early return)
  + 13 (`binarylessProviders["managed_agents"]` early return) + 12 (`eval`/`eval-go` map to
  `nil` = any provider, × 6 binaried providers) + 11 (per-provider matches: claude 3, gemini 2,
  codex 2, opencode 1, pi 2 incl. `pi-go`, motoko 1).
- M1 `ValidateExecutionRoute` accepts **10/104** (`cloudProviderVariants` 9 pairs + the
  `NormalizeExecutionVariant` claude-`""`→`default` cell).
- The 10 ⊂ the 49. 49 − 10 = **39 over-rejects, 0 under-rejects**. Cause split 13/13/12/1 as the
  doc states.

Two instruments now agree. The codex Go parity test remains the acceptance gate, but the
number is no longer in doubt and no plan should be blocked waiting on it.

### R-3 — NEW, and it RAISES severity: both guards now run in `Dispatch`, M1's first.

The worktree has M1 applied. Read from the applied tree:
```
$ grep -n "ValidateExecutionRoute\|checkVariantProviderAgreement" internal/dispatch/cloudrun/dispatcher.go
184:	if err := coordinator.ValidateExecutionRoute(params.AgentID, params.Provider, params.ExecutorVariant); err != nil {
195:	if err := checkVariantProviderAgreement(params.ExecutorVariant, params.Provider); err != nil {
```
M1 does not *replace* dev's guard, it *precedes* it. The effective dispatch accept-set becomes
the **intersection = 10 cells** from the instant M1 lands, and dev's 39 extra cells become
unreachable dead code. This is measured, not inferred, and it is the load-bearing fact for §4.

### R-4 — NEW, and it LOWERS blast radius to ZERO. I read the live production config.

```
$ AILANG_CLOUD_PROJECT=ailang-multivac ./ailang coordinator config get /tmp/ail309/live_cfg.yaml
RC=0 ; gs://ailang-multivac-ailang-config/config.yaml generation 1788171270744533 ; 1277 lines
$ grep -oE '^\s+provider:\s*\S+' live_cfg.yaml | awk '{print $2}' | sort | uniq -c
   2 codex     1 motoko     32 pi
$ grep -oE '^\s+executor_variant:\s*\S+' live_cfg.yaml | awk '{print $2}' | sort | uniq -c
   1 codex     1 codex-go     1 motoko     32 pi
$ grep -nE 'managed_agents|eval-go|executor_variant:\s*eval|pi-go' live_cfg.yaml ; echo rc=$?
rc=1                      # control, same file, same instrument: grep -c 'executor_variant' -> 35
$ grep -cE '^\s+- id:' live_cfg.yaml        -> 35
$ grep -n '^\s*default_provider:' live_cfg.yaml -> pi        # non-empty, so cause C2 unreachable
$ sed -n '73,95p' live_cfg.yaml             -> pipelines: ... bindings: []   # ExpandPipelines adds 0 agents
```
**35 live agents, 4 distinct routes — `(pi,pi)`×32, `(codex,codex)`, `(codex,codex-go)`,
`(motoko,motoko)` — all four inside M1's 10-cell accept set.** Zero `managed_agents`, zero
`eval`/`eval-go`, zero `pi-go`, zero empty provider, zero pipeline-expanded agents.

Consequences:
- M1-as-committed would refuse **nothing currently deployed**. The 39 cells are a *capability*
  regression, not an outage-in-waiting. There is no time pressure.
- The doc's top Risk row ("a live cloud agent is configured with an `eval`/`eval-go` variant")
  has a **measured-false premise** at generation 1788171270744533. Keep the row (config drifts),
  but restate it as a standing check, not a live hazard.
- It also confirms the doc's inherited Rollout §2 claim about the planner lane, first-party:
  `sprint-planner` is `codex`/`codex`.

### R-5 — The "quorum instrument defect" is a DOCS defect, not a code defect.

```
$ python3 -c "import json;d=json.load(open('.ailang/state/mission-quorum/m-coordinator-route-authority-recovery-2026-08-31T12-06-03Z.json'));print(sorted(d.keys()));print('absent_reviewers' in d)"
['controller_in_session', 'doc', 'iso_ts', 'reviewers', 'synthesis']
False
```
There is **no top-level `absent_reviewers` field at all** — it is not "present and null".
Absence *is* recorded, correctly, per reviewer:
`{"model":"oc-glm-5-2","present":false,"absent_reason":"invalid","error":"reviewer returned non-JSON or malformed response..."}`.
So a controller reading the artifact **can** see the absence; the documented rule just names a
field the schema never had. Fix the rule to read `reviewers[].present == false`, not the writer.
Filing this as a broken instrument would have sent someone to fix working code.

### R-6 — The worktree is no longer at pristine base; label any "base rc" measured in it.

At my first call `git status --porcelain` showed one `??` line. Four calls later it shows M1's
13 files **staged**, byte-identical to the commit
(`git show :internal/coordinator/execution_route.go | git hash-object --stdin` →
`c7e5aad77...` == `git rev-parse 3500db0a7:internal/coordinator/execution_route.go`). HEAD is
still `8bb92ae2b`. Therefore my builds are **merged-tree** figures, not base figures:
`go build ./internal/coordinator/...` rc=0, `./internal/dispatch/...` rc=0, `./cmd/ailang` rc=0,
`go build ./...` **rc=1** (`cmd/wasm: function main is undeclared`). They independently
reproduce V10 by a different path; they do **not** re-confirm V20's base numbers.
Housekeeping: `go build ./cmd/ailang` left a 97MB `./ailang` in the worktree root — gitignored
(`.gitignore:44 /ailang`), so `git status` is unaffected; I did not remove it (no writes).

### R-7 — `gemini-3-1-pro`'s round-2 objection is DISCHARGED by measurement, in one call.

```
$ git log --oneline -5 3500db0a7~1..origin/dev -- cmd/ailang/coordinator_cloud.go ; echo rc=$?
rc=0                                        # empty
$ git log --oneline -5 3500db0a7~1..origin/dev -- internal/dispatch/cloudrun/dispatcher.go
23066345e fix(dispatch): refuse a job whose executor binary cannot exist in its image
```
Empty result with a positive control **in the same call, same range, same instrument**. The
claim holds and now has evidence. This is a one-row fix, not a design change. (The controller's
unscoped `8a993bb89 / 2026-08-28` is consistent — it predates `3500db0a7~1`, hence the empty range.)

---

## §1 Is parking right? — YES for the code merge, NO for the iteration

**Recommendation: park the merge of M1/M1r/M2a. Do not park the iteration, and do not treat
D-50 as spent.**

Why parking the merge is correct, not over-caution:
1. The doc is BLOCKED, and the surviving objection is **true** (R-1, measured). Merging over a
   blocked quorum whose objection you have personally confirmed is precisely the guardrail
   standing rule 2 exists for.
2. **D-50 authorized a scope that no longer exists.** Mark approved "apply the already-reviewed
   unit `3500db0a7` + M2a". Under that authorization the loop discovered that the unit carries a
   39-cell regression (R-2) needing **M1r** — a reconciliation that is not in the approved
   narrowing and did not exist when he approved — and that the rollout mitigation cannot fire
   (R-1), needing a **new CLI surface** also not in the narrowing. An authorization for X does
   not authorize X + two new work items. Executing anyway would be forcing scope through a
   human gate, dressed as compliance with it.
3. **R-4 removes the only argument for urgency.** Zero live agents are affected. Nothing is
   burning; a one-iteration delay costs nothing measurable.

Why the authorization is **not** wasted, and must not be reported as such:
- The sprint's most valuable output has already been produced. A clean 3-way cherry-pick
  (rc=0, zero conflicts) and a fully green suite (V10/V11, my rc=0 builds) **both hide all 39
  cells**, because each side's tests exercise only its own table (dev's cases call
  `checkVariantProviderAgreement` directly at `dispatcher_test.go:435`). That is the finding
  the authorization bought. The code was always the cheap part.
- The failure mode to avoid is not "declining to execute", it is "declining to execute **and
  returning nothing**". §5 lists what ships today regardless.

Report to Mark as: *authorization honoured, scope grew under it by two measured items, here is
the re-scoped three-row ask and here is what already landed* — not as *blocked, nothing done*.

---

## §2 SPLIT, not revise. The localisation is unambiguous.

| Round | `gpt5-6-sol` | `gemini-3-1-pro` | `oc-glm-5-2` |
|---|---|---|---|
| 1 | **M2a** — preflight/execute instance identity | **Rollout** — `config diff` unverified | **M1r** — matrix specified as prose only |
| 2 | **Rollout** — `config diff` does not evaluate routes; arity wrong | **Doc hygiene** — Conflict-Surface citation scoped wrong | ABSENT (`present:false`, `absent_reason:"invalid"`) |

Read down the columns and the pattern is decisive:

- **M1/M1r matrix**: objected once, fixed with the 104-cell table, **clean in round 2**.
- **M2a child boundary**: objected once, fixed with the factory-memoization constraint + R6
  pointer-identity gate, **substantively clean in round 2** (gemini's round-2 hit is a citation
  defect about a file M2a touches, now discharged — R-7).
- **Rollout / operator tooling**: objected in **both rounds, by two different reviewers**, and
  it is the objection I independently confirmed. It is the only surface that has not cleared.

Objections have migrated *off* the two implementation surfaces and *onto* one non-implementation
surface. That is the SPLIT signal, and the next round is round 3 — the rule fires on schedule,
not early. A third revision of a 570-line doc would put two clean surfaces back at risk of a
fresh reviewer picking a new thread in either of them.

**Honesty caveat that travels with the split:** M1r's round-2 clean is **N-1 evidence** —
`oc-glm-5-2`, the reviewer who raised the matrix objection in round 1, was absent in round 2 and
never saw the fix. "Clean" here means "unobjected", not "cleared". Say so in the split doc.

### The three documents

| # | Doc | Content | Objection that travels with it |
|---|---|---|---|
| **A** | `m-coordinator-config-route-preflight.md` (NEW, ~120 lines) | §3 below: `ailang coordinator config check`, `diff` validation on both branches, AC arity fix | `gpt5-6-sol` round 2, **verbatim** — it is this doc's reason to exist |
| **B** | `m-coordinator-route-authority.md` (M1 + M1r) | 104-cell table, S1–S5 / P1–P3, `TestRouteMatrixParityWithDispatcherGuard`, `TestAgentRoutePolicy_RefusesEvalVariants`, superseded-markdown annotation, R-3, R-4 census | none open; carries the N-1 honesty caveat + the R-3 both-guards-run finding |
| **C** | `m-opencode-child-preflight.md` (M2a) | `resolveExecutable`, hoisted 15s health preflight, R1–R8 mutation table | `gemini-3-1-pro` round 2 — already discharged (R-7), lands as a verification row, not an open objection |

Cross-doc dependencies: **A has none** (it improves dev's existing tooling; post-B it
automatically gains route checks because it calls `validateCoordinatorConfigBytes`). **B's
rollout AC depends on A.** **C depends on B** only for the `PermanentDispatchError` type (V19).

---

## §3 What replaces the vacuous preflight

**What a real preflight must do** — five properties, each one a thing `diff` fails today:

1. Read the **live** config, not a local candidate (the deploy question is *"would the binary I
   am about to ship refuse what production is running right now?"*).
2. Run **unconditionally** — never short-circuit on byte equality; byte equality is the *normal*
   case for a code-only deploy.
3. Apply defaults exactly as the daemon does (`ApplyCoordinatorConfigDefaults`) and expand
   pipelines, so the evaluated agent set equals the dispatched agent set.
4. Call the **same** `ResolveExecutionRoute` / `ValidateCloudExecutionRoutes` the daemon calls —
   never a re-implementation, or the preflight becomes a third authority table.
5. **Exit non-zero** on any refusal, listing agent id / provider / variant / reason. A printed
   warning inside an rc=0 is not a gate.

**Command surface (concrete):**
```
ailang coordinator config check [<file>]      # no <file> => fetch live via gcsConfigStore
  --json
# rc=0: every cloud-lane agent's route accepted (prints "N agents, N routes accepted")
# rc=1: prints one line per refused agent, then the count
```
Plus two one-line repairs shipping in the same row:
- move `validateCoordinatorConfigBytes(local)` in `coordinatorConfigDiff` **above** the
  `identical` early return, so the existing documented preflight stops being silently vacuous
  for anyone who keeps using it;
- fix doc B's AC/Rollout text to `ailang coordinator config check` (and, wherever `diff` is
  genuinely meant, to `diff <file>`).

Do **not** overload `diff`. `diff` answers "do these bytes match?"; conflating that with route
validity is the exact mistake that produced this defect.

**Acceptance for row A** — fixture config containing one `managed_agents` agent, one `pi-go`
agent, one `eval` agent and one good agent; assert exact per-agent verdicts and rc; one
mutation (delete the per-agent loop body) must red it. Plus a verification row recording the
one-time live census from R-4 (generation `1788171270744533`, 35 agents, 4 routes, 0 refused).

**In scope for this sprint?** **No — separate row, executed FIRST.** It is not in the D-50
narrowing; it is the gate that makes B's rollout safe so it must precede B; it is ~60 LOC plus
tests; and folding it back into B re-creates the surface mixing that caused two blocks.

---

## §4 Sequencing

### M1 and M1r must land as ONE commit. Two measured reasons.

1. **There is no safe intermediate state.** From R-3: post-cherry-pick `Dispatch` calls
   `ValidateExecutionRoute` at `:184` *before* `checkVariantProviderAgreement` at `:195`. The
   moment M1 alone is on dev, the dispatcher accept-set is the 10-cell intersection. There is no
   window in which dev's 49-cell behaviour survives an M1-only landing.
2. **Nothing in CI could detect the interval.** The merged tree is fully green — V11, and my own
   `go build` rc=0 across all three trees. dev's guard tests call `checkVariantProviderAgreement`
   directly, so they cannot see the intersection; M1's tests exercise only M1's table. An
   M1-only commit would put a 39-cell regression on dev that **no gate in the repo can see**.
   Splitting M1 from M1r is splitting a defect from its only detector.

### Order

| Step | Work | Approval needed | Depends on |
|---|---|---|---|
| **0** | *(this iteration)* Split into docs A/B/C; land the dev-side golden accept-set test; ledger rows; VL corrections | none | — |
| **1** | **Row A** — `config check` + `diff` branch fix. Design → quorum → execute | re-scoped D-50 | step 0 |
| **2** | **Row B** — M1 + M1r as **one** commit, gated by the 104-cell parity test (empty exemptions) + the policy test; annotate the two bundled superseded markdowns; rollout preflight = `config check` against live | re-scoped D-50 | A |
| **3** | **Row C** — M2a OpenCode child boundary, R1–R8 mutation protocol | re-scoped D-50 | B (`PermanentDispatchError`) |
| — | M3 / M4 | — | stay OUT: no Firestore emulator in CI |

Note on step 2's risk posture: R-4 shows zero live blast radius, which is an argument for
landing B **soon**, not for landing it **loosely**. The parity gate is still the acceptance bar.

---

## §5 Landable THIS iteration with no further approval

**Ships now:**
1. **The three split design docs** into `design_docs/planned/v0_35_0/`. Doc B's status header
   must record: *quorum rounds 1–2 BLOCKED; split at round 3 per the localisation rule; M1r's
   round-2 clean is N-1 (oc-glm-5-2 absent, `present:false`/`absent_reason:"invalid"`)*.
2. **A dev-side golden accept-set test** — `TestVariantProviderAgreement_AcceptSetIsPinned` in
   `internal/dispatch/cloudrun`: enumerate all 104 cells against the **real**
   `checkVariantProviderAgreement`, assert exactly the 49 accepts, variants read from the real
   `knownVariants` map. This needs **no M1, no approval, and touches no production code** — it
   asserts what dev already does. It is the durable half of the future parity gate (after B, the
   parity test compares two functions against the same pinned table), and it means a future
   change to *dev's* table also reds. This is the concrete answer to "don't return empty-handed".
   *(The full `TestRouteMatrixParityWithDispatcherGuard` cannot land today — it references
   `coordinator.ValidateExecutionRoute`, which only exists with M1.)*
3. **Verification-log corrections** into doc B: the unscoped `coordinator_cloud.go` log with its
   in-call control (R-7); the true `coordinatorConfigDiff` branch reading (R-1); the live-config
   route census (R-4); the both-guards-run dispatch ordering (R-3).
4. **Ledger/mission-log rows**: (a) 39-cell regression hidden by clean-cherry-pick + green suite;
   (b) preflight vacuous in exactly its cited scenario; (c) live census / zero blast radius;
   (d) quorum-rule docs defect (R-5) — *rule* fix, not code.

**NOT landable without the human:** the M1 cherry-pick, M1r, M2a, any
`cmd/ailang/coordinator_config.go` change.

**The single ask for Mark** — one line, no quorum needed to pose it:
> D-50 approved applying the reviewed M1 unit + M2a. Executing under it measured two items that
> were not in the narrowing: **M1r** (M1 alone regresses 39 of 104 route cells; clean cherry-pick
> and green tests both hide it) and a **real config preflight** (the cited `config diff` cannot
> fire in the deploy case it is cited for). Nothing live is affected — all 35 production agents
> are on routes M1 accepts. Re-approve as three sequenced rows (A preflight → B M1+M1r as one
> commit → C M2a)?

No cloud config write, no deploy, and no Terraform is in any of the three rows.

---

## §6 Operational notes for the controller

- The worktree index currently holds the staged M1 cherry-pick (R-6). If the disposition is
  park, **leave it** — do not `git reset --hard` (standing rule 0); the worktree is disposable
  and its index is the executor's working state.
- A gitignored 97MB `./ailang` build artifact sits in the worktree root from my
  `go build ./cmd/ailang`. Harmless, invisible to `git status`, left in place.
- Every "base rc" quoted from this worktree after ~14:10 is a **merged-tree** rc. Re-measure
  base figures from a clean extraction if they are to be quoted as base.
