### Fixed — filesystem deny patterns protect ancestors from rename and removal (#1569)

The shared matcher now refuses rename sources, destinations and removals that can contain
an `fs_deny_write` target. This closes the parent-directory rename bypass for literal file
patterns, higher ancestors of `dir/**`, and mid-glob patterns such as `a/*/x.txt`.
Refusals retain `E_FS_PROTECTED` and identify the original pattern and ancestor relationship.
Basename patterns, unrelated moves, ordinary writes and mkdirs keep their existing behavior.
