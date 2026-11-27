### Fixed — git package checkouts ignore the user's line-ending config (#1720 D4)

- Every package git invocation now runs with `core.autocrlf=false` and `core.eol=lf`, as well as
  `protocol.ext.allow=never`. Before this, a user's `core.autocrlf=true` rewrote `.ail` files to CRLF
  on checkout, so one tag locked to a different `content_hash` on each machine and failed the
  fatal drift check at `ailang check`. `TestGitCheckoutIgnoresUserAutocrlf` covers it.
- A confined check with no clone of a git dependency now names the operator remedy rather than
  telling the agent to "run 'ailang lock'", because a confined lock refuses to fetch.
- `CachedPackagePath` joins with `filepath.Join`, so Windows no longer gets mixed separators.

The edits were left uncommitted in the `p0/pkg-authority-chokepoint` worktree and missed PR #1964.
They were recovered during the 2026-11-27 cleanup from `rescue/pkg-authority-git-eol-hardening`.
