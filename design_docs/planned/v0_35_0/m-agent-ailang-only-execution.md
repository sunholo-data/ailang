# M-AGENT-AILANG-ONLY-EXECUTION — AILANG as the agent's only route to acting, fleet-wide

**Status**: Planned
**Target**: v0.39.0 — filed beside its parent in `v0_35_0/`; both slipped past v0.38.9 and retarget together
**Priority**: P1 — the claim "the agent can only act through AILANG" is made in three places today and is true in none of them
**Estimated**: ~4 days (1 day gate, 1.5 days pi tool + policy plumbing, 1 day fleet + resident cutover, 0.5 day eval lane + bookkeeping)
**Dependencies**: **[M-PI-HARNESS-UPGRADE](m-pi-harness-upgrade.md)** — one pinned pi on both planes, with the extension-execution probe passing *inside a container*. The pi tool this doc ships is an extension; shipping it across two pi majors means testing it twice and trusting it once. Parent story: [M-RESIDENT-AGENT-INSTANCES](../m-resident-agent-instances.md) D6/D8. Reuses the [M-AGENT-SAFE-RUNNER](../v1_1_0/m-agent-safe-runner.md) M1 spike (`ailang policy-check --policy`).

**Created**: 2026-09-16 · **Author**: attended session with Mark

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Admission is a pure function of (program source, policy file). The same submission is admitted or refused identically on the rig, in a Job and in a resident. Today it depends on which of three ad-hoc gates the deployment happens to have. |
| A2: Replayability | +1 | Every banked agent-mode row records the *effective* tool policy and the policy file's digest, so a run can be re-executed under the authority it actually had — not the authority the image default implied. |
| A3: Effect Legibility | +1 | The whole design rests on effect rows being legible: a program is admitted because its declared row is a subset of the policy's `allowed_caps`. A lying signature is rejected by the typechecker before the gate runs (M-AGENT-SAFE-RUNNER spike, demonstrated). |
| A4: Explicit Authority | +1 | Removes ambient authority. `bash` is off in the `ailang_only` profile; what remains is a typed tool whose authority is the policy file. The policy is deployment config and default-deny when absent — the stance `resident-run` already takes. |
| A5: Bounded Verification | +1 | `policy-check` is a typecheck plus a row-subset check: decidable, local, terminating. |
| A6: Safe Concurrency | 0 | No concurrency change. The resident's `RESIDENT_MAX_CONCURRENT_RUNS` ceiling is untouched. |
| A7: Machines First | +1 | The tool returns structured JSON (`admitted`, `error_kind`, `effects_used`, stdout) instead of a shell transcript the model must parse. One fewer round of "the agent read its own bash output wrong". |
| A8: Minimal Syntax | 0 | No language change. One CLI flag (`ailang run --policy`), one registry field, one pi extension. |
| A9: Cost Visibility | +1 | Per-effect budgets in the policy surface in the result; the banked row says which profile a run cost was measured under. |
| A10: Composability | +1 | Retires a JS reimplementation (`resident-run`) in favour of the Go gate that already exists; the pi tool is one more `ailang pi install` asset beside eleven others. No new plane. |
| A11: Structured Failure | +1 | Refusals carry `error_kind` (`policy_violation`, `typecheck_failed`, `not_listed`, `budget_exhausted`), never a bare exit code; the executor maps them to a named `error_category` rather than `api_error`. |
| A12: System Boundary | +1 | The boundary between "agent reasoning" and "agent acting" becomes a single typed tool call rather than an argv string handed to a shell. |

**Net Score: +10** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): admission reads no clock, no ambient env beyond the named policy path
- [x] A3 (Effects): introduces no effects; makes existing ones load-bearing
- [x] A4 (Authority): tightens; never widens. `full` profile is explicit opt-in, never the silent default in a container
- [x] A7 (Machines First): JSON result; human text is a render of it

---

## Problem Statement

**Current State (all measured 2026-09-16 — see Verification Log):**

"AILANG is the agent's only route to acting" is stated as a design property in three places. Each has a different mechanism, and none holds:

