### Added

- `std/io.readLineOpt() -> Option[string] ! {IO}` distinguishes stdin EOF (`None`)
  from blank lines (`Some("")`). Final unterminated content is returned once,
  followed by `None`. Both line APIs share buffered input; existing `readLine`
  behavior is unchanged. Includes a runnable echo loop and EOF-aware teaching prompt.
