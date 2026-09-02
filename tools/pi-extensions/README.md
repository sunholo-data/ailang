# pi extensions — mission executor containment

pi is the mission's sprint executor (`pi:openrouter/deepseek/deepseek-v4-flash-0731`).
It runs with **full user permissions** from a git worktree, and containment has been the
directive's scope fence plus the controller's post-hoc `git -C <main-checkout> status
--short` review — prose plus an audit, with nothing enforcing it. Iteration 168 showed the
cost: a killed executor kept running and overwrote a verified tree mid-evaluation.

pi's own docs name this as extension territory ("permission gates, path protection,
sandboxing"), so containment goes here rather than into a VM.

## Two layers, because one is not enough

| Tool | Fenced by | Mechanism |
|---|---|---|
| `bash` | `sandbox/` (this dir, adapted upstream) | `@anthropic-ai/sandbox-runtime` → Seatbelt (`sandbox-exec`) on macOS |
| `write`, `edit` | **`worktree-fence.ts` (this dir)** | `tool_call` hook, allow-list on the resolved path |
| `read` | — | not a write risk; see Limitations |

**The upstream sandbox extension fences only `bash`.** It registers a replacement
`bash (sandboxed)` tool and hooks `user_bash`; `write` and `edit` are Node `fs` calls
inside the pi process, which is *not* sandboxed, so they bypass it entirely. Verified by
reading its source — that gap is why this extension exists. Use both.

## worktree-fence.ts

Allow-list, not deny-list. The upstream `protected-paths` example blocks a few known-bad
substrings; wrong shape here. We don't know every path worth protecting, but we do know
the one path that is legitimate.

```bash
cd "$WT" && PI_FENCE_ROOT="$WT" pi --mode json --no-session \
  -e tools/pi-extensions/worktree-fence.ts --model "$MODEL" -p "$PROMPT"
```

Root is `$PI_FENCE_ROOT`, else cwd. Headless-safe: it only ever blocks or allows, never
prompts (`ctx.hasUI` guards the notification), because a prompt in the mission's
non-interactive path would wedge the loop.

Fails closed: a write tool whose path argument it cannot find is refused, not waved through.

### Tests

```bash
cd tools/pi-extensions && bun run worktree-fence.test.ts
```

18 arms, driving the real extension through a fake `pi` object. Covers `..` escapes,
symlink escapes, the macOS `/tmp` → `/private/tmp` realpath trap, `/a/bc`-vs-`/a/b`
prefix confusion, fail-closed shapes, and pass-through for non-write tools.

Not wired into `make ci` — it needs `bun`, which CI does not carry. Run it by hand when
changing the fence.

### One trap worth keeping

An early version resolved **relative** paths against the fence root instead of the process
cwd. pi passes relative paths (`{"path":"dbg.txt"}`), so `dbg.txt` resolved to
`<root>/dbg.txt` — inside the fence, allowed — while pi wrote it to `<cwd>/dbg.txt`,
outside. **The unit tests were green throughout**, because they set `root == cwd` and so
could not distinguish the two bases. Only a live run caught it. The suite now has an
explicit `root != cwd` arm, verified to fail when the bug is reintroduced.

## Limitations — read before trusting this

- **Not an exfiltration control.** `read` is unfenced, and pi's own model calls are outside
  the sandbox by design (that is what keeps OpenRouter reachable). This confines *writes*.
- **`bash` needs the separate sandbox extension.** This file deliberately does not parse
  shell; without that extension a `bash` tool call can still write anywhere.
- **SM.B2a-class work (irreversible publish) still stays off this lane.** Sandboxing writes
  is not the same as bounding blast radius on a publish.

## sandbox/ — the bash layer

Upstream's example, adapted for pi 0.73.1. Two forced deltas, both recorded in the file
header: the package rename (`@earendil-works` → `@mariozechner`), and `CONFIG_DIR_NAME`,
which 0.73.1 does not export from the package root (only `"."` and `"./hooks"` are
exported) so it is inlined as `".pi"` — the value read out of the installed
`dist/config.js`. Dependency moved 0.0.26 → ^0.0.71; `SandboxManager.wrapWithSandbox`
and `initialize` were verified present in 0.0.71 before adopting.

Policy: `sandbox.mission.json` here is canonical; installed at
`~/.pi/extensions/sandbox.json`.

### Two gotchas that cost real debugging

1. **`mkdir -p /tmp/claude` is required.** sandbox-runtime pins `TMPDIR=/tmp/claude`
   inside the sandbox — it is built for Claude Code, which creates that directory. Under
   pi nothing does, and `go build` dies with `creating work dir: stat /tmp/claude: no such
   file or directory`. Your own `TMPDIR` does not survive into the sandbox, so exporting it
   is not a workaround.
2. **The Go caches must be in `allowWrite`.** With the stock `[".", "/tmp"]` policy,
   `go build` fails on `~/Library/Caches/go-build: operation not permitted`. Cache dirs
   cannot corrupt the repo, so widening to them costs nothing that matters — but it must be
   deliberate, and re-verified after: widening a policy is how holes get opened.

### Verified live (2026-08-11), running the skill's invocation verbatim

