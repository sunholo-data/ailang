### Added — Guarded shared skill checkout refresh

Mission fires now fast-forward the shared skill checkout with `merge --ff-only origin/dev`
after disabled/overlap guards and before boot stagger. The checkout must be on `dev`,
have zero local commits, no worktree operation or index lock, and no dirty paths
intersecting the incoming range (including rename sides and directory obstructions).
Bounded Git calls and a final safety recheck skip refusals without retrying or
removing locks; no fetch, reset, stash, checkout or rebase is performed.

Dry runs call report only: the `skill-sync=synced:N` field is a computed verdict,
with a `would-sync` note from the helper, and the target stays unchanged. Real fires
log status and note; absent helpers produce an explicit error and continue.
`AILANG_SKILL_SYNC=0` disables inspection; `AILANG_SKILL_SYNC_CHECKOUT` overrides the
resolved mission-control skill symlink. These are shell-only knobs, enabled by default.
