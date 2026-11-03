# Rig: ssa_constant_fold splits harnesses on the same local model (pi 7/7, opencode 1/8)

- **Date**: 2026-11-03
- **Class**: question
- **Recommend**: drop — not actionable from this workspace; the mandated instruments are rig-resident. Re-dispatch to a rig-attended session (see "What would make it actionable" below). No code change is warranted until the session-level evidence is read; do NOT conclude harness-vs-gap from the aggregated numbers.
- **Searched**: `rg ssa_constant_fold` (repo: benchmark spec + frontier refs only, no banked rows); `opencode` in `design_docs/` (`m-eval-cross-harness-comparison`, `m-three-camps-rerun-local-rig`, `m-eval-followups`, `v1-mission.md` #551/#554 precedent); `non_agentic` (`error_categorizer.go`, `v1-mission.md`); `qwen3-8-27b` in `internal/modelreg/models.yml`; prod observatory spans (3,000 scanned — coordinator infra only, zero eval-run chains); `docs/static/benchmarks/os/{latest,history}.json` (aggregated tiers only); git history for `eval_results` commits since 2026-10-01 (none).
- **Source**: prod `user` inbox `inbox_1793718623208_e0255f93`, posted by `rig-value-report`.

## Why

The report asks an empirical question — are opencode's 7 failures on `ssa_constant_fold` (qwen3.8:27b, agent mode, stretch tier) harness friction or an AILANG gap that pi's prompt/loop avoids — and mandates `ailang chains chat` on the opencode rows plus the opencode session DB. Neither instrument exists where this task ran:

- `eval_results/rotation/os-rolling/` is not in this checkout (`eval_results/` holds only `baselines/` and `performance_tables/`; rotation rows are banked on the rig filesystem — the glob in `tools/rig-value-report.sh` confirms the layout).
- The prod observatory (`AILANG_OBSERVATORY_URL`, spans + chains queried 2026-10-19..11-03) carries coordinator infrastructure spans only; local-rig eval runs export to the rig's own local observatory (per the local-ollama-eval skill's state block), so `chains chat` here has nothing on these rows.
- No opencode session storage on this host; `ailang coordinator workers list` shows no rig worker to route a request to.

### What WAS verified from the repo (context for the eventual verdict — hypotheses, not conclusions)

1. **Wall-clock budgets are equal across the disagreeing arms.** `internal/modelreg/models.yml`: `opencode-qwen3-8-27b` and `pi-qwen3-8-27b` both run `ttft_timeout: 1800`, `generation_timeout: 3600`, `hard_timeout_secs: 3600`, `max_tokens_per_bench: 3.0M`. The timeout concentration on opencode rows is not a config asymmetry at the arm level.
2. **The prompt-weight asymmetry is large and measured.** The opencode CLI adds ~100k tokens of framework prompt per request (the AILANG teaching prompt is ~3k of the ~109k observed input tokens per trial — local-ollama-eval skill, "Prompt strategy"), and the dense 27B generates 2.5–3.4x slower than the MoE incumbent the timeouts were tuned against. pi's lean loop pays none of this. This is the leading harness-friction candidate for `timeout 4` on opencode vs `7/7` on pi.
3. **`non_agentic` has strong opencode-lane precedent.** `internal/eval_harness/error_categorizer.go` defines it as a 0-shot answer (1 turn, 0 tool calls) — and the 2026-08-01 nightly incident (`#551`, v1-mission.md iter-128) banked 12/12 `non_agentic` on opencode/qwen3.5 while opencode-qwen3.6 and pi-qwen3.6 passed 4/4 on the same rig the same night; that lane-specific tool-delivery collapse (`#554`) was never root-caused. One `non_agentic` in this report's opencode rows is consistent with that class.
4. **The motoko middle result (4/7, timeout 2 + compile_error 1) is the one arm where an AILANG-language explanation is plausible** — microrag profile, same model — but its sessions are equally unread from here.

## What would make it actionable

Re-dispatch to a session **on the rig** (`studio.eval-rig`), where three things live that this task cannot reach:
1. `eval_results/rotation/os-rolling/v0.54.16..v0.55.0/agent/*ssa_constant_fold*` — the 8 opencode + 7 pi + 7 motoko rows (error_category, `agent_transcript`, duration).
2. The opencode session DB (`~/.local/share/opencode/`) — tool RESULTS, which the banked `agent_transcript` deliberately omits (CLAUDE.md "Look the question up, not the tool").
3. The rig's local chains DB — `ailang chains chat` on the eval-run chains.

Alternatively: have the rig attach the 8 opencode rows' JSON + opencode session IDs to a prod message (the rig already posts via `ailang messages send user`, proven by `rig-value-report`), so remote sessions can read the evidence without rig access. If the eventual answer is "harness friction", it likely routes to the parked `#554` opencode/ollama lane question rather than a new doc; if "AILANG gap", it routes through `design-doc-creator` — neither call can be made from the aggregated pass rates alone.

## Related

- Sibling report same batch: `inbox_1793718624754_c0b782ef` (cp_60166/cp_60159 timeout rows bank `duration_ms=0` — any wall-time analysis of these ssa rows must check for the same instrument defect).
- `#551`/`#554` precedent: `design_docs/v1-mission.md` (iter-128, iter-191 rows).
- Harness prompt-cost analysis: `.agents/skills/local-ollama-eval/SKILL.md` ("Prompt strategy") and `resources/prompt_strategy_findings.md`.