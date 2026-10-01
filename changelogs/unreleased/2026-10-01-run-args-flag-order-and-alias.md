### Fixed — `ailang run` argument footguns (stapledons_godot report)

- **A `run` flag after the file path is now an error.** It used to be passed silently to the program as an
  argument: `ailang run --entry main file.ail --args-json '{…}'` gave the entry no arguments and failed
  later with `ARG_DECODE_MISMATCH … got <nil>`. Now:
  `flag --args-json comes after the file path, so it would be passed to the program instead of to ailang run; put it before the file …`.
  Program arguments that look like flags still work after a literal `--`, and arguments that aren't `run` flags
  are unaffected.
- **Named record types decode as entry arguments.** `export type Args = {…}` with `func main(a: Args)`
  failed with `unsupported type constructor: Args`, while the same record written inline worked. The
  decoder now expands aliases from the entry module's interface, including nested ones such as `[Vec]` inside
  `Args` and chains like `type A = B`. A cyclic alias is an error.
- Three example header comments showed `run` flags after the file; they now show them before it.
