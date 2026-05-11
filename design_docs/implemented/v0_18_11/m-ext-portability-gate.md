# M-EXT-PORTABILITY-GATE: pre-publish durability gate + asset bundling for extension packages

**Status**: IMPLEMENTED
**Target**: v0.18.11 (multi-sprint — has 3 milestones with strict dependencies)
**Priority**: P0 (broken extensions are shipping to the registry today; bug class compounds across every consumer project)
**Estimated**: ~5-7 days (~1200 LOC across AILANG core + ailang-packages + motoko_agent + the registry validator service)
**Dependencies**: ✅ M-MATCH-ADT-XCHECK (v0.18.10) closed the typechecker side; this closes the publish + runtime sides.
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-11
**Source**: motoko_explore bug reports 2026-05-11:
- [msg_20260511_171403_a17c208e](mailto:msg) — `motoko_ext_omnigraph` registers unconditionally; every Omnigraph* tool call requires `${workdir}/omnigraph/` with full schema layout. 90% of consumer projects don't have it. Silent failures with no recovery path.
- [msg_20260511_171353_90c8be04](mailto:msg) — `motoko_ext_mcp` hardcodes `${workdir}/scripts/mcp-call.mjs`. Fresh workdirs hit `Cannot find module` on first ExaSearch call. Affects every MCP-backed extension.
- [msg_20260511_171544_22fb9b81](mailto:msg) — refinement: bridge should be bundled in package, not borrowed from consumer.
- [msg_20260511_171552_b474396c](mailto:msg) — refinement: extensions should self-detect-and-skip when their workdir state is absent.

---

## Problem statement

Extensions in the AILANG registry are passing `ailang publish` (and the v0.18.10 `_smoke.ail` runtime smoke check we shipped earlier today) yet still ship broken behavior because **registration and tool execution have different requirements**:

```
register_with_config(cfg)  →  ExtensionHooks         ← passes _smoke.ail
                                       │
                                       ↓
        consumer agent invokes provided_tool  ← CRASHES on first use
```

Concrete failure modes from production this week:

| Extension | Registration | First tool call |
|---|---|---|
| `motoko_ext_omnigraph@0.2.0` | ✅ returns hooks | ❌ `cd: ${workdir}/omnigraph: No such file or directory` |
| `motoko_ext_mcp@0.2.0` | ✅ returns hooks | ❌ `Cannot find module '${workdir}/scripts/mcp-call.mjs'` |
| `motoko_ext_compaction_ai@0.1.1` | ✅ returns hooks | ❌ runtime "no pattern matched" (fixed in 0.1.2) |

The compaction_ai bug was caught at the typechecker level by M-MATCH-ADT-XCHECK (v0.18.10). The omnigraph + mcp bugs slip through because:
1. They depend on **filesystem state in the consumer's workdir**, which the smoke test (run in the package dir) has access to via the package author's machine.
2. AILANG packages **can't bundle non-`.ail` assets** (see `internal/pkg/tarball.go:49` — only `*.ail` + `ailang.toml` + `AGENT.md` go in the tarball), so packages that need shell scripts / schemas / templates HAVE to leak the dependency to the consumer.

These are TWO problems with one shared cause: **the publish gate isn't strict enough about what self-containedness means**. Fix is layered:

1. **Make `ailang publish` reject packages whose tools crash in an isolated workdir** (no consumer-state assumed).
2. **Let packages bundle their own assets** so they CAN be self-contained instead of borrowing from the consumer.
3. **Give extension authors the runtime API to detect-and-skip** when consumer state legitimately is required (e.g. omnigraph genuinely needs a schema; the right behavior is "advertise nothing if absent").

## Goals

1. **Pre-publish durability gate**: `ailang publish` runs each extension's smoke test in a temp-dir with no workdir state. If any provided_tool crashes, publish is rejected.
2. **Package asset bundling**: AILANG packages can ship arbitrary files in an `assets/` subdirectory, accessible at runtime via a new `std/package.assetPath(pkgName, relPath)` API. Tarball includes them.
3. **Workdir-state detection API**: stdlib helper `std/extension.requireWorkdirFile(path) -> Result[(), ExtensionDisabled]` so register.ail can cleanly skip when consumer state is absent (returns hooks with `provided_tools: []`).
4. **Update affected packages** to use the new APIs: `motoko_ext_omnigraph` self-disables when workdir lacks `omnigraph.yaml`; `motoko_ext_mcp` bundles `mcp-call.mjs` via package assets.
5. **Backward compatibility**: existing packages without smoke tests / asset usage continue to publish (with a warning), but new publishes are encouraged into the new pattern.

