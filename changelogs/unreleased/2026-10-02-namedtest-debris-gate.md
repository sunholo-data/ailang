### Fixed — stale `_namedtest_body_*.ail` files are refused, never published (#1502)

- An interrupted `ailang test` of v0.51.0 or older could leave a `_namedtest_body_<digits>.ail` in a
  package. Each one is a full copy of a test module. Current binaries cannot produce them, but packages
  that already carry one were still affected:
  - `pkg quality` counted it as a compiled file.
  - `publish` put it in the tarball.
  - `git add -A` staged it.
- `pkg quality` and `publish` now raise a **PUB024** gate at every stability level. The gate names each
  file and the fix: delete it, and ignore `_namedtest_body_*.ail`. `pkg quality` exits 2 and `publish`
  refuses.
- These files are also excluded from:
  - the publish tarball
  - the content hash
  - smoke staging
  - package source discovery
  - the `ailang check <dir>` walk, which names what it skipped

  With the files present, tarball bytes, the hash and the `pkg quality` compile count are the same as
  for the clean tree.
- `ailang init package` scaffolds a `.gitignore` containing `_namedtest_body_*.ail`. An existing
  `.gitignore` keeps its lines and gets the entry once.
- The package-authoring guide and the package-publishing guide both describe PUB024.
