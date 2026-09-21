---
title: Agent tool policy — the ailang_only lane
description: How an agent is restricted to executing programs only through AILANG, how the policy is written and delivered, what the runtime enforces beneath it, and how to read it back on a banked row.
---

# Agent tool policy — the `ailang_only` lane

An agent on the **`ailang_only`** tool policy reads and writes files inside an operator-chosen
sandbox and executes programs **only** by submitting AILANG source to an operator policy it cannot
edit. There is no shell. This page is the operator's view; the designs are
`design_docs/implemented/v0_39_0/m-agent-ailang-only-execution.md` (the lane) and
`design_docs/implemented/v0_41_0/m-executor-policy-hardening.md` (the enforcement beneath it).

## The three pieces

| Piece | What it is | Where |
|---|---|---|
| **Gate** | `ailang run --policy <agent-policy.toml> prog.ail` — a **supervised** run: the parent resolves the policy, arms `timeout_ms` before the worker exists, starts this binary as a worker in its own process group with a control pipe for the admission decision, an allowlisted environment (restricted mode) and an output cap; the worker typechecks → entry → the program's declared effect row must be a subset of `allowed_caps`; every widening flag (`--caps`, `--entry`, `--env*`, `--net-*`, `--stream-*`, `--stdlib-path`, `--package-dir`, `--ai*`, `--routing-*`, `--fs-max-bytes`, …) is refused **by name**; denial prints the decision JSON and exits 2 without running; a timeout or output cap exits 3 with a `policy-result:` envelope | `cmd/ailang/run_policy.go`, `run_policy_supervise.go` |
| **Tools** | `ailang_run({path, args_json?})` calls the gate. `ailang_read/ailang_write/ailang_edit` and `ailang_cli({op, …})` are served by **`ailang policy-tool`**, a typed endpoint in the Go binary backed by the same resolved policy: file paths are root-anchored (a path outside the sandbox, or a symlink that leads outside, fails inside the syscall), and each CLI op has a schema — the argv is built in Go from validated fields, never composed by the model or the extension. With no policy attached every tool **refuses with a named reason** (default-deny) | `internal/policytool`, `.pi/extensions/ailang-exec.ts` |
| **Profile** | `tool_policy: ailang_only` ⇒ `--no-builtin-tools --tools ailang_read,ailang_edit,ailang_write,ailang_check,ailang_run,builtins_search,examples_search,ailang_cli` (`ailang pi tool-profile ailang_only` prints it). pi's native `read`/`write`/`edit` — which see the whole host filesystem — are **not** in the lane | `internal/executor/toolpolicy.go`, `internal/executor/pi/toolnames.go` |

## Writing a policy

```toml
security_mode = "restricted"              # absent = restricted; "trusted_host" for host integrations (below)
allowed_caps  = ["IO", "FS"]              # the WHOLE authority a submitted program can have
fs_sandbox    = "/workspace"              # must NOT contain the policy file's own directory (D4)
# net_allow      = ["api.example.com"]    # required when "Net"/"Stream" is allowed; https only unless
# net_allow_http = true                   #   net_allow_http = true; every redirect hop is re-checked
# cli_allow      = ["iface", "docs:search"] # ailang_cli ops; absent = the read-only default set
timeout_ms    = 5000                      # the WHOLE invocation, enforced by the supervisor (positive, ≤ 24h)
# max_source_bytes / max_module_graph_bytes / max_output_bytes / max_fs_transfer_bytes
#                                          # restricted defaults: 1 MiB / 16 MiB / 8 MiB / 8 MiB
entry         = "main"

# [budgets]                               # operator ceilings for the whole run: 0 = zero operations,
# FS = 100                                # absent = unlimited; a source @limit can only tighten
```

