# v1.0.0 roadmap — from the readiness review to a tag

**Status:** ACCEPTED — decisions 1–5 ruled by Mark, attended 2026-11-10 (ledger `D-66`: "go with all
recommendations"). Decisions 6–10 ruled the same day (ledger `D-67`: "yes to all", with decision 8 re-scoped — see §6). Written 2026-11-10.
**Source:** [v1-release-readiness-review-2026-11-09.md](../v1-release-readiness-review-2026-11-09.md).
Each row in §2 is a design doc written by an independent Opus agent from that review, grounded in
the code and with its blast radius measured. This file is the index: order, dependencies, what the
design work changed in the review, and the decisions that unblock it.

## 1. Where we are

- `Release v0.58.0` is committed on `dev` (`f3b2ac26f`) and **not yet tagged**. It carries the three
  BREAKING changes (#2184, #2185, named-test checks). That matches the review's step 2.
- **v0.58.x is the last window for breaking changes.** Everything marked *breaking* below lands
  there or waits for 2.0.
- None of the ten docs has been through `ailang design-quorum` or sprint-planner yet.

**Size, honestly.** The ten docs sum to roughly **95–115 working days** of effort by their own
estimates. The review said "two to three weeks"; that was too low. With the work run in parallel
lanes the calendar is more plausibly **five to seven weeks** including the soak. This is an
estimate built from ten estimates, not a plan.

## 2. The ten docs

| Doc | What it fixes | Breaking | Estimate (days) | Measured blast radius |
|---|---|---|---|---|
| [m-v1-release-gates](m-v1-release-gates.md) | Test gate on releases; validator deploy on tag; rc pipeline; Stable-surface golden diff; dangling hook; race job; CI red | no | ~12 | n/a |
| [m-v1-downstream-compat-gate](m-v1-downstream-compat-gate.md) | `make downstream-check`: baseline vs candidate over every downstream corpus | no | 7–8 | Prototype: 2,070 files, about 8 min; reproduced the hand-counted 106 motoko breaks exactly |
| [m-v1-nominal-type-identity](m-v1-nominal-type-identity.md) | Cross-module ADT unsoundness; undeclared types; duplicate definitions; ambiguous imports; soundness clause | **yes** | 9–14 | Type identity: 0 programs found. Duplicate functions: 5, incl. every cached `ailang_parse` 0.39.2–0.55.5 |
| [m-v1-match-exhaustiveness](m-v1-match-exhaustiveness.md) | Non-exhaustive match becomes an error with a witness; located run-time error; `unreachable` | **yes** | 6–8 | std/benchmarks 0; downstream 6 one-line fixes; **1.5% of passing AI-written programs (150/10,166)** |
| [m-v1-tyvar-hygiene-and-digest-determinism](m-v1-tyvar-hygiene-and-digest-determinism.md) | #2157, #2076; closed and canonically named export schemes | **yes** (hard error on unclosed schemes) | 4–6 | Not measured as a count |
| [m-v1-show-total](m-v1-show-total.md) | Lossless, unambiguous `show`; `showPreview` helper | **yes** | ~7 | 23/394 repo programs change output; 1 benchmark; 1/1,049 banked passes flips (prototype patch) |
| [m-v1-stdlib-surface-polish](m-v1-stdlib-surface-polish.md) | 14 renames, 2 removals, 5 behaviour fixes, `JInt`, 25 functions re-tiered, `@deprecated` aliases | **yes** | 9–11.5 | 0.75% of recent model solutions use any renamed name |
| [m-v1-verification-honesty-and-semantic-decisions](m-v1-verification-honesty-and-semantic-decisions.md) | `check --verify` exit contract; contracts under bytecode; integer overflow; callback effect rule; #1914; #2156 | **yes** (overflow, callbacks, parser) | 15–19 | Per item; see doc |
| [m-v1-public-surface-hygiene](m-v1-public-surface-hygiene.md) | Help text 189 → 47 lines; capability-grant disclosure; Go API; tracked clutter; ollama module; CodeQL | partly (caps, command moves) | ~16 | n/a |
| [m-v1-docs-for-1-0](m-v1-docs-for-1-0.md) | Syntax reference rewrite; doc-sample CI gate; migration guide; security-model page; support window | no | 9–12 | 226 of 342 site code blocks fail `check` today |

## 3. Order and dependencies

**Wave 0 — protect the releases we are about to cut (start now, non-breaking)**

1. Release gates M1: test dependency in `release.yml`; registry validator stops deploying on
   pre-release tags. Until this lands, check the tag commit's CI by hand.
2. Downstream gate, first usable version. Every breaking change after it gets its blast radius
   from a real binary instead of a regex census. **This is the tool the rest of the roadmap is
   measured with**, so it goes first even though it fixes nothing itself.
3. Repair the dangling `pre-push` hook; close duplicate PR #2195.

**Wave 1 — type system (breaking; the reason not to tag now)**

4. Nominal type identity M1 (representation spike). Named fallback if it fails: ban duplicate type
   names across an import closure.
5. Match exhaustiveness — its wiring milestone **waits on step 4**, because it needs each type's
   constructor list keyed by the qualified identity.
6. Type-variable hygiene — independent of 4 and 5, small, can run alongside.
7. Verification honesty, part 1 (`check --verify` exit contract, bytecode contracts, #1914,
   #2156) — small, independently landable.

**Wave 2 — surface and semantics (breaking)**

8. `JInt` and the rest of the stdlib behaviour fixes. **Land `JInt` before exhaustiveness is
   enforced**, or `Json` matches break twice.
9. Stdlib renames with `@deprecated` aliases.
10. Total `show`.
11. Integer overflow and the callback effect rule (each starts with a decision; the overflow doc
    has a quorum round pending).
12. Capability-grant changes and Internal command moves from public-surface hygiene.

**Wave 3 — non-breaking, any time, in parallel**

13. Docs: the sample gate first so fixes stay fixed, then the rewrites, then the new pages.
14. Help text, ollama module replacement, untracking, CodeQL triage.
15. The rc pipeline (see §4) — must be done before step 18.

**Close-out**

16. **Remove the stdlib aliases in the last v0.58.x, before the soak** — not at the 1.0 tag.
17. Paired AILANG/Python baseline on that build, **after** exhaustiveness and `show` have landed,
    since both move pass rates. Scope per `D-67`: one Opus-level model and Haiku on subscription, one
    cheap open-weight model, and the on-device lanes. No full-suite run without Mark's go.
    The open-weight slot is **whatever the current default is when the run is scoped** — GLM 5.3
    Flash as of 2026-11-10 (Mark: "a good default recently but this changes like the wind").
    Re-confirm the pick at run time; do not treat this name as pinned.
18. Soak ≥ 7 days with every downstream pin on the final v0.58.x. A new breaking fragment resets it.
19. `v1.0.0-rc.1`, then `v1.0.0` = that build plus docs and version strings.

**Three things several docs touch, each needing one owner:**

- **The compile-cache key.** Nominal identity bumps it v10 → v11 and others may follow. Two PRs
  collided on this last week (#2184, #2185); serialise the bumps.
- **The teaching prompt.** At least five docs edit it. Batch the edits and re-verify every sample
  with `ailang check` — 46 of its 98 code blocks fail today.
- **The stdlib goldens and `make freeze-stdlib`.**

## 4. What the design work changed in the review

The review was written before these investigations. Where they disagree, the design docs are
right; each has a "differs from the review" section with the evidence.

**Worse than the review said**

- **Integer overflow makes Z3-verified contracts unsound.** The solver models `int` as unbounded;
  the runtime wraps. `ailang verify` reports `✓ VERIFIED` for `ensures { result > x }` on `x + 1`,
  and the program prints `-9223372036854775808`. *Reproduced by the controller.* The teaching
  prompt tells agents Z3 "mathematically proves" their code. This moves overflow from a
  documentation choice to a soundness fix.
- **A second and third type-soundness hole**, neither with an issue: a nonexistent record field
  (`best().bogus`) passes `check` when a declared record return flows through a lambda parameter;
  and lowercase type names parse as type variables, so some declared types are never enforced.
  *Reported, not reproduced by the controller.*
- **The bytecode VM raises no error on an unmatched value** — the match returns Unit and the
  program fails later, somewhere unrelated.
- **Exhaustiveness warnings never reach `check --json` or agent-format output**, so AI authors
  cannot see the warnings that already exist.
- **The evaluator and the VM disagree on JSON** (empty-string keys), and there are three decoders.
- **The rc path does not work today:** `release.yml` skips `v*-*` tags, one code path sorts
  `rc.10` before `rc.2`, and the docs site would show an rc as the stable release.
- **Changing only a function body can move the Stable v2 interface hash** via the `[a]` /
  `list[a]` rendering, contrary to `stability.md`.
- `ailang prompt --compact` exits 1.

**Better than the review said**

- **The Stable `interface_hash` is deterministic** (20/20 cold runs identical). #2076 affects the
  compile cache only — a fix-before-1.0 item, not a blocker.
- **The type-identity fix appears to break nothing downstream.**
- **The docs backlog is about a fifth of the review's figure**: 342 blocks on the live site; 1,660
  of the "about 1,900" are on archived prompt pages.
- Solver timeout and "unknown" already exit 1; the silent pass is missing-Z3 and skipped checks.
- v0.57.4's failed macOS job was a full-suite job; `test` and `test-windows` were green.

**Review recommendations the designs reject**

- **Fewer stdlib renames.** Keep `PI()`/`E()`, `take`/`drop`, `join`, `intToStr`, `find`: `join`
  appears in 853 model-solution rows, `take` in 621.
- **Do not simply untrack `eval_results/`.** That makes the `check-prompt-freeze` gate pass
  vacuously and empties the executor image's benchmark index. Use a bucket plus a tracked index.
- **Nothing internal needs `show`'s truncation**, so no internal caller moves to the new helper.
- **"Mostly macOS timing flakes" was wrong.** Most red CI is real: govulncheck (29),
  `TestQuorumArtifactShellSuite` (28), the launchd mirror (15). True flakes peak at 4.1% of SHAs.
- The soundness bug was already recorded as "M4" in a September design doc, and #1485/#1980
  already had planned docs. It is known work that was never prioritised.

## 5. Not covered by any doc yet

- The two additional soundness holes in §4 need issues and a home (most naturally the type-variable
  hygiene doc, which found them).
- Evaluator/VM JSON parity — partly in the stdlib doc; check it is a milestone, not a footnote.
- The hosted `ailang_check` tool image reportedly has no Z3; unconfirmed, the tool needed
  re-authentication.
- The security items held out of this repository (see the review, §11).

## 6. Decisions that unblock the work

Each doc ends with its own list. These are the ones that gate the most. **Rows 1–5 are RULED as
recommended (`D-66`)**, including the sub-rulings under row 2 (no escape hatch, `unreachable`,
literal-only matches follow the same rule, redundant arms warn and warnings reach JSON output) and
the transitional warning under row 4. **Rows 6–10 are RULED (`D-67`)**, with two changes from the table as first written:

- **Row 8 is re-scoped by Mark.** The paired AILANG/Python baseline is not the full model suite: one
  Opus-level model (not Fable) and Haiku, both on subscription, plus one cheap open-weight model, with
  the on-device lanes as the real focus — the aim is AILANG that works for weaker models too. A
  full-suite run before 1.0 happens only on Mark's explicit go.
- **Row 10: the motoko fork is blocking**, not advisory (it is what the rig runs). World stays advisory.

Sub-rulings carried by `D-67`: a release candidate is never promotable to prod; the syntax reference
page is normative; the support window is the latest 1.x minor, plus 90 days of security fixes for
v0.58.x; `runtime/` and `third_party/goopus` move under `internal/`; `check --json` emits real error codes.

| # | Decision | Recommendation | Unblocks |
|---|---|---|---|
| 1 | Accept this roadmap and its size, or tag sooner with known limitations | Accept. Three soundness holes and a prover/runtime mismatch are what a 1.0 promise is for | everything |
| 2 | Non-exhaustive match: error with no escape hatch, plus `unreachable` | Yes to both | exhaustiveness |
| 3 | Integer overflow: trap, with explicit wrapping ops | Trap — it is the only option that keeps `verify` honest | semantics doc, part 2 |
| 4 | `JInt` now | Yes; it can never be added after 1.0 | stdlib doc M4 |
| 5 | Soundness clause in `stability.md` | Yes, wording in the nominal-identity doc §10 | every future soundness fix |
| 6 | Registry validator moves on promote, not on tag | Yes | release gates |
| 7 | Doc-sample gate, which reverses the existing no-inline-code rule | Yes; the rule is not being followed and the gate is 13 seconds | docs |
| 8 | Budget for a paired baseline on the final v0.58.x | Yes, once, after wave 2 | the announcement's headline number |
| 9 | Go API excluded from the 1.x promise in writing | Yes | public surface |
| 10 | Which downstream repos are blocking in the gate | docparse, email-parse, daneel, ailang-packages blocking; world and motoko advisory at first | downstream gate |

## 7. Implementation waves (2026-11-10)

Each PR was written by an Opus agent in its own worktree, then reviewed by a different Opus agent
that had not seen it, then fixed, and merged only when a review came back clean. **Of the ten
first-round reviews, seven found at least one blocking defect that the author's tests and green CI
had missed** — two of five in wave 1 and all five in wave 2; three passed first time (#2203, #2206,
#2207). Review is not optional for this work.

### Wave 1 — all merged; shipped in v0.59.0

| PR | Scope | Outcome |
|---|---|---|
| #2203 | Release gates M1+M2: hook installer, `release.yml` ci-gate | Merged. Gate also requires `build.yml`'s per-OS jobs (`D-68`); polling window 90 min |
| #2206 | `ollama` Go module replaced by local wire types | Merged. Empty streaming reply is now an error |
| #2207 | Match exhaustiveness core: algorithm package, located RT002, VM silent-Unit fix, warnings in JSON | Merged. Not yet wired into `check` |
| #2208 | `make downstream-check` | Merged after review found five ways it could PASS without measuring |
| #2209 | Type-variable hygiene: #2157, #2076 | Merged after review found a soundness REGRESSION (let-generalisation over open records); the fix unified four free-variable walks into one and closed three older holes |

### Wave 2 — all merged 2026-11-10; on `dev`, tagged in v0.60.0 (tag blocked by the gate, see below)

| PR | Scope | Outcome |
|---|---|---|
| #2220 | Downstream gate: dependency provisioning, 600 s default timeout, candidate runs its own `lock` | Merged after one fix round |
| #2219 | Doc-sample gate (`make check-doc-samples`, in `ci-quick` and on the pre-push path) | Merged after one fix round. 537 samples: 223 pass, 284 fail and are baselined (shrink-only set) |
| #2224 | Canonical module identity (M1b) | Merged after three fix rounds and two reviews |
| #2223 | `JInt`, `decode` rejects trailing data, one shared decoder, prompt v0.16.13, warning `TC_MATCH_005` | Merged after two reviews. **17 downstream sites**, not the 4 the review counted (below) |
| #2222 | Verification honesty part 1: one exit contract, hosted tool probes the solver per call, `--no-verify`, imports note | Merged after three reviews and four fix rounds; each round's fixes introduced the next finding |
| #2218 + infra #17, #21 | Registry validator: final tags only, behind the CI gate; trigger narrowed; the triggers stack applies on push | Merged and **applied**, verified in Cloud Build (positive and two negative controls) |

Also landed directly on `dev` that day, each a CI or tooling fault that was failing good changes:
the auto-push bound (`f2a9807e2`); the gate's false alarm on a sibling checkout edited mid-run
(`dc63d01a5`); `build.yml`'s Windows test budget, out of step with `ci.yml` (`f6988a304`); a 20 ms
sleep-then-assume race in nine daemon tests (`73dac3c4d`); the gate now removes its 3.4 GB of working
trees per run (`e05c600b1`); the maintenance workflow's packages pin (`cfaf5f836`).

**Review tally, both waves: eight of eleven PRs had a blocking defect that the author's tests and green
CI missed.**

### Wave 3 — started 2026-11-11

| Work | Doc | State |
|---|---|---|
| Module-qualified type identity (M2 + resolver half of M3) | m-v1-nominal-type-identity | **Merged** (#2228, b7465ed6a) after four review rounds. Import-only scoping of indirect type names left for M3 |
| Total `show` | m-v1-show-total | **Merged** 2026-11-12 (#2229, becc37909). Prompt v0.16.15; benchmark `run_length_encode` replaced by `_v2` |
| Stdlib renames with `@deprecated` aliases, re-tiering, remaining behaviour fixes | m-v1-stdlib-surface-polish | **Merged** 2026-11-12 (#2230, e52ee05fd). Cache key v12, prompt v0.16.14. Broke scoring for a running eval: a binary run from the repo root loads `./std`, so pin `AILANG_STDLIB_PATH` before merging `std/` changes |
| Non-exhaustive match as an error, `unreachable` | m-v1-match-exhaustiveness | **Merged** 2026-11-12 (#2246, 9eaf0b65e) after a final review of the rebuild. Cache key v13, prompt v0.16.16. Follow-up #2272. 8 downstream sites in 3 repos: motoko fork patched (PR 9 merged); daneel #421 and ailang-packages #195 open — both must merge before any release that contains this |

**Soundness fixes (2026-11-12/13).** Each PR written by one Opus agent, reviewed by another that had
not seen it, sent back until clean. Every first review found a defect the author's tests and green CI
had missed; #2285 took five rounds.

| PR | Issues | State |
|---|---|---|
| #2287 | #2244 match on an alias of an ADT | Merged (d5580243a) |
| #2288 | #2210, #2221 parts 1–3: record rows, occurs check | Merged (5ed4b3a0b); part 4 is #2312 |
| #2289 | #2238 function-literal annotations, `! {}` | Merged (3cef3da7c) |
| #2285 | #2247 class constraints on applied types; primitive aliases | Merged (8797bd79e) |
| #2283 | #2232, #2248 contracts and test bodies type-checked | Merged (287222440) |
| #2286 | #2211, #2240 type names by scope; two declaration rules | Merged (5db322d71) |
| #2284 | #2239 newtype records | In review (third round) |
| #2304 | #2273 signature type variables are binding | Rebasing after first review |

**Rulings (Mark, 2026-11-12):** `==` on a type without `deriving (Eq)` is an error in contracts, as in
bodies; an ill-typed contract in a dependency is fatal for importers (a dependency's ill-typed TEST is
not). **Awaiting a ruling:** a `constraints` field in `iface` output to fix #2290 (class constraints lost
across modules; fix on branch `fix/soundness-exported-constraints`); whether a root module's own
ill-typed test should stop `run`/`serve-api`; pure functions that build an effectful closure (#2322,
checker contradicts the prompt).

**Downstream state.** v0.60.3 (2026-11-12) shipped #2246. Patched: motoko fork (PRs 9, 10), ailang-packages
(#195, #198; eight packages republished). Open: daneel #421; ailang-world #565 (needs world/core 0.1.2);
docparse and daneel pin bumps for the republished packages; motoko fixtures that `check` now rejects
(3 from #2288, 21 from #2289) must move to the fork's compiler-rejected group. The cascade PRs opened by
package agents (ailang-packages #200–#202) delete or weaken contracts and should be closed.

**Found and filed during this stretch, unowned:** #2290, #2303 (VM picks the first loaded ADT for a
by-name match at an unfixed type: silent wrong answer), #2272, #2305, #2312, #2318, #2322–#2327, #2329,
#2308/#2309; Go emitter: #2241, #2259–#2264, #2274, #2292, #2293 (silent wrong answer), #2294, #2299,
#2307, #2317.

**Not started:** integer overflow trap; callback
effect rule; capability-grant changes and Internal command moves; help text; docs rewrites (284
baselined samples) and the migration, security-model and support-window pages; rest of the rc
pipeline; then close-out steps 16–19.

### What the waves changed in the plan

- **New milestone: canonical module identity**, before module-qualified type identity (`D-68`). A
  module inside a package was compiled two or three times under different names and its module-level
  code RUN more than once. The design doc's regex census had found 0 affected programs; a real
  binary found 14 docparse files.
- **Exhaustiveness wiring needs less than §3 says:** a constructor stamp keyed by the canonical
  module, not the whole identity change.
  Confirmed in the wiring PR (exhaustiveness doc §14): the stamp from #2228 plus the type a
  by-name pattern was checked against was enough. A non-exhaustive match is now TC_MATCH_002;
  measured cost 8 downstream sites in 3 repos and 2.1% of passing AI-written programs.
- **The downstream gate is a type-check comparison, not downstream CI.** Exit 0 guarantees the
  candidate's `check` is no worse than the baseline's on every entry, against dependencies provisioned
  from each repo's own pins. It does not run downstream tests. Release runs need `FETCH_DEPS=1`.
  Three entries remain NOT_MEASURED for real bugs in their repos (one unexported import in the motoko
  fork, one invalid manifest in ailang-packages).
- **`VERIFIED` means less than the teaching prompt says**, in two ways now: the solver models `int` as
  unbounded while the runtime wraps (§4), and `requires` is assumed, never checked at call sites, with
  run-time contract checking off by default. Both must be stated in LIMITATIONS, the contracts guide
  and the security-model page before 1.0.
- **A breaking change must be swept with a compiler, not counted.** `JInt` was scoped at 4 downstream
  sites; a sweep of every corpus with a `dev` compiler found 17 (9 in ailang-packages plus 4 frozen
  task-pack fixtures, 3 in the motoko fork, 1 in world). All were fixable with `asNumber`/`getNumber`,
  which exist since v0.54.14, so no release ordering was needed after all. Every remaining breaking
  change gets that sweep **before** merge, and the gate should count the warning: `check` exits 0 on
  these files, so `make downstream-check` cannot see a change that alters behaviour but still type-checks.
- **The roadmap's "last v0.58.x" is stale as a name.** Releases have moved to v0.59.0 and a tagged
  v0.60.0; the breaking window is still open and closes at step 16, whatever that version is called.
- **More than half the code samples in the docs do not pass `check` as written** (284 of 537). The
  number can now only go down.

### Issues filed from review (all pre-existing on `dev`)

- #2210 — a missing record field passes `check` when a declared record return flows through a lambda
  parameter (open-vs-closed record unification never binds the row).
- #2211 — lowercase type names parse as type variables, so the declared type is never enforced.
- #2221 — `check` never terminates on an infinite type through a record; a record update loses the
  other fields (unsound); a user-written `| r` clashes with the internal row name; defaulting keeps a
  separate free-variable walk.
- #2215 — non-blocking follow-ups for #2203, #2206, #2207.

### Found and not yet owned by any doc

- A warm compile cache drops warnings from machine output (`check --json` gives `warnings` on the
  first run and omits the key on the second).
- With plain `--bytecode`, a VM run-time error in a PURE entry falls back and re-runs on the evaluator
  (item B4 in `m-bytecode-vm-parity-bugs.md`).
- `serve-api` / MCP / A2A arguments turn whole-number JSON floats into ints before the program sees
  them.
- A stream that ends without its `done` chunk is still reported as a short success by the ollama
  client.
- The downstream gate's write audit raises a false "could not measure" when another session edits a
  sibling checkout mid-run.
- The hosted `ailang_check` image has no Z3 (needs a non-static base image).
- The infra repo published all of `docs/` to a public Pages site on a push to `main`; the landing page
  now publishes from its own folder (infra #17). Where the AILANG Cloud landing page should live is
  open.

## 7. Suggested next step

Decisions 1–5 are ruled. Send wave 0 and nominal-identity M1 to sprint-planner; the older overflow
design doc's pending quorum round is unblocked by ruling 3. Running
`ailang design-quorum` on the two type-system docs first is worth its cost; on long docs pass
`--max-cost-usd 0.30` or the budget cap drops a reviewer.
