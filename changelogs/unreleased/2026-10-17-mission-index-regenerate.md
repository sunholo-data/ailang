### Added

- `ailang mission rotate-log <name> --index-only [--stream log|status]` rebuilds the index from live and archived records without rewriting either source. Empty live logs work; structural headings are ignored and counted. Explicit `--keep` is incompatible with `--index-only`.

### Fixed

- Both rotation and index regeneration preserve numbered orphan rows verbatim and hand-written blockquote blocks above the table, reporting preserved orphan numbers. Source-backed rows still follow their headings. Archive and index read errors surface before normal rotation mutates sources. References #1306 (index preservation only).