## Non-goals

- **Static analysis of tool call paths** — we're using runtime smoke testing, not `go vet`-style flow analysis. Smoke tests are sufficient and don't require typechecker changes.
- **Sandboxed tool execution at runtime** — that's the existing capability/effect system; this design is about catching bugs at publish time, not changing runtime semantics.
- **Backward-incompatible package format change** — the new tarball format is a strict superset (assets/ is optional).
- **AILANG-side LSP integration for the new error codes** — surface errors via existing typechecker diagnostic plumbing.
- **Force-migrating existing packages** — they keep working; new patterns are opt-in until 1.0.

## Conflict surface

Touches:
- `internal/pkg/tarball.go` — extend the file-include filter to also walk `assets/`. Risk: existing tarball-hash determinism. Mitigation: assets sorted lexically + zero ModTime, same as today.
- `cmd/ailang/pkg_publish.go` — add the durability gate as a step BEFORE upload. Risk: backward compat — existing packages without `_smoke.ail` shouldn't fail publish. Mitigation: warn-only if no smoke; require smoke for packages with `[extension]` block in ailang.toml (new opt-in marker).
- `internal/pkg/publish_validator.go` (NEW) — the gate logic itself. Process-isolated `ailang run` invocations, captures exit code + stderr.
- `cmd/registry-validator/main.go` — server-side validator could also enforce the gate (defense-in-depth). Risk: longer publish latency. Mitigation: cap smoke-test wall time at 30s per package.
- `std/package.ail` (NEW or extend existing) — `assetPath` API. Risk: stdlib API surface. Mitigation: keep API minimal (one function), document opt-in for v1.0.
- `std/extension.ail` (NEW) — `requireWorkdirFile` helper. Same.
- `internal/effects/fs.go` — may need a way for assetPath to resolve to package-cache paths. Minimal change.

**Programs that MUST still work post-change** (regression fixtures):
1. Existing motoko-ext-* packages with no `_smoke.ail` — publish succeeds with a warning.
2. Existing motoko-ext-* packages WITH `_smoke.ail` (compaction_ai 0.1.2) — publish succeeds (smoke passes in temp-dir).
3. `motoko_ext_omnigraph` after self-disable wiring — registers with empty `provided_tools` in temp-dir → smoke passes (no tools to fail).
4. `motoko_ext_mcp` after asset bundling — `assetPath("sunholo/motoko_ext_mcp", "mcp-call.mjs")` resolves to the cached package's asset; smoke passes.

## Solution sketch

Three coordinated milestones, executed in order (each unblocks the next):

### M1 — Asset bundling (~250 LOC AILANG + tests)

**Tarball format extension**:
```
my-pkg/
├── ailang.toml
├── register.ail
├── exec.ail
└── assets/         ← NEW: arbitrary files included in tarball
    ├── mcp-call.mjs
    └── starter-schema.yaml
```

`internal/pkg/tarball.go` change:
```go
// Include: ailang.toml, *.ail, AGENT.md, assets/**
if rel == ManifestFile || strings.HasSuffix(rel, ".ail") || rel == "AGENT.md" {
    files = append(files, rel)
}
if strings.HasPrefix(rel, "assets/") {
    files = append(files, rel)
}
```

**New stdlib API** (`std/package.ail`):
```ailang
module std/package

import std/result (Result, Ok, Err)
import std/fs (fileExists)

-- assetPath returns the absolute filesystem path to an asset file
-- shipped inside an installed package. Resolves via the package cache
-- (~/.ailang/cache/registry/<pkg>/<version>/assets/<rel>).
-- Returns Err if package not installed or asset not present.
export func assetPath(pkgName: string, rel: string) -> Result[string, string] ! {FS} =
  _pkg_asset_path(pkgName, rel)
```

`_pkg_asset_path` builtin in `internal/builtins/pkg.go` resolves to the package cache directory.

**ailang.toml `[assets]` declaration** (optional, for future tooling like docs generation):
```toml
[assets]
files = ["mcp-call.mjs", "starter-schema.yaml"]
```

If `[assets]` is declared but a listed file is missing from the package dir, `ailang publish` rejects.

### M2 — Pre-publish durability gate (~400 LOC AILANG + tests)