1. **The resident** ([M-RESIDENT D6/D8](../m-resident-agent-instances.md)) ships `resident-run`, a default-deny program allowlist that runs `ailang run --caps <declared>`. It is copied to `/usr/local/bin` and **registered as no pi tool** — the only way the agent can invoke it is through pi's `bash` tool, and `bash` is exactly the thing that bypasses it (V1). The three live instances in `ailang-multivac-dev` set **neither `RESIDENT_TOOLS` nor `PROGRAM_ALLOWLIST_FILE`** (V2), so on every deployed resident today `bash` is live and `resident-run` refuses everything. The design doc's own D8 admits the allowlist is "a convenience and not a containment boundary".

2. **The fleet containers** (`agent-pi`, `agent-eval`, and the Jobs the coordinator dispatches to them) run pi with its defaults — `read, bash, edit, write`. `PiExecutor` passes `--tools` only when a caller sets `Task.AllowedTools`, and **no caller does for pi** (V3) except one: the coordinator restricts `question`-kind tasks to `Read, Grep, Glob, WebFetch, WebSearch` — Claude-cased names pi does not have. pi **silently ignores unknown tool names** (V4), so a question routed to a pi agent runs with **no tools and no error**. The agent registry has **no tool-policy field at all** (V5): what an agent may do is an image default nobody wrote down.

3. **The rig's mission executor** has real containment — `tools/pi-extensions/sandbox` (Seatbelt around `bash`) and `worktree-fence.ts` (path allowlist on `write`/`edit`) (V11) — but it *contains* `bash`; it does not replace it. The agent still acts by emitting shell.

Meanwhile the Go admission gate this story needs **already exists and is used by nobody**: `ailang policy-check --policy P file.ail` landed as the M-AGENT-SAFE-RUNNER M1 spike ([internal/policy/](../../../internal/policy/)) with a typed TOML policy (`allowed_caps`, `fs_sandbox`, `net_allow`, `budgets`, `max_source_bytes`) and enum `error_kind`s (V7). `resident-run` is a **second implementation of the same concept in JavaScript**, and it has already drifted: its `KNOWN_CAPS` lists 10 capabilities; `ailang run -caps` accepts 15 (V8). That is the seam CLAUDE.md warns about — one concept, two implementations, the bug living between them.

**Impact:**

- The customer-facing deployment of the resident (Aitana estate, D1b) is being offered a containment property that the image cannot deliver. The resident doc's D11 already says the tool "must not be offered" for confidential material until isolation holds; this is the other half of that sentence.
- Any benchmark that claims to measure "AILANG-only" agents would today measure bash-plus-AILANG agents.
- Question-kind tasks on the pi lane run toolless — a silent fallback of exactly the class CLAUDE.md §2 forbids.

**What is NOT the problem:** AILANG's own controls — effect rows, `--caps`, `AILANG_FS_SANDBOX`, budgets — work (V9). The M-AGENT-SAFE-RUNNER doc names the gap precisely: they protect the runner *given the operator chose the flags*, and here the agent is the operator. This doc moves the choice out of the agent's hands.

---

## Goals

**Primary Goal:** One admission gate (`ailang`, Go), reached through one pi tool, selected by one registry field — so that an agent whose profile is `ailang_only` can read and write files, and can execute programs *only* by submitting AILANG source to a policy it cannot edit.

**Success Metrics:**

