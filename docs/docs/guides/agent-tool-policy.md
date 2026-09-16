---
title: Agent tool policy — the ailang_only lane
description: How an agent is restricted to executing programs only through AILANG, how the policy is written and delivered, and how to read it back on a banked row.
---

# Agent tool policy — the `ailang_only` lane

An agent on the **`ailang_only`** tool policy reads and writes files normally and executes
programs **only** by submitting AILANG source to an operator policy it cannot edit. There is
no shell. This page is the operator's view; the design is
`design_docs/planned/v0_35_0/m-agent-ailang-only-execution.md`.

## The three pieces

| Piece | What it is | Where |
|---|---|---|
| **Gate** | `ailang run --policy <agent-policy.toml> prog.ail` — typecheck → entry → the program's declared effect row must be a subset of `allowed_caps`; caps, `net_allow` and `fs_sandbox` come **from the policy**; `--caps`, `--no-budgets`, `--allow-env` are refused; denial prints the decision JSON and exits 2 without running | `cmd/ailang/run_policy.go` |
| **Tool** | `ailang_run({path, args_json?})` — a pi tool that calls the gate with `$AILANG_AGENT_POLICY`; returns `{admitted, exit_code, decision, policy_digest, stdout, stderr}`; with no policy attached it **refuses with a named reason** (default-deny) | `.pi/extensions/ailang-exec.ts`, shipped by `ailang pi install` |
| **Profile** | `tool_policy: ailang_only` ⇒ `--no-builtin-tools --tools read,edit,write,ailang_check,ailang_run,builtins_search` (`ailang pi tool-profile ailang_only` prints it; `pi.go` and the resident read the same string) | registry field / `RESIDENT_TOOLS` / `eval-suite --tool-policy` |

## Writing a policy

```toml
allowed_caps  = ["IO", "FS", "Process"]   # the WHOLE authority a submitted program can have
fs_sandbox    = "/workspace"              # must NOT contain the policy file's own directory (D4)
process_allow = ["git:pull", "git:status"] # Process narrowed to subcommands (git:* = any)
# net_allow     = ["api.example.com"]      # required when "Net" is allowed; https only unless
# net_allow_http = true                    #   net_allow_http = true
entry         = "main"
```

The fine-grained caps are enforced by AILANG's own handlers (`--process-allowlist`,
`--net-allow-domains`), fed from the policy — a program admitted with `Process` and
`process_allow = ["git:status"]` gets `Ok` from `exec("git", ["status"])` and
`Err(NotAllowed(git push))` from `exec("git", ["push"])` (measured). Rules the gate enforces on
the file itself: every cap must be a real effect; `FS` needs an `fs_sandbox`; `Process` needs a
`process_allow`; `Net` needs a `net_allow`; `[budgets]` are **refused** (nothing enforces them at
run time yet). Keep the lists short: they are the boundary.

## Delivering it

- **Coordinator agents**: `tool_policy: ailang_only` and `policy_path: <file on the coordinator
  host>` on the agent in the registry; `ailang coordinator agents <id>` shows both declared and
  effective. Locally the path is forwarded as `AILANG_AGENT_POLICY`; for Cloud Run Jobs the
  coordinator reads the file and forwards its **content** as `AILANG_AGENT_POLICY_TOML`, which
  `execute-job` materialises read-only under `~/.ailang/agent-policy/` (outside the workspace).
  A missing `policy_path` file fails the dispatch loudly rather than sending a refusing agent.
- **Resident instances**: `resident-instance.sh create|update … --policy-file agent-policy.toml`.
  The file travels as `AILANG_AGENT_POLICY_TOML` and `boot.sh` materialises it **read-only
  outside the sandbox** (`~/.resident/policy/`); with no `bash` there is no `chmod` to undo
  that. Without a policy the resident boots and `ailang_run` refuses.
- **Evals**: `ailang eval-suite --agent --tool-policy ailang_only --policy-file … --models …`.

## Reading it back

Every agent-mode row banks `tool_policy` (the effective list, or `<cli default>`) and
`policy_digest` (sha256 of the policy). **Absent means unmeasured** — rows before 2026-09-16
have neither. Compare `ailang_only` rows only with each other, or against bash-lane rows as an
explicit A/B (`ailang eval-paired`), never pooled. A denied program banks as
`error_category: policy_violation`; `decision.missing_from_policy` names the effect.

## What the model is told

`ailang-exec.ts` injects a lane section — no shell, the five tools, the policy's allowed effects,
the sandbox root, the Net hosts and Process commands, and "narrow the program on a denial" — as
**both** a system-prompt section and a conversation message. Both, because the
`ollama/glm-5.3-flash:cloud` route discards the system role entirely (measured 2026-09-16;
deepseek via ollama and OpenRouter honour it).

## What it does not do

`read`/`edit`/`write` are still pi's own tools: they are bounded by the container and the
FS sandbox for AILANG programs, not by the policy. Per-user isolation on a shared resident is
a separate matter (resident design D11).
