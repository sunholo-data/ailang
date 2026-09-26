---
paths:
  - "internal/ai/ollama/**"
  - "tools/launchd/dev.ollama.serve.plist"
  - "tools/launchd/nightly-eval.sh"
  - "tools/launchd/nightly-lang-eval.sh"
  - "tools/launchd/mission-control.sh"
  - "tools/launchd/test_mission_memgate.sh"
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

## Capping ollama moved the problem; it did not end it

The caps above held. Measured across the three OOM events that followed
(`JetsamEvent` 2026-09-04 05:08, 09-05 08:29, 09-05 09:23), ollama's physical
footprint was **25.77 GB at every one of them, identical to two decimals** — flat,
under load, days apart. It was no longer the growth term.

The machine still filled: ~60 MB free, 46 GB wired, and a compressor holding
**131 GB** of logical pages while only 37 GB was resident across 566 processes. The
largest identifiable population was **ours** — 22 concurrent Claude Code processes,
12.7 CPU-hours — because every mission plist carries `RunAtLoad=true` and a boot or
GUI login fires all four missions within seconds. The motoko plist recorded this
same alignment on 2026-08-17 and fixed only the steady-state half (non-harmonic
`StartInterval`s); the boot half bit us on 09-05.

`tools/launchd/mission-control.sh` now carries both halves of the answer:

| Knob | Default | Does |
|------|---------|------|
| `MISSION_BOOT_WINDOW` | 900 | How long after boot the stagger applies (0 outside it, so steady-state phase is untouched) |
| `MISSION_MIN_AVAIL_GB` | 16 | Refuse to start below this much **available** memory |
| `MISSION_MAX_COMPRESSED_GB` | 48 | Refuse when the compressor already holds this much |
| `MISSION_MEM_WAIT` / `MISSION_MEM_POLL` | 600 / 60 | Wait this long for room before yielding the slot |

Offsets are v1 0s · world 420s · docs 840s · motoko 1260s (`_mc_boot_offset`).
Full derivation and the jetsam numbers: **"Fleet Memory Admission"** in
`docs/docs/guides/debugging.md`.

**Available is `free + inactive + speculative + purgeable`, never `free` alone**:
at the 09-23 event free was 66 MB while 7.7 GB sat reclaimable in `inactive`. The
thresholds are *starting values* — nobody has profiled an iteration's peak — but
they sit two orders of magnitude from both observed states (7.8 GB at each OOM,
104 GB idle), and every fire logs the live numbers so the log can correct them.
`tools/launchd/test_mission_memgate.sh` pins all of it.

**The pin delays the fix.** The loops re-exec out of `~/.ailang-driver-pin/<mission>/`,
a worktree at *committed origin/dev* — so a driver edit changes nothing until it is
committed **and pushed**. Verify with
`git -C ~/.ailang-driver-pin/v1 log --oneline -1`.

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
