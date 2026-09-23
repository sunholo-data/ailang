# Feedback gate classifier → sunholo/decisions (shadow first, then switch)

- **Date**: 2026-09-18
- **Class**: feature
- **Recommend**: design-doc
- **Searched**: `feedbackgate|feedback gate|feedback_gate` in `design_docs/`; `sunholo/decisions` in `design_docs/ docs/`; `estimated_dispatch_value|classifierVerdict` in `internal/feedbackgate/`; `feedback_gate_audit` in `internal/`; `ailang pkg-docs sunholo/decisions` (0.2.0)
- **Source**: Mark's proposal, attended 2026-09-18 (message, task-e8444bf9)

## Why design-doc

The audit already rules the direction: this is **site #1** of `design_docs/planned/m-ai-decide-system-one-audit.md` — "Drop-in — first candidate. Noul / Noul / Choice / Score, exactly", with a recommended order whose step 1 is verbatim this proposal (shadow beside Haiku via an embed shim, log both into the existing audit table, compare on ~200 messages, then a design doc to switch). The proposal ratifies that plan, and the field mapping verifies first-party: `classifierResult` (`internal/feedbackgate/classifier.go`) is two bools (`is_genuine_feedback`, `is_prompt_injection`), an enum (`best_category`), a level (`estimated_dispatch_value`), with `reasoning` read by nobody — `classifierVerdict` never touches it. The package's 0.2.0 `decideOrFallback` exists and its Degraded semantics (confidence forced to 0.0, so `gate` escalates for any t > 0; `MissingKey` not retried) match the proposal's fallback requirement. This is therefore not a new idea needing invention — it is a ratified audit item whose execution still crosses three design-doc gates the proposal itself names, plus deltas the proposal gets wrong.

**The deltas (found, not restated — these are what the design doc must resolve):**

1. **The mapping does not match the gate's contract.** The proposal's mapping reproduces the *package docs' worked example*, not the real gate. Verified against `classifierVerdict` (`internal/feedbackgate/classifier.go`):
   - Injection currently maps to **`ActionReject`** (TTL cleanup), not `ActionFile`. The proposal (and the pkg-docs example) say `p >= 0.2 -> file`. Softening reject→file is a change to the gate's contract (abuse posture), not a transport swap — it must be decided, not silently inherited from the docs example.
   - `estimated_dispatch_value == "none"` currently files **before** the genuine check. The proposal states no threshold for its `value` Score answer at all, and its Score has **3 levels** against the real enum's **4 values** (`high|medium|low|none`) — the "none → file" branch has no home in the proposed mapping.
   - The current category branch is **not** confidence-gated: it files unless `best_category == strippedCategory(in.Category)` — the classifier must *match the user-declared category*. The proposal replaces this with `gate(category, 0.7)` → dispatch on Act. That drops the match-against-declared-category semantics entirely; whether that is intended (and how shadow compares the two arms when their category tests differ) is a design decision.
   - The pkg-docs example's Choice options are `bug/feature/docs/other`; the real schema is `bug/feature/docs/limitation/spam`. "It compiles" is true; "it is the same gate" is not.
2. **The data-boundary ruling gates shadow, not just the switch.** The proposal says "RULING NEEDED BEFORE SWITCHING", but a shadow call already sends the feedback body to TypeSafe via OpenRouter — for those ~200 messages the data flow has switched. The package's own AGENT.md (`ailang pkg-docs sunholo/decisions` → "Data boundary") says the ruling must be extended "before switching, not after", and the audit doc says the same. This is an attended ruling (Mark), and it is the first precondition, not step 3's.
3. **The shadow shim is multi-file and changes a recorded format.** Go route via `internal/embed` (`Engine.Call`/`CallPreserveFloats`, the `budget_checker.ail` precedent) means: a new AILANG policy program, an embed shim wired into the M3 stage of `feedbackgate.Decide`, and — for D6 banking — an extension of `gateAuditPayload` (`internal/coordinator/feedback_gate_audit.go`), whose current payload (`message_id/action/reason/category/from/inbox/est_cost_usd/dry_run/would_reject`) has **no model, id, distribution, usage or degraded-why fields**. Changing the audit row format is a file-format contract change (rubric row 4). Hand-rolling the HTTP call in Go is already ruled out by the audit doc and the package AGENT.md.
4. **Citation note:** the proposal's "AGENT.md says the same" refers to the *package's* agent notes (served by `ailang pkg-docs sunholo/decisions`), not this repo's AGENTS.md, which is silent on data boundaries. The design doc should cite the former and not imply a repo-level ruling exists.

**Not direct-fix:** the estimate is trivially over `DIRECT_FIX_MAX_LINES`/`DIRECT_FIX_MAX_FILES` — a shadow harness touching `internal/feedbackgate`, `internal/coordinator`, plus a new embedded `.ail` program and audit-schema extension. Not `duplicate-of`: the audit doc rules the *direction* but does not resolve the four deltas above, and the proposal adds new decisions (fallback via `decideOrFallback`, reject→file question, Score mapping) the audit predates (it cites 0.1.2; 0.2.0's fallback shipped since).

**Recommended next step:** design doc covering shadow + switch in one document (the proposal's step 3), sequenced after the attended data-boundary ruling, with the mapping deltas as explicit decisions (D-row style) rather than inherited-from-docs defaults. Nothing acts on a Jev answer until the shadow comparison passes — the proposal already says this and it matches the audit doc's gate.

## Sources checked

- `design_docs/planned/m-ai-decide-system-one-audit.md` — site #1, recommended order step 1 (read in full)
- `design_docs/implemented/v0_40_1/m-ai-decide-system-one.md` — D4 (no default threshold), D6 (bank the whole Decision)
- `ailang pkg-docs sunholo/decisions` (0.2.0) — migration example, `decideOrFallback`, data-boundary line
- `internal/feedbackgate/classifier.go` (`classifierResult`, `classifierVerdict`), `decide.go` (M1–M3 staging)
- `internal/coordinator/feedback_gate_audit.go` (`gateAuditPayload` — no D6 fields yet)
- `internal/dashboard_transforms/budget_checker.ail` — the embed precedent exists