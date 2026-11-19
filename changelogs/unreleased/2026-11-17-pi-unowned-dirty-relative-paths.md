### Fixed — pi `unowned-dirty` reported the session's own edits as unowned (2026-11-17)

The extension recorded edit/write paths as absolute (`${ctx.cwd}/${p}`) but compared them with the
repo-relative paths `git status --porcelain` prints, so every own edit was named in the "may sweep N
dirty file(s) this session did not write" warning. The handler now resolves `git rev-parse
--show-toplevel` and relativizes own paths against it (`relativeTo`), resolving symlinks first
(`realOrSelf`, for a cwd under macOS `/var` → `/private/var`). Adds a handler-level test. Same
contract as the Claude Code port's `relativeTo` (tools/claude-mods/unowned-dirty).
Ticket `pi:unowned-dirty-absolute-vs-relative-paths`, ruling D-FLEET-33 A.

**Embedded copy synced:** `cmd/ailang/pi_assets/unowned-dirty.ts` was synced by an attended
session (`make pi-assets`, commit `d2ee7dc03`) because the fleet's push scope excludes
`cmd/ailang/pi_assets/**`; `make verify-pi-assets` (CI `checks`) passes, so `ailang pi setup`
installs the fixed extension.