**New `_smoke.ail` template** (already shipped a basic version in `motoko-ext-compaction-ai`; this extends it):
```ailang
module sunholo/motoko_ext_X/_smoke

import std/io (println)
import pkg/sunholo/motoko_ext_X/register (register_with_config)
import pkg/sunholo/motoko_ext_abi/types (ExtCtx, ToolCallEnvelope)

export func main() -> () ! {Env, FS, IO, Process, AI, Net, SharedMem, Clock, Stream} {
  -- Step 1: register
  let hooks = register_with_config(0);
  println("OK: register_with_config returned");

  -- Step 2 (NEW): exercise each provided_tool with synthetic input.
  -- Tools are expected to return a result (Handled, Delegate, etc.)
  -- WITHOUT crashing, even when the workdir is empty.
  let ctx: ExtCtx = mk_synthetic_ctx();
  for_each(hooks.provided_tools, \tool_name -> {
    let envelope: ToolCallEnvelope = {
      id: "smoke", tool: tool_name,
      arguments: synthetic_args_for(tool_name)
    };
    -- on_tool_handle should NOT crash; either Handled (tool ran) or
    -- Delegate (tool defers) is fine.
    let _ = hooks.on_tool_handle(ctx, envelope);
    println("OK: tool ${tool_name} dispatched without crash")
  });
  println("OK: all smoke checks passed")
}
```

**`ailang publish` change** (`cmd/ailang/pkg_publish.go`):
1. Before upload, look for `_smoke.ail` in the package dir.
2. If present: copy package to a temp-dir (no inherited workdir state), `cd temp-dir`, run `ailang run --caps ... --ai-stub --entry main _smoke.ail` with a 30-second timeout.
3. If smoke fails: `publish` aborts with the smoke output.
4. If no smoke: warn ("package has no _smoke.ail; recommended for v1.0+") but allow publish.
5. If `[extension]` block in ailang.toml: require smoke (extensions are higher-stakes).

**Server-side validator** (`cmd/registry-validator/main.go`): runs the same smoke check inside a sandboxed container as defense-in-depth. Same 30s timeout. Soft-launch behind a feature flag.

### M3 — `requireWorkdirFile` API + omnigraph + mcp updates (~200 LOC)

**New stdlib helper** (`std/extension.ail`):
```ailang
module std/extension

import std/result (Result, Ok, Err)
import std/fs (fileExists)

-- requireWorkdirFile returns Ok if the file exists at the workdir-relative
-- path, or Err with a "this extension is disabled" message that
-- register_with_config should surface as empty provided_tools. Used by
-- extensions whose tools require consumer-side state (e.g. omnigraph
-- needs a schema; mcp needs an MCP server config).
export func requireWorkdirFile(workdir: string, rel: string) -> Result[(), string] ! {FS} {
  let path = "${workdir}/${rel}";
  if fileExists(path) then Ok(())
  else Err("required workdir file not found: ${rel}")
}
```

**`motoko_ext_omnigraph@0.2.1` update** (per refinement msg b474396c, tier 1):
```ailang
export func register_with_config(_cfg: a) -> ExtensionHooks ! {Env, FS} {
  let workdir = getEnvOr("MOTOKO_WORKDIR", ".");
  match requireWorkdirFile(workdir, "omnigraph/omnigraph.yaml") {
    Err(_) => disabled_hooks("omnigraph"),  -- empty provided_tools
    Ok(_) => active_hooks(workdir, ...)
  }
}
```

**`motoko_ext_mcp@0.2.1` update** (per refinement msg 22fb9b81):
```ailang
export func register_with_config(_cfg: a) -> ExtensionHooks ! {Env, FS} {
  match assetPath("sunholo/motoko_ext_mcp", "mcp-call.mjs") {
    Err(_) => disabled_hooks("mcp"),
    Ok(bridge_path) => active_hooks(bridge_path, ...)
  }
}
```

`mcp-call.mjs` ships in the package's `assets/` subdirectory.

### Acceptance

Per-milestone:

**M1 (asset bundling)**:
- [ ] `ailang publish --dry-run` for a package with `assets/foo.txt` includes `assets/foo.txt` in the tarball.
- [ ] `assetPath("pkg", "foo.txt")` resolves correctly when package is installed.
- [ ] Existing packages without `assets/` continue to publish unchanged (tarball hash stable).
- [ ] New unit tests in `internal/pkg/tarball_test.go` cover assets/ include logic.
- [ ] Stdlib doc page for `std/package.assetPath`.

