# Mission Dashboard — V1

_Snapshot, overwritten each iteration. History lives in the charter STATUS block + mission log._

**Last iteration**: 282 · 2026-08-26 · LANDED (pending CI)

## In flight / just landed
- **`78f9f4f42`** — 342 `.ail` files reformatted to `ailang fmt` canonical form, executing Mark's
  **`D-38`(a)** ruling. Corpus canonical **63 → 405** of 450; drift **342 → 0**.
  Comments **7865 → 7865** (0 per-file delta, poisoned control fired); `ailang check` rc unchanged
  on 342/342. Gates green in **both** arms.

## Next picks
1. `m-stdlib-freeze-gate-path-mismatch` — `make test-stdlib-freeze` names `goldens/stdlib` (absent)
   while the script writes `.stdlib-golden` (present); the gate cannot run and is in no workflow.
2. `m-format-comment-brackets-break-wall-scan` — a `{`/`[` in comment *text* is read as the block wall.
3. Formatter **printer** rows — all blocked on `D-39`.

## Parked on Mark — 0 OPEN decisions (attended session 2026-08-26)
**All 7 ruled.** Ledger validates at 40 rows, 0 open. Headlines only — reasoning is on the rows.
- `D-38` → **(a) REFORMAT stands**; #893 correct and merged. Its cited directive is **no longer
  retrievable and that is EXPECTED** — Mark posted it on #852 and then deleted it (confirmed in
  session). A provenance note is on the row so ghost discipline does not re-open this every read.
- `D-39` → **YES, add a line-width limit, then reformat the corpus a SECOND time.** **Scope of
  (a) ruled explicitly: it ratifies the DIALECT and the direction of travel, NOT the line layout.**
  So `m-fmt-typedecl-printer-needs-multiline-emit` is **UNBLOCKED, not re-gated**, and the fmt gate
  must NOT be wired or frozen until the width fix lands and the second pass has run.
- `D-30` → **(b) same-binary `os.Executable()`**, with NO PATH fallback (unlike all 9 in-repo
  precedents) and an injected path in tests.
- `D-31` → **(a) split authoring vs review lanes.** Widening has no candidate today.
- `D-32` → **(b) keep strict** — `inconclusive` stays in the denominator, reported as its own
  named bucket. Not the same axis as `not_applicable`.
- `D-36` → **(c) raise the budget** while findings shrink in kind; park on new-in-kind; cap 5.
- `D-37` → **(b) mode-polymorphic `std/ai.call`.** The (a) stopgap was DECLINED, so
  `verify-examples-toplevel` and local `make ci` stay red until (b) lands — a REGISTERED
  exemption, not a new one. GitHub CI is green and unaffected.
- `D-40` **NEW** → **no independent judge for 5 iterations**; cause is a harness-injected
  instruction absent from every file the loop can read. Driver prompt now carries an explicit
  Agent-tool request. **UNVERIFIED until the next unattended fire** — if the routing block still
  reads `NOT spawned`, the escape-clause theory is REFUTED and `D-40` re-opens.

**Driver defect fixed in the same PR (iteration 282's Gate-0 find):** V1 alone read the
fleet-shared `mission-gh-issue` (**745**, CLOSED) instead of `mission-v1-gh-issue` (**852**, open);
siblings were correctly namespaced. Gate 0 reads Mark's directives from that value.

## Loop health
- Bookkeeping issue: **#852** (namespaced key). ⚠ The **driver** exports `MISSION_GH_ISSUE=745`
  (closed) — `mission-control.sh:68` reads the fleet-shared bare key for `v1` only. Filed.
- ⚠ PATH `ailang` is stale for the **5th** consecutive iteration (ignores `AILANG_MESSAGES_STORE`);
  every iteration must build its own ldflags-stamped binary.
- Running skill is `065a4f16c` behind origin (main checkout 22 behind); delta read each iteration.
- dev CI: green except the standing non-required SonarCloud `new_coverage` red.
- Routing: controller `opus` only. **Agent tool unavailable for the 5th iteration** — no
  designer/planner/executor/evaluator, so no independent judge. metered **$0.00** of $5.
