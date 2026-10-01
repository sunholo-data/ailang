# M-MOTOKO-AILANG-ONLY-LANE — motoko as the harness inside the ailang_only executor boxes

**Status**: PLANNED (2026-10-01; revised after design-quorum rounds 1–3 — per-tool verification, property-based gating, fail-closed loading (met by upstream strict extension loading as a rollout precondition, Mark's ruling), and in round 3: motoko-resolved config instead of a Go re-parse, an enforcement probe instead of trusting the declared flag, and the measured Deny-wins verdict merge) · **Owner**: attended (prioritised by Mark over the motoko mission
loop) · **Repos**: ailang (executor, eval), motoko main (`~/dev/mk-main`, our extension), upstream
arniwesth/motoko_agent (one core ask), ailang-multivac (agent config)

## Goal

Mark, 2026-10-01: *"motoko will be the best harness to write AILANG, and the ailang only executor
boxes will be the most secure place to do it."* Today the `ailang_only` lane runs **pi**. This doc
makes **motoko** able to run that lane with the **same security boundary**, measures it against pi on
the same model, and then switches the lane's default harness from pi to motoko.

Non-goal: changing the boundary itself. The lane's security lives in AILANG (`ailang policy-tool`,
`ailang run --policy`, the v0.41 runtime confinement) and stays exactly as it is; this doc swaps the
harness that calls it.

## Where things stand (verified 2026-10-01)

**The lane (pi).** `ProfileTools("ailang_only")` = 8 tools: AilangRead, AilangEdit, AilangWrite,
AilangCheck, AilangRun, BuiltinsSearch, ExamplesSearch, AilangCLI — no Bash, no native file tools
(`internal/executor/toolpolicy.go:53`). pi is launched with `--no-builtin-tools --tools <allowlist>`
and three embedded extensions (`internal/executor/pi/toolnames.go:84`, `pi/profile_assets.go`).
The tools are thin adapters: `ailang_run` → `ailang run --policy $AILANG_AGENT_POLICY`; read / write
/ edit / cli → `ailang policy-tool --policy <p>` (one JSON request in, one JSON response out;
`cmd/ailang/policy_tool.go`, `internal/policytool`: fileguard root-anchored file ops, a per-command
CLI schema table). The policy file is materialised 0444 outside the workspace, must say
`security_mode = "restricted"`, and the coordinator refuses to dispatch without it. Users: 34 cloud
agents (`ailang-only-executor`, `daneel-executor`, `design-doc-creator-daneel`, ~30 `pkg:` agents and
`package_agent_template`), all pi on `openrouter/z-ai/glm-5.3-flash`. Measurement: one 5-benchmark A/B
on 2026-09-16 that predates the current 8-tool surface — the lane as it stands is unmeasured.

**motoko main.** Seven native tools are always advertised — ReadFile, WriteFile, EditFile, BashExec,
RunTests, Search, MotokoRuntimeStatus (`src/core/tool_catalog.ail:78`). The model's list is
`tools() ++ extension schemas` (`tool_catalog.ail:150`) with **no de-duplication and no removal**:
`ToolConfig` has no allowlist (`src/core/config.ail:45`). ABI 8.0 gives an extension `ToolPolicy`
(per-call Allow / Deny / NoOpinion / Pending; Deny returns `denied_by_policy` to the model,
`tool_phase.ail:111`; verdicts from several extensions merge as **Deny > Pending > Allow >
NoOpinion**, Deny short-circuiting — `ext/runtime.ail:577-596` — so no co-loaded extension can
override a Deny, and both dispatch paths consult it, `tool_phase.ail:567` and
`tool_envelope_dispatch.ail:40`), `ToolProvider` (handle calls to named tools, with the Process effect) and
`DescribeTools` (add schemas). It cannot hide a native tool.

**Consequence found while designing.** A `ToolProvider` that claims a native name adds a second,
name-only schema of that name (`collect_hook_schemas` → `ext_tool_schema`). Our
`motoko-ext-ailang-tools` claims WriteFile / EditFile / ReadFile, so **the cloud profile already
sends those three names twice**. It works on DeepSeek via OpenRouter; providers that require unique
tool names would reject the request. Tracked as part of the upstream ask (D3) and the follow-up below.

**Step 1 is done.** PR #1429: the motoko executor now refuses any task whose `AllowedTools` forbids a
native tool or that carries a program policy, instead of silently running with full tools.

