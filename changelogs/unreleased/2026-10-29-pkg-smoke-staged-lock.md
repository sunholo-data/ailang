### Fixed

- Package smoke staging rewrites path dependencies to their published versions
  and regenerates the execution lock, so registry dependency source and assets
  are tested together. Existing confined git-pin carry-forward and package-root
  authority are preserved.

- Failed package smoke reports now retain staging/setup errors, the first useful
  process diagnostic and timeout context in bounded human/JSON PUB015 Notes,
  including long and non-ASCII output. Deliberate `--no-run` stays distinct.
