---
paths:
  - "internal/ai/ollama/**"
  - "tools/launchd/dev.ollama.serve.plist"
  - "tools/launchd/nightly-eval.sh"
  - "tools/launchd/nightly-lang-eval.sh"
---

# Local Model Rules (ollama on the rig)

## The rig has a memory budget, and ollama ignores it by default

**Measured 2026-09-03: unconfigured ollama kernel-panicked this machine.** It claims
**84% of unified memory** as VRAM (`total="107.5 GiB"` of 128 GiB), reserves nothing
(`overhead="0 B"`), and therefore hands the model its full native context
(`default_num_ctx=262144`). At 256k the 27B runner peaked at **90.39 GiB**, leaving
~38 GB for the desktop, the agent fleet and the harness. Free memory hit 15 MB, the
pager reclaimed 20 of 3,088 pages wanted, and the watchdog reset the box.

Two variables bound it. **Both are required — they do different jobs:**

| Variable | Role | Current |
|----------|------|---------|
| `OLLAMA_GPU_OVERHEAD` | **Admission** control — subtracted from the budget ollama considers when deciding whether a model fits | 32 GiB |
| `OLLAMA_CONTEXT_LENGTH` | **Runtime** bound — KV size is a direct function of context, so this caps how far a *loaded* model can grow | 131072 |

The reservation alone is not enough: a model admitted under budget still grows past
it as KV fills, which is exactly what the 90 GiB peaks were.

`OLLAMA_GPU_OVERHEAD` is **bookkeeping, not an allocation** — nothing is held and
every other process still sees the whole machine:

```
available="45.3 GiB"   free="77.8 GiB"   overhead="32.0 GiB"
```

`free` is real free VRAM; `available = free − overhead` is only what ollama will
*consider*.

## Before changing either value

Sizing is one inequality: **`weights + KV(context) < available`**. Raising context is
safe only while it holds. Check the largest prompt the harness actually sends before
lowering it — the max ever observed is **108,738 tokens**, which is why the cap is
131072 and not 32768. A 1M-context model's KV alone exceeds this machine regardless
of settings; that is hardware talking, not configuration.

Full derivation, the panic numbers and the `free`/`available`/`overhead` distinction:
**"Ollama Memory Budget on the Rig"** in `docs/docs/guides/debugging.md`.

## Two traps that have each cost a machine

- **No swap on the rig.** `/private/var/vm` is empty; ollama logs `free_swap="0 B"` on
  every sample. A machine with swap thrashes audibly before it dies and someone
  notices — this one goes straight from healthy to wedged. There is no warning phase.
- **The kernel's `memoryPressure` flag read `false` throughout the panic.** Never key
  a guard to it. Use free pages and reclaim rate.

`AILANG_EVAL_MAX_RSS` (8 G default) bounds **only generated code under eval**. It
never observes ollama, the agent processes, or the session population — so it cannot
protect the machine from this class of failure.

## The plist in this directory is the source of truth

Installed plists in `~/Library/LaunchAgents/` are **copies, not symlinks**. Editing
only the installed copy means the next reinstall silently reverts the fix; editing
only the repo copy means nothing changes until reinstall. Change both, and verify:

```bash
diff <(plutil -p tools/launchd/dev.ollama.serve.plist) \
     <(plutil -p ~/Library/LaunchAgents/dev.ollama.serve.plist)
```

They drifted before: the repo carried `OLLAMA_KEEP_ALIVE=-1` (models never unload)
while the running config had `60m`.