| check | result |
|---|---|
| bash write to `$HOME` | `Operation not permitted`, exit 1 |
| bash read of `secrets.env` | `Operation not permitted`, exit 1 |
| `write` tool to `/tmp` (outside `$WT`) | blocked by the fence |
| `go build ./internal/messaging/` | `BUILD_OK`, exit 0 |

## models.mission.json — the model registry pi actually reads

Policy: `models.mission.json` here is canonical; installed at `~/.pi/agent/models.json`
(the OpenRouter `apiKey` is a placeholder in the repo copy — the installed file carries
the real key, which is why it lives in the config and not in the env: headless-safe).

**pi has no max-tokens flag.** It reads `maxTokens` per model from that file and falls
back to **16384** for any model that omits it (`model-registry.js`), along with
`contextWindow` 128000 and `reasoning: false`. Our four OpenRouter models were registered
as bare `{"id": "..."}`, so from the day the lane was wired until 2026-08-13:

| | declared in models.yml | actually on the wire |
|---|---|---|
| max output tokens | 65536 | **16384** |
| context window | 1M (deepseek-v4-flash-0731) | **128K** |
| reasoning capability | thinks by default | **`false`** — no `--thinking` level accepted |

DeepSeek V4 Flash 0731 thinks by default and OpenRouter bills reasoning tokens against
`max_tokens`, so thinking and answer shared a 16384 budget nobody chose. That is the
mechanism behind the executor's `"stopReason":"length"` failures: three runs, two
directives written specifically to prevent it, no prompt could have fixed it.

`internal/executor/pi/buildPiArgs` **cannot** forward `task.MaxOutputTokens` — there is no
flag to put it in. The registry value reaches the wire only if this file carries the same
number, which `TestPiModelsConfigMatchesRegistry` (in `internal/eval_harness`) now asserts.

### The trap when fixing it

Setting `reasoning: true` **alone makes it worse.** With `compat.thinkingFormat:
"openrouter"` and no effort passed, pi-ai sends `reasoning: {effort: "none"}` — actively
*disabling* thinking that previously happened by default. The `thinkingLevelMap` here maps
`off` to `null`, which suppresses that branch: no reasoning field is sent unless
`--thinking <level>` is explicitly passed. Levels a model cannot honour are mapped to
`null` too (gemma-4-26b does not list `reasoning_effort` in its OpenRouter
`supported_parameters`). glm-5.3-flash **does** list it, and is still mapped to
`null` deliberately: the same reasoning applies — with no effort passed, default
thinking is preserved, and an explicit `--thinking <level>` is the only way to
override it.

Verified live 2026-08-13 after the change: `stopReason: stop`, a 358-char `thinking` block
present in the response — thinking preserved, not disabled.

### Keeping it honest

`maxTokens` mirrors `max_output_tokens` in `internal/eval_harness/models.yml` (CI-enforced).
`contextWindow`, ceilings and `cost` (per 1M tokens — pi reports `cost: 0` for any custom
model without one, which silently zeroed the metered ledger) come from
`GET https://openrouter.ai/api/v1/models`; re-measure rather than assume, prices move.

### Ollama rows — aligned 2026-08-13, and the numbers were not what we assumed

The local rows were left alone in the first pass on the belief that the three harnesses
disagreed as pi 16384 / opencode 4096 / motoko 8192, and that aligning them would move a
live baseline. Measured, two of those three were wrong:

| harness | assumed | **measured** | how |
|---|---|---|---|
| pi | 16384 | **16384** ✓ | pi's own default; config carried no `maxTokens` |
| opencode | 4096 | **~32000** | banked sessions show qwen3.6 outputs to 31,703 and `length` finishes at exactly 32,000 — the per-model `max_tokens: 4096` in `opencode.jsonc` is **not reaching the wire** (sst/opencode#971 class) |
| motoko | 8192 | **32768** | `motoko-local-*` rows declare 32768; the 8192 belonged to the `pi-`/`opencode-` rows |

So the registry disagreed with **itself** — 32768 on `motoko-local-*`, 8192 on `pi-`/
`opencode-*`, for the same model on the same wire. Aligned at **32768** (one budget per
model), which is the value the lane with the most banked local data was already using, so
motoko does not move and opencode moves ~2%. Only pi actually changes, 16384 → 32768.
`TestPiModelsConfigMatchesRegistry` now fails on registry self-disagreement too.

Why 32768 and not higher: qwen3.6 does hit `length` at ~32k about 0.5% of the time, but
those are runaway generations, not normal turns — and on a time-boxed local rig a larger
ceiling mostly buys longer runaways. `MaxTokensPerBench` already covers the cumulative case.

⚠️ **Baseline boundary: 2026-08-13.** Local qwen rows banked before this date ran on a
per-harness budget and are not comparable across harnesses.

`gemma4:26b` is unchanged at 8192 — it is not established as a heavy reasoner in our data,
so there is no evidence to move it. `gemma4:26b-ailang`'s 4096 in `opencode.jsonc` is
genuinely deliberate (it matches `num_predict` in its Modelfile) and is also untouched; the
qwen models have no `num_predict` in their Modelfiles at all, so that rationale never
applied to them.
