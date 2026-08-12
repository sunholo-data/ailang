# Motoko Mission — iteration log (append-only)

One entry per mission-control iteration, newest LAST (append). Fixed template — keep every
section, write "none" rather than omitting. Same template as
[v1-mission-log.md](v1-mission-log.md); do not diverge it, so cross-mission comparisons parse.

```markdown
## N — YYYY-MM-DD — <headline>
**Picked**: <backlog item + why it was top>
**Reality check**: <what git/code verification of the doc's status found>
**Shipped**: <commits/branches/PRs, evaluator result + score, or "parked: reason">
**Routing evidence**: model=<m> task-class=<design|plan|execute|evaluate|mechanical>
  round1-score=<n> rounds=<n> corrections=<n>
  provider=<p> agent=<a> cost=<$<n>|quota-bucket:weekly-fable|quota-bucket:weekly-opus|unknown>
**Ruled out**: <hypotheses/approaches refuted this iteration — the anti-re-chase ledger>
**Retro lane**: <skill-fix: file+change | process-fix: change | backlog: new doc | none>
**Next**: <what iteration N+1 should pick up>
```

**Mission-specific note on the "Ruled out" ledger.** This mission's history is dense with
hypotheses that felt right and were wrong — "switch to qwen3.6", "it's a model wall", "the docx
loop is one bug". Every motoko conclusion so far has bottomed out in a harness bug. Write the
refutation down with its evidence; the ledger is the point.

**An idle iteration is a valid entry.** While the epic's Phase-0 gate is closed (see the charter's
Guardrails), exhausting the unblocked queue is a correct outcome. Record it as a real entry with
`**Shipped**: parked: Phase-0 gate closed, unblocked queue empty` rather than pulling gated work
forward to look productive.

---

## 0 — 2026-08-12 — charter ratification: quorum blocked twice, all four objections true, one escalated to Mark

**Picked**: Iteration 0 — ratify the rewritten charter. Not a sprint by construction (bootstrap
guide step 6); the deliverable is an agreed bar + queue, run attended with Mark.

**Reality check**: the 2026-06-24 charter was 7 weeks stale — 3 of its 4 stated goals solved or
superseded upstream (canonical prompt delivery solved; context-length superseded by Arni's affine
calibration; empty-response retry superseded by `motoko-ext-empty-stop-guard`). Rewritten rather than
edited; the original is preserved byte-identical in `motoko-mission-status-archive.md`. Bootstrap
verified end-to-end: separate checkout, namespaced state, distinct pidfile (dry-run), kill switch
armed BEFORE `launchctl bootstrap`, `RunAtLoad` fire confirmed to exit at Gate 0 with rc=0 and zero
tokens.

**Shipped**: charter + log + archive + bookkeeping issue #663 (`bc2fc3c2d`, `98ffaf5cf`); clause 6
(executor-fleet graduation) added on Mark's direction; Premise Verification Log V1-V21 added under
quorum pressure. **NOT ratified** — see Ruled out.

**Routing evidence**: model=gpt5-6-sol,gemini-3-1-pro task-class=design-review
  round1-score=blocked rounds=2 corrections=4
  provider=openrouter agent=ailang-design-quorum cost=**$0.115** (round 1 $0.0535, round 2 $0.0613)
  controller=claude-opus-5 (in-session, attended) cost=quota-bucket:weekly-opus

**Ruled out**:
- *"A charter is exempt from premise verification because it is direction, not design."* **Refuted**
  by `gpt5-6-sol` round 1. A charter's claims gate isolation, routing and queue order — they are
  more safety-bearing than a design doc's, not less. V1-V21 now exist.
- *"Inheriting V1's `go-compiler` verification is sound because it is the same repo."* **Refuted** by
  `gemini-3-1-pro` round 2, correctly: `bin/ailang` is a **per-working-tree** artifact and this is a
  separate clone. Running it first-party (V19) then surfaced **V20**, which no reviewer could have
  predicted — `make quick-install` writes the SHARED `~/go/bin/ailang` that V1 and the eval rig
  resolve through. A separate checkout isolates the working tree, **not the installed toolchain**.
  New guardrail written.
- *"Requiring a loud fallback signal of the future `motoko:` lane is sufficient."* **Refuted** by
  `gpt5-6-sol` round 2. The mission runs TODAY on `codex→pi→opus` with no such guarantee, in a
  charter that cites World losing five iterations to precisely that. Measured as V21 and **confirmed**:
  demotion is `log`ged at driver lines 360/392 and posted to the issue at none of its 4 `gh issue
  comment` sites (control: the driver does call `gh`, 8 hits — the instrument works).
- *Not refuted, and worth recording as holding:* the separate-checkout decision. V1 was mid-iteration
  (pid 71129, 70 min elapsed) during this very bootstrap, so a shared tree would have contended
  immediately rather than eventually.

**Retro lane**: backlog — queue item 2 (loud lane-degradation notice in the driver). Deliberately NOT
taken as a skill-fix this iteration: the driver is frozen core, and the defect affects **v1, world
and motoko**, so ownership is Mark's routing call rather than this mission's to assume. No skill edit
spent (Gate 5 budget intact).

**Next**: Mark ratifies (or amends) the bar + queue. Then iteration 1 picks queue item 3
(disposition the 52 fork commits) — the top item that is neither gated on #154 nor awaiting a routing
decision.