**`security_mode`.** `restricted` (the default, and the only mode the `ailang_only` lane accepts)
admits only effects with a confined adapter — `IO`, `FS`, `Net`, `Clock`, `Rand`, `Stream` — and
refuses everything else in the registry with a named migration. It also refuses a configured
HTTP proxy (`E_NET_PROXY_REFUSED`: the destination address cannot be pinned behind one) and has
no localhost/private/metadata grant. `trusted_host` keeps operator-approved host integrations —
`Process` with `process_allow`, `AI` with `ai_provider`, `Env`, `Secret` — with conspicuous
provenance (`security_mode` is banked on the admission line) and **no confinement claim**: the
worker gets the operator's full environment, current proxy semantics, and a loopback entry in
`net_allow` is honoured as an explicit grant. An `ailang_only` agent with a `trusted_host` policy
is refused at dispatch (`executor.CheckLanePolicy`) — the lane's "no shell" claim cannot sit on a
host-integration grant; such an agent declares `tool_policy: full` instead.

**Migration (2026-09-21).** A policy that admits `Process`, `AI`, `Env` or `Secret` with no
`security_mode` now fails at startup: `allowed_caps admits Process, which restricted mode has no
confined adapter for — set security_mode = "trusted_host" … or drop it`. Either drop the grant
(the lane then has no `git` at all — there is no confined Process adapter yet) or set
`security_mode = "trusted_host"` **and** `tool_policy: full`. The deployed lane policies that need
a decision are `pkg-ailang-only.toml`, `ailang-only-executor.toml` (both admit `Process`) and
`daneel-executor.toml` (`Process`, `AI`, `Net`) in the multivac config repo.

**Web search for programs — `std/web`.** `webSearch(query, max)` and `webFetch(url)` are `{Net}`
effects backed by a fixed endpoint on `ollama.com`; the runtime reads `OLLAMA_API_KEY` itself, so the
program never holds a key and cannot name a host. A lane that should search grants `Net` and lists
`ollama.com` in `net_allow` — nothing else. Without `Net` the program is denied at admission
(`missing_from_policy: ["Net"]`); with `Net` but without the host it gets `DisallowedHost(ollama.com)`.

`ai_provider` is the model an `AI`-cap program calls — the lending boundary for the AI effect
(`trusted_host` only). It is required when `AI` is in `allowed_caps` and refused without it;
`"stub"` is the offline test route. Under `--policy`, `--ai` and `--ai-stub` are refused by name.

`cli_allow` names the `ailang` operations `ailang_cli` may perform, in `process_allow` syntax
(`docs:search` admits `docs search` only). Absent means the default set — `check ai-check iface
fmt test docs:search examples builtins pkg-docs tree prompt agent-prompt devtools-prompt
policy-check axioms version` — and an empty list refuses everything. Each op has a typed schema in
`internal/policytool/cli_ops.go` (`check {path}`, `iface {module}`, `docs_search {query}`,
`test {path?, package?}`, …); a flag the schema does not admit, a path outside the sandbox (or a
symlink), or a subcommand with no schema is refused even when listed. `run`, `exec`, `repl`,
`replay`, `watch`, `select-best` are never reachable: execution only goes through `ailang_run`'s gate.

Rules the gate enforces on the file itself: every cap must be a real effect; `FS` needs an
`fs_sandbox`; `Net`/`Stream` need a `net_allow`; `Process` needs a `process_allow` (trusted_host);
`AI` needs an `ai_provider` (trusted_host); `timeout_ms` must be positive; byte caps non-negative;
a budget for an unadmitted effect is a contradiction. Keep the lists short: they are the boundary.

## What the runtime enforces beneath the gate

- **Filesystem**: with `fs_sandbox` set, every FS operation (and `std/zip`, `std/gzip`, `std/tar`)
  goes through one `os.Root` handle — `..`, absolute paths outside, symlinks whose target leaves
  the root (relative or absolute) and a link swapped between check and use all fail inside the
  syscall. Relative symlinks that stay inside keep working; absolute symlinks are refused even when
  they point back inside. Every read and write is capped by `max_fs_transfer_bytes`.
- **Network**: one destination authorizer runs on **every** hop and connection — HTTP, SSE,
  NDJSON and WebSocket — so a redirect to a host outside `net_allow` is refused before any dial,
  the resolved address is validated and pinned, and `Authorization`/`Cookie`/`Proxy-Authorization`
  are stripped across origins. Loopback, private, link-local and metadata addresses are refused in
  restricted mode with no override.
