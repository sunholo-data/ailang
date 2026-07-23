# M-EVAL-FMT-WEAKMODEL-AB — M6: `motoko_ext_fmt`, a profile-scoped AILANG extension

**Status**: DESIGN (buildable spec) — path validated 2026-07-23.
**Supersedes** the risky delivery options in [`…-M5-hardset-results.md`](m-eval-fmt-weakmodel-ab-M5-hardset-results.md).
**Package home**: `sunholo-data/ailang-packages` → `packages/motoko-ext-fmt/`.
**Consumer**: `arniwesth/motoko_agent` (our fork, branch `feat/local-eval-profiles`), via a new A/B profile.

## Why this path (recap)

M4 + M5 proved fmt is delivered cleanly but inert without drift; the drift regime is the weak **local**
models, which the Claude-only hook can't reach. Every naive local-delivery path touched the live rig.
This design instead uses motoko's **own extension mechanism** — and motoko is written in AILANG, so the
treatment is an **AILANG package**. North-star aligned: an *extension* (not a core change) that
*dogfoods AILANG + the package registry*, scoped to a *profile* so the rotation is untouched.

## The proven template: `motoko-ext-microrag`

microRAG already does this exact shape via `on_tool_handle` (NOT `on_pre_step` — an earlier draft of
this doc guessed wrong):

```
on_tool_handle(WriteFile{path: foo.ail, content: code}) →
  1. writeFile(path, content)              -- perform the real write
  2. exec("ailang micro-rag context ...")  -- post-process the written content
  3. Handled(toolResult{ stdout: "wrote N bytes\n[μRAG]: <retrieval>" })
Non-.ail writes → Delegate (motoko's default handler).
```

`motoko_ext_fmt` is the same skeleton with the post-process swapped for the formatter.

## Package: `packages/motoko-ext-fmt/`

### `ailang.toml` (modeled on motoko-ext-microrag)

```toml
[package]
name = "sunholo/motoko_ext_fmt"
version = "0.1.0"
edition = "1"
ailang = ">=0.17.0"
description = "Tier-1 motoko extension: canonically formats .ail files on write, so a weak model re-reads drift-free code. Shells out to `ailang fmt --write`."
license = "Apache-2.0"

[exports]
modules = ["sunholo/motoko_ext_fmt/register"]

[dependencies]
"sunholo/motoko_ext_abi" = "2.2.0"   # ExtensionHooks ABI (match microrag's pin)

[effects]
max = ["IO", "Process", "FS", "AI", "Env", "Net", "SharedMem", "Clock", "Stream"]

[metadata]
tags = ["motoko", "extension", "fmt", "format", "ailang"]
ai_summary = "Intercepts WriteFile/EditFile on .ail files via on_tool_handle: performs the real write, then runs `ailang fmt --write` on the file so the model's next turn re-reads canonical code. The mechanism under test in M-EVAL-FMT-WEAKMODEL-AB — does removing syntax drift reduce compile-stuck spirals in a weak local model. Non-.ail writes are Delegated. Logs each format to the eval fmt_hook_events sink for treatment-integrity."

[stability]
level = "experimental"
```

### `register.ail` (module `sunholo/motoko_ext_fmt/register`)

Imports mirror microRAG (`pkg/sunholo/motoko_ext_abi/types` for `ExtCtx`, `ExtensionHooks`,
`ToolCallEnvelope`, `ToolResultEnvelope`, `Handled`, `Delegate`, `NoOpinion`, `PassThrough`, …), plus
`std/process (exec)`, `std/fs (writeFile)`, `std/json`, `std/string (endsWith)`.

`register_with_config(cfg) -> ExtensionHooks` returns hooks where every field is the neutral default
(`on_tool_policy → NoOpinion`, `on_pre_step → PassThrough`, `on_build_system_prompt → empty patch`, …)
**except `on_tool_handle`**:

