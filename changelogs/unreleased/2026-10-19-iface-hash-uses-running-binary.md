### Fixed — package interface identity (v2) is computed by the running binary

- `ailang publish` / `ailang pkg quality` built the v2 interface hash in a subprocess that was chosen by
  file NAME: only an executable called exactly `ailang` counted as itself, and anything else silently
  used the first `ailang` on PATH. A v0.54.15 build run under another name therefore hashed with an
  installed v0.52.0, and the registry refused sunholo/email 0.10.0 with PUB005 against every
  "upgraded" publisher the author tried. The CLI now pins its own executable at startup
  (`pkg.UseIfaceBinary`); only processes with no pin (the registry validator, tests) look on PATH.
- PUB005 now names both versions: the publisher sends `X-Ailang-Version`, and the refusal says
  whether the publisher is behind, whether both sides ran the same version (not a version gap), or
  — for an older publisher that sent no version — to check `which ailang`. The validator half takes
  effect on the next registry deploy.
