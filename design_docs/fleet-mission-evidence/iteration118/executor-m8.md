Committed `a3f6a5088` on `fleet/i118-rulings-ledger-build` as `fix(mission): rulings check OS-independent paths (Windows CI)`. Not pushed.

- **Fixed all five Windows failures:** `displayPath` now returns slash-separated paths; the three path tests use `t.TempDir()` roots; the TOML `workdir` is written with `filepath.ToSlash`.
- **Choice on `displayPath`:** relative and repo-relative paths are slash-separated. An absolute path outside the root stays native.
- **Other literal paths:** the only other one in the rulings tests was the `/wd/` check, which is fine after the fix.
- **Checks on this host (darwin):** `gofmt -l` empty; `go vet` and `GOOS=windows go vet` rc 0; `golangci-lint` 0 issues; `go test ./internal/mission/` and the `-run 'Mission|Doctor|Ruling'` filter on `./cmd/ailang` pass.
- **Mutation did not fail on this OS.** Reverting the `ToSlash` calls in a scratch copy left every test passing, because `ToSlash` is an identity when the separator is `/`. I added a `filepath.Join` assertion, but it cannot fail here either. The regression is only caught by the Windows CI run, via the three tests above. The file was restored and confirmed identical.
- **Windows not run:** the per-test reasoning is in the commit body.
- **Changelog:** the coding standards want a fragment for every change. I didn't add one because the brief asked for a single commit; say if you want it.
