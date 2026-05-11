# std/extension — helpers for AILANG extension packages

The `std/extension` module (v0.19.0+) provides conventions and small utilities for AILANG packages that plug into a host runtime (typically [motoko_agent](https://github.com/sunholo-voight-kampff/motoko_agent)) via the `[extension]` manifest block. The first helper, `requireWorkdirFile`, is the canonical way for an extension to self-disable when a required consumer-side input is missing.

## When to use

Use `requireWorkdirFile` at the top of your extension's `register_with_config` when the extension's tools depend on filesystem state in the consumer's workdir (a config file, a schema, a profile directory). The pattern lets the host cleanly skip your extension instead of crashing on the first tool dispatch.

If your extension is self-contained — all assets bundled inside the package via `[assets]` — use [`std/package.assetPath`](./std-package) instead and skip `requireWorkdirFile` entirely.

## The self-disable pattern

The shape AILANG extensions converge on (canonical example: `motoko_ext_omnigraph 0.2.1`):

```ailang
import std/extension (requireWorkdirFile)
import std/result (Result, Ok, Err)
import std/env (getEnvOr)

func enabled_in(workdir: string) -> bool ! {FS} =
  match requireWorkdirFile(workdir, "omnigraph/omnigraph.yaml") {
    Ok(_)  => true,
    Err(_) => false
  }

export func register_with_config(_cfg: a) -> ExtensionHooks ! {Env, FS} {
  let workdir = getEnvOr("MOTOKO_WORKDIR", ".");
  let active = enabled_in(workdir);
  let tools = if active then provided_tools() else [];
  {
    id: "omnigraph",
    provided_tools: tools,
    -- ... rest of hooks unchanged
  }
}
```

When `omnigraph/omnigraph.yaml` is absent in the workdir, `provided_tools` is empty. The host (motoko_agent) won't dispatch unadvertised tools, so `on_tool_handle` never fires and the extension cleanly skips itself.

This replaces the older anti-pattern of calling the panicking variant of `std/fs.readFile` and bringing down the host on the first tool call.

## API

### `requireWorkdirFile(workdir, rel) -> Result[(), string] ! {FS}`

Returns `Ok(())` when `<workdir>/<rel>` exists, `Err(message)` otherwise.

```ailang
import std/extension (requireWorkdirFile)
import std/result (Ok, Err)

match requireWorkdirFile("/tmp", "missing-config.yaml") {
  Ok(_)  => println("found"),
  Err(e) => println("not found: ${e}")
}
-- Output: not found: required workdir file not found: /tmp/missing-config.yaml
```

**Parameters**:
- `workdir` — absolute or relative path to the consumer's workdir. Typically read from `MOTOKO_WORKDIR` env var via `getEnvOr("MOTOKO_WORKDIR", ".")`.
- `rel` — path relative to `workdir`. Forward slashes only; AILANG concatenates them as-is.

**Returns**:
- `Ok(())` — the file exists at `${workdir}/${rel}`.
- `Err("required workdir file not found: ${path}")` — the file is absent. The error message includes the full resolved path for triage.

**Effects**: requires the `FS` capability — invocations need `--caps FS` at the CLI.

**Note on directories**: `requireWorkdirFile` returns `Ok(())` for both files and directories — it's just a `fileExists` wrapper. If you need to distinguish, use `std/fs.isFile` or `std/fs.isDir` after the existence check.

## Patterns

### Sentinel-only check

When you only want to know whether an input exists (not read it):

```ailang
match requireWorkdirFile(workdir, "config.yaml") {
  Ok(_)  => active_hooks(workdir),
  Err(_) => disabled_hooks
}
```

### Existence + read

When you also need the file contents:

```ailang
import std/extension (requireWorkdirFile)
import std/fs (readFile)
import std/result (Ok, Err)

let cfg = match requireWorkdirFile(workdir, "config.yaml") {
  Ok(_)  => readFile("${workdir}/config.yaml"),  -- safe: existence already verified
  Err(_) => ""
};
```

### Multiple required inputs

When the extension needs several files all present:

```ailang
let all_present = match requireWorkdirFile(workdir, "schema.yaml") {
  Err(_) => false,
  Ok(_) => match requireWorkdirFile(workdir, "policy.yaml") {
    Err(_) => false,
    Ok(_) => true
  }
};
```

(A future helper could fold a list of required files; for now the nested-match is explicit.)

## Run the example

```bash
ailang run --caps IO,FS --entry main examples/extension_self_disable.ail
# Expected output:
#   ext disabled: required workdir file not found: /tmp/missing-config.yaml
#   provided_tools = []
```

The shipped example uses a tiny stand-in `Hooks` record type so it stays self-contained; real extensions return the full `pkg/sunholo/motoko_ext_abi/types.ExtensionHooks` shape.

## Why not just panic on missing input?

Before v0.19.0, extensions typically called `std/fs.readFile` (the panicking variant) at registration time. If the consumer's workdir lacked the input, the whole motoko_agent process would crash on the first tool dispatch — long after the registration succeeded.

The crash had three downsides:

1. **No graceful skip**: a consumer using 5 extensions, one of which legitimately doesn't apply to their workdir, would have all 5 crash together.
2. **Late discovery**: the failure surfaced at first tool call, not at registration. Took longer to attribute.
3. **No structured signal**: stderr panic vs ExtensionHooks `provided_tools: []` is a categorically different failure mode that the host can't recover from.

The `requireWorkdirFile` + empty-`provided_tools` pattern fixes all three.

## See also

- [M-EXT-PORTABILITY-GATE design doc](https://github.com/sunholo-data/ailang/blob/main/design_docs/implemented/v0_19_0/m-ext-portability-gate.md) — the v0.19.0 sprint that introduced this helper alongside the pre-publish smoke gate and `std/package.assetPath`.
- [`std/package.assetPath`](./std-package) — companion helper for assets that ship inside the package itself rather than living in the consumer's workdir.
- [`std/smoke.dispatchAllTools`](https://github.com/sunholo-data/ailang/blob/main/std/smoke.ail) — the canonical `_smoke.ail` helper that exercises every advertised tool to catch tool-dispatch-time crashes.
- [v0.19.0 / v0.19.1 changelog entries](https://github.com/sunholo-data/ailang/blob/main/changelogs/v0.10-current.md) — full release notes including the M-EXT-PORTABILITY-GATE chain.
