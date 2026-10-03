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
#                                         # host:PORT scopes an entry to one port: "127.0.0.1:7655"
# cli_allow      = ["iface", "docs:search"] # ailang_cli ops; absent = the read-only default set
timeout_ms    = 5000                      # the WHOLE invocation, enforced by the supervisor (positive, ≤ 24h)
# max_source_bytes / max_module_graph_bytes / max_output_bytes / max_fs_transfer_bytes
#                                          # restricted defaults: 1 MiB / 16 MiB / 8 MiB / 8 MiB
# fs_deny_write = [".github/**", ".pi/**", "Makefile", "*.yml"]
#                                          # read-only INSIDE the sandbox: the artifact's own supply chain
# ai_provider   = "gemini-3-5-flash-lite"  # restricted mode admits AI with a pinned provider AND [budgets] AI
entry         = "main"

# [budgets]                               # operator ceilings for the whole run: 0 = zero operations,
# FS = 100                                # absent = unlimited; a source @limit can only tighten
# AI = 50                                 # REQUIRED when AI is admitted in restricted mode
```

**`security_mode`.** `restricted` (the default, and the only mode the `ailang_only` lane accepts)
admits only effects with a confined adapter — `IO`, `FS`, `Net`, `Clock`, `Rand`, `Stream`,
`Process` **only for `process_allow` entries with a confined schema** (`git:status`, `git:diff`,
`git:log` — see below), `AI` **only with a pinned `ai_provider` and an explicit `[budgets] AI`
ceiling**, and `Declassify` (a compile-time information-flow gate with no host reach) — and refuses
every other effect with a named migration. It also refuses a configured
HTTP proxy (`E_NET_PROXY_REFUSED`: the destination address cannot be pinned behind one) and has
no private/metadata grant; loopback is admitted only as a **port-qualified literal** in `net_allow`
(see *Port-scoped entries* below). `trusted_host` keeps operator-approved host integrations —
`Process` with `process_allow`, `AI` with `ai_provider`, `Env`, `Secret` — with conspicuous
provenance (`security_mode` is banked on the admission line) and **no confinement claim**: the
worker gets the operator's full environment, current proxy semantics, and a bare loopback entry in
`net_allow` (`127.0.0.1`, `localhost`, `::1`) is honoured as an explicit grant for **every**
loopback port; a port-qualified one opens only that port. An `ailang_only` agent with a `trusted_host` policy
is refused at dispatch (`executor.CheckLanePolicy`) — the lane's "no shell" claim cannot sit on a
host-integration grant; such an agent declares `tool_policy: full` instead.

**Port-scoped entries (`host:PORT`, #1558).** A `net_allow` entry is `host` (any port) or
`host:PORT` (that port only, compared with the port the request dials — the explicit one, else 443
for `https`/`wss` and 80 for `http`/`ws`). IPv6 literals take brackets with a port (`[::1]:7655`);
wildcards combine with a port (`*.svc.example:9000`). A port-qualified **loopback** entry is itself
the loopback grant for that host and port — the way to let a confined program reach one local mock
without opening the host's other local services:

```toml
allowed_caps   = ["IO", "Net"]
net_allow      = ["127.0.0.1:7655"]   # the mock's port, and no other loopback port
net_allow_http = true
```

The grant is checked at every round trip and at the pinned dial (the dialer connects only to the
authorized port), for Net and for Stream (SSE, NDJSON, WebSocket) alike. It is never extended to a
**redirect hop** — a redirect to any loopback port, the granted one included, is refused — nor
inside a `Net[scope=public]` frame. Names match exactly: `127.0.0.1:7655` does not admit
`localhost:7655`. Restricted mode refuses, at policy load, a bare loopback entry (it would open every
port), a loopback **name** (`localhost:7655` — list the literal), and private, link-local (the
metadata server included), unspecified and multicast literals; every mode refuses a malformed entry
(`127.0.0.1:0`, `http://host`, an unbracketed IPv6 with a port). `--net-allow-localhost` stays
refused under `--policy`.