- **Authority**: the policy is decoded once into an immutable value and its digest is banked; the
  worker freezes source reads, so the module graph the gate typechecked is the one that runs
  (`module_graph.digest` on the admission line); the decision travels on a control pipe, never
  parsed from program stdout; the entrypoint is the policy's.
- **Limits**: `timeout_ms` bounds the whole invocation from before source loading (the worker's
  process group is killed; a `trusted_host` `Process` child dies with it); combined stdout+stderr
  stops at `max_output_bytes`; `[budgets]` is an aggregate ceiling independent of source `@limit`s.

Residual trust assumptions the runtime does **not** close are listed in
[LIMITATIONS](../../LIMITATIONS.md#execution-policy-residuals).

## Delivering it

- **Coordinator agents**: `tool_policy: ailang_only` and `policy_path: <file on the coordinator
  host>` on the agent in the registry; `ailang coordinator agents <id>` shows both declared and
  effective. Locally the path is forwarded as `AILANG_AGENT_POLICY`; for Cloud Run Jobs the
  coordinator reads the file and forwards its **content** as `AILANG_AGENT_POLICY_TOML`, which
  `execute-job` materialises read-only under `~/.ailang/agent-policy/` (outside the workspace).
  A missing `policy_path` file, or a policy the lane cannot run (`trusted_host`, an unadmitted
  effect), fails the dispatch loudly rather than sending a refusing agent.
- **A local pi session (developing a package on your own disk)**: nothing attaches a policy for
  you, so every lane tool refuses with `AILANG_AGENT_POLICY is unset` — that is the gate working,
  not a missing feature. Attach one explicitly, and keep the binary current:

  ```bash
  make quick-install                                   # the lane extension shells out to `ailang` on PATH
  cat > /tmp/dev-policy.toml <<'EOF'
  allowed_caps  = ["IO", "FS"]
  fs_sandbox    = "/path/to/your/checkout"             # must NOT contain /tmp/dev-policy.toml (D4)
  entry         = "main"
  EOF
  AILANG_AGENT_POLICY=/tmp/dev-policy.toml pi $(ailang pi tool-profile ailang_only)
  ```

  `messages`, `publish`, `install` stay refused on the lane by design: an executor never touches
  the plane or the registry itself.
- **Resident instances**: `resident-instance.sh create|update … --policy-file agent-policy.toml`.
  The file travels as `AILANG_AGENT_POLICY_TOML` and `boot.sh` materialises it **read-only
  outside the sandbox** (`~/.resident/policy/`).
- **Evals**: `ailang eval-suite --agent --tool-policy ailang_only --policy-file … --models …`.
- **By hand**: `echo '{"op":"summary"}' | ailang policy-tool --policy agent-policy.toml` shows
  exactly what the lane's tools will and will not do under a policy.

## Reading it back

Every agent-mode row banks `tool_policy` (the effective list, or `<cli default>`) and
`policy_digest` (sha256 of the policy bytes). The admission line on a run's stderr carries
`security_mode`, `caps`, `entry`, `timeout_ms`, `budgets`, `limits` and `module_graph`
(`digest`, `files`, `bytes`). **Absent means unmeasured** — rows before 2026-09-16 have neither.
A denied program banks as `error_category: policy_violation`; `decision.missing_from_policy`
names the effect. A supervisor limit is exit 3 with
`policy-result: {"version":1,"stage","reason":"timeout"|"output_limit",…}`.

## What the model is told

`ailang-exec.ts` injects a lane section — no shell, the eight tools, the policy's mode and allowed
effects, the sandbox root, the Net hosts, the `ailang_cli` ops, the timeout, and "narrow the
program on a denial" — as **both** a system-prompt section and a conversation message. The text
is built from the Go summary (`policy-tool` op `summary`), never from parsing the TOML.

## What it does not do

The confined-execution claim covers the mediated effects and the lane's tools on Linux and macOS.
It does not make an operator-approved `trusted_host` integration safe, does not bound CPU or
memory (host isolation is the container's job), and does not sever hard links, device nodes or
bind mounts the launcher seeds into the sandbox. Windows and `GOOS=js` refuse restricted mode.
