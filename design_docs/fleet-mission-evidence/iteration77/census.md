# Iteration 77 — census for `agent-tool:workspace-routing-discovery-timeout`

Measured 2026-11-02 by the controller (claude-opus-5-5), read-only, on the rig.

## Instrument

- Store: `~/.codex/sessions/2026/10/*/*.jsonl`, 1,049 session files, first `2026-10-01T01-51-14`, last
  `2026-10-30T21-35-48` (no `2026/11` directory exists). This is the only codex session store on the rig.
- Script: [`census-classify.py`](census-classify.py) (stdlib, run `python3 -I`). It prints one row per JSONL record
  that contains `workspace routing discovery`, with the session date, cwd, record type and a context snippet.
  Output: [`census-occurrences.tsv`](census-occurrences.tsv), 65 records in 27 files.
- Wording variants: `workspace routing discovery`, `routing discovery` and `workspace routing` each match the
  same 27 files, so the census does not rest on one wording.
- Repo: `git grep -i 'workspace routing\|routing discovery'` at `ffabe3278` finds no harness file. The two hits are
  unrelated v0.9.0 design docs about message routing. Driver logs `/tmp/ailang-mission-*.log` (14 files): 0 hits.

## Classification (from the snippet column)

Most of the 65 records QUOTE the ticket: sessions reading `ailang mission ticket open` output, or the stapledon
controller writing the ticket. Only records whose text is an error event or a sub-agent's final message are
occurrences:

| When (local) | Session | Originator / cli | Event | Text |
|---|---|---|---|---|
| 10-02 10:29 | stapledon codex sub-agent `/root/evaluator_transport` | `codex_exec` 0.159.2 | `task_complete.error` | `workspace routing discovery timed out` (`codex_error_info: other`) |
| 10-02 10:15 | stapledon codex controller (parent) | `codex_exec` 0.159.2 | sub-agent FINAL_ANSWER | `Agent errored: workspace routing discovery timed out` |
| 10-09 12:14–15:56 | 3 attended stapledon sessions | `codex-tui` 0.162.0 | 6 × `task_complete.error`, 2 × FINAL_ANSWER | `workspace routing discovery unauthorized (401)` |

So the ticketed timeout is ONE event (the sub-agent's error, plus the parent receiving it). The 401 variant on 10-09 is
attended `codex-tui` work, not a mission fire, and nobody ticketed it.

**Denominator after the last occurrence**: 652 session files dated after 10-09. 39 of them made 107 `spawn_agent`
calls (`payload.type == function_call`, `name == spawn_agent`; up to 10-09: 29 sessions, 161 calls). None of the 652
has an error or final-message occurrence. The only post-10-09 hits are ticket quotes (10-10, 10-29). Window ends
10-30T21:35; codex-cli is now `0.164.0`.

## The chain inside the ticketed fire (stapledon iteration 9, 10-02)

From the parent session `rollout-2026-10-02T10-15-07-…` and the sub-agent `rollout-2026-10-02T10-29-17-…`:

1. 08:18:16Z the codex controller called `spawn_agent` with `model: "sonnet"` for the evaluator. That is the D-FLEET-24
   path: a bare alias handed to the native host. Its paired `function_call_output` (same `call_id`) reads
   ``Unknown model `sonnet` for spawn_agent. Available models: gpt-6.1-sol, gpt-6-astra, gpt-6-sol, gpt-6-luna, gpt-5.6-sol``.
   (Six seconds earlier the designer spawn with `pi:ollama/kimi-k3:cloud` got the same refusal; the controller then used
   a `designer_transport` sub-agent the same way.)
2. 08:28:55Z it called `spawn_agent` with `task_name: evaluator_transport`, `model: gpt-6.1-sol`, i.e. a codex
   sub-agent used only to carry the Claude CLI.
3. 08:49:56Z that sub-agent ran
   `env -u ANTHROPIC_API_KEY -u ANTHROPIC_AUTH_TOKEN claude-sub --model claude-sonnet-5-5 … --print --output-format json`.
4. 09:03:15Z the sub-agent's codex turn ended `error: workspace routing discovery timed out`. This is codex's own
   multi-agent transport, not ours.
5. 09:04:32Z the parent called `followup_task` on `evaluator_transport`. 09:19:56Z the resumed turn completed:
   *"Actual independent Sonnet5.5 result: PASS, 91/100, no hard fails."*

Both sessions ran with `sandbox_policy: danger-full-access`, `approval_policy: never` (their `turn_context`). That is
how the driver launches every codex controller: `tools/launchd/mission-control.sh:2431`
`--dangerously-bypass-approvals-and-sandbox`.

## What this means for D-FLEET-24

- The ticket is a symptom of D-FLEET-24, not a separate defect. The transport existed only because the bare `sonnet`
  alias could not be spawned natively. Option A (reroute to a provider-pinned fallback) removes the transport. Option B (run
  the `claude:` recipe through the shell) removes the need for it, but the skill's "heavy roles never inline" invariant
  (`gate-3-route.md:8-12`) may still lead a codex controller to wrap the recipe in a sub-agent (unmeasured). The error is
  codex-internal and can hit any codex sub-agent: the 10-09 `401` hit non-transport sub-agents (judge, iteration 77).
- Option B's stated premise, "`claude-sub` working from inside the codex sandbox (unmeasured)", now has one first-party
  reading. In a session whose `turn_context` reads `danger-full-access` (the policy the driver's launch sets),
  `claude-sub --model claude-sonnet-5-5` ran in a codex sub-agent's background shell and returned a verdict. That is one
  event with codex-cli 0.159.2. That this fire came from the driver is inferred (the ticket calls it an "API scheduled
  invocation"). Under a *restricted* codex sandbox it is still unmeasured.
