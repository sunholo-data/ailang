# Frozen live trial inputs

Implementation: `48e72ef7e`. Existing guide review authority: `97e695be542c6580d603ff3d6f9729bd4ecc6541`.
Additional task input/authority source: `e6b54faf0c857553d55cfac171d3f0dd49c7d6b9`.
Both additional tasks independently use that source and remain undispatched until the
existing guide is accepted and its completed replay proves no new provider calls.

Generated inputs and logs are retained in `/private/tmp/ailang-reliability-trials`.
Original immutable canary state remains in `/private/tmp/ailang-docs-canary`.
Model snapshots are retained in that trial directory; their digest is frozen in
`tasks-frozen.json`. No inference credentials are stored in these artifacts.

Combined new trial cap $5: existing review $2, each additional task $1.50.
Preparation uses isolated temporary HOME bindings and dry-run; it cannot activate work.
Live runs use `mission activation run` with the actual host's owned installation.
