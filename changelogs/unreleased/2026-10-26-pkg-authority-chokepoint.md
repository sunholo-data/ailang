### Fixed — security: one authority gate for package network and `$HOME` writes under confinement (#1720)

A confined agent (`AILANG_AGENT_POLICY` set, e.g. through `ailang policy-tool`'s `lock`
op) could still reach the network and write outside its sandbox through the package
commands. v0.53.0 (#1607/#1608) gated three registry-client functions; an audit of every
package verb and the package loader found 22 more unguarded sites, measured live:

- `ailang lock` / `lock --check` / `install --path` with a git dependency cloned into
  `~/.ailang/cache/git` (a `file://` URL copies any repository the host user can read,
  and later checks compiled against that clone);
- `ailang check`/`test`/`iface`/`run` created `~/.ailang/cache/git` for any project
  with a git dependency;
- `lock` with a prior lock opened the messaging store, creating
  `~/.ailang/state/collaboration.db`;
- `pkg info`, `pkg versions`, `pkg stats` and `pkg history` made registry requests;
- publish upload, unpublish, `pkg key`, the cascade Pub/Sub publisher, `[bin]` shim
  writes and removals, and `notify-upgrade`/`publish` messaging had no guard.

Now `internal/pkg/authority.go` is the single gate per authority (network,
`~/.ailang` writes): package HTTP clients refuse at the transport, git runs only
through a hardened, classified runner (`--` before URLs, `ext::` disabled), home
directories are computed without being created, and the messaging store and Pub/Sub
have one gated opener each. Operator verbs (`publish`, `unpublish`, `pkg key`,
`install`, `add`, `notify-upgrade`, `bin uninstall`, `pkg cascade`) are refused at
dispatch. Every refusal reads `confined by AILANG_AGENT_POLICY: … is operator
authority — <remedy>`; the v0.53.0 registry text is unchanged.

Behaviour changes worth knowing:

- **Confined `lock` with git dependencies** never runs git. An entry already pinned in
  `ailang.lock` (same `git_url`, tag or rev, `subdir`) whose provisioned copy still
  matches its hashes is carried forward; anything else is refused. Locks now record
  `git_tag`; regenerate an older lock once outside the sandbox.
- **`AILANG_PACKAGE_ROOT` covers git packages**, at `<root>/<vendor>/<name>/<version>`,
  with no HOME fallback.
- **Git `content_hash` is verified at `ailang check` and drift is fatal**, like registry
  packages. The git cache is shared per URL, so two projects pinning different tags of
  one repository now get "content changed … run 'ailang lock'" instead of silently
  compiling against whichever tag was checked out last.
- Manifest `git`/`tag`/`rev`/`subdir` values beginning with `-` are rejected.
- Confined `lock` skips the silent-ratchet messaging check and says so on stderr.

Guarding the class, not the list: a Go AST test fails CI when package code uses a raw
HTTP client, `exec.Command` git, `os.UserHomeDir`, `messaging.OpenStore` or
`pubsub.NewClient` outside the gate, and an end-to-end matrix runs every `ailang pkg`
verb plus the loader entry points confined against a loopback recorder and a git shim
(zero requests, zero clones, byte-identical HOME/sandbox) and fails when a verb has no
row. Design: `design_docs/planned/v0_54_19/m-pkg-authority-chokepoint.md`.
