# `tools/motoko/` — probes that run against the motoko fork, kept in the anchor repo

The motoko sources live in `arniwesth/motoko_agent` (see [MOTOKO.md](../../MOTOKO.md)). We are
**guests** there, so anything we write purely to answer one of our own questions lives here instead
of upstream. Committing them here is what makes a measurement reproducible by the next iteration
rather than a number in a log entry.

## `r8_headroom_band.ail`

Settles disposition row **R8** (`design_docs/planned/m-motoko-fork-disposition.md`): *is the band
between the compaction ladder's 70% target and the seal's 95% permission reachable in practice?*

Answer, measured 2026-08-14 against `main_dst@6c06b08`: **yes**, and the seal permits it —
79% of the window is sent with 54,905 tokens of headroom against a 65,536 output cap. Row → PORT.

### Running it

The probe imports both the extension (`compact_for_pre_step`) and the phase core
(`seal_compacted_payload`), so it needs an upstream worktree with resolvable packages:

```bash
CLONE=~/dev/arniwesth/motoko_agent
WT=~/dev/mk-r8-main-dst                 # a SIBLING of the other mk-* worktrees, never /tmp
git -C "$CLONE" worktree add --detach "$WT" 6c06b08

cd "$WT"
bash scripts/sync-extension-packages.sh   # generates .packages/ (gitignored)
ailang lock                               # REQUIRED — see the gotcha below
cp <ailang-repo>/tools/motoko/r8_headroom_band.ail scripts/dst/
ailang run --entry main --caps IO scripts/dst/r8_headroom_band.ail
```

**Gotcha that costs ten minutes if you skip it:** the committed `ailang.lock` in `main_dst` carries
**absolute** paths from the upstream maintainer's dev container (`/workspaces/motoko_agent/...`, 19
of 19 entries). From any local worktree every package import fails with *"package directory not
found"*, which reads like a broken checkout rather than a stale lock. `ailang lock` rewrites them;
confirm with `grep -c /workspaces ailang.lock` → 0, against `grep -c '"path":'` → 19 as the control.
`.packages/` is gitignored and generated, so the sync script must run first.

### Reading the output

Three arms, and the two controls are the point — an `Ok` from the seal only means something if you
have shown the seal can say `Err`:

| Arm | History | Expect |
|---|---|---|
| **B** negative control | small | `PassThrough`, seal `Ok`, large headroom |
| **A** the question | one large **user** message ≈79% of window | `tier=floor`, seal `Ok`, headroom < 65,536 |
| **C** over-band control | ≈158% of window | seal `Err(SealExhausted)` |

The load-bearing detail is *why* arm A reaches the floor tier: `elide_walk` only rewrites messages
with `role == "tool"`, so a large **user** message is invisible to all four ladder tiers. They each
produce near-identical output (the ladder removes ~1%), the unconditional floor branch fires, and
the payload is sent unchanged. No amount of keep-last aggressiveness moves this case — only a
reserve at the seal does.