**Confined git (restricted mode).** `process_allow = ["git:status", "git:diff", "git:log"]`
keeps working under restricted mode, and it is now a boundary rather than a prefix match. A
subcommand prefix is not enough for git: the repo's own config can name commands
(`core.fsmonitor`, `diff.external`, `diff.<driver>.textconv`, `core.pager`, `core.hooksPath`) and
read-only subcommands carry flags that reach outside the clone (`--no-index`, `--output`, `-c`,
`--git-dir`, `-C`, `--exec-path`). So under restricted mode `exec("git", …)` runs through a
confined adapter (`internal/effects/process_confined.go`): the argv is **built** from a
per-subcommand schema (only the admitted flags; revs must look like revs; pathspecs stay inside the
clone), the invocation is hardened (git by absolute path, cwd = sandbox root, the caller's `GIT_*`
stripped, global/system config disabled, `-c core.fsmonitor=false -c
core.hooksPath=/dev/null/ailang-hooks-disabled -c core.pager=cat -c diff.external= -c
safe.bareRepository=explicit`, `--no-ext-diff --no-textconv` on diff/log, `--no-optional-locks`),
and **`.git/` is read-only** (case-folded: `.GIT/` too, for case-insensitive filesystems) to the
program's FS effect and to the lane's tools — the repo config is the launcher's clone config and
nothing else. Repository discovery is **bounded at the sandbox**: `GIT_CEILING_DIRECTORIES` is
the sandbox's parent, so a `.git` at the sandbox root (the clone-root case) is found but git never
walks up into an enclosing repository — a sandbox that is a repo subdirectory gets `not a git
repository`, and `git diff` there is refused outright (outside a repository it would compare
arbitrary paths). Restricted `Process` therefore requires `FS` with `fs_sandbox` (the clone root);
a policy without one is refused at resolution. Confined mode is `exec`-only
(`spawnProcess`/`asyncExecProcess` are refused). Any other `process_allow` entry (`git:push`,
`git:*`, `sh`, `gh:…`) is refused at startup by name.

**AI in restricted mode.** The AI effect's destination is the operator's (`ai_provider` pins the
registry entry; `--ai`/`--routing-*` are refused) and the program cannot read the credential (no
`Env`); what a program *can* do is spend. So restricted mode admits `AI` when `ai_provider` is
set **and** `[budgets] AI = N` states the ceiling (`AI = 0` permits none). The restricted worker
receives only the pinned provider's credential variables (`GOOGLE_API_KEY`/`GEMINI_API_KEY`/ADC
for Google, `OPENROUTER_API_KEY` for OpenRouter, …) — never another provider's key; on Cloud Run
the Google provider needs none (ADC). Residual, documented: the AI client is not the Net
authorizer (the host is not program-controlled).

**`fs_deny_write`.** Paths inside the sandbox the program and the lane's tools may read but not
write — the artifact's own supply chain, which would otherwise run with CI's or the next
session's authority once committed: `".github/**"`, `".pi/**"`, `"Makefile"`, `"*.yml"`. A
pattern is a glob for one path (matched against the whole relative path and its base name) or
`<dir>/**` for a subtree. `.git/**` is always implied in restricted mode. Refusals are
`E_FS_PROTECTED` from every mutating FS op and a named refusal from `ailang_write`/`ailang_edit`.

**The program file must be inside `fs_sandbox`.** `ailang run --policy` refuses an entry file
outside the sandbox (the module root the imports resolve from would otherwise be anywhere on the
host); `ailang_run` says so before shelling out.

**Migration (2026-09-21).** A policy that admits `Env`, `Secret`, `Process` beyond the three
confined git entries, or `AI` without a `[budgets] AI` ceiling, with no `security_mode`, now
fails at startup with a message naming the entry and the `trusted_host` migration. Either fix the
grant or set `security_mode = "trusted_host"` **and** `tool_policy: full` (the lane does not
accept `trusted_host`). Of the deployed lane policies in the multivac config repo,
`pkg-ailang-only.toml` and `ailang-only-executor.toml` need **no change**; `daneel-executor.toml`
needs `[budgets] AI = <n>` added.

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
`fs_sandbox`; `Net`/`Stream` need a `net_allow` whose entries are well-formed `host`/`host:PORT`; `Process` needs a `process_allow` (trusted_host);
`AI` needs an `ai_provider` (trusted_host); `timeout_ms` must be positive; byte caps non-negative;
a budget for an unadmitted effect is a contradiction. Keep the lists short: they are the boundary.

## What the runtime enforces beneath the gate

- **Filesystem**: with `fs_sandbox` set, every FS operation (and `std/zip`, `std/gzip`, `std/tar`)
  goes through one `os.Root` handle — `..`, absolute paths outside, symlinks whose target leaves
  the root (relative or absolute) and a link swapped between check and use all fail inside the
  syscall. Relative symlinks that stay inside keep working; absolute symlinks are refused even when
  they point back inside. Every read and write is capped by `max_fs_transfer_bytes`. In restricted
  mode `.git/` is read-only.
