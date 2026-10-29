---
reviewBy: 2027-01-18
---

# std/package — bundled asset resolution

The `std/package` module (v0.19.0+) lets AILANG packages ship arbitrary helper files — shell scripts, JSON schemas, templates, prompts — and resolve them at runtime to absolute filesystem paths. Removes the "consumer must vendor `scripts/mcp-call.mjs`" failure mode that motoko_ext_mcp 0.2.0 demonstrated in production.

## When to use

Use `assetPath` when your package needs a non-`.ail` helper file (Node script, YAML schema, markdown template) that must be present at runtime. The asset ships inside the package tarball and is installed into `~/.ailang/cache/registry/<vendor>/<name>/<version>/assets/`.

If your package only needs other `.ail` modules, just use ordinary `import pkg/...` paths — `std/package` is for non-AILANG resources.

## Bundling assets

Declare assets in the package's `ailang.toml`:

```toml
[assets]
files = ["mcp-call.mjs", "schemas/tool-call.json"]
```

Place the files under the package's `assets/` subdirectory:

```
my-pkg/
├── ailang.toml
├── register.ail
└── assets/
    ├── mcp-call.mjs
    └── schemas/
        └── tool-call.json
```

At `ailang publish` time, `VerifyDeclaredAssets` rejects the publish if any file listed in `[assets].files` is missing from the package's `assets/` directory — a typo fails loud rather than shipping a package whose runtime `assetPath()` lookups all return `Err`.

The undeclared case is also supported: any file under `assets/` is bundled in the tarball even without a `[assets]` declaration. Declare assets to opt into publish-time existence verification; leave undeclared when you want the convenience without the guard.

## API

### `assetPath(pkgName, relPath) -> Result[string, string] ! {FS}`

Resolves to the absolute filesystem path of an asset shipped inside an installed package.

```ailang
import std/package (assetPath)
import std/result (Result, Ok, Err)

func bridge_path() -> Result[string, string] ! {FS} =
  assetPath("sunholo/motoko_ext_mcp", "mcp-call.mjs")
```

**Parameters**:
- `pkgName` — canonical `vendor/name` (e.g. `"sunholo/motoko_ext_mcp"`). Must contain exactly one `/`.
- `relPath` — path relative to the package's `assets/` subdirectory. Must not be absolute or contain `..`.

**Returns**:
- `Ok(absolutePath)` — the package is installed and the asset exists. The path is under the locked dependency directory (registry, path, or git source), or the fallback registry cache.
- `Err("invalid package name: ...")` — `pkgName` is not in `vendor/name` form.
- `Err("invalid relative path: ...")` — `relPath` is absolute or contains `..`.
- `Err("package not installed: ...")` — no version is cached, or the locked dependency is unavailable. A missing locked target names the package/version and install command; it never selects another version.
- `Err("no valid installed versions: ...")` — fallback found only unparseable version directories.
- `Err("asset not found: ...")` — the package is installed but the asset is absent (file was renamed, deleted, or never bundled).

**Version selection**: the running program's resolved package graph selects the same directory its modules were compiled from (registry, git or path dependency, and the root package itself). A pinned `0.3.0` wins even when `0.4.0` is also cached. Missing locked targets return `Err` naming the pinned version and install command, without fallback or network fetch; existing import validation may reject a missing dependency before evaluation.

For packages absent from the lock, or when there is no lock context (bare programs and the REPL), lookup chooses the highest installed semantic version (`0.10.0` beats `0.9.0` and `0.2.0`) under `~/.ailang/cache/registry`, or `AILANG_PACKAGE_ROOT` when set. Prereleases sort before their release. Unparseable directory names are skipped; a cache containing only invalid versions returns `Err`.

Engine targets are scoped to that engine and retained across calls and context replacements. The root package's own assets resolve directly from its source directory, including during authoring before a lockfile exists.

**Security**:
- Requires the `FS` effect — invocations need `--caps FS` at the CLI.
- The `_pkg_asset_path` builtin validates package names and relative paths before lookup. Sandboxed contexts are refused before any lookup because package assets are outside the FS sandbox.
- The resolved path is verified to exist via `os.Stat` before return — a missing file produces `Err`, not a dangling path.

## Examples

### Resolving a bundled helper script

```ailang
import std/package (assetPath)
import std/process (exec)
import std/result (Result, Ok, Err)

func run_bridge(args: [string]) -> () ! {FS, Process} =
  match assetPath("sunholo/motoko_ext_mcp", "mcp-call.mjs") {
    Ok(p) => {
      let _ = exec("node", [p] ++ args);
      ()
    },
    Err(_) => ()
  }
```

### Self-disable on missing asset

The canonical pattern for extension packages that need a bundled helper:

```ailang
import std/package (assetPath)
import std/result (Result, Ok, Err)

export func register_with_config(_cfg: a) -> ExtensionHooks ! {FS} =
  match assetPath("sunholo/my_ext", "tool-runner.mjs") {
    Ok(path) => active_hooks(path),
    Err(_)   => { provided_tools: [], ... }   -- self-disable
  }
```

If the package isn't installed correctly (or someone manually deleted the cache directory), the extension cleanly advertises no tools instead of crashing on first dispatch.

### Run the example

```bash
ailang run --caps IO,FS --entry main examples/asset_path.ail
# Expected:
#   not installed: package not installed: nonexistent/pkg
```

The `examples/asset_path.ail` shipped with this release uses a non-existent package so the example is self-contained; substitute your own `vendor/name` to see the `Ok` branch.

## See also

- [M-EXT-PORTABILITY-GATE design doc](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/v0_19_0/m-ext-portability-gate.md) — the v0.19.0 sprint that introduced asset bundling, the pre-publish smoke gate, and `std/extension.requireWorkdirFile`.
- [`std/extension.requireWorkdirFile`](./std-extension) — companion helper for extensions whose assets live in the consumer's workdir rather than the package itself.
- [v0.19.0 changelog entry](https://github.com/sunholo-data/ailang/blob/dev/changelogs/v0.18-v0.25-eval-harness.md) — full v0.19.0 release notes.

## Registry module resolution and confinement

For `import pkg/...` module resolution and `ailang pkg-docs`, hosts can set
`AILANG_PACKAGE_ROOT` to a read-only registry snapshot containing
`<vendor>/<name>/<version>/ailang.toml` and package source files. The configured
root is exclusive: a missing root, package, or docs guide is an error, with no
HOME cache fallback or automatic root creation. Without this variable, registry
modules and docs use `~/.ailang/cache/registry` as before. Locked `std/package.assetPath` targets follow this setting through the package
loader; no-lock fallback asset lookup continues to use the HOME cache.

Under `AILANG_AGENT_POLICY`, registry network fetches and cache writes refuse
with a named confinement error. Pre-provisioned versioned dependencies can be
locked offline; legacy transitive path dependencies that need an index refuse.
Provision a root outside confinement:

```bash
ailang install sunholo/oauth@0.1.0
mkdir -p /srv/ailang/packages
cp -R ~/.ailang/cache/registry/. /srv/ailang/packages/
chmod -R a-w /srv/ailang/packages
export AILANG_PACKAGE_ROOT=/srv/ailang/packages
```

Registry `.ail` source content must match the lock's `content_hash` in the
selected root or HOME cache. A mismatch makes `ailang check` exit non-zero in
this release, with no deprecation window; missing registry content also fails.
Path dependency drift continues to warn. Operators should regenerate a lock
only after verifying and provisioning the intended package content.
