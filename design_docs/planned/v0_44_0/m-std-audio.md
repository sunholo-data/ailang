# M-STD-AUDIO: `std/audio`, PCM to WAV in pure AILANG plus a pure-Go Ogg Opus encoder

**Status**: M0-M2 implemented (sprint M-STD-AUDIO, 2026-09-25); M3 (resample) still planned
**Target**: v0.44.0
**Priority**: P2 (Low). The requester placed it below the serve-api WebSocket and `--host` tickets.
**Estimated**: about 3.5 agent-days. M1 takes 0.5, M2 takes 2, M3 takes 1 and is optional.
**Dependencies**: One design-freeze ruling from Mark on the third-party dependency and the Go 1.27 toolchain bump (F1 and F2 below). There are no code dependencies.
**Planner-Lane**: codex-ok for M1. M2 is opus-required because it adds a dependency and a container writer, and it carries a determinism contract.
**Requested by**: Daneel, inbox message `inbox_1790359273273_d0dfc4f7` (2026-09-25).

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Today the only way to compress is `afconvert` through the Process effect, which is an OS binary whose output is unpinned. M2 replaces it with a pure function that returns identical bytes for identical input. A spike measured this with a pinned Ogg serial number (V14). The random serial in the upstream container writer (V15) is the one nondeterminism source, and M2 removes it on purpose. Cross-architecture byte identity is **UNVERIFIED** (R2), and the acceptance tests cover it. |
| A2: Replayability | 0 | The functions are pure and add no effects or trace-shape changes. |
| A3: Effect Legibility | +1 | Compression moves from `! {Process}` (an OS binary with ambient power) to a pure function with no effect row. |
| A4: Explicit Authority | +1 | Daneel no longer needs `--caps Process` to make a brief small. |
| A5: Bounded Verification | +1 | `durationMs` carries an `ensures` clause that Z3 verifies (V12). The header builders carry `requires` contracts. |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | 0 | Errors come back as `Result` text. The API mirrors Daneel's requested shape. |
| A8: Minimal Syntax | 0 | No syntax changes. |
| A9: Cost Visibility | 0 | — |
| A10: Composability | +1 | The module composes with `std/bytes` and `std/fs.writeFileBytes`. M1 needs no new builtin, and M2 adds one builtin. |
| A11: Structured Failure | +1 | Unsupported rate, channel count or bitrate returns a typed `Err` and is never clamped silently (§Solution M2). |
| A12: System Boundary | −1 | The language closure gains a third-party dependency with a single maintainer (`tphakala/go-opus`). §Risks mitigates this with a pin, a vendored container writer and a golden test. |

**Net Score: +5**, so the decision is to **proceed**, subject to the rulings in Design Freeze F1 and F2.

### Hard Violation Check

- [x] A1: This introduces no implicit nondeterminism. The upstream random serial is removed by design in M2.
- [x] A3: No hidden side effects. All functions are pure.
- [x] A4: No ambient access. The design removes a Process requirement.
- [x] A7: Not optimising for humans over machines.

## Quorum trigger check

- **Trigger 1 fires.** F1 (accept a new third-party dependency into the language closure) and F2 (bump the Go toolchain from 1.26.6 to 1.27) need Mark's ruling.
- **Trigger 4 fires.** Two load-bearing premises are about external systems: the maturity of the library, and whether the file plays back on the requester's phone (V17).
- Triggers 2 and 3 do not fire. The design overrides no shared machinery and touches no cost, KPI or banked-data schema.

**Recommended:** run `ailang design-quorum ... --max-cost-usd 0.30` before sprint planning.

---

## Problem Statement

Daneel's morning and evening briefs are narrated with Vertex Gemini TTS or Live, which return raw PCM: 16-bit, 24 kHz, mono. Measurements:

