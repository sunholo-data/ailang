### Fixed — registry confinement and content verification (Refs #1607, #1608)

- Confined registry clients refuse network fetches and cache writes. Package reads no longer create the HOME registry cache; unconfined docs/install/lock retain their fetch workflows.
- Hosts can provide an exclusive, read-only `AILANG_PACKAGE_ROOT` (`<vendor>/<name>/<version>`), with no fallback to HOME or legacy lock paths when configured.
- Registry content is verified against the lock before compilation. Tampering or missing registry content fails `ailang check` immediately; path dependency drift still warns.
- Known gap: a confined `ailang lock` with a git dependency still clones into `$HOME/.ailang/cache/git`, and `pkg info`/provenance skip the confinement guard; tracked in #1720.
