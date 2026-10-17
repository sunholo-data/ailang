### Fixed — `fs_deny_write` was bypassable through an in-root symlink or hard link (2026-10-17)

**Security.** The protection check for the policy-tool's write/edit and for the FS effect's
mutating ops matched the path *as spelled*. The root handle then followed any symlink that stayed
inside the sandbox. With `fs_deny_write = ["tests/**", "*_test.go"]`, writing `lnk/new.py`
(`lnk -> tests`) created `tests/new.py`, and writing `alias.txt` (`-> pkg/a_test.go`) rewrote the
protected test. A dangling link was enough. Reported by AILANG World (row 156 design), reproduced
on v0.52.1 and v0.54.11.

`fileguard.Root.Resolve` now follows in-root symlinks component by component, through
`os.Root.Lstat`/`Readlink`, keeping not-yet-existing components as spelled.
`Root.CheckPath` matches the spelled name and then the resolved one. policy-tool and
`internal/effects` both use it, so the two cannot drift. A rename or remove judges a final
symlink by its own name, because it acts on the link, not the target. A write to a regular file
with more than one hard link is refused while protection applies, since a hard link is a second
name no walk can find (Unix only; Windows does not expose the count through `Stat`). Refusals
name the resolved path (`… (resolves to tests/new5.py)`).

### Fixed — `**` in `fs_deny_write` patterns meant `*` (one directory level) (2026-10-17)

A `**` that was not a trailing `/**` went to `path.Match`, where it is just `*`.
`**/*.test.ts` protected `src/foo.test.ts` but not `a.test.ts` or `src/deep/x.test.ts`. A `**`
component now matches zero or more directories. A pattern with a slash stays anchored at the
sandbox root, so `tests/**` does not cover `cli/tests/` (write `**/tests/**` for that). A
malformed glob now fails closed instead of matching nothing. Reported by AILANG World.

### Added — `policy-tool` `delete` op (2026-10-17)

`{op: "delete", path}` removes one file or symlink entry inside the sandbox, never the link's
target. It refuses directories, empty ones included, and is gated by the rename/remove
protection, so a protected file, a directory holding protected paths, and `.git/` cannot be
removed. Without it, an agent confined to policy-tool could not remove a helper it had created.
In AILANG World episode tps-c (2026-10-17) the agent rewrote a test file to undo one. The pi
`ailang_only` lane does not expose the op: its prompt still tells the model it has no delete tool.
