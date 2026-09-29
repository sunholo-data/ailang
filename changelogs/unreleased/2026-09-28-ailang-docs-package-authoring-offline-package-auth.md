### Added — `ailang docs package-authoring`: offline package authoring guide (2026-09-28)

An embedded guide covering package style, contracts, native tests, effect ceilings and budgets,
validation and publishing. It works without a source checkout, a stdlib path or the network.
Both `ailang-packages` skill variants now separate compilation, native tests, quality evidence and
publication. Their `validate_package.sh` runs lock, `check --package`, `test --package` and
`pkg quality --strict`, and says that inline test blocks in source modules still need an explicit
`ailang test <module>`. The branch's own `pkg quality` inventory (written 2026-09-08) is dropped in
favour of the M-PKG-QUALITY-LADDER command that has since shipped.
