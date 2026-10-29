# sprint-planner and the coordinator

Moved from SKILL.md to keep it under its context-doc baseline.

- **PLAN LOCATION:** Put plans and associated files in `design_docs/planned/<version dir>/` or the planned root as `sprint-plan-<slug>.md`, never in `design_docs/planned/ailang-core-triage/`, even when the design row lives there; see [template rules](sprint_plan_template.md#placement-and-human-decision-rules).
- **HUMAN-DECISION GATES:** Require a still-OPEN decision cited by design section or decision-ledger row ID; copy already-ruled decisions as rulings, never gates; each gate says **STOP: commit current work, report `BLOCKED: <decision>`, and end the run** (see [template rules](sprint_plan_template.md#placement-and-human-decision-rules)).

**Merging a `coordinator/task-*` sprint-plan PR is the approval:** the cloud daemon (`internal/coordinator/daemon_landed_cards.go`, every ≤10 min) approves the task and fires the handoff to sprint-executor, which writes code. Do not merge a plan PR you only meant to read.

### Receiving Handoffs from design-doc-creator

The sprint-planner receives:
```json
{
  "type": "design_doc_ready",
  "correlation_id": "task-123",
  "design_doc_path": "design_docs/planned/v0_6_3/m-semantic-caching.md",
  "session_id": "claude-session-abc"
}
```

### Sending Tasks to sprint-planner

```bash
# Direct task (skip design-doc-creator)
ailang messages send sprint-planner "Plan sprint for M-CACHE feature" \
  --title "Sprint: M-CACHE" --from "user"

# Reference existing design doc
ailang messages send sprint-planner '{"design_doc_path": "design_docs/planned/v0_6_3/m-cache.md"}' \
  --title "Sprint: M-CACHE" --from "design-doc-creator"
```
