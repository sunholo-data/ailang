### Fixed — `ailang-lens` kept a stale ✗ after the package manifest changed (2026-10-07)

The lens re-checked a module only when the module file itself changed. In a package, a
`pkg/...` import resolves through `ailang.toml` `[exports]`, so adding the export fixed the check
without touching the module, and the pane kept showing `module "…" is not exported by package`.
A shown module is now re-checked when its package's `ailang.toml` or `ailang.lock`, or a module it
imports by relative path (`./data/stars`), changes, and an Edit/Write of any `.ail` or
`ailang.toml` in a package re-checks the other modules shown from it. Reported on stapledons' `sim`
package.