## Design

### D1 — A new extension, `motoko-ext-ailang-policy`, provides the lane's tools under NEW names

It registers, via `DescribeTools` (full schemas) + `ToolProvider`, the eight lane tools with the
**same names and argument shapes as the pi lane** (the canonical names map exactly as pi's
`toolnames.go` maps them). **It does not claim native names**, so there are no duplicate schemas.
Each handler mirrors the pi implementation exactly — verified per tool on 2026-10-01:

| Tool | pi implementation (`.pi/extensions/`) | motoko handler |
|---|---|---|
| `ailang_read` / `ailang_write` / `ailang_edit` | `ailang policy-tool --policy <p>`, one JSON request on stdin (`ailang-exec.ts:79-101,365-408`) | same call, same request shape |
| `ailang_cli` | `policy-tool` op per the Go CLI schema table; run/exec/repl refused there (`ailang-exec.ts:409`, `internal/policytool/cli_ops.go:49-75`) | same call |
| `ailang_run` | `ailang run --policy <p> …` (`ailang-exec.ts:331-357`) | same argv |
| `ailang_check` | under a policy: `policy-tool` op=check, path validated in Go (`ailang-lsp-lite.ts:108-148`) | same call; the no-policy branch is NOT ported — the extension refuses to register without a policy (D4) |
| `builtins_search` | fixed argv `ailang builtins list --json`, filtered in-process (`ailang-lsp-lite.ts`) — reads no file, takes no path | same fixed argv |
| `examples_search` | reads `*.ail` under a fixed examples directory in-process (`examples-search.ts:20-161`); never the workspace | same: a fixed, read-only examples root resolved at registration, no model-supplied path |

The boundary is therefore one implementation shared with pi: file, CLI and run access go through
`ailang policy-tool` / `ailang run --policy`; the two search tools take no path and touch no
workspace file. The extension holds no security logic of its own.

Rationale: re-using `ailang policy-tool` keeps the boundary in one place (AILANG Go), already tested
and hardened; the extension adds no security logic of its own. Matching pi's names makes the A/B
compare harnesses, not tool vocabularies.

### D2 — The same extension DENIES every native tool, as defence in depth

Its `ToolPolicy` returns `Deny` for ReadFile, WriteFile, EditFile, BashExec, RunTests, Search and
MotokoRuntimeStatus, with a reason that names the lane tool to use instead. Until D3 lands the model
still SEES the native schemas; a lane prompt line says they are disabled. This costs turns (measured
in D6).

**The Deny is only as good as the extension being loaded, so the lane FAILS CLOSED on that — by an
upstream precondition, not by a mechanism of ours (Mark, 2026-10-01, after design-quorum round 2).**
motoko stores `extensions.strict` but upstream `main` does not enforce it: an `extensions.order` name
that resolves to no installed extension is skipped silently, and an extension that registers no
capability is omitted with only a warning (`registry_generated.ail` `parse_tokens`; a malformed
registration already exits 2). **We upstreamed the fix** — [arniwesth/motoko_agent#205](https://github.com/arniwesth/motoko_agent/pull/205) — under `extensions.strict = true` both
cases exit 2 before any session starts, with a JSON error line; non-strict keeps today's behaviour
(plus a warning for the unresolved name). It is verified on real motoko runs (unknown name → refused;
`ailang_tools` with `"enabled": false` → refused) and by a new `verify_strict_extensions` target in
`check_core`, mutation-checked against the unpatched registry. **The hardened motoko lane does not
roll out until that PR is merged upstream.** The lane profile sets `extensions.strict = true`, and the
executor's property check (D4) refuses a lane profile without it. An extension that crashes while
registering kills the AILANG runtime, so it cannot start a session either.

A runtime-level alternative was spiked and ruled out for now (**spike A**, 2026-10-01): running
motoko's AILANG runtime itself under `ailang run --policy` in restricted mode. Measured blockers:
(1) admission refuses motoko's entry row — `Env`, `Rand`, `SharedMem`, `Trace` have no confined
adapter (`missing_from_policy` in the admission decision); (2) under a policy the program's own files
must sit inside `fs_sandbox`, but motoko's code and the agent workspace are separate roots;
(3) confined Process admits only read-only git, so the lane tools cannot call `ailang`; (4) the
decisive one — motoko's runtime depends on its Node env-server over localhost (forbidden in restricted
mode), and the env-server itself runs delegated shell commands with `execSync`
(`src/tui/src/env-server.ts:1304`), outside any AILANG policy; (5) a policy pins one AI provider while
motoko also calls a compaction model. So an AILANG-runtime boundary cannot contain motoko, nor any
harness whose tools run in Node (pi included). The harness-independent boundary is **box-level
hardening** of the ailang_only executor image — no shell binaries, read-only root filesystem with only
the workspace writable, a non-root user, egress only through the existing proxy — filed as its own
design. Spike side-finding: the policy loader rejects `Rand` as an unknown capability while restricted
mode's error message lists it as admitted, and `SharedMem` is in neither list (AILANG backlog).

