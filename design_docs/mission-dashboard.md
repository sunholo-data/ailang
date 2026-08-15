# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-15 ~06:50 local (iteration 205)

## Now
- **v0.33.1** · `dev` @ `c095f1f0e` — squash of PR #724. Gate 3b green (21 checks, 4/4 required,
  `SonarCloud` `success`, so iteration 204's fix holds at the gate, not just in argument).
- **The `-coverpkg` decision is made, and the answer is NO.** Design doc landed
  (`planned/v0_33_2/m-coverage-cross-package-attribution.md`, Planned): keep own-package
  (**LOCALITY**) semantics for the gated/badged/Sonar metric; add a separate, non-gating
  `test-coverage-xpkg` diagnostic for triage.
- **The crux inverts the intuition** — iteration 204's defect was a function with *no own-package
  test*, so `-coverpkg` would have painted it **green**, killing the true positive. Sonar was right.
- **Two of the queue row's own premises are REFUTED** (A/B, 105 packages, 3 replicates/arm): runtime
  **89/78/82s → 92/79/83s** (~+1–4%, not "material"); `total:` **45.5% → 48.1%** — it moves **UP**,
  both arms far above the 29% gate. The hazard is the gate silently **loosening**, not breaking.
- **The real cost was never named**: merged profile **5.7 MB → 599 MB** (~105×), and the Sonar step
  is `continue-on-error: true` (`ci.yml:258`) — an ingest failure would be **silent**.
- Quorum **BLOCKED ×2**, both reviewers present both rounds, `metered=$0.1454`; round 2 closed under
  the narrow-refinement carve-out, both objections **measured and confirmed** (`XC1` was vacuous —
  two packages sit in the profile with *every* counter zero). Separate finding: **CI runs the
  coverage suite twice** — 492s of a 1127s critical-path `test` job.

## Next
1. **`D-COV-1` parked on Mark — one word.** Does the coverage number mean **LOCALITY** or
   **EXECUTION**? Recommendation LOCALITY. **No sprint runs on the doc until he answers.**
2. `#717` (module-only allowlist skips expiry) · `#709`/`#649` correctly open · `#610` infra-gated
   · `#613` blocked on `D-1`.

## Loop
- launchd, fired from the driver pin. Controller **opus** · designer **claude-fable-5** (rotation,
  `claude-sub`) · quorum `gpt5-6-sol` + `gemini-3-1-pro` · planner/executor/evaluator **not fired**
  (NEW-DOC lane — the quorum is the judge).
- Skill drift **CLOSED** (`cmp`-identical to `origin/dev`; pin == `origin/dev`, 0 ahead). Nightly
  2026-08-15 self-declared **INVALID** (`infra_outage`, 5/12 unmeasured) — no verdicts, nothing owed.

## Parked on Mark (issue #635)
**`D-1`–`D-14`** unchanged · **`D-COV-1`** new · `D-15`/`D-16`/`D-17` remain discharged.