- A 50 s evening brief is 2.38 MB of PCM (Daneel's figure). A local reproduction, a 50.39 s `say` render at 24 kHz s16 mono, gave 2,418,712 PCM bytes (V13).
- The weekly brief is about 3 minutes, which comes to about 8.6 MB as WAV. That is too big to attach and heavy for a phone over the tailnet.

Current workarounds:

- **The WAV header is hand-written.** Daneel builds the 44-byte RIFF header himself with `std/bytes.fromInts`. The `fromInts` builtin's own docs cite this use case, with the example `"RIFF" (WAV header magic)` (`internal/builtins/bytes_ints.go:33-40`).
- **Compression is not possible without the Process effect.** Today it needs Process plus macOS `afconvert`. The Studio has no ffmpeg, and the fleet does not want Python.

**Done when** (from the requester): a 50 s 24 kHz PCM brief becomes a playable `.ogg` or `.m4a` under 1 MB, made from AILANG alone.

**Spike result.** The pure-Go encoder recommended below turned the 50.39 s reproduction into a **229,763-byte Ogg Opus file in 0.12 s**, at a 24 kbps target (36.5 kbit/s effective). macOS CoreAudio decoded it back to a 50.39 s, 24 kHz WAV (V14). At a 16 kbps target the file was 123,553 bytes. Both results clear the 1 MB bar by 4–8×. A 3-minute weekly brief at 36.5 kbit/s works out to about 820 KB, which is also under 1 MB.

---

## Goals

**Primary goal:** a pure AILANG program turns TTS PCM into (a) a valid WAV and (b) an Ogg Opus file under 1 MB, with no capabilities beyond `FS` for the final write.

Success metrics:

1. `std/audio.wavFromPcm` output is byte-identical to a golden 44-byte header followed by the PCM.
2. `std/audio.encode(pcm, 24000, 1, OggOpus(24000))` on the 50 s fixture produces less than 400 KB.
3. That output decodes, through the library's own Ogg Opus decoder in a Go test, to within ±1 frame of the input sample count.
4. Encoding the same input twice produces identical bytes on the same platform.
5. `make build-wasm` stays green, and the repository still contains no cgo.

---

## Routing (PROGRAM.md §4)

| Piece | Lane | Why |
|---|---|---|
| `wavHeader`, `wavFromPcm`, `durationMs` | **AILANG stdlib, pure `.ail`, no builtin** | This is arithmetic and byte concatenation over existing builtins (`std/bytes.fromInts`/`concat`/`length`, `std/bytes.ail:77,91,63`). A working draft was type-checked and run end-to-end in 0.02 s (V11). It is **not** a separate package: M2 needs the `std/audio` module anyway, and splitting 30 lines of WAV helpers from the encoder into a separate package would give one concept two homes. |
| `encode` (Ogg Opus) | **AILANG fix: a new pure Go builtin** | PROGRAM.md:48 routes "missing builtin" to the AILANG-fix lane. The default bias toward extensions (PROGRAM.md:52) cannot hold here, because **AILANG packages are pure AILANG and cannot carry Go** (`internal/pkg` has no native-code path; `bin.go:15-22` only generates shims that `exec ailang run`). The two package-shaped alternatives both fail. **Process + afconvert** is darwin-only and nondeterministic, and it is exactly what the requester is escaping. **An encoder in pure AILANG** is impractical: one trivial `map` pass over the 50 s PCM through `toInts`/`fromInts` took **3.47 s and 925 MB peak RSS** (V16), and a CELT encoder does a few hundred times more work per sample. |
| `resample` (24 kHz to 16 kHz for Live input) | **AILANG fix, builtin, optional M3** | This has the same cost argument as `encode`. The Opus encoder does not need it, since Opus accepts 24 kHz natively (V18), so it serves only the Live-input use case. |
| AAC/m4a | **Deferred (rejected for core)** | See the encoder evaluation: LGPL licensing plus 44.1/48 kHz only in pure Go, or cgo. |

**Simplicity program check** (`design_docs/planned/m-v1-simplification-program.md`):

- **Stated non-goal.** The program lists `std/` changes as a non-goal (line 410), so this design does not collide with any of its phases.
- **Closure metric.** The gated metric is non-leaf *internal* packages in the language closure (line 90). M2 adds a file to `internal/builtins` and no new internal package, so the metric is unchanged.
- **Leak roots.** The third-party leak-root list (`tools/simplicity_metrics.sh:53`) covers sqlite, otel, grpc, websocket and GCP clients, and does not include audio codecs. `closure_leak_roots` therefore stays 0.
- **What the new dependency adds.** It adds third-party packages to the core closure: `go-opus/{opus,oggopus,internal/*}` plus `tphakala/simd` and `golang.org/x/sys`. It also costs about **197 KB** of binary size, measured with a hello-world versus hello-world-plus-`oggopus` build (V19). That is the A12 cost, accepted in F1.
- **Precedent.** `internal/builtins` already links three third-party runtime libraries: `gopkg.in/yaml.v3`, `golang.org/x/net/html` and `github.com/ollama/ollama/api` (V20).

---

## Encoder evaluation (UNVERIFIED marks what was not measured here)

| Option | Pure Go? | Result in this checkout | Verdict |
|---|---|---|---|
| **`github.com/tphakala/go-opus` v1.1.0** (BSD-3, libopus derivative) | Yes, zero cgo | **Spiked.** Encodes 24 kHz mono directly. 50 s gives 230 KB in 0.12 s. CoreAudio decodes the result. Builds for `js/wasm`, `windows/amd64` and `linux/amd64 CGO_ENABLED=0` (V14, V18, V21). Its README claims the CELT encoder is byte-identical to libopus 1.6.1 (UNVERIFIED here). | **Recommended**, with caveats. The **encoder is CELT-only**, which is not libopus's choice for low-bitrate speech, so quality per bit is lower than SILK. The size bar is still met at a 16 kbps target. The library is **2 months old, has 3 stars and one maintainer** (created 2026-07-16, V22). It needs **`go 1.27`**, while the repo and CI pin 1.26.6 (V23). **The Ogg serial is random** and not configurable (V15). |
| `github.com/kazzmir/opus-go` (BSD-3, ccgo-transpiled libopus, 23 stars) | Yes | **Spike FAILED.** The CLI accepts only 48 kHz. `-bitrate 16000/24000/32000` all gave ~830–880 KB, meaning the bitrate had no effect. Its own decoder aborted on its own output (`silk/resampler.c:193 assertion failed`) (V24). | Rejected. |
| `github.com/skrashevich/go-opus` (ccgo transpile) | Yes (per README) | Not spiked. UNVERIFIED. | Fallback candidate only. |
| cgo libopus (e.g. `hraban/opus`) | No | The repo has **zero** cgo (V25). Adding it breaks `CGO_ENABLED=0` cross builds, the darwin/amd64-on-arm64 release job and `js/wasm` (`make/build.mk:100-103`). | Rejected. |
| darwin AudioToolbox (cgo) or `afconvert` via Process | No | afconvert AAC at 32 kbps gave 214 KB (V26). It is darwin-only, needs build-tag stubs everywhere else, and its output bytes are not pinned. | Rejected for core. Daneel can keep using it as a stopgap. |
| `github.com/tphakala/go-aac` (pure Go AAC-LC) | Yes | Per the README, the encoder supports only 44.1/48 kHz (so a 2× resample would be needed), and it is **LGPL-2.1**. AILANG is Apache-2.0 and ships a statically linked binary (V27). | Rejected. Licensing needs a human ruling that is not worth taking for P2. |
| IMA-ADPCM WAV in pure AILANG (4:1, ~600 KB) | n/a | `afconvert -d ima4` into WAVE fails (`'fmt?'`, V26), so CoreAudio playback of the output is doubtful. The pure-AILANG cost argument in V16 also applies. | Rejected. |
| **Upstream: Cloud Text-to-Speech with Gemini-TTS voices and `audioEncoding: OGG_OPUS`/`MP3`** | n/a | Not probed. A public issue reports that *streaming* Gemini-TTS ignores `OGG_OPUS` and returns LINEAR16 (GoogleCloudPlatform/generative-ai#2480). Batch behaviour is UNVERIFIED. It does not help the Live API path. | **Open question Q1.** If batch synthesis returns MP3/Opus, the narrated-brief use case needs no AILANG change at all. |

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Accept `tphakala/go-opus` into the language closure (F1) | This is a supply-chain dependency with one maintainer, and it is hard to remove once programs rely on `encode` | human | design | high |
| Bump `go` from 1.26.6 to 1.27 in `go.mod` and every CI `go-version` (F2) | It affects every build lane (`.github/workflows/build.yml:51`, `ci.yml:95,511,618,664`) | human | design | med |
| Write the Ogg container ourselves over `opus.Encoder`, rather than use `oggopus.NewEncoder` | Determinism: the upstream writer uses a random serial (`oggopus/encoder.go:127,322`) | agent (recommended: own writer) | design | low |
| API shape: `encode(pcm, rate, channels, codec: Codec)` with `Codec = OggOpus(int)` | Adding codecs later must be additive | agent | design | med |
| Inputs are s16le only; any other bit depth returns `Err` | This keeps the builtin small. TTS output is s16 | agent | design | low |

### Design Freeze

- [x] **F1** (ruled 2026-09-25: vendor under `third_party/goopus`): accept `github.com/tphakala/go-opus` (BSD-3), pinned to an exact version, into `internal/builtins`. The alternative is to vendor its `opus` + `internal/*` packages under `third_party/`, which lets us hold our own copy if upstream disappears.
- [x] **F2** (ruled 2026-09-25: no Go bump; v1.1.0 already declares `go 1.26` and builds on 1.26.6): the Go 1.27 toolchain bump. The alternative is to vendor the code and lower its `go` directive, if it does not actually use 1.27 features (UNVERIFIED, since the spike used `GOTOOLCHAIN=auto` and go1.27.0).
- [x] **F3**: the codec set is Ogg Opus only. AAC/m4a is deferred (see Non-Goals).

## Deferred Decisions

- The Ogg serial value, for example a fixed constant or a hash of the PCM. The agent may choose, provided it is a pure function of the input.
- Frame duration (20 ms default) and complexity (default 10). The agent may choose. Neither is exposed in v1 of the API.
- Resampler algorithm in M3. The agent may choose, provided it uses **integer fixed-point** arithmetic. Go may fuse `x*y+z` into FMA on arm64 but not on amd64, so float filters can differ across architectures.
- The builtin-name suffix (`_audio_encode_ogg_opus`). The agent may choose. Both names are currently unallocated (V10).

---

## Solution Design

### M1: `std/audio` in pure AILANG (no Go)

`std/audio.ail` is embedded automatically (`std/embed.go:10`, `//go:embed *.ail`). The block below is the draft that was type-checked, and its WAV half was run end-to-end on the 50 s fixture (V11). `afinfo` read the output as `1 ch, 24000 Hz, Int16, 50.389833 sec`.

```ailang
module std/audio

import std/bytes (fromInts, concat, length)
import std/result (Result, Ok, Err)

-- Compressed output formats; the payload is the target bitrate in bits/s.
-- NOTE: `type Codec = OggOpus` (nullary, one ctor) parses as a type ALIAS, not an ADT.
export type Codec = OggOpus(int)

pure func le16(n: int) -> [int] = [n & 255, (n >> 8) & 255]
pure func le32(n: int) -> [int] = [n & 255, (n >> 8) & 255, (n >> 16) & 255, (n >> 24) & 255]

-- 44-byte canonical RIFF/WAVE (format tag 1 = integer PCM) header for dataLen PCM bytes.
export pure func wavHeader(dataLen: int, rate: int, channels: int, bitsPerSample: int) -> bytes
requires { dataLen >= 0 && dataLen <= 4294967259 && rate > 0 && channels > 0 && bitsPerSample > 0 && bitsPerSample % 8 == 0 }
{
  let blockAlign = channels * (bitsPerSample / 8);
  fromInts([82, 73, 70, 70] ++ le32(36 + dataLen) ++ [87, 65, 86, 69]
    ++ [102, 109, 116, 32] ++ le32(16) ++ le16(1) ++ le16(channels)
    ++ le32(rate) ++ le32(rate * blockAlign) ++ le16(blockAlign) ++ le16(bitsPerSample)
    ++ [100, 97, 116, 97] ++ le32(dataLen))
}

export pure func wavFromPcm(pcm: bytes, rate: int, channels: int, bitsPerSample: int) -> bytes
requires { rate > 0 && channels > 0 && bitsPerSample > 0 && bitsPerSample % 8 == 0 }
{
  concat(wavHeader(length(pcm), rate, channels, bitsPerSample), pcm)
}

-- Duration in whole milliseconds (integer: no float rounding to argue about).
export pure func durationMs(pcmLen: int, rate: int, channels: int, bitsPerSample: int) -> int
requires { pcmLen >= 0 && rate > 0 && channels > 0 && bitsPerSample > 0 && bitsPerSample % 8 == 0 }
ensures { result >= 0 }
{
  pcmLen * 1000 / (rate * channels * (bitsPerSample / 8))
}
```

For the fixture, the golden header bytes (from `xxd` on the V11 output, `dataLen = 2418712`, 24000/1/16) are:

```
52 49 46 46 3c e8 24 00 57 41 56 45 66 6d 74 20
10 00 00 00 01 00 01 00 c0 5d 00 00 80 bb 00 00
02 00 10 00 64 61 74 61 18 e8 24 00
```

`durationMs` returns an `int` in milliseconds, whereas Daneel asked for `duration(...)`. A float return would carry rounding the caller never asked for, and an integer keeps the Z3 proof in scope. Callers who want seconds can compute `intToFloat(ms) / 1000.0`.

### M2: `encode`, backed by a pure Go builtin `_audio_encode_ogg_opus`

```ailang
-- in std/audio (signature type-checked in V11; body is the builtin call)
export pure func encode(pcm: bytes, rate: int, channels: int, codec: Codec) -> Result[bytes, string] {
  match codec {
    OggOpus(bitrate) => _audio_encode_ogg_opus(pcm, rate, channels, bitrate)
  }
}
```

The Go side, `internal/builtins/audio.go` (about 250 LOC plus tests), follows the `deflate.go` pattern:

- **Registration.** `RegisterEffectBuiltin(BuiltinSpec{Module: "std/audio", IsPure: true, Effect: ""})` (`deflate.go:115-120`), typed with `T.Func(T.Bytes(), T.Int(), T.Int(), T.Int()).Returns(Result[bytes,string])`. It takes raw `bytes` rather than the base64 strings `std/deflate` uses. The newer `bytes` convention is `bytes_ints.go:26-27`.
- **Validation, returning `Err` and never clamping.** The rate must be one of 8000/12000/16000/24000/48000. Channels must be 1 or 2. `len(pcm) % (2*channels) == 0`. Bitrate must be in [6000, 510000]. PCM must be at most 100 MB, the same cap as deflate/gzip (`std/deflate.ail:22`).
- **Encoding.** It builds `opus.NewEncoder(opus.EncoderConfig{SampleRate, Channels, Bitrate})` and feeds 20 ms frames. The final partial frame is zero-padded, and the true length is recorded through the Ogg granule position (end trimming, RFC 7845 §4.4).
- **Ogg container.** We write our own (about 120 LOC): `OpusHead` and `OpusTags` pages, audio pages, CRC-32 with polynomial 0x04C11DB7, and the pre-skip taken from `Encoder.PreSkip()`. The **serial is a pure function of the input**. The upstream `oggopus.NewEncoder` is not used because it calls `rand.Uint32()` for the serial (`oggopus/encoder.go:322`, V15).
- **Wiring.** `internal/pipeline/testdata/builtin_types.golden` gets the new line. `internal/stdlib/integration_test.go` gets a module entry beside `std/regex` (line 39).

### M3 (optional): `resample`

`_audio_resample(pcm, fromRate, toRate, channels) -> Result[bytes, string]` handles s16le only, with an integer polyphase FIR. The initial pairs are 24000→16000 and 16000→24000. Any other pair returns `Err`. The Live API wants 16 kHz input, and a pure-AILANG version is ruled out by V16.

---

## Conflict Surface

The design touches no parser, typechecker or codegen code. The names it adds:

- **Module path `std/audio`.** It is unallocated, since no `std/audio.ail` exists (V10).
- **Builtin names `_audio_encode_ogg_opus` and `_audio_resample`.** A grep for `_audio_` over `internal/` and `std/` returns nothing (V10).
- **Type `Codec` and constructor `OggOpus`.** Both are module-scoped, so they cannot collide with user types unless imported.

**Programs that must still work:** the existing `std/bytes` tests, `examples/` (`make verify-examples`) and the WASM build (`make build-wasm`).

**Dialect trap found while drafting.** `export type Codec = OggOpus` is parsed as a **type alias**, not a one-constructor ADT: `ailang iface` shows `"alias": "OggOpus"`, and importing `OggOpus` fails with IMP010 (V11). The design uses `OggOpus(int)`, which carries the bitrate. The trap itself belongs in a separate ailang-core triage item.

**Separate finding (not in scope, needs its own triage row).** `bytes` is missing from the parser's builtin-type list (`internal/parser/parser_type.go:66-69`, which lists int/float/string/bool/unit/char). In a source annotation, `bytes` therefore parses as a **type variable**. The consequence: `pure func f(b: bytes) -> bytes = b` accepts `f(42)` without error, while the `string` control correctly fails (V9). `std/audio` still gets the right `bytes` types because its bodies call `bytes`-typed builtins, but any pure-AILANG function whose body does not do so is silently polymorphic. The fix is one line plus regression tests, and it should land before or with M1.

---

## Examples

```ailang
module use_audio

import std/audio (wavFromPcm, durationMs, encode, Codec, OggOpus)
import std/fs (writeFileBytes)
import std/bytes (length)
import std/result (Result, Ok, Err)

-- Daneel's brief: raw 16-bit 24 kHz mono PCM in, a WAV and an Ogg Opus out.
export func saveBrief(pcm: bytes, stem: string) -> Result[int, string] ! {FS} = {
  writeFileBytes("${stem}.wav", wavFromPcm(pcm, 24000, 1, 16));
  match encode(pcm, 24000, 1, OggOpus(24000)) {
    Ok(ogg) => { writeFileBytes("${stem}.ogg", ogg); Ok(durationMs(length(pcm), 24000, 1, 16)) },
    Err(e) => Err(e)
  }
}
```

This example was checked against the draft module with `ailang check` (V11). It ships as `examples/audio_brief.ail` together with a `manifest.json` entry.

---

## Success Criteria

**M1**

- [ ] The `parser_type.go` `bytes` fix has landed, or has a triage row and an agreed order.
- [ ] `std/audio.ail` exports `wavHeader`, `wavFromPcm` and `durationMs`.
- [ ] `ailang verify std/audio.ail` reports `durationMs` VERIFIED.
- [ ] A golden test asserts `wavHeader(2418712, 24000, 1, 16)` equals the 44 bytes above exactly. It also asserts a stereo 48 kHz case (blockAlign 4, byteRate 192000).
- [ ] A round-trip test: `wavFromPcm` output has length `len(pcm)+44`, and slicing from offset 44 returns the PCM unchanged.

**M2**

- [ ] A size test: the checked-in 5 s fixture at `OggOpus(24000)` is under 40 KB. A 50 s synthetic fixture (generated in the test, not checked in) is under 400 KB.
- [ ] A round-trip test: Go-side `oggopus.NewDecoder` decodes the output, and the decoded sample count after pre-skip and end-trim equals the input count ±1 frame. The capture-pattern header parses as `OpusHead`, version 1, with the correct channels and input rate.
- [ ] A determinism test: two encodes of the same input are byte-identical (`bytes.Equal`). A golden SHA-256 of the 5 s fixture's output is checked on **both** linux/amd64 CI and darwin/arm64; see R2 for the rule when they differ.
- [ ] A negative test for each rejected input: rate 44100, 3 channels, odd byte length, bitrate 0 and bitrate 1_000_000. Each returns `Err` with a message naming the field.
- [ ] `make build-wasm`, `GOOS=windows go build ./...` and `CGO_ENABLED=0 go build ./...` all stay green.
- [ ] Manual: the output plays in macOS QuickTime/afplay, and on the requester's phone (Q2).

**M3 (optional)**

- [ ] Resampling 24k→16k of 1 s gives exactly 32,000 bytes (mono s16). A 1 kHz sine keeps its dominant FFT bin at 1 kHz. The output is identical across amd64 and arm64.

**All milestones**

- [ ] `make test`, `make lint` and `make verify-examples` pass.
- [ ] The CHANGELOG is updated, `docs/docs/reference/std-audio.md` is added, and a `stdlib.md` row is added.

## Testing Strategy

- **Go.** `internal/builtins/audio_test.go` holds the golden, round-trip, size, determinism and negative tests.
- **AILANG.** `tests/audio_wav_test.ail` covers the pure M1 functions (header golden, length round-trip).
- **Mutation check.** Revert the pinned-serial change so the upstream random serial comes back, and confirm the determinism test goes red.

---

## Non-Goals

- **AAC/m4a.** This would require LGPL (`go-aac`) or cgo (AudioToolbox). It can be reopened if Q2 finds that Ogg does not play on the target phone.
- **Decoding, playback, MP3, Vorbis and streaming (incremental) encode.**
- **Exposing Opus tuning knobs** (complexity, frame size, VBR, DTX). They are internal defaults.

## Timeline

| Day | Work |
|---|---|
| 0.5 | M1, plus the `bytes` parser fix (if routed here) |
| 2 | M2: dependency, Ogg writer, builtin, tests, WASM/cross checks |
| 1 | M3 (optional) |

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| R1: `go-opus` is abandoned or breaks (one maintainer, 2 months old) | Pin the exact version. The golden tests catch drift. Vendoring is the F1 alternative. The API surface we use is tiny (`opus.NewEncoder`, `Encode`, `PreSkip`). |
| R2: float CELT is not bit-identical across amd64 and arm64 | Measure it in M2. If the goldens differ, keep per-architecture goldens and state "deterministic per platform" in the std docs. Round-trip and size tests stay architecture-independent. |
| R3: CELT-only speech quality at 16–24 kbps is poor | Default to `OggOpus(24000)`, which is about 230 KB for 50 s, and let the caller raise it. There is 4× headroom under 1 MB. |
| R4: the Go 1.27 bump breaks another lane | F2 is a human ruling. The vendoring alternative avoids the bump. |

## Open Questions for Mark

1. **Q1.** Has Daneel tried batch Cloud Text-to-Speech with Gemini-TTS voices and `audioEncoding: MP3/OGG_OPUS`? If that works, the brief use case needs only M1, or nothing at all, and M2 can wait. Live API output stays PCM regardless.
2. **Q2.** What phone and player will open the attachment? macOS CoreAudio decodes the Ogg Opus output (V14), but iOS playback of `.ogg` is UNVERIFIED. If it fails there, the fallback is Opus in a CAF container, which Apple supports natively, as an extra `Codec` constructor. That is roughly another 100 LOC and is UNVERIFIED.
3. **F1 and F2.** Rule on the dependency, and choose between the Go 1.27 bump and vendoring.

---

## Related Documents

- `design_docs/planned/m-v1-simplification-program.md`: closure and dependency constraints.
- `design_docs/PROGRAM.md` §4: routing.
- Precedent for a pure-Go codec builtin: the `std/deflate`, `std/gzip` and `std/zip` builtins (`internal/builtins/deflate.go`).

## Verification Log

| # | Claim | Evidence |
|---|---|---|
| V1 | `fromInts`, `concat` and `length` exist as pure `bytes` functions | `std/bytes.ail:91,77,63` |
| V2 | The `fromInts` builtin documents the WAV-header use | `internal/builtins/bytes_ints.go:33-40` |
| V3 | The pure-builtin pattern is `IsPure: true, Effect: ""` | `internal/builtins/deflate.go:115-120` |
| V4 | Stdlib `.ail` files are embedded automatically | `std/embed.go:10` |
| V5 | `builtins` is in the WASM build | `GOOS=js GOARCH=wasm go list -deps ./cmd/wasm \| grep -c internal/builtins$` returns 1 |
| V6 | Packages cannot carry Go | `internal/pkg/bin.go:15-22`, where bins are `ailang run` shims. There is no native-code path in `internal/pkg` |
| V7 | PROGRAM.md routes a missing builtin to the AILANG-fix lane | `design_docs/PROGRAM.md:48,52` |
| V8 | The simplicity program excludes `std/`, gates internal non-leaf packages, and does not list codecs as leak roots | `m-v1-simplification-program.md:410,90`; `tools/simplicity_metrics.sh:53` |
| V9 | `bytes` in an annotation parses as a TypeVar | `parser_type.go:66-69` omits `bytes`. `f(b: bytes) -> bytes = b; g() = f(42)` passes `ailang check`, while the `string` control fails with `No instance for Num[string]` |
| V10 | `std/audio` and `_audio_*` are unallocated | `grep -rn "std/audio\|_audio_" internal std docs/docs` returns empty. The positive control is the same grep form hitting `_bytes_from_ints` |
| V11 | The draft module and example type-check. WAV output is valid. `type X = C` is an alias | `ailang check` on scratch `std/audio.ail` and `use_audio.ail` gives "No errors found". `ailang run` wrote a 2,418,756 B WAV, which `afinfo` reads as 50.389833 s. `ailang iface` shows `"alias": "OggOpus"`, and the import gave IMP010 |
| V12 | The `durationMs` contract verifies | `ailang verify` reports `✓ VERIFIED durationMs` |
| V13 | 50 s at 24 kHz s16 mono is about 2.4 MB | `say --data-format=LEI16@24000` then `afinfo` gives 2,418,712 audio bytes over 50.39 s |
| V14 | go-opus: 24 kHz input, 230 KB, 0.12 s, CoreAudio decodes it, byte-identical with a pinned serial | `wav2opus -bitrate 24000` gives 229,763 B (36.5 kbit/s). `afconvert` back to WAV gives 50.389833 s at 24 kHz. With `randomSerial` patched to a constant, two runs were identical under `cmp` |
| V15 | The upstream Ogg serial is random, so identical runs differ | Unpatched, `cmp` shows the files differ at byte 15, with 95 differing bytes (the serial plus the CRC on every page). `oggopus/encoder.go:127,322` (`rand.Uint32()`). `oggopus.Config` has no serial field |
| V16 | A pure-AILANG pass over the PCM is too expensive | `fromInts(map(\x. 255 - x, toInts(pcm)))` on 2.4 MB: 3.47 s real, 925,712,384 B max RSS (`/usr/bin/time -l`) |
| V17 | iOS playback of Ogg Opus | **UNVERIFIED** (Q2) |
| V18 | go-opus accepts 8/12/16/24/48 kHz | `opus/encoder.go:47-51` (EncoderConfig doc), plus the V14 run at 24 kHz |
| V19 | The dependency costs about 197 KB of binary | 2,429,698 B vs 2,627,154 B for hello-world vs hello-world plus `oggopus.NewEncoder` |
| V20 | Existing third-party imports in builtins | grep over `internal/builtins/*.go` finds `gopkg.in/yaml.v3`, `golang.org/x/net/html` and `github.com/ollama/ollama/api` |
| V21 | go-opus cross builds | `GOOS=js GOARCH=wasm go build ./oggopus ./opus`, `GOOS=windows` and `CGO_ENABLED=0 linux/amd64` all OK |
| V22 | go-opus maturity | `gh api repos/tphakala/go-opus` shows 3 stars, created 2026-07-16, pushed 2026-09-17, tags v1.1.0 / v1.0.0 / v0.1.3 |
| V23 | Toolchain mismatch | go-opus `go.mod` says `go 1.27`. Ours says `go 1.26.6` (`go.mod:3`), and CI pins `1.26.6` (`build.yml:51`, `ci.yml:95,511,618,664`) |
| V24 | kazzmir/opus-go defects | At 16k/24k/32k targets the outputs were 835,163 / 855,233 / ~880 KB. `oggopus2wav` on its own output hits `Fatal (internal) error in ../silk/resampler.c, line 193` |
| V25 | There is no cgo in the repo | `grep -rl 'import "C"' --include='*.go' .` returns empty |
| V26 | afconvert baselines | AAC m4a at 32 kbps gives 214,361 B. `-f WAVE -d ima4` gives `ExtAudioFileCreateWithURL failed ('fmt?')` |
| V27 | go-aac limits | README (fetched 2026-09-25) says the encoder handles 44.1/48 kHz only and is LGPL-2.1-or-later. `gh api` license is LGPL-2.1. The ailang `LICENSE` is Apache 2.0 |

## References

- RFC 6716 (Opus), RFC 7845 (Ogg encapsulation for Opus), RFC 3533 (Ogg)
- https://github.com/tphakala/go-opus, https://github.com/kazzmir/opus-go, https://github.com/tphakala/go-aac, https://github.com/skrashevich/go-opus
- GoogleCloudPlatform/generative-ai#2480 (Gemini-TTS streaming ignores OGG_OPUS)

## Future Work

- A `CafOpus(int)` codec constructor if iOS needs it (Q2).
- A streaming encoder for Live output, together with the serve-api WebSocket work.
