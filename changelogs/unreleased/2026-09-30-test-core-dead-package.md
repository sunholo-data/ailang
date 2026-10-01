### Fixed — `make test-core` could not run at all

`CORE_PKGS` in `make/test.mk` still listed `./internal/module/...`, a package deleted in
`184a852df` (M-STDLIB-ROOT-RESOLUTION, #1324). `go test` fails the whole invocation on an
unmatched pattern — `pattern ./internal/module/...: lstat ./internal/module/: no such file or
directory` … `[setup failed]` — so the language inner loop that CLAUDE.md tells you to run between
edits exited non-zero no matter what the code did, and had done since that commit. Removed the dead
entry; the remaining 17 package trees pass.
