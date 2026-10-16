### Fixed — `ailang test --package` passed with a type error in an untested module (#1845) (2026-10-16)

Package mode compiled only `*_test.ail` files and what they import, so a module no test
touched could fail to type-check while `test` printed "All tests passed" and exited 0. It now
type-checks every non-test source module after the tests run (modules the tests already
compiled are cache hits), prints the errors on stderr, and exits 1, as `go test ./...` does.
The publish-only STRICT_FALLBACK rule is still `ailang check --package`'s job.

### Fixed — `std/package.assetPath` ignored the lockfile and sorted versions as strings (#1772) (2026-10-16)

`assetPath` listed the registry cache and took the string-largest version, so `0.9.0` beat
`0.10.0`, and a program locked to `0.3.0` read `0.4.0`'s assets when both were cached, so the
same program could read different data on two machines. It also ignored `AILANG_PACKAGE_ROOT`
and never found path or git dependencies. The runner now hands the effect the directories the
program's `ailang.lock` pins (plus the root package), and the fallback without a lock picks the
highest semver under the configured package root. The reference doc's claim that string order
matched semver was wrong and is corrected.
