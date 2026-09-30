### Changed — `ailang run` is quiet by default; progress goes to stderr

`ailang run` no longer prints "→ Type checking...", "→ Effect checking..." and
"✓ Running <file>" unless asked. `--verbose` brings them back, and they now go to
**stderr**, so stdout is exactly the program's output. `--quiet` is still
accepted (it is the default now). Before this, the lines went to stdout, so any
program whose output is compared byte for byte, such as a quine, saw them mixed
into its own output. Eval models were told to verify with plain `ailang run`, and
a local motoko quine run spent several steps looking for a way to suppress them.
Scripts that parsed the preamble from stdout should read program output directly,
or pass `--verbose` and read stderr.
