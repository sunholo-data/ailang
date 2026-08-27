# Mission Dashboard — V1

_Snapshot after iteration 291 (2026-08-27). Overwritten each iteration; history lives in the charter STATUS block and the mission log._

## Latest
- **Release**: v0.34.0 · `origin/dev` @ `ed5600da6`
- **Landed this iteration**: [#936](https://github.com/sunholo-data/ailang/pull/936) → squash `ed5600da6` — M1 of `m-prompt-version-freeze-on-first-bank` (decision **D-41(c)**)
- **Live defect repaired**: `prompts/versions.json` recorded a stale hash for `aver`, so the loader **failed outright** on it. Registry audit **58 ok / 1 bad → 59 ok / 0 bad**.
- **Migration**: all 59 prompt versions marked — 19 `banked`, 39 `legacy`, 1 mutable (`v0.16.6`, the active version, zero banked uses).

## In flight / next
1. **M2** — CI gate (`make check-prompt-freeze`, merge-base immutability, mirror-registry check)
2. **M3** — close the agent-mode verification hole (found this iteration: `internal/prompt` never compares `Hash`, and `langreg` converts a load failure into a **success** attributed `"default"`)
3. **M4** — bank-time `prompt_sha256` byte evidence
4. `m-fmt-gate-freeze` (was queue head; deferred by the D-41 directive)

## Loop health
- Cadence: launchd, ~6h slot · driver **PINNED** this fire (`~/.ailang-driver-pin/v1`) after six consecutive unpinned fires
- Routing: controller `opus` · designer **rotation** · planner `opus` (derived) · executor `codex:gpt-5.6-sol` · evaluator `sonnet`
- **Designer lane `pi:ollama/kimi-k3:cloud` FAILED** on first real use (`wall_timeout`, 1802s, 73 tool calls, **0 files written**) → fell back to `claude:claude-fable-5`
- Evaluator: **PASS 91/100, zero blocking**, in its own worktree. generator≠judge held on both axes.
- metered **$0.25** of $5 (two quorum rounds)

## Parked on Mark
- **D-42** (only open row) — standing authorization to reconcile this checkout to `origin/dev` unattended?
- **SonarCloud** red on dev ≥6 analysed commits — inherited, non-required, named not fixed (60.1% coverage-on-new-code, B security rating)
- Two external `mcp-public` `.eml` bug reports + a parser error-cascade report left unacked for an attended session

## Recently resolved
- **D-41 → (c)** answered 2026-08-27, implemented same iteration
