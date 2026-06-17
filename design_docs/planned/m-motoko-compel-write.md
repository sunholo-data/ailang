# M-MOTOKO-COMPEL-WRITE: Compel Reliable Solution-Writing in Motoko

**Status**: REVERTED 2026-06-17 — A/B net +16pp but the loop guard fired 0/18 (non-functional),
`config_file_parser` regressed 3/3→0/3, and pi has no loop guard anyway. Re-scoped to a
request-param (temperature/thinking) approach — see the analysis log's 2026-06-17 A/B entry.
Kept as the record of why the "compel" approach was set aside in favour of "coax".
**Target**: v0.25.x (AILANG side) + draft PR to `arniwesth/motoko_agent` (motoko side)
**Mission item**: #2 — close the motoko-vs-pi AILANG gap (79% → ~88%+)

## Evidence (proven, not inferred)

Targeted transcript diagnostic, 6 failing AILANG benchmarks: **100% correlation between
`WriteFile` and pass.** Every pass called WriteFile; every failure did not — either
explored (`mkdir`/`ReadFile`) then stopped, or emitted 0 tool calls (prose). Same model
under pi writes+iterates reliably (88%). Root cause = **reliability of agentic engagement**,
not path/capture/model-capability. pi's edge: mandatory tool-result feedback + a loop that
doesn't give up early. motoko has **no "definition of done" guard** — it finalizes the
moment the model emits no tool call, even with nothing written.

## The three changes (per routing rule)

### 1. [motoko PR] Definition-of-done loop guard — `src/core/agent_loop_v2.ail`
The deterministic fix. When the model emits no tool call and is about to finalize, if it has
**written nothing** and budget remains, inject feedback and recurse instead of stopping.

- **`LoopTotals`** (line 164) + `zero_totals` (173): add `wrote_files: int`, `write_nudges: int`.
- **`new_totals`** (line 1031): `wrote_files: totals.wrote_files + count_writes(result.tool_calls)`
  (a `pure func count_writes(calls) = #{c in calls : c.name == "WriteFile" || c.name == "EditFile"}`
  — tool name is `call.name`, per line 531/550/603); carry `write_nudges`.
- **Guard at the `NoDecision => … None => done` site** (lines 1144–1155 — exactly the path the
  failures take: `dispatch_solver_candidate → NoDecision → dp7_gate None → done`):
  ```
  NoDecision => {
    if should_nudge_write(new_totals, step_budget) then {
      emit "write_nudge" event;
      let fb: Message = { role:"user", content: WRITE_NUDGE_TEXT, tool_calls:[], tool_call_id:"" };
      loop_v2(… msgs_with_assistant ++ [fb] … {new_totals | write_nudges: +1} …)
    } else { <existing dp7_gate/done> }
  }
  ```
  `should_nudge_write = getEnvOr("MOTOKO_REQUIRE_WRITE","")=="1" && t.wrote_files==0 && t.write_nudges < 3 && step_budget > 2`.
  Reuses the `ContinueWithFeedback` message shape (lines 1135–1140). Opt-in (env-gated) so
  non-eval motoko use is unaffected; capped at 3 nudges + bounded by `step_budget` (no infinite loop).
- `WRITE_NUDGE_TEXT`: "You have not written any solution file yet. Use the WriteFile tool to
  write your full implementation to the solution file, then run it before finishing."

### 2. [motoko PR] Tool description — `src/core/tool_catalog.ail` (line 39)
`WriteFile` description → add "Use this to write your solution implementation before finishing."

### 3. [AILANG dev] Enable the guard + strengthen the task prompt
- The motoko executor (`internal/executor/motoko`) sets `MOTOKO_REQUIRE_WRITE=1` in the agent
  subprocess env for eval runs (turns the guard on for benchmarks only).
- The AILANG agent task prompt (agentprompt) gains an explicit imperative: "You MUST write your
  full solution to the solution file using WriteFile before finishing. You are done only when
  the file is written and the tests pass."

## Validation (gating the PR)
- **A/B** on the 6 failing benchmarks ×3 trials, before vs after, **lock-respecting** (the rig
  lock serialises against the rotation). Metric: WriteFile rate + pass rate. Before captured
  first (motoko unmodified); after with all 3 changes.
- `ailang check` the edited `.ail`; smoke-run one motoko benchmark end-to-end before the A/B.
- Land AILANG changes on `dev`; open a **draft** PR to `arniwesth/motoko_agent` only if the
  A/B shows a real lift.

## Risk
- `.ail` change to motoko core could break ALL motoko runs → compile-check + smoke-run first.
- Guard must respect `step_budget` + nudge cap (≤3) so a refusing model can't loop forever.
- Env-gated so it never affects non-eval motoko usage.

## References
- Diagnosis: [motoko-harness-analysis-log.md](../motoko-harness-analysis-log.md) (2026-06-17 transcript entry)
- pi mechanism: mandatory feedback + no early give-up (investigated 2026-06-17).
