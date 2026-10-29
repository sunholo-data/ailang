### Fixed

- Package smoke staging rewrites path dependencies to their published versions
  and regenerates the execution lock, so registry dependency source and assets
  are tested together. Existing confined git-pin carry-forward and package-root
  authority are preserved.
