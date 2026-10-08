### Fixed — motoko lane tasks run on the `ailang_only` profile when none is given

A coordinator agent on the `ailang_only` lane with `provider: motoko` would have refused every
task. The motoko executor took its profile from the cloud job's `MOTOKO_CONFIG=dogfood` unless the
task carried `motoko_profile`, and coordinator tasks never do. The lane gate requires
`ailang_policy` first with strict loading, which `dogfood` fails. A lane task with no explicit
`motoko_profile` now runs on `ailang_only` (an explicit one, as the eval harness sets, still wins).
The executor writes the choice into the task's metadata, so the gate and the run judge the same
profile.
