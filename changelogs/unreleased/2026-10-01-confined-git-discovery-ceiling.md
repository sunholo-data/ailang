### Fixed — confined git walked up out of the fs_sandbox into an enclosing repository (2026-10-01)

Security audit 2026-10-01 F-A1. Under a restricted policy whose `fs_sandbox` was a repository
*subdirectory* (or unset), `git:log`/`git:status` discovered the ancestor repository and printed its
commits, authors and file names. Confined git now runs with `GIT_CEILING_DIRECTORIES` set to the
symlink-resolved sandbox's **parent** (git ignores a ceiling equal to the cwd), so a `.git` at the
sandbox root — the clone-root case every deployed policy uses — is still found, and a subdirectory
sandbox gets `not a git repository`. Related hardening in the same adapter:

- Restricted `Process` without `FS` + `fs_sandbox` is now a named policy-resolution error (no
  deployed policy is affected: all three Process-granting policies set `fs_sandbox = "${WORKSPACE}"`).
- `git diff` is refused when the sandbox is not a repository root (outside a repository git treats
  `git diff <a> <b>` as `--no-index` of two arbitrary paths), and rev tokens with a `..` path segment
  (`d/../../etc/hosts`) are refused; ranges like `A..B` are unaffected.
- `-c safe.bareRepository=explicit`: a sandbox cannot be turned into a planted bare repository.
- `--no-ext-diff --no-textconv` on diff/log. This also fixes confined `git diff` patch output, which
  always died with "external diff died" because git runs `diff.external=` (empty) as a program.
- `core.hooksPath=/dev/null/ailang-hooks-disabled` (F-A8): a path no one, root included, can create.
- `.git` write protection (FS effect and policy-tool) is case-folded: on macOS's case-insensitive
  APFS, `.GIT/config` *is* `.git/config` and was writable.
