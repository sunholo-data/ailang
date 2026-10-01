### Added — `ailang run --chdir` and `ailang policy-tool --request-file`

Two flags for callers that cannot set a child process's working directory or
pipe its stdin. The motivating caller is an AILANG program using
`std/process.exec`, as motoko's `ailang_only` lane extension does:
- `ailang run --chdir DIR` runs as if started in `DIR`, so the entry path and
  module names resolve from it. Under `--policy`, `DIR` must be inside
  `fs_sandbox`, because it becomes the module root; otherwise the run is refused.
- `ailang policy-tool --request-file FILE` reads the JSON request from a file
  instead of stdin. The policy still decides every request.

Found while testing: when a policy does not admit `FS`, it resolves no
sandbox root, so the rule that the entry file must be inside the sandbox (and
the new `--chdir` rule) has nothing to check against. Lane policies always
admit `FS`.
