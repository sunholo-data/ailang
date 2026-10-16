### Fixed

- Named tests and properties use parser-recorded closing positions when removing
  test blocks from a module. Braces in strings, JSON fragments, interpolation,
  character literals and comments cannot truncate those ranges or cause valid
  modules accepted by `ailang check` to fail during `ailang test`.
