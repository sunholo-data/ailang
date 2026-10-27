### Changed — CI `test` job split in two; the FMA gate builds in one batch

- **`test` → `test` + `checks`, side by side.** `test` keeps the Go suite and
  what reads its coverage profile (the coverage gate, the badge numbers,
  SonarCloud). The other 60-odd gate steps (examples, determinism, stdlib,
  hygiene gates, fuzz, imports…) move to a new `checks` job. One job ran them
  back to back: 1312-1446s on dev (runs 40495946401, 40497105921, 40498455123).
  Probe run 40507150837: all jobs green, `test` 786s and `checks` 822s in
  parallel, whole run 988s against 1468-1565s for those three.
- **`build` now waits for `checks` instead of `test`, and always runs.** Branch
  protection requires `test`, `lint` and `build`, not `checks`, so `build` fails
  unless `changes` and `checks` both succeeded. That includes the docs-only lane:
  there it skips the compile but still refuses a red `checks`, because a skipped
  required job counts as passed.
- **`make check-no-fma`:** one `go test -c` per arch for all four packages, both
  arches at once, objdumps in parallel. Local 33.6s → 8.9s, but NOT faster on
  CI's 4-core runner (124s in the probe vs 93-121s before). It now also fails if
  a package stops producing a test binary, so the scan cannot shrink silently.
  Mutation-checked: an unwrapped product in `mathx.expmulti` is still caught,
  same output as before.
