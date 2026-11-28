# M-PKG-PARALLEL-AGENTS: Packages as the unit of parallel agent work

**Status**: In progress. Phase 1 shipped in v0.64.0 (Sprint 1). Phase 2, re-exports, merged in #2475 (Sprint 2). Phase 4, moves, is next (Sprint 4). It depends on Phase 2 only. Phase 3, checkpoints and leases, waits on the full Phase 0 run. Design freeze ratified 2026-11-25 (D1–D4, D7 as recommended), after quorum round 2 (1 pass, 3 reject, all objections folded in).
**Target**: v0.63.0 (Phases 1–2), v0.64.0 (Phases 3–4)
**Priority**: P1. This finishes the coordination half of [M-PKG](../../implemented/v0_9_5/m-pkg-package-system.md), which shipped its packaging half.
**Estimated**: ~11–14 days across 4 phases, plus a 0.5-day Phase 0 spike
**Dependencies**: M-PKG (v0.9.5), M-PKG-MSG (v0.9.9), M-PKG-AUTONOMOUS-UPDATES (v0.10.0), M-PKG-CASCADE-DETERMINISTIC-FIRST (v0.16.0), interface hash v2 signatures (registry metadata `InterfaceSignatures`)
**Source**: Field work, 2026-11-25. Splitting `sunholo/ailang_parse` into a generators-only core for a downstream package (specstore). See the companion doc in that repo: `ailang-parse/design_docs/planned/v0_62_0/v0_62_0_docparse_core_split.md`.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Under checkpoint resolution, a dependent builds against bytes identified by the content hash its lock already records, not against a sibling directory that another agent is editing. |
| A2: Replayability | +1 | A lock plus the checkpoint store reproduces exactly what a dependent compiled against at any point in a multi-agent session. |
| A3: Effect Legibility | 0 | No effect-row change. A re-export carries the original symbol's effects unchanged, so a facade cannot hide or launder them. |
| A4: Explicit Authority | +1 | Package leases make "which agent may write which package" a declared, listable fact instead of a convention. The facade's `[effects].max` must still cover anything it re-exports. |
| A5: Bounded Verification | +1 | Each agent verifies its own package against fixed interfaces of its neighbours, which is the property M-PKG claimed for packages. |
| A6: Safe Concurrency | +2 | This is the doc's purpose. Several agents share one checkout and cannot break each other's builds, so no per-task git worktree is needed for package-scoped work. |
| A7: Machines First | +1 | Change classes come from measured signature diffs and name the affected symbols. Module moves travel as structured import rewrites, not prose. |
| A8: Minimal Syntax | 0 | One new form, `export import M (names)`, built from the existing import grammar. It was chosen over a new keyword (see Conflict Surface). |
| A9: Cost Visibility | +1 | Additive changes stop routing to the AI cascade path. Today every in-module addition is classed C and dispatched to AI (V6, V7). |
| A10: Composability | +1 | Re-exports let packages be refactored behind a stable facade. Checkpoints reuse the tarball file set, and the cascade reuses the existing message kinds. |
| A11: Structured Failure | +1 | New typed diagnostics: MOD016 (re-export cycle), MOD017 (checkpoint missing), MOD018 (checkpoint corrupt: digest mismatch on read), MOD019 (checkpoint with a live dependency), PUB025 (claimed release kind contradicts measured class). |
| A12: System Boundary | +1 | A nested `ailang.toml` becomes a real package boundary (today it is silently absorbed into the outer package, V3). |

**Net Score: +11** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): checkpoint resolution removes a source of nondeterminism (live sibling state). It adds none.
- [x] A3 (Effects): re-exports keep effect rows identical, and the facade package's ceiling check still applies.
- [x] A4 (Authority): leases add declared authority. Nothing gains ambient write access.
- [x] A7 (Machines First): every new artefact is structured (signature diffs, rewrite maps, lease records).

---

## Problem Statement

M-PKG (v0.9.5) gave packages four roles: a **reasoning** boundary, an **ownership** boundary ("one agent per package avoids conflicts"), a **verification** boundary and an **authority** boundary. Its Parallelism Rules say that "Class A changes on different packages can run simultaneously". The registry, effect ceilings, interface hashes and the autonomous cascade all shipped. What did not ship is anything that makes **concurrent agents on one codebase** cheaper than giving each one a git worktree:

- The coordinator still creates a worktree per task (`internal/coordinator/worktree.go:126`, `git worktree add -b …`).
- It still defaults each package agent to `max_concurrent_tasks: 1` (`agent_config.go:353-354`).

### What a worktree is actually buying

A worktree isolates three different things. Packages can take over two of them:

| Job | What goes wrong without it | Can packages replace it? |
|-----|----------------------------|--------------------------|
| **Edit isolation**: two agents never write the same file | Lost edits | **Yes, by ownership.** One agent per package directory. This needs leases (C3), because nothing records ownership today (V9). |
| **Build isolation**: agent A's half-finished edit can't break agent B's build | B's `check`/`test` goes red for reasons B didn't cause; B "fixes" A's code | **Not today.** A path dependency compiles live sibling source, even when the lock already knows that source drifted (V1). This is the gap C2 closes. |
| **VCS isolation**: commits don't interleave | Shared index; concurrent git commands collide | **Partly.** `git commit -- <pkgdir>` commits one package's paths and leaves other staged changes alone. Concurrent git commands still fail on `index.lock` (V10), so commits must be serialised and retried. |

Worktrees still win for cross-package edits of the same files, and for long-lived branches that need separate history. The goal is to make them **unnecessary for package-scoped work**, which is the common case in a well-cut repo.

### What breaks when the code has to move

The ailang-parse split showed that moving code across a package boundary always breaks dependents, and that absorbing the break is manual:

