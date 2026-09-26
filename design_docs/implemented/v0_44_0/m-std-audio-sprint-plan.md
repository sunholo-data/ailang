# Sprint Plan: M-STD-AUDIO

**Design doc:** [m-std-audio.md](m-std-audio.md)
**Progress file:** `.ailang/state/sprints/sprint_M-STD-AUDIO.json`
**Duration:** about 2 agent-days (M0 0.25, M1 0.25, M2 1.25, buffer 0.25). M3 is deferred.
**Risk:** medium. A vendored third-party codec enters the language closure.

## Rulings taken in place of Design Freeze F1/F2

The caller ruled on the doc's two open freezes before planning:

- **F1 → vendor.** go-opus is copied under `third_party/`, keeping its BSD-3 LICENSE.
- **F2 → no Go bump.** Build on the repository's 1.26.x. If go-opus cannot build on 1.26, stop M2
  and report the compiler error.

Planning measurement that changes the doc's premise: the doc says go-opus needs `go 1.27`
(V23). That is true of upstream `main`, but the **v1.1.0 tag declares `go 1.26`**, and
`opus` + `oggopus` build on go1.26.4 and go1.26.6 natively, for `js/wasm`, `windows/amd64` and
`linux/amd64 CGO_ENABLED=0`. So F2 needs no directive lowering at all.

Vendoring shape: a nested module plus a `replace` directive would keep upstream import paths, but a
`replace` in the main module breaks `go install github.com/sunholo-data/ailang/cmd/ailang@latest`
(documented in CONTRIBUTING.md) and the Dockerfiles' `COPY go.mod go.sum` + `go mod download`
layer. The plan vendors into the ailang module instead, with import paths rewritten.

Also measured: go-opus is **fixed-point** (a transliteration of libopus built `FIXED_POINT`), so the
doc's risk R2 (float CELT differing across architectures) should not arise. M2 verifies it rather
than assuming it.

## Milestones

### M0: `bytes` in a type annotation is a type, not a type variable (~80 LOC)

The parser's primitive-name map (`internal/parser/parser_type.go:66-69`) omits `bytes`, so
`f(b: bytes) -> bytes` is polymorphic and accepts `f(42)`.

Systemic audit (CLAUDE.md principle 3): the names appear in at least five hand-written switches
(`internal/types/typechecker.go` `astTypeToType`, `internal/elaborate/file_funcs.go`
`astTypeToInternalType`, `types/normalize.go`, `types/type_head.go`, `types/helpers.go`), none of
which is a list the parser could import (`parser` imports only `ast`, `errors`, `lexer`). The one
list both sides can see is in `internal/ast`.

- Add `ast.BuiltinTypeNames()` / `ast.IsBuiltinTypeName()`; the parser uses it.
- Tests that range over that list, so a name without a converter mapping fails a test:
  `internal/types` (`astTypeToType` returns a `TCon` for every name) and `internal/pipeline`
  (every primitive except int/float rejects an int argument, end to end).
- Regression: the int-passed-as-bytes case, plus a positive control.
- `make test`, `make verify-examples`, `make verify-stdlib`: any stdlib interface that changes
  is a real mistyping being corrected. Re-freeze and list each one; do not weaken the check.

Acceptance: the regression test fails with `bytes` removed from the list (mutation check).

### M1: `std/audio` WAV helpers in pure AILANG (~70 LOC + tests)

- `std/audio.ail`: `Codec = OggOpus(int)`, `wavHeader`, `wavFromPcm`, `durationMs` (with
  `requires`/`ensures`), from the doc's type-checked draft.
