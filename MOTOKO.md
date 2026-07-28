# motoko_agent on this machine — the map

**One-line answer:** evals run `/Users/voightkampff/dev/mk-ast`, branch
`integration/sync-ast-20260624`, and every extension comes from the **online
registry** (`registry.ailang.sunholo.com`). Nothing else is the eval version.

> The branch name is a leftover from a one-off sync and says nothing about what
> it is. Renaming it to `sunholo/eval-canonical` is pending (the command is in
> §8); update this file and the `motoko` shim in the same breath.

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
| **`dev/mk-ast`** | **`integration/sync-ast-20260624`** | **CANONICAL — what `motoko` on PATH runs** |
| `dev/mk-prwork` | `fix/reliable-compaction` | PR #97 only |
| `dev/mk-ast-upstream-fix` | `fix/ailang-0.30-message-images` | PR #96 only |
| ~~`dev/mk-sync`~~ | `integration/sync-clean-20260624` | fully superseded by mk-ast — worktree pending removal |
| ~~`dev/mk-integration`~~ | `integration/editdecl-timeout` | fully superseded by mk-ast — worktree pending removal |

"Fully superseded" was verified by commit-subject comparison, and for the one
commit whose subject was absent (`register ollama/qwen3 context limit =
262144`) by confirming the same content is present in mk-ast under a different
lineage. Removing those worktrees deletes no unique work; the branches survive
regardless.

`dev/sunholo-data/motoko_explore` is an unrelated repo. Ignore it.

## 2. How the eval harness finds motoko

`~/go/bin/motoko` is a shim, not a binary:

```bash
exec /Users/voightkampff/dev/mk-ast/scripts/run-agent.sh "$@"
```

`run-agent.sh` prefers a repo-local `ailang/bin/ailang` if one exists — it
currently does **not**, so motoko uses the `ailang` on PATH. If motoko ever
behaves like an old compiler, check for that file first.

## 3. Extensions: the registry is the source of truth

`mk-ast/ailang.toml` pins every extension to a **published registry version**.
There are deliberately **no `{ path = ... }` overrides** — those were the cause
of the local-vs-published drift that made this repo hard to reason about.

Workflow when changing an extension:

1. edit in `dev/sunholo-data/ailang-packages/packages/motoko-ext-*`
2. bump `version` in that package's `ailang.toml`
3. `ailang publish` (the server does a **stricter** compile than
   `--dry-run` — trust the server, not the dry run)
4. repin in `mk-ast/ailang.toml`, then `ailang lock`
5. `ailang generate-extension-registry`
6. `make check_core && make verify_extensions`

To validate a change across the *real* dependency graph before publishing,
temporarily point `mk-ast/ailang.toml` at local paths, converge, then restore
the registry pins. Per-package `ailang check` **under-reports** — it does not
see the full graph.

## 4. Profiles decide which extensions actually load

Compiled-in ≠ enabled. A profile's `extensions.order` is what loads at runtime.

| Profile | Extensions | Verify gate | Used by |
|---|---|---|---|
| `cloud` | compaction_ai, context_mode, ailang_docs, microrag | `ailang check benchmark/solution.ail` | all cloud `motoko-*` models |
| `ollama` | compaction_ai, context_mode | — | `motoko-local-*` baseline |
| `ollama_docs` / `ollama_microrag` / `ollama_fmt` / `ollama_dp7` | ollama + one variable | varies | A/B arms |
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
2. **A zombie holding port 8080** (`lsof -i :8080`) — motoko pins `ENV_PORT=8080`.
3. **Stale `ailang.lock` vs cache** — the log says "dependency … content changed".
   Fix: `rm -rf ~/.ailang/cache/registry/sunholo/<pkg>` then `ailang lock`.

A green boot is:

```bash
cd /Users/voightkampff/dev/mk-ast && make check_core && make verify_extensions
```

## 7. Staying mergeable with upstream

Arni is actively refactoring `origin/main` (~18 open PRs). Our branch carries
~45 commits they don't have; we are behind on theirs. Deliberate choices that
keep the merge cheap:

- **Never run `ailang fmt` across motoko sources.** It reflows whole
  expressions and inserts blank lines between imports, producing hundreds of
  lines of conflict surface for no benefit. Keep diffs signature-sized.
- Query our delta with `git log origin/main..sunholo/eval-canonical`.
- Rebase deliberately, not reflexively — check what Arni has in flight first
  (`gh pr list --repo arniwesth/motoko_agent`).

## 8. Pending manual steps

These mutate git refs/worktrees and were left for a human to run:

```bash
cd /Users/voightkampff/dev/arniwesth/motoko_agent
git worktree remove --force /Users/voightkampff/dev/mk-sync
git worktree remove --force /Users/voightkampff/dev/mk-integration
git branch -m integration/sync-ast-20260624 sunholo/eval-canonical
```

After the rename, update the branch name in §1 of this file and in the
`~/go/bin/motoko` shim header.

The clone itself still sits on the stale `feat/local-eval-profiles` with one
uncommitted line in `src/tui/src/runtime-process.ts`
(`AILANG_OLLAMA_HTTP_TIMEOUT_SEC` forwarding). That change is **already in
mk-ast** (line 408), so it is redundant and safe to discard — but it is real
work, so discarding it is a human's call.
