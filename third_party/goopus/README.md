# third_party/goopus — vendored go-opus

A vendored subset of [github.com/tphakala/go-opus](https://github.com/tphakala/go-opus),
a pure-Go, fixed-point port of libopus. It backs `std/audio.encode`
(`internal/builtins/audio.go`).

- **Version:** v1.1.0, commit `0f3a5416f14945b7fedd0102eb04b7da4039c084`.
- **License:** BSD-3-Clause, see [LICENSE](LICENSE). Provenance of the libopus
  derivation: [THIRD_PARTY.md](THIRD_PARTY.md).
- **Runtime dependency:** `github.com/tphakala/simd` (MIT), pinned in the root
  `go.mod`.

## What was changed

1. Only the packages the encoder and decoder need are kept: `opus`, `oggopus`,
   and `internal/{celt,fixedmath,opusdec,opusenc,packet,rangecoding,silk,silkmath}`.
   Upstream tests, test data, commands, tools and the cgo reference harness
   (`internal/reftest`) are not vendored.
2. Import paths are rewritten from `github.com/tphakala/go-opus/...` to
   `github.com/sunholo-data/ailang/third_party/goopus/...`, so the code is part
   of the ailang module. A `replace` directive would have kept the upstream
   paths but breaks `go install github.com/sunholo-data/ailang/cmd/ailang@latest`.
3. No other edits. The code builds with the repository's Go 1.26.6 toolchain
   (upstream v1.1.0 declares `go 1.26`; later upstream commits moved to 1.27).

`std/audio` does not use `oggopus.Encoder`: its Ogg serial is random, and
`std/audio.encode` must be deterministic. The container is written by
`internal/builtins/audio_ogg.go`. `oggopus` is kept for its decoder, which the
round-trip tests use.

## Updating

Re-copy the same file set from a new upstream tag, rewrite the import paths
the same way, then run `go test ./internal/builtins -run 'TestAudio|TestOgg'`.
The golden SHA-256 in `audio_test.go` changes whenever the codec's output
does; update it deliberately and say so in the changelog.

This directory is excluded from golangci-lint and SonarCloud (upstream code, not
ours). `go vet` still covers it.