The extension must be **first** in the lane profile's `extensions.order`, and the profile must load
**nothing that can execute outside the policy**: no `compose` (its snippet path runs `ailang run
--caps` without `--policy`), no `context_mode` (runs an external binary), no `microrag` hook that
shells out unpolicied, no `ailang_tools` (its `ailang check` runs via `bash`). `hybrid: false`.
Allowed alongside: `empty_stop_guard`, `compaction_ai` (model calls only), `repetition_guard`.

### D3 — Upstream ask: a profile can hide native tools

One small core change in arniwesth/motoko_agent: a `tools.native` allowlist (or `tools.disable`
list) in `ToolConfig`, applied to `tools()` before `tools_with_extensions`. Also: when an extension's
`ToolProvider` claims a native name, its schema should replace the native one rather than be appended
(fixes the duplicate-schema defect for everyone). We are guests: an issue first, then a PR if Arni
wants one. The lane does NOT wait on it — D2 is the interim — but D3 removes the dead schemas and the
wasted turns.

### D4 — Executor wiring (AILANG)

- `internal/executor/motoko` accepts a task with `AllowedTools == ProfileTools("ailang_only")` only
  when **the properties that make the lane safe hold, checked on the profile's contents, not its
  name** (quorum round 1), **as motoko itself resolves them, not as a Go re-parse of the file**
  (quorum round 3 — a parser differential would let the check pass on a config motoko reads
  differently). Before spawning, the executor runs motoko's own resolver under the task's env —
  `ailang run --caps IO,FS,Env --entry print_config_json src/core/config.ail -- --workdir <ws> --profile <p>` in the motoko tree (`config.ail:667-671`; verified 2026-10-01 — it returns the profile's resolved `extensions.order`/`strict`/`tools.hybrid`, while without the two arguments it silently returns defaults, so the executor must pass both and check the order is non-empty)
  — and refuses unless the resolved JSON shows `extensions.order[0] == "ailang_policy"`, every other entry is on the
  lane allowlist (`empty_stop_guard`, `compaction_ai`, `repetition_guard`), `extensions.strict ==
  true`, `tools.hybrid == false`, and the extension's package exists in the motoko tree; AND `Task.PolicyPath` is set and parses as
  `restricted` (`CheckLanePolicy`). The same property check runs in the image test (D5).
- **Enforcement, not declaration** (quorum round 3): `strict == true` means nothing on a motoko build
  that predates #205. Once per executor process, the executor runs motoko's own
  `scripts/verify_strict_extensions.ail` against a throwaway profile that names an uninstalled
  extension with `strict = true`, and requires **exit 2**. If the script is absent or the probe
  starts, the build lacks enforcement and every lane task is refused (`motoko build does not enforce
  extensions.strict`). The image build (D5) runs the same probe and fails without it.
- After spawn, the executor also reads motoko's startup `loaded_extensions` line and kills a run whose
  first extension is not `ailang_policy` — defence in depth on top of strict loading, not the claim.
- It forwards `AILANG_AGENT_POLICY=<PolicyPath>` to motoko (and adds it to `buildChildEnv`'s
  allowlist on our motoko branch so the AILANG runtime and the extension see it). Every other
  restricted list stays refused (PR #1429's rule).
- The result banks `ToolPolicy` = the 8 lane tools and `PolicyDigest = executor.PolicyDigest(path)`.
- The extension refuses to register (loud error at startup) if `AILANG_AGENT_POLICY` is unset or the
  file does not parse as `restricted` — a lane run without a policy must never start.

### D5 — Cloud image and agent config

`docker/Dockerfile.agent-motoko` already builds motoko main (#1413). It must also carry the new
extension (in-repo package on our branch) and z3 (as agent-pi does for `ailang_cli`). Add an image
test equivalent to agent-pi's: a lane run whose session JSONL shows **no native tool executed**.
Agent switch (ailang-multivac `config.cloud.yaml`): `provider: motoko`, `motoko_profile: ailang_only`
on ONE lane agent first (`ailang-only-executor`), then `package_agent_template` and the `pkg:` agents,
then Daneel — each after its own smoke. Daneel's `BLOCKED:` / summary contract and
`acknowledge_only` behaviour are re-checked under motoko before its switch.

### D6 — Measurement gates the switch

Paired `ailang eval-paired`, pi-lane vs motoko-lane, **same model** (`openrouter/z-ai/glm-5.3-flash`),
the current 8-tool surface, **≥ 20 benchmarks**, same window; harness defects excluded first and
listed. Tool use is verified from session logs (pi `tool_execution` events, motoko session JSONL —
`ailang chains chat`), never from prose. Report pass rate, discordant pairs, turns and cost per
solved task, `denied_by_policy` count (the D2 tax), and **zero native-tool executions** as a hard gate.

**Switch criterion**: motoko-lane pass rate ≥ pi-lane's (no worse within the discordant-pair
evidence), zero native executions, zero policy violations. Below that, the lane stays on pi and the
gap is worked; the switch is Mark's call with the numbers in front of him.

## Milestones

| # | Milestone | Owner | Done when |
|---|---|---|---|
| M1 | Executor refuses unenforceable restrictions | ailang | **DONE** — PR #1429 |
| M2 | `motoko-ext-ailang-policy` (D1 + D2) + lane profile | motoko (our branch) | each of the 8 tools behaves as its D1 row (one test per tool, incl. a write outside the root refused by policy-tool and `examples_search` refusing a path argument); **each** of the 7 native tools returns `denied_by_policy` (one test per tool, incl. via any hybrid/delegated route); `make verify_extensions` green |
| M2b | Strict extension loading upstream (D2) | Arni (our PR [#205](https://github.com/arniwesth/motoko_agent/pull/205), opened 2026-10-01) | #205 is **merged upstream**; on the rebased tree, with `ailang_policy` removed from the tree, misnamed in the order, or declining to register, a lane run exits before any session starts — one test per case. **Rollout gate**: M6 does not start before this |
| M3 | Executor wiring (D4) | ailang | properties read from motoko's `print_config_json`, never a Go re-parse; the enforcement probe refuses a pre-#205 motoko tree; refusal tests for each violated property (extension not first, a bypassing extension present, hybrid on, package missing, no policy, non-restricted policy); `AILANG_AGENT_POLICY` reaches the extension; `ToolPolicy` + `PolicyDigest` banked |
| M4 | Upstream issue (D3) | Arni | **FILED** 2026-10-01 as [arniwesth/motoko_agent#204](https://github.com/arniwesth/motoko_agent/issues/204), with the duplicate-schema evidence; bounded wait 14 days, then record and proceed on D2 |
| M5 | Paired A/B (D6) | ailang eval | the D6 report exists and is recorded here |
| M6 | Image + one cloud agent (D5) | ailang + multivac | `ailang-only-executor` on motoko completes one real cloud task in test with zero native executions; then Mark decides the wider switch |

## Risks

- **Container reliability.** motoko runs a bun env-server per session; pi has months of production
  history. M5/M6 surface it; per-run `ENV_PORT` is already in place.
- **D2 tax.** Visible-but-denied native tools may cost turns or confuse weaker models. Measured in
  M5; D3 removes it.
- **Policy drift between harnesses.** Avoided by design: both harnesses call the same
  `ailang policy-tool`; the extension has no policy logic.
- **Upstream timing.** D3 (hide native tools) may not land soon; D2's Deny covers it. The strict
  loading PR (M2b) is a hard rollout gate: if it does not merge, the lane stays on pi.

## Follow-up outside this milestone

`motoko-ext-ailang-tools` duplicates WriteFile / EditFile / ReadFile schemas in the cloud profile
today. Until D3, consider moving its behaviour to new-name tools or accept the duplicate on
providers that tolerate it — decide with the M5 data.