- `tests/stdlib/audio_wav_test.ail` under `make test-stdlib-ail` (bump
  `STDLIB_AIL_SUITES_EXPECTED`): golden mono 24 kHz header (the doc's 44 bytes), stereo 48 kHz
  header (blockAlign 4, byteRate 192000), PCM round trip, `durationMs`.
- `ailang verify std/audio.ail` reports `durationMs` VERIFIED.

### M2: `encode` over vendored go-opus with our own Ogg container (~350 LOC Go + tests)

- Vendor `opus`, `oggopus`, `internal/*` (non-test files, `.s` included) to `third_party/goopus`,
  rewrite imports, add `github.com/tphakala/simd` v1.8.0 to `go.mod`. Exclude the directory from
  golangci-lint and SonarCloud; `go vet` still runs.
- `internal/builtins/audio_ogg.go`: OpusHead/OpusTags pages, audio pages (whole packets, at most
  1 s per page), RFC 3533 CRC, pre-skip from `Encoder.PreSkip()`, zero-padded last frame, silent
  frames until coded audio covers pre-skip + input, end-trimmed final granule. Serial = FNV-1a of
  (rate, channels, bitrate, PCM).
- `internal/builtins/audio.go`: `_audio_encode_ogg_opus : (bytes, int, int, int) ->
  Result[bytes, string]`, pure; validation returns `Err` naming the field.
- Wiring: `builtin_types.golden`, `internal/stdlib/integration_test.go`, stdlib freeze.
- Tests: golden SHA-256 of a synthetic (integer-only) 5 s fixture, determinism, a different
  input gets a different serial, 5 s < 40 KB, 50 s < 400 KB, round trip through
  `oggopus.DecodeInterleaved` with exact sample count at six shapes, negative cases, CRC vector.
- Cross-architecture: run the golden under `GOOS=js GOARCH=wasm` (scalar kernels) locally;
  linux/amd64 CI checks the same golden.
- Mutation checks: random serial makes the determinism test red; removing the end trim makes the
  round-trip test red.

Acceptance (DONE WHEN): a worktree-built binary turns a ~50 s 24 kHz mono PCM file into a
< 1 MB `.ogg` from an AILANG program alone, byte-identical across two runs, and `afconvert`
decodes it to the right duration.

### Docs and examples (all milestones)

- `examples/runnable/std_audio_brief.ail` + manifest entry (runnable `main`, plus a `convert`
  entry for real PCM files).
- `docs/docs/reference/std-audio.md`, `stdlib.md` row, `third_party/goopus/README.md`.
- `changelogs/v0.32-current.md` (the root CHANGELOG is an index).
- Gates: `make fmt`, `make lint`, `make vet`, `make check-file-sizes`, `make verify-examples`,
  `make verify-stdlib`, `make test-stdlib-ail`, `make build-wasm` (or `check-wasm-build`),
  `CGO_ENABLED=0 go build ./...`, `GOOS=windows go build ./cmd/ailang`.

### M3 (deferred): `resample`

Optional in the doc and not needed by the encoder (Opus accepts 24 kHz). Left as a follow-up:
integer polyphase FIR, 24k↔16k only.

## Open questions carried from the doc

- Q1 (does batch Cloud TTS return Opus/MP3 directly) and Q2 (iOS playback of `.ogg`) are for the
  requester; neither blocks this sprint.

## Status (2026-09-25)

- [x] M0: `bytes` annotation fix, one primitive-name list in `internal/ast`, stdlib re-freeze
  (net, process, stream, sem corrected; math picks up #1274's `isNaN`). Mutation-checked.
- [x] M1: `std/audio` WAV helpers; `durationMs` VERIFIED; `tests/stdlib/audio_wav_test.ail` 13/13.
- [x] M2: `encode` over vendored go-opus v1.1.0; golden hash equal on darwin/arm64 and js/wasm;
  DONE WHEN met (50.96 s PCM, 2,446,018 B → 236,967 B `.ogg`, identical across two runs,
  `afconvert` decodes to exactly 2,446,018 B). Mutation-checked (serial, end trim).
- [x] Docs, example + manifest, changelog.
- [ ] M3: deferred (follow-up).
