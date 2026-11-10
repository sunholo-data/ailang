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

## 7. Suggested next step

Decisions 1–5 are ruled. Send wave 0 and nominal-identity M1 to sprint-planner; the older overflow
design doc's pending quorum round is unblocked by ruling 3. Running
`ailang design-quorum` on the two type-system docs first is worth its cost; on long docs pass
`--max-cost-usd 0.30` or the budget cap drops a reviewer.
