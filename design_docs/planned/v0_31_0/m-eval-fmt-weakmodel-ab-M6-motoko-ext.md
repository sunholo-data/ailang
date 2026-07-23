# M-EVAL-FMT-WEAKMODEL-AB — M6: fmt as a motoko AILANG extension (profile-scoped)

**Status**: DESIGN — path validated 2026-07-23, not yet built.
**Supersedes** the risky delivery options in [`…-M5-hardset-results.md`](m-eval-fmt-weakmodel-ab-M5-hardset-results.md).

## Why this is the right path (vs the M5 options)

M5 concluded fmt can only be tested where drift happens (weak local models), and that every naive
delivery path touched the live rig. This design avoids all of that by using motoko's **own extension
mechanism** — which is **AILANG all the way down**:

- **motoko_agent is written in AILANG.** `src/core/ext/{types,runtime}.ail`, `tool_runtime.ail`,
  `agent_loop_v2.ail` are `.ail`. Extensions are AILANG packages `pkg/sunholo/motoko_ext_*`
  (omnigraph, microrag, ailang_docs, compose, …), each exporting
  `register_with_config(cfg) -> ExtensionHooks`.
- **North-star aligned**: this is an **extension**, not a core-floor change (PROGRAM.md's default bias),
  and it **dogfoods AILANG + the package registry** — the fmt treatment is itself an AILANG program.
- **Profile-scoped = safe**: profiles pick extensions via `--ext-order` / the profile's `extensions`
  config (`src/core/config.ail:144`). The fmt ext goes in a **new A/B profile only**; the rotation's
  profile is byte-unchanged, so the live rig is never at risk.

## The extension

New package **`pkg/sunholo/motoko_ext_fmt`** with a `register` module returning `ExtensionHooks`:

- **Hook**: `on_pre_step(ExtCtx, [Msg]) -> PreStepDecision ! {IO, Process, FS, AI, Env, ...}`.
  Fires before each agent step. Side-effect only: run `ailang fmt --write` on the workspace's `.ail`
  file(s) (idempotent; `Parse(fmt(x)) ≡ Parse(x)`), then return `PassThrough` (no message change).
  This delivers canonical formatting **between turns**, so after the model edits in step N, step N+1
  presents it canonical — exactly the drift-removal mechanism the hypothesis is about.
  - Alternative considered: `on_tool_handle` per Edit/Write. Rejected — it would have to re-implement
    edit application to post-process; `on_pre_step` composes without owning the write path.
- **Treatment-event logging**: append `{status, file, exit_code, detail}` to
  `<workdir>/.claude/fmt_hook_events.jsonl` (the existing sink `ReadFmtHookSink` consumes), so the
  eval harness banks identical `fmt_hook_events` and the M-void-clause integrity check works unchanged.
- **Gating**: enabled purely by profile membership (present in the A/B-ON profile, absent in A/B-OFF).
  No env var needed — the profile IS the toggle. OFF profile == today's rotation profile.

## Wiring

1. `pkg/sunholo/motoko_ext_fmt/{register,types}.ail` — the extension (AILANG).
2. `src/core/ext/registry_generated.ail` — add the `resolve("fmt", cfg)` arm + import.
3. Two profiles under motoko's profile dir: `fmt_on` (ext-order includes `fmt`) and `fmt_off`
   (identical minus `fmt`). Everything else — model, docs, microrag, dp7 — held constant.
4. Rebuild motoko (`ailang` compiles the `.ail` core); point the eval `motoko-local-*` model's
   `MOTOKO_CONFIG` at each profile for the two arms.

## Run

`ailang eval-suite --agent --models motoko-local-qwen3-6-35b-a3b-mxfp8` twice (fmt_on / fmt_off
profiles), on a **drift-verified** benchmark subset (per M4/M5's lesson: confirm current compile-stuck
> 0 before selecting — qwen's live-rotation `compile_error`s already evidence drift). N≥5, same set both
arms. Primary metric + void clause identical to the M5 prereg.

## Scope

This is a proper sprint: a new AILANG package, registry wiring, two profiles, a motoko rebuild, and
isolated treatment-integrity verification before the A/B. It is **not** a one-turn build. Recommend
running it via sprint-planner/executor (or a mission iteration) with clean context, since it spans the
motoko fork + the ailang package registry.
