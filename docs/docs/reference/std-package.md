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
- `Ok(absolutePath)` — the package is installed and the asset exists. Absolute path under `~/.ailang/cache/registry/<vendor>/<name>/<version>/assets/<rel>`.
- `Err("invalid package name: ...")` — `pkgName` is not in `vendor/name` form.
- `Err("invalid relative path: ...")` — `relPath` is absolute or contains `..`.
- `Err("package not installed: ...")` — no version of `pkgName` is in the registry cache.
- `Err("asset not found: ...")` — the package is installed but the asset is absent (file was renamed, deleted, or never bundled).

**Version selection**: when multiple versions of the package are installed, `assetPath` picks the lexically-highest version. This matches semantic versioning for the small versions used in practice (e.g. `0.10.0 > 0.2.0` lexically and semantically).

**Security**:
- Requires the `FS` effect — invocations need `--caps FS` at the CLI.
- Path validation in two layers: surface check (`assetPath` rejects bad inputs before calling the builtin) plus the underlying `_pkg_asset_path` builtin re-validates so a malicious caller bypassing the wrapper can't escape `assets/`.
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
- [v0.19.0 changelog entry](https://github.com/sunholo-data/ailang/blob/main/changelogs/v0.10-current.md) — full v0.19.0 release notes.