```
on_tool_handle(ctx, call):
  if call.tool not in {WriteFile, EditFile}      → Delegate
  path := call path arg;  if not endsWith(path, ".ail")  → Delegate
  writeFile(path, content)                          -- the real write (respects AILANG_FS_SANDBOX)
  r := exec("ailang", ["fmt", "--write", path])     -- bounded; format in place, idempotent
  emit_fmt_event(ctx.workdir, path, r.exitCode)     -- append to .claude/fmt_hook_events.jsonl
  Handled(toolResult{ stdout: "wrote N bytes; formatted (exit " + r.exitCode + ")" })
```

Notes:
- **Idempotent + meaning-preserving**: `Parse(fmt(x)) ≡ Parse(x)`; a fmt failure leaves the file as
  written (atomic `--write`) and logs `status:"error"` — never wedges the turn.
- **`EditFile` support**: microRAG only handles `WriteFile`. motoko's edit tool set includes `EditFile`;
  confirm its envelope shape from `src/core/tool_catalog.ail` and handle both (Delegate anything else).
- **Treatment-event sink**: append `{status, file, exit_code, detail}` JSONL to
  `<workdir>/.claude/fmt_hook_events.jsonl` — the exact sink `eval_harness.ReadFmtHookSink` consumes, so
  banks carry identical `fmt_hook_events` and the M-void-clause integrity check works unchanged.

## Consumer wiring (`motoko_agent` fork)

1. **Registry**: add `motoko_ext_fmt` to `src/core/ext/registry_generated.ail` — an import + a
   `resolve("fmt", cfg)` arm. This file is *generated* from the `ExtRegistry` (`ailang.toml`
   `registry_import` / `output`), so add the dependency and regenerate rather than hand-edit if the
   generator is wired; otherwise mirror the existing arms.
2. **Two profiles** under motoko's profile dir:
   - `fmt_off` — identical to today's rotation profile (`ext-order` = current set).
   - `fmt_on`  — same, plus `fmt` in `ext-order`.
   Everything else — model, ollama_docs, microrag, dp7, budgets — held constant. The **profile is the
   A/B toggle**; no env var needed.
3. **Build**: `ailang` compiles the `.ail` core + the new ext package (local-path override via
   `../ailang-packages`, per motoko's `ailang.toml`). Rebuild the `motoko` binary.

## Run (the actual A/B)

`ailang eval-suite --agent --models motoko-local-qwen3-6-35b-a3b-mxfp8 --langs ailang --trials 5`,
twice, with `MOTOKO_CONFIG` pointed at `fmt_on` then `fmt_off`, on a **drift-verified** benchmark subset.
Per M4/M5's hard lesson: **confirm current compile-stuck > 0 on the chosen benchmarks BEFORE selecting**
(qwen's live-rotation `compile_error`s already evidence drift, but re-check on the exact set). Primary
metric, CI, and void clause identical to the [M5 prereg](m-eval-fmt-weakmodel-ab-M5-hardset-prereg.md).

## Verification gates (before trusting any A/B number)

1. **Ext loads**: `motoko` with `fmt_on` lists `fmt` among active extensions.
2. **Treatment integrity**: a single `fmt_on` run banks `fmt_hook_events > 0` with `status:"formatted"`;
   a `fmt_off` run banks `0`. (Same gate that correctly classified M5.)
3. **No leak into the rotation**: the rotation's profile is byte-unchanged; a rotation cycle still banks
   `fmt_hook_state` absent / `0` events.

## Scope

A new AILANG package + registry wiring + two profiles + a motoko rebuild + isolated integrity
verification. Two repos (`ailang-packages`, `motoko_agent` fork). A proper sprint — run via
sprint-planner/executor or a mission iteration with clean context. Both repos publish/override through
the AILANG package registry, so this also exercises the package-dogfooding path end to end.

## Open questions to resolve at build time

- `EditFile` envelope shape + whether motoko emits `MultiEdit` (handle or Delegate).
- Is `registry_generated.ail` regenerated by a target, or hand-maintained? (affects step 1).
- `ailang fmt` timeout bound inside the ext (microRAG caps its exec; match that — never unbounded).