- **Process**: restricted mode runs only confined read-only git (above); `trusted_host` keeps the
  prefix-matched allowlist with a cwd and no further claim.
- **Network**: one destination authorizer runs on **every** hop and connection — HTTP, SSE,
  NDJSON and WebSocket — so a redirect to a host outside `net_allow` is refused before any dial,
  the resolved address is validated and pinned, and `Authorization`/`Cookie`/`Proxy-Authorization`
  are stripped across origins. Private, link-local and metadata addresses are refused in
  restricted mode with no override; loopback is reachable only on a port named by a port-qualified
  `net_allow` entry, and never through a redirect.
- **Authority**: the policy is decoded once into an immutable value and its digest is banked; the
  worker freezes source reads, so the module graph the gate typechecked is the one that runs
  (`module_graph.digest` on the admission line); the decision travels on a control pipe, never
  parsed from program stdout; the entrypoint is the policy's.
- **Limits**: `timeout_ms` bounds the whole invocation from before source loading (the worker's
  process group is killed; a `trusted_host` `Process` child dies with it); combined stdout+stderr
  stops at `max_output_bytes`; `[budgets]` is an aggregate ceiling independent of source `@limit`s.

Residual trust assumptions the runtime does **not** close are listed in
[LIMITATIONS](https://github.com/sunholo-data/ailang/blob/main/docs/LIMITATIONS.md#execution-policy-residuals).

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

## The agent child's environment

Every executor (claude, codex, pi, opencode, motoko) launches its CLI with an environment built by
one function, `executor.BuildEnvironment` (M-EXECUTOR-ENV-HARDENING), and that environment is
**default-deny**, on every lane — not only `ailang_only`:

| Layer (later wins) | What it holds |
|---|---|
| inherited | host variables on the allowlist (PATH, HOME, locale, toolchain dirs, `AILANG_*`/`OTEL_*`/`CLAUDE_*`/`PI_*`/`MOTOKO_*` configuration, …) — **minus every credential-shaped name** (`*_KEY`, `*_TOKEN`, `*SECRET*`, `SSH_AUTH_SOCK`, `AWS_*`, …) unless granted |
| harness-injected | rig lease, stdlib pin, `PWD`, trace context, correlation IDs, `AILANG_AGENT_POLICY` (from the task's policy path), OTEL wiring, GCP project/location, the per-task git credential config |
| `Task.ExtraEnv` | e.g. a benchmark's `agent_env` — validated: loader, shell start-up, proxy, `GIT_*`, `OTEL_*`, `AILANG_AGENT_POLICY*` and harness-owned names are refused by name |
| executor-required | the executor's own variables (motoko's `MODEL`, `MOTOKO_CONFIG`, …) |

**Grants.** Each executor gets its inference credential and nothing else: claude none (OAuth
from its credentials file; `ANTHROPIC_API_KEY` only under `AILANG_AUTH_MODE=apikey`); codex
`OPENAI_API_KEY`/`CODEX_API_KEY`; pi and opencode the key of the model's `provider/` prefix;
motoko `OPENROUTER_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`. A task with a program policy
also grants the credentials that policy's worker may select (its pinned `ai_provider`, std/web's
key when `net_allow` admits its host) — the worker takes its keys from the child's env, so a key
the child does not hold cannot reach it. `GITHUB_TOKEN` and `AILANG_REGISTRY_API_KEY` are granted
to no executor. A `trusted_host` program launched by an agent therefore also sees only these.

**Git.** In a cloud job the parent clones, pushes and opens the PR. Its global credential helper
reads `$GITHUB_TOKEN` at call time and answers nothing in the child, whose env has no token. With
`AILANG_CHILD_GIT_CREDENTIALS=repo` (default) the child gets a 0600 credential file outside the
workspace that git consults only for the task's own repository; `=none` gives it nothing.

**Operator lever.** `AILANG_EXECUTOR_ENV_INHERIT=NAME1,NAME2` (host env, never a task) forwards
extra names, credentials included. Every build prints the credential names it withheld
(`executor-env: withheld from the pi child: GITHUB_TOKEN, …`) — names, never values.

**In the data.** `Result.EnvNamesDigest` (eval rows: `env_names_digest`) is the sha256 of the
child env's sorted names. It changes when a lane starts inheriting a new name.

**Limit.** The child runs as the same UID as the parent, so this is defence in depth: secrets
leave `printenv`, tool and MCP subprocess inheritance and `~/.gitconfig`, but a same-user process
can still read the parent's `/proc/<pid>/environ` or a file the parent can read. The boundary for
that is a UID split (audit H-6) and the egress lock.