**M2 (publish gate)**:
- [ ] `ailang publish` for a package with broken `_smoke.ail` fails BEFORE upload, with the smoke output included in the error.
- [ ] `ailang publish` for a package without `_smoke.ail` succeeds with a warning.
- [ ] Smoke runs in `os.MkdirTemp` (no consumer-state contamination).
- [ ] 30-second timeout enforced; timeout failure surfaces as "smoke timeout, likely a hang".
- [ ] Regression: motoko_ext_compaction_ai@0.1.2 smoke still passes (existing well-formed package).
- [ ] Regression: motoko_ext_omnigraph@0.2.0 smoke FAILS (the bug we're trying to catch); 0.2.1 (with self-disable) PASSES.
- [ ] Regression: motoko_ext_mcp@0.2.0 smoke FAILS; 0.2.1 (with asset bundling) PASSES.

**M3 (extension updates)**:
- [ ] `std/extension.requireWorkdirFile` shipped + documented.
- [ ] `motoko_ext_omnigraph@0.2.1` published using the new pattern.
- [ ] `motoko_ext_mcp@0.2.1` published using the new pattern + bundled `mcp-call.mjs` asset.
- [ ] motoko_agent's `ailang.toml` + `ailang.lock` bumped to 0.2.1 versions.
- [ ] motoko_explore (the originally-affected workdir) verified working: `motoko "test exa search"` no longer crashes; `motoko "test omnigraph"` shows omnigraph as silently disabled in this workdir.

## Why this matters for AI-author workflows

Same compounding-effect logic as M-MATCH-ADT-XCHECK: AI agents iteratively write extensions → publish → consumer projects discover bugs at runtime → multiple-version package iterations to fix downstream. Today's chain for the omnigraph bug:

```
Author writes register.ail
  → ailang check passes (typecheck OK)
  → _smoke.ail (v0.18.10 era) passes (registration returns)
  → ailang publish succeeds
  → consumer pulls package
  → consumer's first OmnigraphStatus call CRASHES
  → consumer files bug report
  → author publishes 0.2.1 with workdir guard
  → consumer pulls + retries
  → finally works
```

With M-EXT-PORTABILITY-GATE shipped, the chain becomes:

```
Author writes register.ail
  → ailang check passes
  → ailang publish runs smoke in temp-dir
  → smoke calls OmnigraphStatus → cd: omnigraph: no such file
  → publish FAILS with "tool OmnigraphStatus crashed in temp workdir; either guard with std/extension.requireWorkdirFile or bundle a starter schema in assets/"
  → author fixes BEFORE upload
  → consumer pulls a working package on first try
```

Pairs with downstream-side `make verify-extensions` that we shipped in arniwesth/motoko_agent#16 — that's the consumer's safety net; THIS is the publisher's prevention.

## Risks

| Risk | Mitigation |
|---|---|
| Smoke tests need real environment (Process effect, etc.) — can't run hermetically | Allow smoke to use `--ai-stub` (no real AI calls) + document that Process-effect extensions may need to mock. Tools that LEGITIMATELY need a real environment can opt out of smoke via `[smoke] required = false` in ailang.toml. |
| 30s timeout too aggressive for large extensions | Per-package override in ailang.toml: `[smoke] timeout_seconds = 60`. |
| Asset bundling bloats package size | Document soft 1MB tarball limit (existing limit). Larger assets should live in OCI registries or Git LFS, referenced by URL. |
| `assetPath` API encourages packages to vendor large helpers | Same — soft size limit + reviewer judgment at publish time. |
| Ecosystem fragmentation: some packages on old format, some on new | Both work simultaneously. New format is opt-in until v1.0. Documentation makes the new pattern obvious. |
| Server-side validator latency increases publish time | Run smoke in parallel with the existing tarball-validation steps. Soft-launch behind a feature flag; gather telemetry before making mandatory. |

## Out of scope (future work)

- **Sandboxed tool dispatch** at runtime (different concern — capability system already provides isolation).
- **Cross-package smoke-test dependencies** (e.g. testing a tool that calls another extension's hooks). Defer to a v1.x sprint.
- **OCI-backed asset distribution** for large binaries. v1.x.
- **Snapshot testing** of tool outputs (golden-file style). Useful but separate sprint.

## Refs

- Source bugs: `motoko_explore` messages a17c208e, 90c8be04, 22fb9b81, b474396c (2026-05-11)
- Pairs with: arniwesth/motoko_agent#16 (`make verify_extensions` consumer-side check, shipped earlier today)
- Builds on: M-MATCH-ADT-XCHECK (v0.18.10) typechecker enforcement
- Related design docs:
  - `design_docs/planned/v0_10_0/m-pkg-metadata-urls.md` — package URL metadata (different concern; complementary)
  - `design_docs/planned/v1_0_0/m-pkg-compat-ailang-version-gate.md` — AILANG version requirements per package (different concern)
  - `design_docs/planned/v1_0_0/m-dx-package-dogfooding.md` — package author DX issues (parent meta-doc; this sprint addresses a major sub-issue)