1. In a container built from `docker/Dockerfile.agent-pi` with `tool_policy: ailang_only`, a directive that asks the agent to run `echo` produces **no `bash` tool call** in the NDJSON (measured by the executor's tool-call list, not by prose), and a directive that asks it to run a `.ail` program produces an `ailang_run` call whose result carries `admitted: true`.
2. A program declaring an effect outside `allowed_caps` is refused with `error_kind: policy_violation` and banked under a **named** `error_category` — never `api_error`.
3. `resident-run` is deleted; the resident image passes `test-image.sh` §4 against `ailang run --policy` instead, and `RESIDENT_TOOLS`'s default is the `ailang_only` profile, not `read,edit,write,bash`.
4. Every agent-mode banked row carries `tool_policy` and `policy_digest`; a row without them is distinguishable from a row that ran `full`.
5. The coordinator's question-kind restriction produces the pi-cased list on the pi lane and a **loud error** for a name pi does not know — the silent-ignore in V4 is closed at our boundary.
6. `ailang coordinator agents <id>` shows `tool_policy` as declared vs effective, and an agent with none declared shows the default it inherited.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1** — The gate is the `ailang` binary (`ailang run --policy P prog.ail`, admission via the existing `internal/policy`), and `resident-run` is retired | Two implementations of admission drift (V8). One gate means the resident, the Job, the rig and Aitana's estate refuse the same program for the same reason | human | design | high |
| **D2** — The agent reaches the gate through a **pi tool** (`ailang_run`, `ailang_check`) shipped in `cmd/ailang/pi_assets/` via `ailang pi install`, not through `bash` | Without a tool, "no bash" means "no execution". With one, `bash` becomes removable. It is also the A7 win: JSON in, JSON out | human | design | high |
| **D3** — Tool policy is a **registry field** (`tool_policy: full \| ailang_only \| [explicit list]`) plumbed to `Task.AllowedTools` → `--no-builtin-tools --tools …`; the *image* carries no policy | Authority must be declared where the operator can read it (`ailang coordinator agents <id>`), not in a Dockerfile default. Also the only way Aitana's deployment and ours can differ without forking the image (D1b) | human | design | high |
| **D4** — Policy files are **deployment config**, default-deny when absent; the agent's own tools cannot write the policy path | The `resident-run` stance, kept. An agent that can edit its policy has no policy. The `fs_sandbox` in the policy must exclude the policy file's directory, and the pi tool refuses to start if it does not | human | design | high |
| **D5** — The `ailang_only` profile is a **new eval lane**, banked with `tool_policy`, never a silent change to existing lanes | Same reasoning as M-PI-HARNESS-UPGRADE D2: a tool change is a harness boundary. Comparing AILANG-only rows to bash-era rows without the field is the ollama-boundary mistake again | human | design | high |
| **D6** — Fleet default stays `full`; `ailang_only` is opt-in per agent — **except** the resident, whose default flips | The resident's whole claim (D6 of its doc) is this property; the coordinator's code agents need `bash` for `go test` today. Flipping the fleet default would break every pipeline agent at once | human | design | med |
| **D7** — Tool-name case is normalised **at our boundary**: `AllowedTools` is canonical (Claude-cased) in `executor.Task`, and each executor maps to its CLI's vocabulary, erroring on a name it cannot map | Closes V4 for every future caller, not just the question-kind one. The alternative — lower-casing at the one call site — is the patch-per-case anti-pattern | agent | compile | low |

**D3's rule (so the agent has a bright line):** `ailang_only` expands to exactly `--no-builtin-tools --tools read,edit,write,ailang_check,ailang_run`. `read`/`edit`/`write` stay because the story is "write `.txt`/`.md`/`.ail` as normal, execute only through AILANG" — file writes are bounded by `AILANG_FS_SANDBOX` on the AILANG side and by the container/`worktree-fence` on the pi side. No `grep`/`find`/`ls` — pi 0.85 has none of those as builtins; if the model needs them it writes a `.ail` that declares `FS`.

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **D1** — `ailang run --policy` is the one gate; `resident-run` is deleted, not kept "for compatibility"
- [ ] **D2** — the pi tool ships in the embedded `pi_assets` suite (so `ailang pi install` in every Dockerfile delivers it), not as a resident-only file
- [ ] **D3** — `tool_policy` is a registry field on `AgentConfig`, with `ailang coordinator agents <id>` showing declared vs effective
- [ ] **D4** — policy files are mounted/injected config, default-deny, and the pi tool refuses a policy whose `fs_sandbox` contains the policy file
- [ ] **D5** — `ailang_only` evals bank under a new lane field, and the first such run is preceded by a boundary note in the charter and runbook
- [ ] **D6** — fleet default `full`; resident default `ailang_only`
- [ ] **Parent freeze** — M-PI-HARNESS-UPGRADE D1 (cut both planes over) is ratified. **Mark, 2026-09-16: "update the fleet to use the newest pi version"** — which is the substance of D1, but the parent pins **0.84.4** and the rig has since moved to **0.85.1** (V10). Decide the pin there before this sprint starts; this doc's extension is tested against whichever version the parent lands.

---

## Solution Design

### Overview

Four layers, each replacing an ad-hoc thing that exists today with the one thing that should:

| Layer | Today | After |
|---|---|---|
| Admission | `resident-run` (JS, 10 caps) **and** `internal/policy` (Go, unused) | `ailang run --policy` (Go) |
| Agent → gate | `bash` → `resident-run` / `bash` → `ailang run` | pi tool `ailang_run` / `ailang_check` |
| Who decides | image default (`read,bash,edit,write`) | registry `tool_policy` → `Task.AllowedTools` |
| Evidence | nothing banked | `tool_policy`, `policy_digest` on every agent-mode row |

Milestones are ordered so the **instrument lands before the change it measures** (M1 banks the field before M4 flips anything), matching the parent doc's ordering rule.

### Architecture

```
registry (config.yaml)                  executor.Task                        pi CLI
  tool_policy: ailang_only  ──►  AllowedTools (canonical)  ──►  --no-builtin-tools --tools read,edit,write,ailang_check,ailang_run
                                        │
                                        ▼
                             ProviderData.tool_policy      ◄── banked, with policy_digest
                                        
pi extension  cmd/ailang/pi_assets/ailang-exec.ts
  registerTool("ailang_run",  {file, args?})  ──►  spawn: ailang run --policy $AILANG_AGENT_POLICY <file> --json
  registerTool("ailang_check",{file})          ──►  spawn: ailang check <file> --json
  (refuses to load if $AILANG_AGENT_POLICY unset or unreadable — default-deny, D4)

ailang run --policy P
  internal/policy.Check(source, P)  → admitted | {error_kind}
  then the existing run path with --caps = P.allowed_caps, AILANG_FS_SANDBOX = P.fs_sandbox, budgets = P.budgets
```

**Components:**

- **`ailang run --policy <toml>`** — new flag on the existing run path. Runs `internal/policy.Check` first; on admission, derives `--caps`, the FS sandbox and budgets **from the policy**, overriding any `--caps`/env the caller passed (the caller is the agent; its flags are not trusted). Emits the M-AGENT-SAFE-RUNNER `program-result` shape with `--json`. Verify (not assume) that the spike's `Check` handles transitive import effects — its own doc lists that as "still missing".
- **`ailang-exec.ts`** — a pi extension registering two tools. Each is a thin spawn of the `ailang` binary already in every image (`Dockerfile.agent-pi:32` runs `ailang pi install`, so the binary is there). Returns the JSON verbatim as the tool result. Registers **no** other tool and reads no config beyond `AILANG_AGENT_POLICY`.
- **`AgentConfig.ToolPolicy`** — `yaml:"tool_policy"`, values `full`, `ailang_only`, or a list. Resolved to `Task.AllowedTools` in `provider_executor.go`, next to where the question-kind list is set today. Defaults per D6.
- **Executor name mapping** (D7) — `PiExecutor` owns a `canonical → pi` table (`Read→read`, `Write→write`, `Edit→edit`, `Bash→bash`, `AilangRun→ailang_run`, …). An unmapped name is an error at `buildArgs`, named in the message. Claude and the other executors keep their tables; only pi's is new because only pi ignores unknowns.
- **Banked provenance** — `Result.ToolPolicy []string` and `Result.PolicyDigest string` → `RunMetrics` as `omitempty`; absent ⇒ unmeasured, in the same voice as `executor_version` from the parent doc.
- **Resident cutover** — `docker/resident/lib/pi.mjs` maps `RESIDENT_TOOLS=ailang_only` to the same flag expansion (share the string constant via the image's `ailang` binary: `ailang pi tool-profile ailang_only` prints it, so there is one source). `resident-run` and `allowlist.example.json` are deleted; `test-image.sh` §4 targets `ailang run --policy`.

### Implementation Plan

**M1: Bank the policy before changing it** (~3 hours)
1. `Result.ToolPolicy`, `Result.PolicyDigest`; `RunMetrics.tool_policy`, `policy_digest` (`omitempty`, absent ⇒ unmeasured).
2. `PiExecutor` stamps the *effective* list it passed (or `["<pi default>"]` when `AllowedTools == nil`) — so the pre-cutover rows say what they ran under too.
3. Test: a row with nil `AllowedTools` banks the sentinel, not `[]` and not absent.

**M2: One gate** (~1 day)
1. `ailang run --policy`: wire `internal/policy.Check` ahead of execution; derive caps/sandbox/budgets from the policy; `--json` emits the `program-result` shape. Refuse `--policy` combined with `--caps` (the agent must not be able to widen).
2. Close the spike's listed gap: transitive import-effect closure in `Check`, with a test where the entry row is clean and an imported module declares `Net`.
3. Verify the `error_kind` enum is exhaustive against `ailang run -caps`'s 15 capabilities — the V8 drift must not recur inside the Go side either; derive the list from one table.
4. Map `error_kind` → executor `error_category` (new named categories; none reuse `api_error`).

**M3: One tool** (~4 hours)
1. `cmd/ailang/pi_assets/ailang-exec.ts`: `ailang_run`, `ailang_check`; default-deny on a missing policy; refuses a policy whose `fs_sandbox` contains the policy path (D4).
2. Container probe, in the same shape as the parent's extension-execution probe: a real `tool_execution_start` with `toolName: ailang_run` inside `agent-pi`, not a `pi list`.
3. `ailang pi tool-profile <name>` prints the flag expansion, so `pi.go` and `pi.mjs` read one string.

**M4: One field** (~4 hours)
1. `AgentConfig.ToolPolicy` + defaults (D6); resolve in `provider_executor.go`; `ailang coordinator agents <id>` shows declared vs effective.
2. D7 name mapping in `PiExecutor.buildArgs`; convert the question-kind list to canonical names; test that an unmapped name errors.
3. Negative test: `tool_policy: ailang_only` + a directive "run `echo hi`" → zero `bash` tool calls in the NDJSON.

**M5: Resident cutover and the eval lane** (~5 hours)
1. Delete `resident-run`, `allowlist.example.json`; `pi.mjs` default `ailang_only`; `test-image.sh` §4 rewritten; `resident-instance.sh update` accepts `--policy-file` and mounts it read-only.
2. Redeploy `resident-pi-ailang` with a real policy; `resident-instance.sh verify` gains "a bash request is refused, an admitted `.ail` runs".
3. Eval: `--tool-policy ailang_only` on `eval-suite`/`eval-loop`, banked per M1; record the boundary in the charter and `docs/internal/harness-upgrade-runbook.md` **before** the first banked run (D5). Run a small comparator set both ways and report with `ailang eval-paired`.

### Files to Modify/Create

**New files:**
- `cmd/ailang/pi_assets/ailang-exec.ts` — the two tools, default-deny load (~120 lines)
- `internal/executor/pi/toolnames.go` — canonical→pi table + error on unmapped (~40 lines)
- `internal/executor/pi/testdata/v<pinned>/ailang_run.ndjson` — fixture with a real `ailang_run` tool call (~40 lines)
- `docs/docs/guides/agent-tool-policy.md` — profiles, policy file, how to read `tool_policy` on a banked row (~80 lines)

**Modified files:**
- `cmd/ailang/main_run.go` (where `-caps` is parsed) — `--policy`, refuse `--policy`+`--caps`, JSON result (~+90)
- `internal/policy/check.go` — transitive import effects; caps table shared with the runtime (~+60/−10)
- `internal/executor/executor.go` — `Result.ToolPolicy`, `Result.PolicyDigest` (~+10)
- `internal/executor/pi/pi.go` — profile expansion, name mapping, stamp effective policy (~+50/−8)
- `internal/eval_harness/metrics.go` — two `omitempty` fields (~+12)
- `internal/coordinator/agent_registry.go` — `ToolPolicy` field + defaults (~+25)
- `internal/coordinator/provider_executor.go` — resolve `tool_policy`; canonical question-kind list (~+20/−1)
- `cmd/ailang/coordinator_agents_list.go` — show `tool_policy` declared vs effective (~+15)
- `cmd/ailang/pi_setup.go` — `tool-profile` subcommand (~+20)
- `docker/resident/lib/pi.mjs` — profile via `ailang pi tool-profile`; default `ailang_only` (~+15/−10)
- `docker/resident/Dockerfile`, `docker/resident/test-image.sh` — drop `resident-run`; §4 targets the Go gate (~+20/−30)
- `ailang-multivac/scripts/resident-instance.sh` — `--policy-file` (~+25)
- `design_docs/planned/m-resident-agent-instances.md` — D6/D8 amended to point here (~+10/−4)

**Deleted files:**
- `docker/resident/resident-run`, `docker/resident/allowlist.example.json`

---

## Examples

### Example 1: what the agent sees under `ailang_only`

Directive: *"Count the lines in `notes.txt` and write the number to `count.txt`."*

```
tool_execution_start  toolName=read        {path: "notes.txt"}
tool_execution_start  toolName=write       {path: "count_lines.ail", content: "module ... export func main() -> () ! {FS, IO} = ..."}
tool_execution_start  toolName=ailang_run  {file: "count_lines.ail"}
tool_execution_end    toolName=ailang_run  {admitted: true, exit_code: 0, effects_used: ["FS","IO"], budget_report: {...}}
```

No `bash` in the stream. The agent wrote a program, declared `FS` and `IO`, and the policy (`allowed_caps = ["FS","IO"]`, `fs_sandbox = "/workspace"`) admitted it. Had it tried `writeFile("/etc/passwd", …)` the FS handler refuses at runtime; had it declared `Net` the gate refuses before running with `error_kind: policy_violation`.

### Example 2: the same directive today, on the resident

```
tool_execution_start  toolName=bash  {command: "wc -l notes.txt > count.txt"}
```

Correct output; the containment story is false. This is Success Metric 1's negative arm.

---

## Success Criteria

- [ ] SM1–SM6 above, each with the measurement named
- [ ] `resident-run` no longer exists in the tree; `grep -rn resident-run docker/` is empty
- [ ] A policy with `allowed_caps = []` refuses every program (default-deny proven, not assumed)
- [ ] `ailang run --policy P --caps Net f.ail` exits non-zero naming the conflict
- [ ] Question-kind task on the pi lane runs with `read` enabled (V4 closed) — asserted on the built args, not on prose
- [ ] Boundary note committed before the first `ailang_only` row is banked
- [ ] All tests passing; `make simplicity-audit` shows no new env-var route (the policy path is one env var, registered in `internal/config`)
- [ ] Documentation updated: guide, resident doc D6/D8, CHANGELOG

---

## Testing Strategy

**Unit tests:**
- `internal/policy`: transitive import closure; caps table parity with the runtime list (a test that fails if either side gains a capability the other lacks)
- `internal/executor/pi`: profile expansion string; name mapping incl. the error arm; effective-policy stamping incl. the nil sentinel
- `internal/coordinator`: `tool_policy` defaults per agent kind; question-kind resolves to canonical names

**Integration tests (container, Cloud Build `test-resident-pi` step + a new `test-agent-pi-tools` step):**
- Positive: `ailang_run` tool call observed in NDJSON with `admitted: true`
- Negative: `bash` requested under `ailang_only` → no `bash` tool call
- Negative: policy file inside `fs_sandbox` → extension refuses to load, run fails loudly
- These steps must **not** be `allowFailure: true` — the parent doc measured what that costs

**Manual / live:**
- `resident-instance.sh verify` extended (M5.2) against the redeployed `resident-pi-ailang`
- One comparator set through `eval-paired` across the `full`/`ailang_only` boundary

---

## Deferred Decisions

- Exact `error_category` names for the four `error_kind`s — **agent may choose**; must be new, must not be `api_error`
- Whether `ailang_check` is also policy-gated (it executes nothing) — **agent may choose**; default is ungated
- Whether the pi tool exposes `ailang iface` as a third tool — **agent may choose**; the resident's session-persistence work suggests yes, eval lanes suggest no
- Which comparator benchmarks form the `ailang_only` re-baseline set — **agent may choose**, stated in the sprint plan before it runs
- Whether opencode/codex get an equivalent profile in this sprint — **agent may choose**; the registry field lands for all, the mapping table only for pi is mandatory

## Non-Goals

- **Sandboxing `read`/`edit`/`write` further.** `worktree-fence.ts` and `AILANG_FS_SANDBOX` already bound them; this doc removes `bash`, it does not rebuild the fence.
- **Per-user isolation on a shared resident.** That is M-RESIDENT D11 and stays there.
- **A runner daemon / message-bus submission** (M-AGENT-SAFE-RUNNER M3). The gate is invoked in-process by the tool; the daemon is a later shape.
- **Flipping the fleet default.** `full` remains the default for code agents (D6).
- **Choosing the pi version.** That is the parent's D1; this doc inherits it.
- **Migrating the Claude/codex/gemini executors' tool vocabularies.** They do not silently ignore unknown names, so D7 is pi-only until measured otherwise.

## Timeline

| Day | Work |
|---|---|
| 1 | M1 + M2 (gate, transitive effects, caps parity, error mapping) |
| 2 | M3 (extension + container probe) · M4 start (registry field) |
| 3 | M4 finish (name mapping, negative test) · M5.1–5.2 (resident cutover, redeploy, verify) |
| 4 | M5.3 (eval lane, boundary note, comparator run) · docs · changelog |

Realistic: 4 days assumes the parent has landed and the container extension probe already passes. If it has not, add a day and expect to fight the same `EEXIST` the parent documents.

---

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Models under `ailang_only` fail more because they reach for `bash` reflexively | That is the measurement (D5). The prompt served to that lane names the tools it has; `ailang prompt` already exists for this |
| The spike's `policy.Check` has gaps beyond the listed one | M2.2/M2.3 are tests first; a gap found is a row in the Verification Log, not a silent pass |
| `--no-builtin-tools` semantics differ between 0.73.1 and the pinned version | Dependency on the parent removes the two-version window; V6 shows the fix dated 0.70.0, and the probe runs in the container regardless |
| The policy path env var becomes a new `os.Getenv` route | Registered in `internal/config` (forbidigo enforces); `make simplicity-audit` is a success criterion |
| Aitana's estate expects `resident-run` | Their doc consumes A2A, not the binary (M-RESIDENT D2); confirm with a grep of their repo before deleting — listed as V13, PENDING |

---

## Verification Log

| # | Claim | Method | Result |
|---|---|---|---|
| V1 | `resident-run` is reachable only through `bash` | `grep -rln registerTool docker/resident/` → none; `Dockerfile:59` copies it to `/usr/local/bin`; `pi.mjs` passes `--tools` from `RESIDENT_TOOLS` only | **Confirmed** — no tool registers it |
| V2 | Live residents set neither `RESIDENT_TOOLS` nor `PROGRAM_ALLOWLIST_FILE` | `gcloud beta run instances describe resident-pi-ailang --project ailang-multivac-dev --region europe-west4` env list, 2026-09-16 | **Confirmed** — 12 env vars, neither present; same for the two per-user instances |
| V3 | No caller sets `AllowedTools` for pi tasks | `grep -rn AllowedTools internal/ cmd/ --include=*.go` (non-test) → `executor.go:61` (field), `provider_executor.go:137` (question-kind), `agent_runner.go:67` (Claude eval default) | **Confirmed** — `pi.go:614` `nil ⇒ pi defaults` is the path every pi run takes |
| V4 | pi silently ignores unknown tool names in `--tools` | `agent-session.js:655` (0.85.1): *"Only tools in the registry can be enabled. Unknown tool names are ignored."*; `:739` filters by `_toolRegistry.has(name)` | **Confirmed** — `--tools Read,Grep,…` on pi ⇒ zero tools, exit 0 |
| V5 | `AgentConfig` has no tool-policy field | read `internal/coordinator/agent_registry.go` field list (lines 20–190) | **Confirmed** — negative existence |
| V6 | `--no-builtin-tools` keeps extension tools; `--tools` covers extension tools | `pi --help` 0.85.1; CHANGELOG entry under `[0.70.0]` (#3592) | **Confirmed on 0.85.1; on 0.73.1 by changelog only — PENDING container check** |
| V7 | A Go admission gate exists and no policy-gated *run* exists | `ailang policy-check --help` → `-policy`; `ls internal/policy/` → `check.go policy.go`; `grep -n policy cmd/ailang/exec.go` → only routing policy | **Confirmed** both halves — negative existence for the run path |
| V8 | `resident-run`'s caps list has drifted from the runtime's | `KNOWN_CAPS` = 10 (`resident-run:27`); `ailang run --help -caps` = 15 (`SharedMem, SharedIndex, Trace, DOM, Msg, Cog` missing) | **Confirmed** |
| V9 | `AILANG_FS_SANDBOX` is a registered, live control | `internal/config/compiler.go:19`; `ailang sandbox-check` exists | **Confirmed** |
| V10 | pi versions and the rig | `npm view … time`: 0.84.4 (08-28), 0.85.0 (09-04), 0.85.1 (09-05); rig `pi --version` = 0.85.1 | **Confirmed** — parent pins 0.84.4, rig is past it |
| V11 | Existing rig containment contains `bash` rather than removing it | `tools/pi-extensions/README.md` table: `bash` → sandbox (Seatbelt); `write`/`edit` → `worktree-fence.ts` | **Confirmed** |
| V12 | The eval pi lane runs with pi defaults | V3 + `agent_runner.go:67` is the Claude runner's list, not passed to `executor.Task` for pi | **Confirmed** |
| V13 | Aitana's estate does not invoke `resident-run` by name | grep of the aitana/platform repo | **PENDING** — do before M5.1 |
| V14 | `policy.Check` lacks transitive import-effect closure | `grep -in "import\|transitive\|closure" internal/policy/check.go` → only the Go `import (` block; no module-import walk | **Confirmed** — negative existence; M2.2 is real work |

---

## Quorum

**Triggers fired: 3 of 4** — (1) design-freeze items D1–D6; (2) overrides shared machinery — `Task.AllowedTools` semantics change for every executor (D7) and `resident-run` is deleted; (3) banked-data schema (`tool_policy`, `policy_digest`) and a new eval lane. Trigger 4 (external systems) is inherited from the parent (pi's flag semantics), already reviewed there.

**Not yet run.** Before round 0: close V13 (one grep of the aitana repo), and settle the parent's version pin so V6 is a container measurement rather than a changelog citation. Reviewers as the parent used: `gpt5-6-sol`, `gemini-3-1-pro`, `oc-glm-5-2`.

---

## Related Documents

- [M-PI-HARNESS-UPGRADE](m-pi-harness-upgrade.md) — **parent / hard dependency**: one pinned pi, `executor_version` banked, extension probe in a container. Its D7 (no workspace trust) is what makes shipping this tool in the image-owned global dir safe.
- [M-RESIDENT-AGENT-INSTANCES](../m-resident-agent-instances.md) — D6 (the allowlist), D8 (bash is outside it), D11 (isolation) — this doc is the follow-up D8 asks for
- [M-AGENT-SAFE-RUNNER](../v1_1_0/m-agent-safe-runner.md) — the Go gate reused here (M1 spike landed); its M3 runner daemon is a later shape of the same boundary. **Distinct**: that doc builds the gate; this one makes an agent unable to go around it
- [M-ANTHROPIC-SANDBOX](../v0_29_0/m-anthropic-sandbox.md) — a different executor, not a tool policy; noted for the neural-search hit, no overlap
- `tools/pi-extensions/README.md` — the rig's existing two-layer containment that this composes with rather than replaces
- `docs/internal/harness-upgrade-runbook.md` — where the `ailang_only` boundary note goes

## References

- pi 0.85.1 `--help`: `--no-builtin-tools`, `--tools` (built-in, extension, custom), `--exclude-tools`
- pi CHANGELOG `[0.70.0]` #3592 — `--no-builtin-tools` keeps extension tools
- `docker/resident/resident-run` header — the ⚠️ that states this doc's premise: *"This gate is only as good as the premise that AILANG is the agent's SOLE route to acting"*
- CLAUDE.md §2 (no silent fallbacks), §3 (audit before patching), memory: *verify the seam, not the artifact*

## Future Work

- M-AGENT-SAFE-RUNNER M3: submission over the message bus, so the gate runs in a different process from the agent
- Equivalent profiles for opencode/codex once their tool vocabularies are measured
- A `question` profile (`read` + `ailang_check` only) for the coordinator's question-kind tasks, replacing the ad-hoc list
- Per-user policy files on the resident once D11 isolation lands
