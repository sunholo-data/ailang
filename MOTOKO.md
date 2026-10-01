# motoko_agent on this machine — the map

**One-line answer (since 2026-09-28):** evals run `/Users/voightkampff/dev/mk-main`, branch
`sunholo/main-dst` = Arni's `main` (DST core, extension ABI 8.0) plus our local commits
(`cloud` and `ollama_microrag` profiles, `motoko_ext_ailang_tools`, rig-lease forwarding).
Its extensions are the in-repo `packages/` copies. The ABI 2.2 fork (`sunholo/eval-canonical`)
and its worktrees were removed on 2026-09-28; the branch survives on our fork. See §9.

This file exists because "which motoko are we actually running?" cost a full
audit once. Read it before touching any `~/dev/mk-*` directory.

---

## 1. There is only ONE clone

```
/Users/voightkampff/dev/arniwesth/motoko_agent     <- the clone
  remotes:  origin = arniwesth/motoko_agent            (upstream, Arni's)
            fork   = sunholo-voight-kampff/motoko_agent (ours)
```

Every `~/dev/mk-*` directory is a **git worktree** of that one clone — not a
separate checkout, not a separate fork. `git worktree list` is the truth.

| Path | Branch | What it is |
|---|---|---|
| `dev/arniwesth/motoko_agent` | `feat/local-eval-profiles` | the clone; stale branch, do not eval from it |
| **`dev/mk-main`** | **`sunholo/main-dst`** | **CANONICAL — what `motoko` on PATH runs** (Arni's `main`, ABI 8.0; see §9) |

Branches `integration/sync-clean-20260624` (was mk-sync) and
`integration/editdecl-timeout` (was mk-integration) still exist but their
worktrees were removed on 2026-07-28: both are fully superseded by mk-ast.

"Fully superseded" was verified by commit-subject comparison, and for the one
commit whose subject was absent (`register ollama/qwen3 context limit =
262144`) by confirming the same content is present in mk-ast under a different
lineage. Removing those worktrees deletes no unique work; the branches survive
regardless.

`dev/sunholo-data/motoko_explore` is an unrelated repo. Ignore it.

## 2. How the eval harness finds motoko

`~/go/bin/motoko` is a shim, not a binary:

```bash
exec /Users/voightkampff/dev/mk-main/scripts/run-agent.sh "$@"
```

`run-agent.sh` prefers a repo-local `ailang/bin/ailang` if one exists — it
currently does **not**, so motoko uses the `ailang` on PATH. If motoko ever
behaves like an old compiler, check for that file first.

## 3. Extensions: the in-repo `packages/` are the source of truth

Upstream `main` resolves every extension by path (`mk-main/ailang.toml`:
`{ path = "packages/motoko-ext-*" }`). The ABI 2.2 registry packages
(`sunholo/motoko_ext_*`) were unpublished on 2026-09-28 — see
`ailang-packages/packages/MOTOKO_EXTENSIONS_RETIRED.md`. Edit an extension in
`mk-main/packages/`, then `make check_core && make verify_extensions`. Ours
(`motoko-ext-ailang-tools`) goes upstream as a PR (arniwesth/motoko_agent#200).

## 4. Profiles decide which extensions actually load

Compiled-in ≠ enabled. A profile's `extensions.order` is what loads at runtime.

| Profile | Extensions | Verify gate | Used by |
|---|---|---|---|
| `cloud` | empty_stop_guard, compaction_ai, context_mode, ailang_docs, ailang_tools, microrag | `ailang check benchmark/solution.ail` | all cloud `motoko-*` models |
| `ollama_microrag` | the `cloud` set, compaction on the local model | — | `motoko-local-qwen3-8-27b-microrag` (full local stack) |
| `ollama` | compaction_ai, context_mode (upstream's, 50 steps) | — | `motoko-local-*` lean arms |
| `dogfood` | compaction_ai, context_mode, exa_search | `make check_core` | motoko's own self-hosting work |

`dogfood` is motoko's **development** profile — its `make check_core` gate only
makes sense inside the motoko repo. It is not for benchmarks. Cloud eval models
previously fell through to it by default, which is why their runs had neither
the AILANG-knowledge extensions nor a meaningful verify gate.

Profile selection lives in `internal/eval_harness/models.yml` as
`motoko_profile:`. **Every** motoko model now sets it explicitly — no implicit
defaulting.

## 5. Known deferral: motoko_ext_a2a is not compiled in

`a2a`'s delegate path calls `uuid4()` (Rand), but `motoko_ext_abi` 2.2.0
declares **closed** effect rows on `ExtensionHooks` that exclude Rand, so it
cannot type-check under the post-`1282767ca` effect checker. Re-enabling it
needs an ABI bump (add Rand to the hook rows), which forces a re-annotation
through every extension. No profile references a2a, so nothing is lost today.

## 6. When motoko "won't start"

Startup crashes are silent in the eval output — check the stderr log first:

```bash
ls -t $TMPDIR/motoko-stderr-*.log | head -1 | xargs tail -20
```

Historic causes, in order of likelihood:

1. **Effect-checking failure after an AILANG upgrade.** AILANG's checker gets
   stricter over time; motoko's declared rows must widen to match. This killed
   motoko for six days in July 2026 (`1282767ca`) and 72 runs were banked as
   failures before anyone noticed.
2. **stdlib drift** — the stderr log reads "stdlib version mismatch". The executor
   now exports `AILANG_STDLIB_PATH` only for a std/ identical to the binary's own.
3. **423 from the rig gateway** — a local model called without the rig lease
   (`/Users/Shared/ailang/rig-gate.jsonl`, decision `refused-none`).

Port 8080 is no longer a cause: the executor gives every run its own `ENV_PORT`.

A green boot is:

```bash
cd /Users/voightkampff/dev/mk-main && make check_core && make verify_extensions
```

## 7. Staying mergeable with upstream

Arni is actively refactoring `origin/main` (~18 open PRs). Our branch carries
~45 commits they don't have; we are behind on theirs. Deliberate choices that
keep the merge cheap:

- **Never run `ailang fmt` across motoko sources.** It reflows whole
  expressions and inserts blank lines between imports, producing hundreds of
  lines of conflict surface for no benefit. Keep diffs signature-sized.
- Query our delta with `git log origin/main..sunholo/main-dst`.
- Rebase deliberately, not reflexively — check what Arni has in flight first
  (`gh pr list --repo arniwesth/motoko_agent`).

## 8. Housekeeping

2026-07-28: worktrees for two superseded branches removed; the eval branch
renamed `integration/sync-ast-20260624` -> `sunholo/eval-canonical`.

2026-09-28: worktrees `mk-ast`, `mk-prwork`, `mk-ast-upstream-fix` removed. Every
branch tip is on our fork; `mk-prwork`'s uncommitted snapshot of older upstream
files was committed to `backup/mk-prwork-staged-20260928` first.

Still outstanding: the clone sits on the stale `feat/local-eval-profiles`.

## 9. Migration to upstream `main` (ABI 8.0) — in progress since 2026-09-26

Plan (completed 2026-09-28/30): `design_docs/implemented/v0_48_0/m-motoko-dst-refactor-migration.md` (per-commit verdicts:
`m-motoko-fork-disposition.md`). **The shim was switched to `mk-main` on 2026-09-28 (Mark):**
we are adopting it, so it is measured on its own (before/after per change), not A/B'd
against the old fork.

- `~/dev/mk-main` builds on AILANG **≥ v0.44.1** (v0.42.0–v0.44.0 fail on a `deriving (Eq)`
  regression; v0.44.1 fixed it). `make check_core`: 60/60 core, 9/9 extensions.
- Our delta on top of upstream: `git log origin/main..sunholo/main-dst` (cloud and
  `ollama_microrag` profiles, `motoko_ext_ailang_tools` = PR #200, rig-lease forwarding, and
  one local-only commit that drops `src/core/ailang.toml`). #191, #193 and #198 are merged.
- `make dst` needs **GNU make 4** (`gmake`, via `brew install make`). Stock macOS make 3.81
  skips the parallel targets and still prints "all targets passed".
- Headless runs need `MODEL` set explicitly: without it motoko ignores the profile's model and
  falls back to `anthropic/claude-sonnet-4-6`. The eval executor always sets it.
- **The rig lease.** Since 2026-09-28 the rig's ollama sits behind a gateway that, under a held
  rig lock, refuses long work without the holder's token (M-RIG-GPU-ADMISSION-GATEWAY). The token
  reaches motoko's AILANG child as `AILANG_RIG_LEASE`; mk-main's `buildChildEnv` allowlist dropped
  it, so every local-model step got 423. Fixed on `sunholo/main-dst` (3e09f986, with a test). It
  also needs an ailang whose clients attach the lease (dev after adc59d45b; not in v0.47.2).
- Open questions to Arni (package source of truth, where AILANG extensions live, a post-tool
  hook, versioning): arniwesth/motoko_agent#192. Arni confirms ABI 8.0 is stable.
- Not yet ported from the fork: compact-interface auto-read, per-edit `ailang check`,
  whitespace-tolerant `EditFile`, fmt, `MOTOKO_MAX_STEPS`, the step-0 resolved-config event.
  ABI 8.0 CAN wrap native tools: a `ToolProvider` advertising `"WriteFile"`/`"EditFile"`/
  `"ReadFile"` receives those calls and `Delegate` falls through to the next provider, then
  native (see `motoko_ext_microrag`). It cannot APPEND to a native result, so the extension
  performs the write/edit itself. Not blocked on Arni.
- Local motoko (`motoko-local-qwen3-8-27b`) is OUT of the GPU rotation until `mk-main` is stable.

### To do (agreed 2026-09-28)
- **Cloud executors:** once motoko main is stable locally, update the AILANG cloud executor images
  to motoko main + `motoko_ext_ailang_tools` (they still carry the old fork). Mark: "once you are
  happy with motoko locally".
- **Registry packages: RETIRED 2026-09-28 (Mark).** All 14 ABI 2.2 `sunholo/motoko_ext_*`
  packages (96 versions) were unpublished; see `ailang-packages/packages/MOTOKO_EXTENSIONS_RETIRED.md`.
  Still to do: port `fmt` to ABI 8.0 (likely inside `motoko_ext_ailang_tools`).
- **Local motoko in the rotation:** re-add `motoko-local-qwen3-8-27b` with a longer canary once the
  above is stable.