1. **No re-exports.** You cannot keep the old import path alive while code moves.
   - `export import …` is a parse error (`PAR_EXPORT_REQUIRES_FUNC`).
   - A constructor imported through a facade module fails with `IMP010` (V2).
   - Every move is therefore a breaking change for every dependent, even when nothing changed behaviourally. In ailang-parse this forced "clean break, no shims" (that doc's D3) across three downstream repos.
2. **Change classes aren't measured at publish.**
   - The classifier in `internal/messaging/pkg_events.go:155` already does a v2 signature-set diff (same → A, superset → B, otherwise → C).
   - The registry already stores each version's `InterfaceSignatures`.
   - But `ailang publish` builds the dependent notification without signatures (`cmd/ailang/pkg_publish.go:438-446`), so it falls back to hash comparison. `mapChangeClassToSchema` then says outright: "cannot distinguish 'added function' from 'removed function' … assume breaking" (V6).
   - Result: adding one function to an existing module is classed **C** and dispatched to the AI cascade path instead of the deterministic one (`classifyDispatchPath`: A/B → deterministic, C → AI) (V7).
   - `[release] kind` is only checked for presence. Nothing compares the author's claim with the measured class (V8).
3. **No structured migrations.** A `migration-request` carries `from_version`, `to_version` and `block_reason` (`validateMigrationRequest`), but there is no field for "import X now lives at Y". The cascade can't apply a move mechanically, even though a move is the most mechanical breaking change there is (V11).
4. **A nested package isn't a boundary.**
   - M-PKG's Workspace Model shows path-linked sibling packages. It never addresses a package *nested inside* another package, which is the shape you get when you extract a core from an existing root package.
   - Today that nested package is swept into the outer compile and tarball (MOD010; the tarball went from 2 to 3 files, 560 to 685 bytes) (V3).
5. **Smaller friction for an agent joining the graph:**
   - ~~`[exports]` names differ from what dependents must import~~. *Refuted during Sprint 1 (V4):* a dependent can import either the bare `[exports]` name or `pkg/<vendor>/<name>/…`, because bare imports resolve via the module_prefix map. The real friction is narrower: the loader lets several dependencies share a prefix and tries each, so only the `pkg/` form says which package is meant, and nothing tells an agent that. (The loader's own comment cites `sunholo/docparse` as sharing `docparse`. That is stale: its manifest uses `docparse_api`, and it is not in the registry.)
   - Inline tests are not discovered when a package is published (observed in `ailang publish --dry-run` on ailang-parse).

**Impact:**
- Every multi-agent setup pays for a full checkout per task, plus a later merge.
- Every package refactor becomes a coordinated breaking release.
- Every additive change costs an AI cascade run.

The coordination machinery is in place: 11 message kinds, interface hash v2, cascade, `affected-by`, provenance. It is being fed coarser information than the compiler already has.

## Goals

**Primary Goal:** Several agents can work at the same time in **one checkout**, each owning a package. No agent's in-progress state can break another's build, and a cross-package change (add, move, remove) reaches dependents through the cascade with the right class and without hand edits.

**Success Metrics:**
- **Shared checkout:** 3 agents, 1 checkout, 3 path-linked packages, no worktrees. Agent A leaves a type error in its package. Agents B and C stay green on `check` and `test` throughout, and see A's work only after A checkpoints (TST-PKGPAR-07, -08).
- **Classes:**
  - Adding a function to an existing exported module is classed **B** and dispatched deterministically.
  - Removing one is classed **C**, and the notification names the removed signature (TST-PKGPAR-01..03).
- **Moves:** moving a module between packages with `ailang pkg move` needs **zero** dependent edits at the release that moves it. Dependents get a `TC_DEPRECATED_001`-style warning naming the new path, and the cascade applies the import rewrite deterministically (TST-PKGPAR-10..12).
- **Real-world check:** the ailang-parse `docparse_core` split ships as a **non-breaking** minor release of `sunholo/ailang_parse`.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Re-export syntax and identity semantics | New surface syntax that every facade will use. The identity rule (alias or copy) decides whether pattern matching and type equality hold across the facade. | human | design | high |
| D2: Is checkpoint resolution opt-in per dependency, or the default for path deps? | The default changes what every path-linked monorepo compiles against. | human | design | high |
| D3: Checkpoints as their own store, or as a local registry with prerelease versions? | Decides whether "local publish" and "registry publish" share one code path and identity. | human | design | high |
| D4: Should AILANG own a git helper (`ailang pkg commit`), or only document the pathspec pattern? | Owning it means AILANG takes on VCS behaviour (retry, serialisation); not owning it leaves every harness to re-learn the `index.lock` race. | human | design | med |
| D5: PUB025 severity schedule (badge, then gate) | Same grace pattern as PUB001/PUB002 (`ReleaseGateHardFrom`) | agent | compile | low |
| D6: Lease store (messaging store vs a file under `.ailang/`) | Local-only versus visible to the collaboration hub | agent | compile | low |
| D7: Add contract lines to the v2 signature set? | Changes the signature set (not `interface_hash_v2`) for **every** package: each gains the `meta:contracts:v1` marker. Every package's first post-D7 publish therefore classes as U (review) across the ecosystem, not only packages with contracts. | human | design | high |

### Design Freeze

**Ratified 2026-11-25 (Mark, attended: "ok these look good - lets continue to implementation"), after quorum round 2:** D1–D4 and D7 as recommended below.

- [x] **D1, re-export form.** Recommend `export import lib/color (Color, Red, Green)`:
  - It reuses the import grammar.
  - The source of every re-exported name is written on the same line, which serves A7.
  - Re-exported symbols keep the identity of the original. They are not copies, so `Red` through the facade is the same constructor.
  - Only explicit name lists are allowed. Whole-module `export import M` is deferred (see Deferred Decisions).
- [x] **D2, resolution default.** Recommend **opt-in**:
  - Per dependency, `{ path = "../core", resolve = "checkpoint" }`, or repo-wide via `[workspace-resolution]` in the dependent's manifest.
  - Live stays the default, because single-agent editing across packages is the common solo workflow and must not gain a step.
- [x] **D3, checkpoint identity.** Recommend a **separate content-addressed store** keyed by content hash, with no version number:
  - A checkpoint is "the last green state", not a release.
  - Minting prerelease versions would pollute `history`/`versions`, and would require a version bump on every save.
- [x] **D4, git helper.** Recommend a **thin helper**, `ailang pkg commit -m … [--agent ID]`:
  - It runs `git commit -- <pkgdir>`, retrying on `index.lock` with bounded backoff.
  - It refuses if the package is leased to another agent.
  - AILANG takes on no branching, merging or push semantics.
- [x] **D7, contracts in signatures.** Recommend **yes**, as lines in the signature set only. If declined, packages with contracts stay on legacy C classification (see Risks):
  - `interface_hash_v2` stays unchanged, so registry identity checks (PUB005 skew) are unaffected.
  - Without it, C4 would make contract tightenings auto-apply (V24).
  - The cost is one U-classed (review) cascade on the first post-D7 publish of **every** existing package, which is the price of making contract changes visible.

## Solution Design

### Overview

There are five components, in dependency order. Each can ship on its own and is useful alone.

```
  ┌──────────── one checkout ───────────────┐   ┌─ ~/.ailang/checkpoints/<checkout-id>/ ─┐
  │ packages/core (lease A)                 │   │ demo/core/                                │
  │   edits ─► ailang pkg checkpoint ─ 1 snapshot ─► .tmp-x/ ─ 2 check+test the snapshot  │
  │                                          │   │        3 hash + signatures ─ 4 rename ─►  │
  │                                          │   │ <digest>/  5 HEAD {digest, seq}           │
  │ packages/app (lease B)                   │   │        ▲ read + verify hash (MOD018)     │
  │   ailang check ── resolve="checkpoint" ──┼───┼────────┘                                  │
  │   ailang lock --refresh-checkpoints --dry-run  →  class A/B/C + signature diff to HEAD   │
  └─────────────────────────────────────────┘   └──────────────────────────────────────────┘
  no messages per checkpoint (a pkg: inbox send dispatches a task, V22)
  publish (registry) = same diff + PUB025 + migration-request rewrites
```

### Components

**C1: Re-exports (language).** `export import <module> (<names>)` in the import header.
- Each listed name becomes part of the facade module's exports, with the original's identity: type, constructor tag, effects, contracts and `@deprecated` status.
- An optional trailing `@deprecated("pkg/new/path")` on the line extends `TC_DEPRECATED_001` (`internal/pipeline/warn_deprecated.go`). That warning already fires in a *dependent package* (V18). But its fix text assumes the replacement lives in the same module ("replace hello with greet (import it from pkg/demo/core/lib)"), and `@deprecated` attaches only to functions today. C1 therefore adds a "moved" variant: the replacement is a module path, the fix text reads "import X from <new path>", and it applies to re-exported types and constructors as well as functions. This is an extension of the existing warning, budgeted in Phase 2, not reuse for free.
- **Effect ceiling over re-exports:** a facade's `[effects].max` must cover the effects of every function it re-exports, so a package's advertised ceiling stays honest. (Dependents are protected either way, because their own reachability check walks into the original function.) Phase 2 owns this check (TST-PKGPAR-16).
- The facade's interface hash v2 includes re-exported signatures. Re-exporting a symbol is therefore an interface fact, and removing a re-export is a C change.
- Cycles (`a` re-exports from `b`, which re-exports from `a`) are an error: **MOD016**.
- `MOD003 "Re-export not supported"` is defined in `internal/errors/codes.go:75,324` but no code emits it (V5). Remove it when C1 lands, rather than repurposing a code whose meaning inverts.

**C2: Path-dependency checkpoints (tooling; this is what replaces the worktree).**

*Store location (fixed):* `~/.ailang/checkpoints/<checkout-id>/<vendor>/<name>/`.

*Identity.* A checkpoint is keyed by a **checkpoint digest**, not by the content hash. The digest is sha256 over:
- the staged files, including `ailang.toml` in canonical form with path dependencies in their *source-relative* spelling (so the digest does not depend on where the checkout lives);
- the **pinned dependency closure**: for every dependency, its immutable identity (a registry version plus hashes, or a dependency checkpoint's digest).

The content hash is kept as a recorded field, because it is what the lock's drift warning and C4 compare. A manifest-only change (a dependency added or re-pinned) therefore yields a new digest and a new checkpoint (round-1 objection; TST-PKGPAR-21).
- `<checkout-id>` is the sha256 of the absolute path of the git top level containing the package. Outside git, it is the sha256 of the package directory's absolute path.
- It is computed **once, from the source package**, when `pkg checkpoint` runs. It is never computed from a snapshot's location: a snapshot under `~/.ailang/checkpoints/` sits outside any repo and would hash its own temp path.
- The snapshot's pinned lock records, for each pinned path dependency, its digest **and its absolute store directory**. Resolution from inside a snapshot follows those recorded paths and never derives a checkout ID (round-2 Carbon objection; TST-PKGPAR-22).
- The store must live **outside every package directory**: any `.ail` file under a package, including one inside `.ailang/`, changes that package's content hash (V20). An in-repo store would therefore invalidate the hash it is keyed by.
- The store is per checkout by construction, because the key is the checkout path.

*Writing a checkpoint (`ailang pkg checkpoint`)* validates the bytes it publishes, never the live tree:
1. **Snapshot first.** Copy the publishable file set into `…/<name>/.tmp-<random>/` in the same store directory, reusing `copyPackageContents`/`shouldStageFile`, the copy step the publish validator already uses (V21).
   - **Every dependency must be pinned immutably.** A registry or git dependency is pinned as today. A path dependency must itself be `resolve = "checkpoint"`, and the snapshot pins it to that dependency's checkpoint digest at the moment of the copy.
   - A path dependency that would resolve **live** makes `pkg checkpoint` fail with **MOD019**, which names the dependency and the fix: checkpoint it, or mark it `resolve = "checkpoint"`. Otherwise a transitive live edit could break a dependent already locked to this checkpoint (round-1 objection; TST-PKGPAR-22).
   - The pinned closure is written into the snapshot as its own lock. The publish validator's `RewritePathDeps` (to registry/git pins) is not used.
   - The content hash covers `.ail` files only (V25), so pinning leaves the snapshot's content hash equal to the live package's at the moment of the copy. The checkpoint digest, which covers the manifest and closure, is what changes.
2. **Validate the snapshot.** Run `check` and package tests with the snapshot as the package root, resolving dependencies only through the pinned closure. Edits to the live tree after step 1 cannot affect the result, which closes the gap between validating and copying.
3. **Measure the snapshot.** Compute the checkpoint digest, content hash, interface hash v2 and signatures from the snapshot. Write `CHECKPOINT.json` (digest, hashes, pinned closure, signatures, agent ID, time) into the temp dir last.
4. **Publish atomically.** `os.Rename` the temp dir to `<checkpoint_digest>/`, a directory rename within one filesystem. If that directory already exists, every input is identical. Discard the temp dir and **do not advance HEAD**, so `--dry-run` never reports progress that did not happen.
5. **Advance HEAD** (only when step 4 created a new directory), **under an exclusive lock**. Create `…/<name>/HEAD.lock` with `O_CREATE|O_EXCL`, retrying with bounded backoff; read HEAD; write `{checkpoint_digest, seq: prev+1, prev_digest}` by temp file + rename; remove the lock.
   - Leases are advisory, so two `pkg checkpoint` runs on one package can overlap. Without the lock, both would read seq 14 and both write 15, and the loser's digest would silently drop out of the history (round-2 Kimi objection).
   - With the lock, HEAD's `prev_digest` chain is a total order, and every published digest is on it (TST-PKGPAR-25).
   - A lock older than the checkpoint timeout is reported, never silently broken.
   - "Newest" is defined by HEAD and never by wall-clock time.
6. A red result at step 2 removes the temp dir and leaves HEAD unchanged.

*Reading a checkpoint.*
- A dependency marked `resolve = "checkpoint"` records the checkpoint digest in the dependent's lock, next to the content hash. It loads from `<checkpoint_digest>/` and never from the live directory. Today the lock records a content hash and warns on drift (V1); C2 adds pinned bytes and a pinned closure behind it.
- The checkpoint's pinned closure enters the dependent's flat lock as the transitive entries for that dependency. A clash with the dependent's other pins is the existing flat-resolution conflict error, never a silent pick.
- Before loading, the resolver **recomputes the stored directory's checkpoint digest**. A mismatch, a missing `CHECKPOINT.json` or a missing pinned dependency is **MOD018** (checkpoint corrupt), and nothing is compiled from it. A missing directory is **MOD017**, which names `ailang pkg checkpoint`. There is no silent fallback to live source.

*Seeing newer work.*
- `ailang lock --refresh-checkpoints` advances each checkpoint-resolved dependency to its HEAD.
- `--dry-run` prints, per dependency, the C4 class and the signature diff from the locked checkpoint to HEAD. A changed closure with unchanged signatures prints `deps changed` and is classed at least **U** (review), never folded into A.
- `ailang pkg checkpoints <pkg>` lists the history.
- **Checkpoints send no messages.** A send to a `pkg:` inbox *creates a coordinator task* (`ailang messages inboxes`: `DISPATCHES pkg:sunholo/ailang_parse … creates a task`, V22). A message per checkpoint would turn every save into an agent run. Checkpoint traffic therefore never touches the message store, the cascade topic or `classifyDispatchPath`. Agents poll with `--dry-run` at their own cadence.
- `upgrade-available` would not fit anyway: it requires `from_version`/`to_version` (`validateUpgradeAvailable`, `pkg_schema.go:166-177`), and checkpoints have none (D3).

*Publishing a package that has checkpoint-resolved dependencies.* Today `ailang publish` runs `RewritePathDeps`, which replaces each path dependency with the dependency's **local manifest version**. Its regex matches only a bare `{ path = "…" }`, so a `resolve` key would make publish fail with "unsupported manifest spelling" (V34). For `resolve = "checkpoint"` dependencies, the rule becomes:
- Publish pins the **registry version whose recorded content hash equals the locked checkpoint's content hash**. If several match, it takes the lowest such version, because the bytes are identical and determinism beats recency.
- If no registry version matches (the tested checkpoint is ahead of everything published), publish refuses with **PUB026**: "checkpointed dependency <pkg> has no registry version with content sha256:…; publish <pkg> first". Nothing is published, and there is no fallback to "latest" or to the local manifest version.
- This is the same rule that makes the ailang-parse acceptance run sound: publish `docparse_core` first, then `ailang_parse` (TST-PKGPAR-24).

*Version skew.* The manifest loader **silently accepts unknown keys and tables**: `resolve = "checkpoint"` and a made-up `[workspace-resolution]` table both lock without complaint today. The root package's `ailang = ">= …"` floor is not enforced locally either: a floor of 9.0.0 still locked and ran on 0.61.1 (V33). An older binary would therefore build a checkpoint-resolved dependency **against live source, silently**. Mitigation:
- Any lock containing a checkpoint entry is written as `schema: ailang.lock/v2`.
- C2 makes an unrecognised `resolve` value an error.
- Older binaries refuse a v2 lock (V35, measured): `check`, `run` and `lock` all exit 1 with `unsupported lock file schema: ailang.lock/v2`, and `lock` does not overwrite it. An older binary therefore cannot silently build a checkpoint-resolved dependency from live source while the v2 lock exists. The one remaining path is deleting the lock and re-locking with an older binary, which writes v1 and ignores `resolve`; the guide says so.

**C3: Package leases (coordination).**
- `ailang pkg claim <pkg> --agent ID [--ttl 2h]`, plus `release` and `claims`.
- Leases are advisory records, listable by both agents and the hub.
- **Ownership is the nearest enclosing `ailang.toml`.** A package's files are those under its directory minus every nested package root, so the root package of ailang-parse does not own `packages/docparse_core/`.
- The PreToolUse hook template refuses writes outside the claimed package's owned files. It is shipped as a documented hook for Claude Code and pi, not enforced inside AILANG.
- `ailang pkg commit` (D4) refuses on another agent's lease. Its pathspec is `-- <pkgdir> ':(exclude)<nested-root>'…`, one exclude per nested package, so a root-package commit cannot take a child's staged or unstaged edits. Without the excludes, `git commit -- .` at the root would commit them (round-2 Codex objection; V36).
- Today only messages, GitHub issues and mission attempts have claims (`ClaimMessage`, `ClaimIssue`, `ClaimMissionAttempt`); packages have none (V9).

**C4: Measured change classes (tooling).**
- `ailang publish` passes old signatures (registry metadata `InterfaceSignatures`) and new ones (the local v2 hasher) into `PackageVersionInfo.Signatures`. The existing `classifyChange` then returns real A/B/C instead of legacy C.
- `mapChangeClassToSchema` uses the same result instead of its module-count heuristic. Its first line, `if oldInfo.InterfaceHash == "" || oldInfo.InterfaceHash == newInfo.InterfaceHash { return "A" }`, is **removed whenever both sides carry signatures**: contract lines are deliberately kept out of the hash (D7), so equal hashes say nothing about contracts. Without this, every contract-only change would still be classed A (round-2 Carbon catch; V31; TST-PKGPAR-28). `classifyChange` in `pkg_events.go` has no such short-circuit; it compares signatures first.
- The envelope's existing `AffectedExports` field carries the symbol-level diff: removed and changed signatures.
- **Contracts are invisible to every interface identity today.** v2 signatures carry name, type, class constraints, literals and effects. `HashProjection` adds IFC labels. Neither carries `requires`/`ensures` (V24).
  - On signatures alone, tightening a `requires` with the type unchanged would be classed **A** and auto-applied deterministically. That is worse than today's conservative C.
  - C4 therefore adds to the signature set **one set-level marker line, `meta:contracts:v1`, in every post-D7 set**, plus a `mod:contract:fn:<normalized requires/ensures>` line for each function that has contracts. The marker records that the set was produced by a contract-aware hasher. The contract lines record what the contracts are. These two facts must not share a signal: if they did, a package's first `requires`, or the removal of its last `ensures`, would look like a format change (round-2 Carbon objection).
  - Classification rule: **any** difference in contract lines → C. Contract lines never count toward the superset (B) rule, because adding a `requires` is a tightening, not an addition.
  - If `meta:contracts:v1` is absent on **either** side, the pair is format-incomparable and classed **U**, regardless of the signature rule. This covers a pre-D7 package that drops all its contracts on its first post-D7 publish, which would otherwise read as A.
  - With the marker on both sides, any difference in `mod:contract:` lines → C, including 0 → 1 (first `requires`) and 1 → 0 (last `ensures` removed).
  - U is already review-only end to end. The coordinator's `ClassifyChange` maps `ChangeClass == "U"` to `ChangeClassC` before the kind switch ("Unknown never auto-applies, whatever the kind"), and `classifyDispatchPath` sends anything other than A/B to the AI path (V26, V7). The lowercase `classifyChange` (`pkg_events.go`) is the producer; it already emits U when one side has v2 signatures and the other doesn't.
  - Changing identity for every package with contracts is D7 (human).
- **PUB025**: `[release] kind` ∈ {fix, security, feature} with a measured class C → refuse (badge first, per D5). `breaking` with a measured class A → warning only.
- The same diff runs for C2 checkpoints, against the previous checkpoint.

**C5: Module moves as cascade migrations (tooling).** A move has two releases, with the facade in between.

*`ailang pkg move <module> --to <pkgdir>` (local, one command):*
- moves the file and rewrites its `module` line;
- updates `[exports]` in **both** manifests;
- **adds the target package as a dependency of the source package** (a path dependency in-repo, `resolve = "checkpoint"` if the source uses checkpoints), because the facade's `export import pkg/<target>/…` must resolve against a declared dependency;
- adds the `export import … @deprecated("<new path>")` facade at the old location (C1);
- rewrites imports inside the repo;
- appends a **persistent `[[migrations]]` entry** to the source manifest: `{from, to, symbols, to_package, since}`. The move survives in package metadata until the facade is removed, and is not reconstructed from git history.
- It refuses if the target directory is not a package.

*Release 1, the move* (source package, minor):
- The facade keeps every symbol at its old path and only adds a dependency, so the measured class is **B** and the cascade bump is deterministic. Dependents compile unchanged and see `TC_DEPRECATED_001` "moved" warnings.
- **Publish order is target first.** Publishing the source refuses with **PUB027** ("move target <pkg>@<ver> is not in the registry; publish it first") until the target version it depends on exists. With checkpoint dependencies, PUB026 applies the same way.
- A dependent's flat lock gains the target package as an ordinary transitive entry on `ailang lock`, with no manual edit (TST-PKGPAR-26).

*Release 2, facade removal* (source package, major; measured class **C**):
- `ailang publish` reads the `[[migrations]]` entries whose facades are gone and emits a `migration-request` whose new `import_rewrites: [{from, to, symbols, add_dependency: {name, constraint}}]` field comes from them.
- **Dispatch:** today `classifyDispatchPath` sends every C to AI (V7). It changes so that a **C-class `migration-request` with non-empty `import_rewrites` goes deterministic-first**: add the dependency if missing, rewrite imports, `ailang lock`, check, test. It falls back to AI only if a step fails. Every other C is unchanged (TST-PKGPAR-27).
- Mechanical moves then never consume an AI run, in either release.
- `[[migrations]]` is validated by the new loader. The old loader would silently ignore it (V33), which is harmless, because only the new publisher reads it.

**C6: Boundary hygiene (small fixes, shipped with Phase 1).**
- A directory with its own `ailang.toml` is excluded from the enclosing package's compile, discovery and tarball (V3).
- ~~`IMP` diagnostics for an `[exports]`-style path suggest the `pkg/` form~~. Dropped: the bare path resolves (V4 refuted).
- `ailang pkg docs` prints a copy-pasteable import line per exported module.
- Inline tests are discovered in package mode. This is **delegated** to the existing planned design [m-package-test-discovery.md](../m-package-test-discovery.md) (issue #959) rather than duplicated here; TST-PKGPAR-23 is its acceptance check from this doc's side.

### Implementation Plan

**Phase 0: Shared-checkout spike** (short run done 2026-11-25; the full 10-minute run is still to do, and it gates Phase 3)
- [x] Short run (90 s; V12). Setup: 3 concurrent `ailang run` loops and 1 `ailang lock` loop, all in the **same** package directory (worse than the design's one-agent-per-package), plus a writer changing the dependency's source 427 times. Results:
  - **Compile cache:** safe but lossy. 175 `CACHE_INVALID … reason=ARTIFACT_INVALID; recompiling`, all self-healed. No run ever printed wrong output.
  - **`ailang.lock`:** **unsafe.** 16 of ~9,700 runs failed with `failed to parse …/ailang.lock: unexpected end of JSON input`. `LockFile.Save` writes with a plain `os.WriteFile` (`internal/pkg/lockfile.go:87`), so a concurrent reader can see a truncated file. This is fixed in Phase 1.
- [ ] Full run: 10 minutes, with the loops in **sibling** packages (the design's real shape), after the atomic-lock fix. Expect 0 hard failures.

**Phase 1: Measured classes and boundary hygiene** (~3 days; done, Sprint 1, v0.64.0)
- [x] Atomic lock write: temp file in the same directory, then `os.Rename`. There is no shared helper in `internal/pkg`, and the four existing ones are all private to their own packages (V17), so add `pkg.writeFileAtomic` there and use it in `LockFile.Save` (TST-PKGPAR-14).
- [x] C4: thread signatures through the publish emitter; replace the module-count heuristic; fill `AffectedExports`.
- [x] C4: contract lines in `SignatureSet`, with the contract-aware classification rule (D7; TST-PKGPAR-19).
- [x] C4: PUB025 as a badge (D5).
- [x] C6: nested-manifest exclusion; `pkg docs` import lines. The IMP hint was dropped (V4 refuted). Package-mode inline test discovery (TST-PKGPAR-23) is not done: it is delegated to [m-package-test-discovery.md](../m-package-test-discovery.md) (#959).

**Phase 2: Re-exports** (~4 days; done, Sprint 2, merged in #2475: identity is kept by a load-time rewrite of importers to the origin)
- [x] C1: parser (import header loop), elaborator/import resolution (identity-preserving aliasing), interface hash v2 inclusion, MOD016, MOD003 removal.
- [x] C1: "moved" variant of `TC_DEPRECATED_001` (module-path replacement; applies to types and constructors) (V18).
- [x] C1: facade effect ceiling covers re-exported functions (TST-PKGPAR-16).
- [x] Engine parity: evaluator and strict VM.

**Phase 3: Checkpoints and leases** (~3–4 days; after Phase 0)
- [ ] C2: `pkg checkpoint` (snapshot → validate snapshot → measure → rename → HEAD), `resolve = "checkpoint"` with read-side hash verification, MOD017/MOD018, `lock --refresh-checkpoints [--dry-run]`, `pkg checkpoints`.
- [ ] Full Phase 0 run, extended with a checkpoint loop: dependents `check` in a loop while the dependency checkpoints 200 times (TST-PKGPAR-15).
- [ ] C3: claim/release/claims; nearest-`ailang.toml` ownership; hook template; `pkg commit` with nested-root excludes (if D4 is accepted).
- [ ] C2: HEAD lock (TST-PKGPAR-25); publish rule for checkpoint deps (PUB026, TST-PKGPAR-24); lock schema v2 for checkpoint entries (V35: older binaries refuse it).

**Phase 4: Moves** (~2–3 days; next, Sprint 4. Needs Phase 2 only; the checkpoint branches of C5 follow Phase 3)
- [ ] C5: `pkg move` (both manifests, target dependency, facade, `[[migrations]]`); PUB027 target-first gate; `import_rewrites` + `add_dependency` schema and validation; `classifyDispatchPath` deterministic-first for C-class migration-requests with rewrites (TST-PKGPAR-26, -27).

### Files to Modify/Create

**Modified:**
- `cmd/ailang/pkg_publish.go`: pass signatures; replace `mapChangeClassToSchema` logic, including removing its equal-hash short-circuit when signatures are present (~80 LOC); emit `import_rewrites` from `[[migrations]]`; PUB026/PUB027 (~90 LOC)
- `internal/pkg/publish_rewrite.go`: the checkpoint-dependency pin rule (content-hash match against registry metadata) (~60 LOC)
- `internal/pkg/manifest.go`: `resolve` values, `[[migrations]]`, rejecting unknown `resolve` values (~50 LOC)
- `internal/messaging/pkg_events.go`, `pkg_schema.go`: `AffectedExports` from the diff; `import_rewrites` on migration-request (~90 LOC)
- `internal/pkg/quality.go`, `changelog.go`: PUB025 (~50 LOC)
- `internal/parser/parser_file.go` (import header loop, `:67`), `parser_decl.go` (`EXPORT` branch, `:470-500`): `export import` (~60 LOC)
- `internal/loader/loader_errors.go` (IMP010 at `:34`) and its resolution callers, plus `internal/link/report.go` (IMP010 at `:37`): identity-preserving re-exports (~200 LOC) (V29)
- `internal/pkg/hasher_v2.go`: include re-exported signatures (~30 LOC)
- `internal/iface/json.go`, `hash_projection.go`: contract field and `mod:contract:` signature lines, kept out of `HashProjection` (~70 LOC)
- `internal/messaging/pkg_events.go`: contract-aware `classifyChange` (~40 LOC)
- `internal/pkg/resolver.go`, `lockfile.go`, `lockcheck.go`: checkpoint resolution (~180 LOC)
- `internal/pkg/discover.go`, `tarball.go`, `loader_layout.go`: nested-manifest exclusion (~40 LOC)
- `cmd/ailang/coordinator_cloud_cascade.go`: `classifyDispatchPath` admits C-class migration-requests with rewrites to the deterministic path; the deterministic rewrite step (~90 LOC)
- `internal/errors/codes.go`: MOD016, MOD017, MOD018, MOD019, PUB025, PUB026, PUB027; remove MOD003
- `internal/pipeline/warn_deprecated.go`, `warn_deprecated_ast.go`: "moved" variant for re-exports (~80 LOC)
- `internal/pkg/loader.go` (ceiling check, `effect ceiling violation` at `:303`): count re-exported functions (~40 LOC) (V29)
- `cmd/ailang/pkg_quality.go`: package-mode inline test discovery (C6; the source of the "inline test blocks are not yet discovered in package mode" note) (~60 LOC) (V30)
- Engine parity for re-exports: Phase 2 runs every C1 test on both the evaluator and the strict VM, by the existing dual-engine test convention. No engine-specific code is expected, because re-exports resolve before core lowering
- `cmd/ailang/pkg_docs.go`: print an import line per exported module (~20 LOC)
- `internal/pkg/publish_validator.go`: export `copyPackageContents`/`shouldStageFile` for reuse by checkpoints (~10 LOC)

**New:**
- `cmd/ailang/pkg_checkpoint.go` (~150 LOC), `cmd/ailang/pkg_claim.go` (~150 LOC), `cmd/ailang/pkg_move.go` (~250 LOC)
- `docs/docs/guides/parallel-agents-one-checkout.md`, plus the hook template

## Conflict Surface

(Required: Phase 2 touches `internal/parser/`.)

### Syntactic positions touched
1. The **file import header**: `parser_file.go:67` loops `for p.curTokenIs(lexer.IMPORT)`. A line starting with `export` currently ends that loop.
2. The top-level **`EXPORT` branch** in `parser_decl.go`. It accepts `FUNC`/`PURE`/`TYPE`, gives a dedicated error for `LET`, and otherwise gives `PAR_EXPORT_REQUIRES_FUNC`.
3. Error-recovery sync tokens: `parser_func.go:518` resyncs on `EXPORT` and `IMPORT`.

### What else lives here
- `export func`, `export pure func`, `export type` (the same EXPORT branch).
- `@annotation` + `export func`. Annotations precede `func`/`export func` (`PAR_ATTR_REQUIRES_FUNC`), so an annotation before `export import` must stay an error.
- `import M (names)` and `import M as A`, including the `(f as g)` aliasing forms. `export import` must accept exactly that grammar and nothing more.

### Disambiguation strategy
- In the header loop, treat `EXPORT` followed by `IMPORT` (one token of lookahead) as a re-export import, and keep looping.
- `EXPORT` followed by anything else ends the header, exactly as today. No current program contains `export import`, because it is a parse error today (V2), so nothing that parses now changes meaning.
- `@deprecated(...)` is accepted only as a trailing suffix on an `export import` line. Annotations before `export import` stay `PAR_ATTR_REQUIRES_FUNC`.

### Programs that MUST still work
- `examples/wasm_friendly_patterns.ail` (4× `export pure func`), `examples/progress_bar.ail` (`export pure func`), and the `export type` ADT declarations in `std/json.ail`, `std/sem.ail` and `std/ai.ail` (V28).
- `@deprecated` in the standard library (`std/array.ail`, `std/bytes.ail`) and its warning test, `internal/pipeline/warn_deprecated_test.go`.
- The deprecation fixtures behind `TC_DEPRECATED_001`.
- `internal/pipeline/match_alias_test.go` "re-exported alias, constructors matched by name". Re-exporting through `export type Hue = Color` must keep matching.
- `internal/pipeline/effect_ceiling_reachable_test.go` "sibling module via pkg/<self>/ wrapper (re-export shape)".

### What deliberately changes
- `export import` moves from a parse error to valid syntax.
- `PAR_EXPORT_REQUIRES_FUNC`'s message gains `import` in its list of accepted followers.
- MOD003 is removed.

## Examples

### Example 1: a move that breaks nobody

**Before** (today): moving `types/document` from `sunholo/ailang_parse` to `sunholo/docparse_core` breaks every `import pkg/sunholo/ailang_parse/types/document (Heading, …)` in three repos. The fix is hand edits, timed with the release.

**After:**
```bash
cd ailang-parse
ailang pkg move docparse/types/document --to packages/docparse_core
```
```ailang
-- docparse/types/document.ail, generated facade at the old path
module docparse/types/document
export import pkg/sunholo/docparse_core/types/document
  (Block, Heading, Text, Table, ParsedDocument) @deprecated("pkg/sunholo/docparse_core/types/document")
```
Dependents compile unchanged and get `TC_DEPRECATED_001` naming the new path. When the facade is later removed, `migration-request.import_rewrites` lets the cascade rewrite them deterministically.

### Example 2: three agents, one checkout
```bash
# agent A
ailang pkg claim sunholo/docparse_core --agent A --ttl 2h
# ... edits; type errors for 20 minutes ...
ailang pkg checkpoint          # snapshot validated → ~/.ailang/checkpoints/<checkout>/sunholo/docparse_core/<digest 9c1…>; HEAD seq 14
ailang pkg commit -m "core: add letterhead section option"

# agent B, the whole time, in ailang-parse root (dependency resolve = "checkpoint")
ailang check docparse/        # green, against A's last checkpoint, never A's live files
ailang lock --refresh-checkpoints --dry-run   # sunholo/docparse_core: seq 11 → 14, class B, +generateDocxSection(...)
ailang lock --refresh-checkpoints && ailang check docparse/
```

## Success Criteria

- [ ] An added function is classed B and dispatched deterministically; a removed one is classed C and the removed signature appears in `affected_exports` (TST-PKGPAR-01, -02, -03).
- [ ] PUB025 fires on `kind = "fix"` plus a measured C, and does not fire on `feature` plus a measured B (TST-PKGPAR-04).
- [ ] A nested `ailang.toml` is absent from the outer tarball and the outer compile (TST-PKGPAR-05).
- [ ] Re-exported constructors match and construct identically to the originals, on both engines (TST-PKGPAR-06, -09).
- [ ] A dependent with checkpoint resolution stays green while its dependency's live source does not type-check (TST-PKGPAR-07, -08).
- [ ] Checkpoints are atomic, verified on read, validated on the bytes they publish, and ordered by HEAD (TST-PKGPAR-15, -17, -18). They are keyed by their full inputs and pin their closure (TST-PKGPAR-20, -21, -22). A facade's ceiling covers what it re-exports (TST-PKGPAR-16).
- [ ] `pkg move` produces a facade with which dependents compile unchanged and warn (TST-PKGPAR-10, -11, -12).
- [ ] Concurrent lock reads never see a truncated file (TST-PKGPAR-14). The 10-minute run in sibling packages records 0 hard failures, and V12 is closed.
- [ ] `docs/docs/guides/parallel-agents-one-checkout.md` exists, and its examples run under the doc-example guard.

## Testing Strategy

| ID | Level | Given / When / Then |
|----|-------|---------------------|
| TST-PKGPAR-01 | unit | Given old and new v2 signature sets where new ⊃ old, when the publish emitter classifies, then the class is **B** and `breaking` is false. |
| TST-PKGPAR-02 | unit | Given a function removed from an exported module, when classified, then the class is **C** and `affected_exports` contains its old signature. |
| TST-PKGPAR-03 | integration | Given a class-B envelope, when `classifyDispatchPath` runs, then the path is deterministic (no AI dispatch). |
| TST-PKGPAR-04 | unit | `kind = "fix"` plus a measured C → PUB025. `kind = "breaking"` plus a measured A → warning, not a gate. |
| TST-PKGPAR-05 | integration | Given `pkg/` containing `packages/sub/ailang.toml`, when `ailang publish --dry-run` runs at the root, then the tarball file list has no `packages/sub/` entries and no MOD010 is raised for them. |
| TST-PKGPAR-06 | integration | Given `export import lib/color (Color, Red)` in a facade, when a dependent imports `Red` from the facade and matches it against `lib/color.Red`, then the match succeeds. (Today: IMP010.) |
| TST-PKGPAR-07 | integration | Given `resolve = "checkpoint"` and a green checkpoint, when the dependency's live source gains a type error, then the dependent's `check` and `run` succeed. (Today it fails; V1.) |
| TST-PKGPAR-08 | integration | Given no checkpoint for a checkpoint-resolved dependency → MOD017 naming `ailang pkg checkpoint`. A checkpoint attempt on red code writes nothing. |
| TST-PKGPAR-09 | unit | A re-export cycle → MOD016. Re-exporting a name the source doesn't export → the existing IMP010. |
| TST-PKGPAR-10 | e2e | `pkg move` of a module with an ADT → the old path still compiles for dependents, and `TC_DEPRECATED_001` names the new path. |
| TST-PKGPAR-11 | unit | A `migration-request` with malformed `import_rewrites` (empty `from`/`to`) is rejected by schema validation. |
| TST-PKGPAR-12 | integration | Given a migration-request with rewrites, the deterministic cascade step rewrites the imports, check passes, and no AI dispatch occurs. |
| TST-PKGPAR-13 | integration | Given a lease held by agent A, `ailang pkg commit --agent B` in that package refuses. With no lease, it commits only `<pkgdir>` and other staged paths stay staged. Given a root package with a nested `packages/child/ailang.toml`, a root edit, a staged child edit and an unstaged child edit, the root commit's diff holds only the root edit, and both child edits keep their staging state. |
| TST-PKGPAR-15 | integration | Given a checkpoint-resolved dependent running `ailang check` in a loop while the dependency checkpoints 200× (alternating valid edits), then 0 checks fail and none reads a partial store. Given a stored file altered after publish → MOD018, and nothing is compiled. |
| TST-PKGPAR-16 | integration | Given a facade with `[effects].max = []` that re-exports a function with `! {Net}`, then the facade's ceiling check fails and names the re-exported function. |
| TST-PKGPAR-17 | integration | Given `pkg checkpoint` started on green source, when a type error is written to the live tree after the snapshot is taken (test hook between steps 1 and 2), then the checkpoint succeeds and its bytes equal the snapshot, not the live tree. With the error written *before* the snapshot, nothing is published and HEAD is unchanged. |
| TST-PKGPAR-18 | unit | `lock --refresh-checkpoints` selects HEAD's `content_hash` regardless of directory mtimes or `CHECKPOINT.json` times; two refreshes with the same HEAD produce identical locks. |
| TST-PKGPAR-19 | unit | Both sides have `meta:contracts:v1`: same type with `requires` added → C (not A, not B); a function's last `ensures` removed → C; a package's first `requires` → C; identical contract lines → classified by the signature rule alone. Either side lacks the marker → U, including a pre-D7 package with contracts whose new version has none (no contract lines on either side), and even when the signature rule alone would say A or B. |
| TST-PKGPAR-20 | integration | Given a package with `{ path = "../core", resolve = "checkpoint" }`, `pkg checkpoint` validates the snapshot with `core` resolved only through its pinned checkpoint digest, never `../core` live. The checkpoint's recorded content hash equals the live package's hash at copy time. |
| TST-PKGPAR-21 | integration | Given a checkpoint, when only `ailang.toml` changes (dependency added, zero `.ail` edits), then `pkg checkpoint` publishes a new digest directory and HEAD advances once. `--dry-run` shows `deps changed`, classed U. Re-running with no change publishes nothing and HEAD does not move. |
| TST-PKGPAR-22 | integration | Given `app` → `core` → `base` with `core` checkpointed (its `base` pinned to a checkpoint) and `app` locked to it, when `base`'s live source gains a type error, then `app` stays green. Given `core` with a live-resolved path dependency, `pkg checkpoint` fails with MOD019 naming it. |
| TST-PKGPAR-23 | integration | Given a package whose only tests are inline `tests [...]` blocks, `ailang pkg quality --json .` counts them (non-zero `tests`), and the "not yet discovered" note is absent. |
| TST-PKGPAR-24 | integration | `ailang_parse` with `docparse_core` as `resolve = "checkpoint"`: (a) a registry version with matching content hash → publish pins exactly that version; (b) the checkpoint is ahead of the registry → PUB026, and nothing is published; (c) two registry versions match → the lowest is pinned; (d) a newer non-matching version exists → never chosen. |
| TST-PKGPAR-25 | integration | Two concurrent `pkg checkpoint` runs on one package with different edits → both digests are published, HEAD's `prev_digest` chain contains both in one total order, and `seq` has no duplicate. |
| TST-PKGPAR-26 | e2e | `pkg move` of a module with an ADT from A to B, then publish B, then publish A (minor). A dependent of A runs `ailang lock && ailang check` with no manual edits; its flat lock now contains B; it gets "moved" warnings. Publishing A before B → PUB027. |
| TST-PKGPAR-27 | unit | `classifyDispatchPath` sends a C-class `migration-request` with non-empty `import_rewrites` to the deterministic path, and an identical envelope without rewrites to AI. A deterministic rewrite whose check fails falls back to AI. |
| TST-PKGPAR-28 | unit | Old and new versions with equal `interface_hash_v2` but different `mod:contract:` lines (both carrying `meta:contracts:v1`) → `mapChangeClassToSchema` returns C, not A. |
| TST-PKGPAR-14 | integration | Given one goroutine calling `LockFile.Save` in a loop and four calling `LoadLockFile` on the same dir for 5 s, then 0 loads fail to parse. (Today: the 16/~9,700 rate in V12.) |

**Regression surface:** the Conflict Surface "must still work" list, each as an existing test run unchanged.

## Verification Log

All runs on `ailang v0.61.1` (dev, 2026-11-25), using scratch packages in the session scratchpad.

| # | Claim | How verified | Result |
|---|-------|--------------|--------|
| V1 | Path deps compile **live** sibling source. The lock records `content_hash`, warns on drift, and still compiles the drifted code. | Scratch `demo/core` and `demo/app` with a path dep; added a type error to an *unused* function in core; ran app | `Warning: dependency demo/core content changed (locked: sha256:f498…, current: sha256:a853…)`, then `Error: type error in pkg/demo/core/lib … int vs string`. Lock entry: `{content_hash, interface_hash, path, source:"path"}`, with no bytes stored. **Confirmed** |
| V2 | No re-exports. Constructors cannot be imported through a facade, and `export import` does not parse. | Scratch `lib/color`, plus facade `lib/facade` (`export type Hue = Color`) | `import lib/facade (Hue, Red, …)` → `IMP010: symbol 'Red' not exported by 'lib/facade'`. `export import lib/color (Color)` → `PAR_EXPORT_REQUIRES_FUNC`. Alias plus a direct constructor import works (prints `red`). **Confirmed** |
| V3 | A nested `ailang.toml` is swept into the outer package | Scratch root package plus `packages/sub/ailang.toml`; `ailang publish --dry-run` before and after (ailang-parse session, 2026-11-25) | MOD010 on the nested module paths; tarball went from 2 to 3 files and 560 to 685 bytes. **Confirmed** |
| V4 | `[exports]` names differ from the dependent import path | Original claim, from the ailang-parse session. **Re-checked 2026-11-25 (Sprint 1, M6):** scratch dependent of ailang-parse, `import docparse/services/markdown_parser (parseMarkdown)` runs and prints `1` on both v0.61.1 and dev; a missing module under the prefix gives `LDR001: module not found`. Bare imports resolve via `ModuleLoader.SetModulePrefixMap` (`internal/loader/loader.go:103-119`, commit `c59f8f916`), which tries each package sharing a prefix. Its doc comment's example (`sunholo/docparse` sharing `docparse`) is stale: `docparse/ailang.toml` has `module_prefix = "docparse_api"`, and `ailang pkg info sunholo/docparse` → not found | **Refuted.** Both forms resolve. The remaining gap is that the canonical `pkg/` form is not surfaced; `ailang pkg docs` now prints it (M6). The IMP hint was dropped |
| V5 | MOD003 is defined but never emitted | `grep -rn MOD003 internal --include='*.go'`, excluding tests and `codes.go` | No emitters. **Confirmed** |
| V6 | Publish classifies without signatures | Read `cmd/ailang/pkg_publish.go:368-379` (patch/minor by lockfile interface hash), `:436-446` (`depInfo` has no `Signatures`), `:512-534` (`mapChangeClassToSchema` comment: "assume breaking") | **Confirmed.** A signature-aware `classifyChange` exists (`internal/messaging/pkg_events.go:155`) but gets legacy input. The registry stores `InterfaceSignatures` (`cmd/registry-validator/main.go:354,397`) |
| V7 | Class C goes to AI dispatch, A/B to deterministic | Read `classifyDispatchPath` (`cmd/ailang/coordinator_cloud_cascade.go:46-53`) | **Confirmed** |
| V8 | `[release] kind` is checked only for presence | `grep -n '"breaking"\|Kind ==' internal/pkg/quality.go internal/pkg/changelog.go cmd/registry-validator/*.go` | Only `quality.go:329` (`Kind == ""`). The `changelog.go:13` comment anticipates a measured check, but no code exists. **Confirmed** |
| V9 | No package-scoped lease or claim | `grep -rn 'func.*Claim\|func.*Lease' internal/messaging internal/coordinator` | Only `ClaimMessage`, `ClaimIssue*`, `ClaimMissionAttempt`. **Confirmed** |
| V10 | Pathspec commits isolate a package; concurrent git commands collide | Scratch repo: staged `a/x`, modified `b/y`, ran `git commit -- b`; then with `.git/index.lock` present | The commit contained only `b/y`, and `a/x` stayed staged. The second command failed: `Unable to create '…/.git/index.lock': File exists`. **Confirmed** |
| V11 | `migration-request` has no import-rewrite payload | Read `PackageRef` fields and `validateMigrationRequest` (`internal/messaging/pkg_schema.go:71-100,220-225`) | Fields: versions, hashes, `change_class`, `effect_delta`, `breaking`, `affected_exports`, contracts, `lockfile_ref`, … No rewrite field. **Confirmed** |
| V12 | Compile cache and lock tolerate concurrent agents in one checkout | **Measured** in a 90 s spike: 3 `ailang run` loops plus 1 `ailang lock` loop in one package dir, while the dependency's source was changed 427 times (`scratchpad/spike/loop.sh`, `churn.sh`) | **Cache: confirmed safe but lossy.** 175 `CACHE_INVALID … recompiling`, 0 wrong outputs; artifacts are written by rename (`cache_artifacts.go:99`), and a corrupt manifest resets to empty (`cache_store.go:82-89`). **Lock: refuted.** 16 hard failures, `unexpected end of JSON input`, because `LockFile.Save` writes non-atomically (`lockfile.go:87`); the fix is in Phase 1. The 10-minute run in sibling packages is still PENDING |
| V13 | Coordinator isolates every task in a worktree, and defaults agents to 1 concurrent task | Read `internal/coordinator/worktree.go:126`, `agent_config.go:353-354` | **Confirmed** |
| V14 | Proposed codes are unallocated | `grep -rn` for MOD016, MOD017, PUB025 across `internal/` and `cmd/` (MOD015 is taken: `import_local_collision_test.go`) | None found. **Confirmed** |
| V15 | Test tag is free | `grep -rn "TST-PKGPAR"` | None found. **Confirmed** |
| V16 | Imports are parsed in a header loop, separate from declarations | Read `parser_file.go:67`; `parser_decl.go` EXPORT branch | **Confirmed.** This is why C1 must extend the header loop, not the EXPORT branch alone |
| V18 | `TC_DEPRECATED_001` fires in a dependent package, but assumes a same-module replacement | Scratch: `demo/core` has `@deprecated("greet") export func hello`; `demo/app` imports `hello`; ran `ailang check` and `ailang run` in app | `TC_DEPRECATED_001 at app/main.ail:4:43 — pkg/demo/core/lib.hello is deprecated; use greet. Fix: replace hello with greet (import it from pkg/demo/core/lib)`. **Confirmed** cross-package. The same-module fix text is why C1 adds a "moved" variant |
| V19 | `[release] kind` enum | Read `internal/pkg/changelog.go:21` | `ReleaseKinds = {security, fix, feature, breaking}`. **Confirmed**; PUB025's trigger set uses exactly these |
| V20 | Any `.ail` under a package dir, including `.ailang/`, changes its content hash | Scratch: copied `lib.ail` into `core/.ailang/checkpoints/x/`, then ran app | `Warning: dependency demo/core content changed (locked: sha256:e4d2…, current: sha256:2a2f…)`. Hasher walks every `*.ail` (`hasher.go`, `filepath.Walk` + `.ail` suffix). **Confirmed**; this fixes the store outside package dirs |
| V21 | The publish validator already snapshots before validating | Read `internal/pkg/publish_validator.go:77` (`stagePublishedPackage(packageDir, tmpDir)`), `:160-180` (`copyPackageContents`, `shouldStageFile`), `publish_stage.go:13-40` | **Confirmed.** Copy, then validate the copy. C2 reuses the copy step, not `RewritePathDeps` |
| V22 | A send to a `pkg:` inbox creates a coordinator task | `ailang messages inboxes` | `DISPATCHES pkg:sunholo/ailang_parse — creates a task for pkg-sunholo-ailang-parse on sunholo-data/ailang-parse`. **Confirmed**; checkpoints therefore send no messages |
| V23 | MOD018 is unallocated | `grep -rln '"MOD018"\|MOD018:' internal cmd` | None found. **Confirmed** |
| V24 | Contracts are in neither v2 signatures nor the interface hash projection | Read `internal/iface/hash_projection.go:109-144` (`SignatureSet`: `mod:func:name:type[:constraints][literal]:effects`, `mod:type`, `mod:ctor`), `:23-75` (`HashProjection`: types, funcs, effects, IFC labels; tier/deprecation stripped), `internal/iface/json.go:37-60` (`FuncJSON` has no contract field); `grep -n -i 'contract\|requires\|ensures'` over both files and `internal/pkg/hasher*.go` | No contract field anywhere. **Confirmed**; this drives D7 |
| V25 | The content hash covers `.ail` files only, not `ailang.toml` | Read `internal/pkg/hasher.go` content-hash walk (`.ail` suffix, named-test bodies excluded) | **Confirmed**, so the snapshot's manifest path rewrite leaves the content hash unchanged |
| V26 | Class U never auto-applies | Read `internal/coordinator/autonomy_router.go:51-70`: `if env.Package.ChangeClass == "U" { return ChangeClassC }`, placed before the kind switch, with the comment "Unknown never auto-applies, whatever the kind". Read `classifyDispatchPath` (V7): only A/B are deterministic. Producer: `internal/messaging/pkg_events.go:155-167` returns U when exactly one side has signatures | **Confirmed** |
| V27 | MOD019 is unallocated | `grep -rln '"MOD019"\|MOD019:' internal cmd` | None found. **Confirmed** |
| V28 | The "must still work" fixtures exist | `grep -c '^export pure func' examples/wasm_friendly_patterns.ail examples/progress_bar.ail` → 4, 1; `grep -ln '^export type [A-Z]… = …\|' std/*.ail` → `std/ai.ail`, `std/sem.ail`, `std/json.ail`; `grep -rln '@deprecated(' std/*.ail` → `std/array.ail`, `std/bytes.ail` | **Confirmed** |
| V29 | IMP010 and ceiling-check sites | `grep -rn '"IMP010"' internal` → `internal/loader/loader_errors.go:34`, `internal/link/report.go:37`; `grep -rln 'effect ceiling violation' internal` → `internal/pkg/loader.go:303` | **Confirmed** |
| V30 | Inline tests are not discovered in package mode | `ailang publish --dry-run` on scratch `demo/core` printed "no *_test.ail files; inline test blocks are not yet discovered in package mode (m-package-test-discovery)"; the note's source is `cmd/ailang/pkg_quality.go` | **Confirmed** |
| V31 | `mapChangeClassToSchema` short-circuits to A on equal interface hashes | Read `cmd/ailang/pkg_publish.go:512-515`: `if oldInfo.InterfaceHash == "" \|\| oldInfo.InterfaceHash == newInfo.InterfaceHash { return "A" }`. `classifyChange` (`pkg_events.go:155-175`) checks signatures before anything else | **Confirmed.** It must go when signatures are present (C4) |
| V32 | The `ReleaseGateHardFrom` grace precedent exists | `internal/pkg/changelog.go:23-26` (`ReleaseGateHardFrom = "0.41.0"`, PUB001/PUB002 block from it), used at `:111` and `quality.go:77` | **Confirmed**; D5 follows it |
| V33 | The manifest loader silently accepts unknown keys and tables; the root `ailang` floor is not enforced locally | Scratch `demo/app`: added `resolve = "checkpoint"` to the dependency → `✓ Generated ailang.lock`; appended `[workspace-resolution]` → same; set `ailang = ">= 9.0.0"` → lock and `run` both succeeded on v0.61.1 | **Confirmed.** This drives the version-skew mitigation |
| V34 | Publish pins path deps to the local manifest version and matches only `{ path = "…" }` | Read `internal/pkg/publish_rewrite.go:20-55` (regex `"<name>"\s*=\s*\{\s*path\s*=\s*"<path>"\s*\}`; replacement is `local.Package.Version`; mismatch → "unsupported manifest spelling"); called from `cmd/ailang/pkg_publish.go:195` | **Confirmed** |
| V35 | Older binaries refuse an `ailang.lock/v2` lock | **Measured** 2026-11-25: v0.61.1 built from its tag, path-dependency fixture. With the v1 lock, `check` passes and `run` prints the dependency's output (positive control). With `schema` edited to `ailang.lock/v2`: `check`, `run` and `lock` all exit 1 with `unsupported lock file schema: ailang.lock/v2 (expected ailang.lock/v1)`; `lock` leaves the file untouched. Cause: `schema.Accepts` (`internal/schema/registry.go:24`) accepts only `v1` and `v1.x` | Confirmed |
| V36 | A pathspec exclude keeps a nested package out of a root commit | Scratch repo: root edit, staged `packages/child/c.ail`, unstaged `packages/child/d.ail`; `git commit -- . ':(exclude)packages/child'` | Commit contained only `root.ail`; afterwards `M  packages/child/c.ail` (still staged) and ` M packages/child/d.ail` (still unstaged). **Confirmed** |
| V37 | PUB026 and PUB027 are unallocated | `grep -rln '"PUB026"\|PUB026:'` (and the same for PUB027) over `internal` and `cmd` | None found. **Confirmed** |
| V17 | No shared atomic-write helper is reachable from `internal/pkg` | `grep -rn '^func [A-Za-z]*[Aa]tomic[A-Za-z]*(' internal cmd` | Only package-private copies: `eval_analysis/dashboard_io.go:154`, `mission/render.go:307`, `mission/activation/files.go:189`, `cmd/ailang/fmt.go:261`. **Confirmed.** Consolidating them is out of scope |

## Deferred Decisions

- **Whole-module re-export** (`export import M`, with no list). Deferred until explicit lists have been used in practice. The human decides; the default is no.
- **Checkpoint garbage collection** (keep the last N per package, or anything referenced by a lock). The agent may choose.
- **Lease store** (D6). The agent may choose.
- **PUB025 grace length** (D5). The agent may choose, following the `ReleaseGateHardFrom` pattern.

## Non-Goals

- **Merging divergent edits to the same file.** That is what worktrees and branches are for. This doc removes the need for them only for package-scoped work.
- **Multiple versions of one package in a single build.** Resolution stays flat. Re-exports and moves reduce the pressure for it.
- **Installing only what a dependent reaches.** Effect checks are already reachability-based (`effect_ceiling_reachable_test.go`); install-time pruning is a separate design (see Future Work).
- **Push, PR or branch automation in `pkg commit`.** The cascade wrapper already owns that.

## Timeline

- **Week 1:** Phase 0 spike (0.5d); Phase 1 (3d); start Phase 2.
- **Week 2:** Finish Phase 2 (4d total); Phase 3 (3–4d).
- **Week 3:** Phase 4 (2–3d); guide and doc-example guard; ailang-parse split as the acceptance run.

**Total: ~11–14 days plus the spike.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Shared on-disk state corrupts under concurrency | High (removes "no worktrees" for builds) | Short spike (V12): the cache self-heals, but the lockfile does not (non-atomic write); fixed in Phase 1. The full run gates Phase 3. If anything else turns up, fall back to per-agent cache dirs |
| Re-export identity breaks type equality across facades | High | TST-PKGPAR-06, -09 on both engines; follow the existing alias-match path that already holds (`match_alias_test.go:84`) |
| Signature-based classing auto-applies a contract tightening inside an unchanged type | High | Contracts are in neither the signatures nor the hash projection (V24). C4 adds contract lines (D7); any contract difference → C; a pair missing contract lines on one side → U (TST-PKGPAR-19). If D7 is declined, signatures cannot see contracts. C4 must then **not** use signature classing for any package whose exports carry contracts; those packages keep today's legacy C for every interface change. Signature classing (A/B) applies only to contract-free packages |
| Leases are advisory and an agent ignores them | Med | Hook template refuses writes; `pkg commit` refuses; `claims` makes violations visible. Not a security boundary, and the doc does not claim it is |
| Checkpoint-resolved dependents lag behind live work | Low | `lock --refresh-checkpoints --dry-run` shows what is newer, with class and diff; refreshing is one command; live mode stays the default (D2). Push notification is deliberately absent (V22) |
| Version skew: an older AILANG silently ignores `resolve = "checkpoint"` and builds live | High | Lock schema v2 for checkpoint entries; unknown `resolve` values are an error from this version on; older binaries refuse v2 (V35, measured); the guide warns that deleting the lock and re-locking with an older binary drops the pins |
| Checkpoint store grows without bound | Low | GC is a deferred decision; content-addressed dirs make "keep HEAD plus anything a lock references" safe to compute |

## Quorum

Trigger 1 fires (design freeze D1–D4, D7). Triggers 2–4 do not:
- Resolution is opt-in, so shared machinery is reused, not overridden.
- No cost or banking schema is touched.
- The only external premise is git behaviour, which is verified in V10.

The user approved the quorum on 2026-11-25. A short concurrency run (V12) went first so that the main premise was measured rather than assumed. The 10-minute run in sibling packages is labelled PENDING; it does not hold up review.

**Round 0 (2026-11-25): BLOCKED, 3 of 3 reject** (oc-glm-5-3, oc-kimi-k3, codex:gpt-6.2-sol; artifact `.ailang/state/mission-quorum/m-pkg-parallel-agents-2026-11-25T14-15-54Z.json`). All three objections hit C2, and all were accepted:
- **GLM: the checkpoint store had no atomicity or read-side verification.** Now: temp dir + rename, HEAD written by rename, hash recomputed on read → MOD018 (TST-PKGPAR-15). Also accepted:
  - V18: cross-package deprecation verified; the fix-text gap is now budgeted.
  - V19: release-kind enum.
  - The facade effect-ceiling check (TST-PKGPAR-16).
  - `pkg_docs.go` added to Files to Modify.
  - Deterministic HEAD (TST-PKGPAR-18).
- **Kimi: checkpoint → `upgrade-available` was unverified and could not validate (no versions).** Checking the inbox table showed it was worse: a `pkg:` send dispatches a task (V22). Checkpoints now send no messages; polling is via `--dry-run`. The contradictory store location is now fixed outside package dirs (V20).
- **Codex: race between validating and copying.** Checkpoints now snapshot first, then validate, hash and publish the snapshot. This reuses the publish validator's copy step (V21; TST-PKGPAR-17).

**Author sweep before round 1.** Every load-bearing claim was re-read, which found two more gaps:
- Contracts are invisible to v2 signatures, so signature-only C4 would have auto-applied contract tightenings. The old risk row said the opposite. Fixed by V24, D7 and TST-PKGPAR-19.
- A snapshot's relative path dependencies would dangle from the store. Fixed by V25 and TST-PKGPAR-20.

**Round 1 (2026-11-25): BLOCKED, 3 of 3 reject** (same seats; artifact `.ailang/state/mission-quorum/m-pkg-parallel-agents-2026-11-25T14-21-40Z.json`). Objections accepted and folded in **after** the round. The guardrail is spent, so they have not been re-reviewed:
- **Kimi and Codex: a checkpoint keyed by `.ail` content hash ignores the manifest and the dependency closure.** A manifest-only change was discarded as a duplicate. Path dependencies inside a snapshot still resolved live, so a transitive edit could break a locked dependent. Now:
  - Checkpoints are keyed by a digest over files, canonical manifest and pinned closure.
  - A live path dependency is refused with MOD019.
  - A duplicate never advances HEAD.
  - A changed closure reports `deps changed` (U).
  - Tests: TST-PKGPAR-20, -21, -22.
- **GLM:**
  - "U is review-only" lacked a log row. Now V26, quoting `autonomy_router.go`.
  - The fallback if D7 is declined left contract tightenings classed A. Now packages with contracts stay on legacy C if D7 is declined.
  - The C4 rule contradicted TST-PKGPAR-19. Now a pair that lacks contract lines on one side is format-incomparable → U.
  - D7 was missing from the trigger-1 list. Fixed.

**Handover.** Every remaining objection was answerable in-session and has been answered, but none of the fixes has been re-reviewed. Two ways forward:
- (a) The human reviews the C2 identity rules and the D7 transition directly while ratifying D1–D4 and D7.
- (b) The human explicitly authorises a round 2.

Both C2 rounds found real defects, so (a) should focus on C2's identity and closure rules.

**Single review, gemini-4-carbon (2026-11-25, after round 1; user-requested addition to the pool).** Reject, $0.082, 16.3K tokens in and 0.8K out. Accepted and folded in:
- Contract-line presence was doing double duty. Now there is a `meta:contracts:v1` marker, and D7's cost is restated for every package (TST-PKGPAR-19).
- The checkout ID was derived inside the snapshot. Now it is computed once from the source package, and pinned store paths are recorded.
- The Conflict Surface placeholder is replaced by real fixtures (V28).
- Unnamed modification sites are now named (V29, V30), and test discovery has TST-PKGPAR-23.

**Round 2 (2026-11-25, user-authorised; new pool: codex:gpt-6.2-astra, gemini-4-carbon, oc-glm-5-3, oc-kimi-k3, claude-fable-5-2@claude-p benched as author vendor; `--seats 4` so the Google seat sat): BLOCKED, GLM pass and 3 reject** (artifact `.ailang/state/mission-quorum/m-pkg-parallel-agents-2026-11-25T14-43-10Z.json`). C2's core drew no objection. Accepted and folded in, not yet re-reviewed:
- **Kimi:**
  - Publishing with checkpoint dependencies was unspecified. Now there is a content-hash pin rule plus PUB026 (V34; TST-PKGPAR-24).
  - Concurrent HEAD writes could lose a digest. Now HEAD is written under an exclusive lock with a `prev_digest` chain (TST-PKGPAR-25).
  - Manifest tolerance of unknown keys was unverified. It is verified now, and it is worse than assumed: unknown keys are silently ignored (V33). This adds the version-skew risk; V35 has since measured that older binaries refuse a v2 lock.
- **Codex Astra:** a root commit swept in nested packages. Now ownership is the nearest `ailang.toml`, and commits use pathspec excludes (V36; TST-PKGPAR-13 extended).
- **Carbon:**
  - C5 was inconsistent end to end. It is now redesigned as two releases: the move releases as B with the target dependency added and PUB027 enforcing target-first publish; facade removal releases as C with `[[migrations]]`-sourced `import_rewrites` and deterministic-first dispatch for C-class migration-requests (TST-PKGPAR-26, -27).
  - Catch: the equal-hash short-circuit in `mapChangeClassToSchema` would mask contract changes (V31; TST-PKGPAR-28).
- **GLM (pass):** C5's dependency side, which overlaps Carbon's point and is fixed by the same redesign. V32 records the D5 precedent.

## Related Documents

Neural search on "pkg parallel agents": all scores are below 0.5. No duplicate exists, and each overlap below is distinct.

**Implemented (this doc completes them):**
- [M-PKG](../../implemented/v0_9_5/m-pkg-package-system.md): source of the ownership and parallelism promises and the Change Classification table. Its Workspace Model covers sibling packages only, not nested ones.
- [M-PKG-MSG](../../implemented/v0_9_9/m-pkg-msg-package-messaging-graph.md): the 11 message kinds. This doc adds one field (`import_rewrites`) and no kinds.
- [M-PKG-AUTONOMOUS-UPDATES](../../implemented/v0_10_0/m-pkg-autonomous-updates.md) (0.48): cascade, with worktree isolation cited under A6. This doc is the alternative for package-scoped work.
- [M-PKG-AUTONOMOUS-CASCADE-SAFE](../../implemented/v0_16_0/m-pkg-autonomous-cascade-safe.md), [M-PKG-CASCADE-DETERMINISTIC-FIRST](../../implemented/v0_16_0/m-pkg-cascade-deterministic-first.md): the deterministic dispatch that C4 and C5 feed.
- [M-PKG-SMOKE-STAGED-LOCK](../../implemented/v0_56_0/m-pkg-smoke-staged-lock.md): "staged" already means the published view for smoke tests, which is why this doc says "checkpoint".

**Planned (checked for overlap):**
- [M-PKG-INFLIGHT](../v1_1_0/m-pkg-inflight.md) (0.44): registry-side visibility of in-progress publishes. That is complementary: a checkpoint is local, and in-flight is cloud-side.
- [M-RESIDENT-AGENT-INSTANCES](../m-resident-agent-instances.md) (0.42): where agents run, not how they share a codebase.
- [M-CONCURRENCY-LEVERAGE](../v0_29_0/m-concurrency-leverage.md) (0.41): runtime `Fork()` parallelism, which is unrelated.

## References

- [Design Axioms](/docs/references/axioms)
- ailang-parse `design_docs/planned/v0_62_0/v0_62_0_docparse_core_split.md`: the field case, with boundary analysis and three downstream repos

## Future Work

- **Install only what a dependent reaches.** If an install fetched only the modules a dependent reaches (and their dependencies), "split a package to shed dependencies" would mostly disappear. This is the other answer to the specstore request.
- **Per-module effect ceilings** in `[exports]`, so a single package can advertise an FS-only surface.
- **Coordinator integration:** schedule Class-A/B work on different leased packages into one shared checkout instead of N worktrees, once Phase 3 and the Phase 0 spike have landed.

---

**Document created**: 2026-11-25
**Last updated**: 2026-11-25
