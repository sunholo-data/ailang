### Fixed — `std/math` and float kernels returned different bits on arm64 and x86_64 (#1465) (2026-10-02)

The same pure program printed `exp(0.2064590551107192)` as `1.229317398921793` on
arm64 and `1.2293173989217931` on x86_64, so replay goldens had to be kept per
architecture. All three backends called Go's `math` package, which makes no
cross-architecture promise. On arm64 it runs an FMA assembly `Exp`, and the compiler
fuses `x*y+z` inside `Log`, `Sin`, `Cos`, `Tan`, `Asin`, `Atan` and `Pow`. On amd64 it
runs assembly `Exp` and `Log`, and that `Exp` changes path depending on whether the CPU
has FMA.

- New `internal/mathx` holds the fdlibm-derived pure-Go algorithms from Go's `math`
  package, with every product wrapped in `float64(...)`. The Go spec says this
  conversion stops fusion. It covers `exp log log10 sin cos tan asin acos atan atan2
  pow`, and the interpreter, `--strict-bytecode` VM and `--emit-go` all use it.
  Compiled programs get the same source, embedded and emitted as `ailmathx_*` helpers.
  `sqrt`, `floor`, `ceil`, `round` and `abs` are exact operations and still use host
  `math`.
- The same `float64(...)` change applies to `_vec_dot`, `_vec_axpy`, `_array_f_dot`,
  `_array_f_axpy`, `_rand_float` (affects seeded replays) and SharedIndex similarity
  scores. On arm64 these compiled to fused multiply-adds.
- Golden-bit tests (`internal/mathx/golden_test.go`) pin SHA-256 digests of more than
  220,000 outputs, plus 112 exact bit patterns. They check the same bits on every
  architecture, and they matched on darwin/arm64 and on js/wasm. On js/wasm, Go's own
  `math` has no assembly and no fusion, and it agrees with `mathx` on every input.
- New CI gate `make check-no-fma` (`scripts/check_no_fma.sh`) cross-compiles the
  runtime packages for arm64 and amd64/v3 and fails on any fused multiply-add
  instruction. A new `float determinism (arm64)` CI job runs the golden tests on
  `ubuntu-24.04-arm`.

**Values change by at most 1 ulp on some inputs.** The canonical value is the Go
pure-Go algorithm with no fusion, not x86_64's old assembly output: that output was not
a single value set, because `Exp` changed with the CPU's FMA support. The reported
input now prints `1.229317398921793` on every machine. arm64 values change for most
functions on a small fraction of inputs. x86_64 values can change for `exp`, `log`,
`log10` and `pow`, whose host versions were assembly. Regenerate float goldens once,
after which a single set serves every architecture. `--emit-go` arithmetic written by
the user (`x * y + z`) can still fuse on arm64 (see `docs/LIMITATIONS.md`). Design:
`design_docs/implemented/v0_51_1/m-cross-arch-float-determinism.md`.
