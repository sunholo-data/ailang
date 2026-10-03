### Fixed — `fs_deny_write` was case-sensitive: case variants bypassed it on APFS/NTFS (2026-10-03)

Under `fs_deny_write = [".claude/**", ...]`, a running program (`writeFile(".CLAUDE/settings.json", ...)`)
and `policy-tool write` on `.CLAUDE/y.json` both succeeded, and on a case-insensitive volume (macOS
APFS by default, Windows NTFS) the write landed in the deny-listed `.claude/` — editor hook config
that runs outside the sandbox. Only `.git` was case-folded, by two separate copies of the check.
There is now ONE matcher, `fileguard.Protection` (`internal/fileguard/protect.go`), used by the FS
effect and by policy-tool for both `.git` and `fs_deny_write`. It folds case (by Unicode case-folding
orbit, so `ſ`/`s` and the Kelvin sign/`k` match too) and NFC-normalizes, on every platform: over-denying
an odd spelling on a case-sensitive Linux volume is the fail-closed direction. (#1559)

### Fixed — `policy-tool fmt` with the `write` flag ignored `fs_deny_write`; flag values and duplicate request keys were ignored (2026-10-03)

`{"op":"fmt","path":".claude/x.ail","flags":{"write":""}}` rewrote a deny-listed file that the `write`
and `edit` ops refuse. Every write policy-tool performs now passes the same gate before the child
runs: `fmt` with `write`, `lock` (`ailang.lock`) and `design_quorum` (`.ailang/state/mission-quorum/`).
A boolean flag accepts only `""` or `"true"` — `"false"` used to mean `--write` and is now refused.
Requests are decoded strictly (`policytool.DecodeRequest`): keys must be spelled exactly (`FLAGS` no
longer aliases `flags`), may appear once (a second `flags` object used to merge into the first), and
unknown keys or trailing data are refused; a malformed request is a refusal response, not an exit 1.
The CLI child's compile and prompt cache now go to a private temp dir outside the sandbox
(`AILANG_CACHE_DIR`), so `check`/`test` no longer write `.ailang/cache/` into it. `test` scaffolding
(`_namedtest_body_*`) already lives in a private temp dir since #1502; a new end-to-end test pins that
a `test` op on a package inside a deny-listed directory resolves its sibling imports and writes
nothing in the sandbox. (#1554)
