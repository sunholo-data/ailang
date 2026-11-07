# M-STD-AUDIO-WAV-READ: std/audio learns to read what it writes

**Status**: Planned
**Target**: v0.54.17
**Priority**: P1 (Medium — a consumer ships and maintains its own WAV parser today, but a working `toInts` re-implementation exists downstream)
**Estimated**: 2–3 days (6h impl + 4h tests + 2h docs, 2x buffer)
**Dependencies**: None

## Problem Statement

`std/audio` (M-STD-AUDIO, v0.44.0) is write-only: it can wrap raw PCM in a
canonical 44-byte RIFF/WAVE header (`wavHeader`/`wavFromPcm`) and compress it
to Ogg Opus (`encode`), but it cannot read a single byte of a WAV file back.
There is no header parse (rate, channels, bits, data-chunk offset, frames) and
no PCM16 decode to `[float]`.

Every consumer that *measures* audio — RMS/peak levels, silence detection,
duration from a real file rather than from caller-known parameters —
re-implements the RIFF walk over `std/bytes.toInts`. Reported by the
sunholo-audio package (stapledons-godot, `design/sunholo-audio`):

- `toInts` materialises a **list of one int per byte** — a 1.5M-sample file
  (3 MB) becomes a ~3M-element intermediate list, just to pair bytes back
  into s16 samples.
- Measuring 19 files (10 MB) this way takes **~10 s on the bytecode VM**
  (reported externally, not re-measured here). It works — but it is the
  same "map pass over megabytes in interpreted AILANG" that M-STD-AUDIO
  already ruled out for the codec: *"only the codec lives in Go, because …
  a single map pass over 50 s of PCM already costs 3.5 s"*.
- Until std ships a read side, sunholo-audio **carries its own WAV parser**,
  a private re-implementation of a std-shaped problem — the exact
  fragmentation this repo's std exists to prevent.

`durationMs` — the closest existing thing — is unusable for this: it needs
`rate`/`channels`/`bits` as *inputs*, which is precisely what a header parse
is supposed to produce.

**Current State (all verified, see Verification Log):**
- `std/audio.ail` (71 lines) exports exactly four things: `Codec`,
  `wavHeader`, `wavFromPcm`, `durationMs`, `encode`. Nothing reads.
- `grep -rn 'wavInfo\|WavInfo\|parseWav' std/ internal/ examples/ docs/` →
  empty. No RIFF reader exists anywhere in the repo (V2).
- The only WAV parser bytes in the tree are the ones `wavHeader` *writes*
  (`internal/builtins/bytes_ints.go` is `fromInts`/`toInts`, not a parser).

**Impact:** any AILANG program handed a WAV file (a tool's output, a
recording, a fixture) must either know the parameters out-of-band or
re-implement the parser. Affected today: sunholo-audio (external); any
future eval fixture that wants to assert on real audio files.

## Goals

**Primary Goal:** `std/audio` gains a read side symmetric with its write side:
`wavInfo(bytes) -> Result[WavInfo, string]` and
`pcm16ToFloats(bytes, offset, samples) -> Result[[float], string]`, such that
`wavInfo(wavFromPcm(pcm, r, c, b))` recovers `r`, `c`, `b` and the decode is
bit-exact `s16/32768`.

**Success Metrics:**
- `wavInfo` returns rate/channels/bits/dataOffset/frames for canonical WAVs
  **and** for non-canonical layouts (fmt after data; unknown chunks before,
  between and after; odd-size chunks with pad byte) — verified fixtures below.
- `pcm16ToFloats` is bit-exact: `-32768 → -1.0`, `32767 → 0.999969482421875`,
  `1 → 3.0517578125e-05` (division by 32768.0, a power of two, is exact).
- The decode is a single Go pass with **no intermediate `[int]` list**;
  the 19-file/10 MB measure corpus drops from ~10 s (reported) to
  < 1 s on the VM (measured at sprint time).
- Identical output on evaluator and `--bytecode` (byte-for-byte; the VM
  adapted-builtin path calls the *same* Go `Impl`).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|----------------|-----------|----------|-------------|
| `wavInfo` is pure AILANG over `std/bytes` (no new builtin) | Sets the split between "std-shaped logic in AILANG" vs "bulk work in Go" for the whole module | human (module precedent) | design | low |
| `pcm16ToFloats` is a Go builtin (`_audio_pcm16_to_floats`) | One pass over MBs vs an interpreted map pass (3.5 s precedent for 50 s PCM) | human (module precedent) | design | med |
| `WavInfo` is a closed record `{rate, channels, bitsPerSample, dataOffset, frames}` | The record shape is the public API every consumer binds to; add-only later | human | design | high |
| `frames` = `min(declared dataLen, bytes present) / frameSize` (floor) — report what the file *holds*, not what the header *claims* | Truncated-file policy: silent-vs-loud behaviour for every measurement consumer | human | design | med |
| RIFF walk is general (any chunk order, skip-by-size, odd-size pad), not canonical-layout assumption | Wrong assumption = rejects real tool output; Err names the first broken invariant | human | design | med |
| Decode window signature takes `samples` (int16 count), not `frames` | Channel-agnostic primitive; stereo callers pass `frames * channels` (explicit at the call site) | human | design | low |
| No `GoCodegenSpec` for the new builtin (parity with `encode`) | A spec requires a `_audio_` row in `stdlibBuiltinFamilyPrefix` + a second hand-written Go body | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] `wavInfo` in pure AILANG; `pcm16ToFloats` as the only new Go builtin.
- [x] `WavInfo` record shape and the truncation policy (`frames` from bytes
      present; the declared `dataLen` is never trusted past EOF).
- [x] General chunk walk with Err messages naming the broken invariant.
- [x] `Result[[float], string]` for the decode (matches `encode` and
      `_vec_decode_f32le/f64le`; nothing clamps, nothing aborts).
- [x] fmt tag 1 (integer PCM) only; other tags (incl. 0xFFFE extensible) are
      `Err` naming the tag — not a silent fallback.

## Deferred Decisions

The following are intentionally left open for the implementer:

- New Go file (`internal/builtins/audio_pcm16.go`) vs extending
  `internal/builtins/audio.go` — agent may choose (audio.go is 90 lines;
  a sibling file matches `float_codec.go`'s one-concern-per-file pattern).
- Exact wording of the `Err` strings — agent may choose, provided each names
  the field/invariant it broke (the module's existing convention).
- Whether the AILANG-side `u16At`/`u32At` helpers stay private in
  `std/audio.ail` or are also exported — agent may choose (default: private).
- Test fixture placement: extend `tests/stdlib/audio_wav_test.ail` vs a new
  `tests/stdlib/audio_wav_read_test.ail` — agent may choose.

## Solution Design

### Overview

Two additions to `std/audio`, mirroring the write side:

1. **`wavInfo(b: bytes) -> Result[WavInfo, string]`** — pure AILANG over
   `std/bytes.byteAt`. A ~45-line recursive RIFF chunk walk: verify the
   `RIFF`/`WAVE` magic, then walk 8-byte chunk headers (`id`, `size`), skip
   unknown chunks by size (with the odd-size pad byte), capture the first
   `fmt ` chunk's fields and the first `data` chunk's window, and finish with
   an `Err` naming whatever is missing. `O(chunks)`, never touches the PCM.

2. **`pcm16ToFloats(b: bytes, offset: int, samples: int) -> Result[[float], string]`**
   — a Go builtin that decodes `samples` little-endian signed 16-bit values
   starting at byte `offset`, each divided by exactly `32768.0`, in one pass,
   into a `[float]` list. Domain violations return `Err` naming the field,
   matching `encode` and the `_vec_decode_*` codecs.

### Architecture

**Components:**

1. **`WavInfo` record (std/audio.ail)** — closed record, what every consumer
   needs from a header:
   `{rate: int, channels: int, bitsPerSample: int, dataOffset: int, frames: int}`.
   `frames` is *whole frames present*: for a stereo 16-bit file, one frame is
   `channels * (bitsPerSample / 8)` = 4 bytes.
2. **The walk (std/audio.ail, private)** — `u16At`/`u32At` little-endian
   readers over `byteAt` (each `Option[int]` unwrap via `getOrElse` with 0 —
   the guards below make the `None` branch unreachable), and `walk`, which
   carries `rate/channels/bits/dataOff/dataLen` forward. Sentinel encoding:
   `channels == 0` ⇒ "no fmt seen", `dataOff < 0` ⇒ "no data seen" — both
   impossible for a *parsed* fmt chunk (fmt validation rejects `ch <= 0`).
3. **`_audio_pcm16_to_floats` (Go)** — registered like the float codecs
   (`internal/builtins/float_codec.go:85`): type
   `T.Func(T.Bytes(), T.Int(), T.Int()).Returns(T.App("Result", floatListType(), T.String()))`,
   impl unpacks `int16(binary.LittleEndian.Uint16)` → `float64(v) / 32768.0`
   (a power-of-two divide — exact for every int16), and returns
   `resultOk(floatsValue(xs))` / `resultErr(msg)`.

Chunk-id comparison detail (verified, V6): AILANG has **no `Eq[bytes]`**
(`ailang check` rejects `==` on bytes), so chunk ids are compared as the
little-endian `u32` the readers already produce, against hex magic constants:
`RIFF` = `0x46464952`, `WAVE` = `0x45564157`, `fmt ` = `0x20746D66`,
`data` = `0x61746164`.

**Proposed `std/audio.ail` addition (this exact code `ailang check`s clean
and passes every fixture below on evaluator and `--bytecode`, V4–V7):**

```ailang
-- New imports for the module (byteAt joins the existing std/bytes import):
import std/bytes (fromInts, concat, length, byteAt)
import std/option (Option, Some, None, getOrElse)
import std/string (intToStr)

-- What a consumer needs from a RIFF/WAVE header: the format fields, where
-- the PCM starts, and how many whole frames the file actually holds.
export type WavInfo = { rate: int, channels: int, bitsPerSample: int, dataOffset: int, frames: int }

-- Little-endian unsigned readers for the header walk (O(chunks), tiny data).
-- The guards in wavInfo/walk keep every index in range, so the getOrElse
-- default is unreachable.
pure func u16At(b: bytes, off: int) -> int
requires { off >= 0 && off + 2 <= length(b) } {
  getOrElse(byteAt(b, off), 0) + 256 * getOrElse(byteAt(b, off + 1), 0)
}

pure func u32At(b: bytes, off: int) -> int
requires { off >= 0 && off + 4 <= length(b) } {
  u16At(b, off) + 65536 * u16At(b, off + 2)
}

-- Walk RIFF chunks from `off`. First fmt chunk supplies the fields, first
-- data chunk the payload window. channels > 0 marks "fmt seen", dataOff >= 0
-- marks "data seen" (fmt validation rejects channels <= 0, so the sentinels
-- are unambiguous). Chunks are skipped by their declared size; odd sizes
-- carry one pad byte per RIFF.
pure func walk(b: bytes, off: int, rate: int, channels: int, bits: int, dataOff: int, dataLen: int) -> Result[WavInfo, string] {
  if off + 8 > length(b) then
    if dataOff < 0 then Err("WAV has no data chunk")
    else if channels == 0 then Err("WAV has no fmt chunk")
    else {
      let frameSize = channels * (bits / 8)
      let present = if dataLen < length(b) - dataOff then dataLen else length(b) - dataOff
      Ok({rate: rate, channels: channels, bitsPerSample: bits, dataOffset: dataOff, frames: present / frameSize})
    }
  else {
    let tag = u32At(b, off)
    let size = u32At(b, off + 4)
    let next = off + 8 + size + (size % 2)
    if tag == 0x20746D66 then
      if channels > 0 then walk(b, next, rate, channels, bits, dataOff, dataLen)
      else if off + 24 > length(b) then Err("fmt chunk is truncated")
      else {
        let fmt = u16At(b, off + 8)
        let r = u32At(b, off + 12)
        let ch = u16At(b, off + 10)
        let bi = u16At(b, off + 22)
        if fmt != 1 then Err("fmt chunk: format tag ${intToStr(fmt)} is not 1 (PCM)")
        else if ch <= 0 || r <= 0 || bi <= 0 || bi % 8 != 0 then Err("fmt chunk: bad rate/channels/bitsPerSample")
        else walk(b, next, r, ch, bi, dataOff, dataLen)
      }
    else if tag == 0x61746164 then
      if dataOff >= 0 then walk(b, next, rate, channels, bits, dataOff, dataLen)
      else walk(b, next, rate, channels, bits, off + 8, size)
    else walk(b, next, rate, channels, bits, dataOff, dataLen)
  }
}

-- Parse a RIFF/WAVE (fmt tag 1, integer PCM) header. frames counts whole
-- frames present in the file (min of the declared data size and the bytes
-- after dataOffset, floored to whole frames) — a truncated file reports the
-- frames it holds, not the frames its header claims.
export pure func wavInfo(b: bytes) -> Result[WavInfo, string] {
  if length(b) < 12 then Err("too short for a RIFF header")
  else if u32At(b, 0) != 0x46464952 then Err("not a RIFF file")
  else if u32At(b, 8) != 0x45564157 then Err("RIFF file is not WAVE")
  else walk(b, 12, 0, 0, 0, -1, -1)
}

-- Decode `samples` little-endian signed 16-bit values starting at byte
-- `offset` (for stereo, pass frames * channels), each divided by exactly
-- 32768.0. Bit-exact: 32768 is a power of two, so the division is exact for
-- every int16; -32768 -> -1.0, 32767 -> 0.999969482421875.
-- Err names the field; nothing is clamped or resampled.
export pure func pcm16ToFloats(b: bytes, offset: int, samples: int) -> Result[[float], string] {
  _audio_pcm16_to_floats(b, offset, samples)
}
```

**Go builtin (sketch — follows `float_codec.go`'s decode registration):**

```go
// internal/builtins/audio_pcm16.go
err := RegisterEffectBuiltin(BuiltinSpec{
    Module: "std/audio", Name: "_audio_pcm16_to_floats",
    NumArgs: 3, IsPure: true,
    Type: func() types.Type {
        T := types.NewBuilder()
        return T.Func(T.Bytes(), T.Int(), T.Int()).Returns(
            T.App("Result", floatListType(), T.String())).Build()
    },
    Impl: audioPcm16ToFloatsImpl,
    Metadata: &BuiltinMetadata{ ... },
})

func audioPcm16ToFloatsImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
    b, ok := args[0].(*eval.BytesValue)     // + Int checks for args 1,2, as in audioEncodeOggOpusImpl
    offset, samples := ...
    switch {
    case offset < 0:
        return resultErr("_audio_pcm16_to_floats: offset < 0"), nil
    case samples < 0:
        return resultErr("_audio_pcm16_to_floats: samples < 0"), nil
    case offset+2*samples > len(b.Value):
        return resultErr(fmt.Sprintf(
            "_audio_pcm16_to_floats: window [offset, offset+2*samples) exceeds %d bytes", len(b.Value))), nil
    }
    xs := make([]float64, samples)
    for i := range xs {
        xs[i] = float64(int16(binary.LittleEndian.Uint16(b.Value[offset+2*i:]))) / 32768.0
    }
    return resultOk(floatsValue(xs)), nil
}
```

**Why no extra size cap (unlike `encode`'s 100 MB):** the decode window is
already bounded by the enforced `offset+2*samples <= len(b)` — the caller
cannot make the builtin touch more bytes than it already holds. `encode`'s
cap exists because *encoding* allocates output proportional to input and
runs a heavy codec; a straight copy-divide needs no second bound.

**VM and transpile behaviour (both verified against the mechanism, V8):**
- Pure registry builtins whose parameters are scalars and whose return
  converts through `std Option/Result` are **auto-adapted**
  (`internal/bytecode/builtin_adapt.go:166`): the VM converts args, calls
  the registered Go `Impl` (the same function the evaluator calls), converts
  back. No native VM port is needed and output is byte-identical by
  construction. `bytes,int,int -> Result[[float],string]` qualifies.
  `TestAdaptedBuiltinsNeverPanic` will call it with sample args
  (bytes "a", 1, 1): `offset+2*samples=3 > 1` → `Err(...)` — fine, must not
  panic.
- The coverage ratchet (`internal/vm/builtin_coverage_test.go`) passes
  automatically once the builtin is in the adapted set — but the sprint must
  run `make test` to confirm the new name didn't land in `unportedPure`
  (it won't: `AdaptReason` returns "" for this signature).
- **No `GoCodegenSpec`** (AILANG→Go transpile): parity with `encode`, which
  has none — `std/audio` is not transpilable today and fails loudly when
  tried ("Missing codegen is allowed and fails loudly during compilation",
  `stdlib_codegen_resolution_test.go`). Adding a spec would additionally
  require a `"std/audio": "_audio_"` row in `stdlibBuiltinFamilyPrefix`
  *and* a second hand-written Go body to keep in sync — the exact
  dual-implementation drift M-CODEGEN-SUSTAINABILITY exists to prevent.
  Making `std/audio` transpilable is future work, done once for both
  directions with parity tests.

### Implementation Plan

**Phase 1: `std/audio.ail` — wavInfo (~2 hours)**
- [ ] Add `WavInfo`, `u16At`, `u32At`, `walk`, `wavInfo` exactly as above
      (already `ailang check`-clean in this doc's verification).
- [ ] `ailang verify std/audio.ail` still proves `durationMs`'s `ensures`.

**Phase 2: Go builtin (~4 hours)**
- [ ] Register `_audio_pcm16_to_floats` (file placement deferred).
- [ ] Table tests in `internal/builtins/audio_*_test.go`: edge values
      (-32768, -1, 0, 1, 32767 — compare with the oracle below), bounds Errs
      (negative offset/samples, window past EOF), zero-sample decode → `Ok([])`.
- [ ] Confirm `make test` (VM coverage ratchet + never-panic + resolution
      tests) stays green.

**Phase 3: integration + docs (~4 hours)**
- [ ] Extend `tests/stdlib/audio_wav_test.ail` (or sibling) with the fixture
      set from this doc's Verification Log (V4–V7): canonical round-trip,
      adversarial layout, truncated file, non-RIFF Err, bad-window Err.
- [ ] New runnable example `examples/runnable/std_audio_measure.ail`:
      read a WAV with `std/fs.readFileBytes`, `wavInfo`, decode, report
      duration + peak/RMS via `std/math.sqrt` + `foldl`, printed with
      `std/string.formatFixed` (lands with M-STD-STRING-FORMAT-FIXED).
- [ ] Update `docs/docs/reference/std-audio.md`: two new table rows + a
      `wavInfo`/`pcm16ToFloats` section with the Err conventions.
- [ ] Changelog entry in `changelogs/v0.32-current.md` under v0.54.17.

### Files to Modify/Create

**Modified files:**
- `std/audio.ail` (+~65 LOC) — `WavInfo`, readers, `walk`, `wavInfo`, `pcm16ToFloats` wrapper, import additions.
- `internal/builtins/audio.go` or new `audio_pcm16.go` (+~90 LOC) — builtin registration + impl + metadata.
- `internal/builtins/audio_test.go` or new test file (+~80 LOC) — table tests.
- `tests/stdlib/audio_wav_test.ail` (+~60 LOC) — RIFF-parse fixtures (or a new sibling file).
- `docs/docs/reference/std-audio.md` (+~40 LOC) — function rows + semantics section.
- `changelogs/v0.32-current.md` (+~15 LOC) — v0.54.17 entry.

**New files:**
- `examples/runnable/std_audio_measure.ail` (~40 LOC) — the consumer pattern end-to-end.

## Conflict Surface

### Syntactic positions touched

No parser, lexer, type, elaboration, codegen or eval changes. The touched
"positions" are three namespaces:

1. `std/audio`'s export set (new names: `WavInfo`, `wavInfo`, `pcm16ToFloats`).
2. The builtin registry name `_audio_pcm16_to_floats`.
3. `internal/vm`'s adapted-builtin index (new entry, appended automatically
   from the registry — `BuiltinNames` indices unchanged).

### What else lives in that position

| Position | Existing occupants | Coexistence |
|---|---|---|
| `std/audio` exports | `Codec`, `wavHeader`, `wavFromPcm`, `durationMs`, `encode` | Add-only; no shadowing (V10: names free repo-wide) |
| `_audio_*` builtin family | `_audio_encode_ogg_opus` | Sibling naming; no dispatch ambiguity (registry is keyed by exact name) |
| Adapted builtins (`AdaptedBuiltinNames`) | e.g. `_vec_decode_f64le` | Index appended in sorted order at init; VM dispatch table rebuilt from the same slice — no manual index to keep in sync |
| `Result[[float], string]` decode returns | `_vec_decode_f32le/f64le` (`float_codec.go`) | Same shape; `sampleValue`-driven never-panic test covers the new entry automatically |

### Disambiguation strategy

None needed beyond the registry's exact-name keying: `wavInfo`/`pcm16ToFloats`
are fresh identifiers, the builtin is called only through its wrapper (the
`StdlibIndex` maps `std/audio.pcm16ToFloats → _audio_pcm16_to_floats` only if
a `GoCodegenSpec` claims it — deliberately none, V8), and the VM adapted set
is derived, not hand-listed. `walk`'s sentinels (`channels == 0`,
`dataOff < 0`) cannot collide with parsed values because fmt validation
rejects `ch <= 0` before a value can be carried (and `dataOff` starts at the
guarded `off + 8 <= length(b)` product, so `dataOff >= 0` always).

### Programs that MUST still work

- `tests/stdlib/audio_wav_test.ail` — the write-side goldens (`monoHeaderIsGolden`,
  `stereoHeaderIsGolden`, `wavLength`, `pcmSurvives`, `monoMs`): exact
  expected bytes must not shift. (Read; asserted values verified V11.)
- `examples/runnable/std_audio_brief.ail` — the write+encode demo
  (`saveBrief`, `convert` entrypoints): unchanged output.
- `examples/runnable/ai_call_speech.ail` — TTS → `wavFromPcm` path.
- `std/ai.ail`'s `callSpeech` docs referencing `std/audio` (line 162).
- `std/audio.ail` itself: `ailang verify std/audio.ail` must keep proving
  `durationMs`'s `ensures { result >= 0 }`.

### What deliberately changes

Nothing. No existing program changes meaning: both additions are new names;
the write side is untouched byte-for-byte. Anything not listed here that
breaks is a regression, not an intentional change.

## Examples

### Before (sunholo-audio's private parser, ~40 lines re-implemented per consumer)

```ailang
-- no std support: walk bytes.toInts(wav) by hand, pair up samples,
-- hardcode offsets 22/24/34, pray fmt comes before data...
```

### After (this design — verified end-to-end in V4–V7)

```ailang
import std/audio (wavInfo, pcm16ToFloats)
import std/fs (readFileBytes)
import std/list (foldl, length)
import std/math (sqrt, abs_Float, intToFloat)
import std/result (Ok, Err)

match readFileBytes("clip.wav") {
  Err(e) => println("read failed: ${e}"),
  Ok(wav) => match wavInfo(wav) {
    Err(e) => println("not a WAV I can read: ${e}"),
    Ok(info) => match pcm16ToFloats(wav, info.dataOffset, info.frames * info.channels) {
      Err(e) => println("decode failed: ${e}"),
      Ok(xs) => {
        let n = intToFloat(length(xs))
        let energy = foldl(\acc x. acc + x * x, 0.0, xs)
        println("${info.frames} frames @ ${info.rate} Hz, ${info.channels} ch")
        println("rms=${formatFixed(sqrt(energy / n), 4)}")
      }
    }
  }
}
```

Expected on the V4 fixture (24 000 Hz mono 16-bit, 4 frames):
`rate=24000 ch=1 bits=16 off=44 frames=4`, decode
`[-1, 0.999969482421875, 3.0517578125e-05, 6.103515625e-05]`.

## Success Criteria

- [ ] `wavInfo` parses canonical, adversarial and truncated fixtures (V4–V7 set) with the exact expected fields.
- [ ] `pcm16ToFloats` is bit-exact `s16/32768` on the edge table (-32768/-1/0/1/32767).
- [ ] Every domain violation returns `Err` naming the field (no clamps, no aborts, no silent zeros).
- [ ] Evaluator and `--bytecode` outputs byte-identical on the example.
- [ ] `make test` green (incl. VM coverage ratchet, never-panic, resolution tests); `make fmt`, `make lint`.
- [ ] `docs/docs/reference/std-audio.md` and the v0.54.17 changelog entry updated.
- [ ] Decode+measure over a 10 MB corpus < 1 s on the VM (measured; was ~10 s reported via `toInts`).

## Testing Strategy

**Unit tests (Go):**
- Table: `(-32768)→-1.0`, `(32767)→0.999969482421875`, `(1)→1/32768`,
  `(0)→0.0`, `(0x8001 = -32767)→-0.999969482421875`; mixed stereo interleave.
- Bounds: `offset<0`, `samples<0`, `offset+2*samples > len` → `Err` naming
  the field; `samples=0` → `Ok([])`.
- `TestAdaptedBuiltinsNeverPanic` (existing) sweeps the new entry with
  `("a", 1, 1)` — expected `Err`, never a panic.

**Integration tests (AILANG):**
- The V4–V7 fixtures as `tests` cases: round-trip
  `wavInfo(wavFromPcm(...))`, adversarial layout (LIST odd-size chunk, fmt
  after data, stereo 48 kHz → off=34 frames=2), truncated file (frames from
  bytes present), non-RIFF `Err`, oversized window `Err`.
- `ailang run` and `ailang run --bytecode` on the new example: byte-identical
  stdout.

**Regression-surface tests:**
- Existing `tests/stdlib/audio_wav_test.ail` goldens unchanged (write side).
- `examples/runnable/std_audio_brief.ail` and `ai_call_speech.ail` still run
  (`make verify-examples`).

**Manual testing:**
- `ailang repl`: `wavInfo(readFileBytes("real.wav"))` on a real file from
  any tool (ffmpeg/python `wave`/CoreAudio output).

## Verification Log

Premises checked before this doc was written (binary v0.54.16, std v0.54.16,
2026-10-22; no Go toolchain in the reporting environment — language claims
verified with the installed `ailang` binary, mechanism claims by reading the
cited code):

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | `std/audio` ships write-side only (`wavHeader`, `wavFromPcm`, `durationMs`, `encode`) | read `std/audio.ail` in full (71 lines) | Confirmed |
| V2 | No RIFF reader / `wavInfo` / PCM decode exists anywhere in the repo | `grep -rn 'RIFF\|WavInfo\|parseWav\|wavParse'` over `std/ internal/ examples/ tools/ docs/` → only *write-side* hits: `std/audio.ail` (the writer), `internal/builtins/bytes_ints.go` (a `fromInts` metadata example that *builds* RIFF bytes), `std/bytes.ail` doc comment, `examples/runnable/http_put_bytes.ail` (uploads 4 RIFF magic bytes), docs | Confirmed |
| V3 | Consumers re-implement over `toInts`; ~3M-element list for 1.5M samples; 19 files/10 MB ≈ 10 s | reported by sunholo-audio (external); list scale re-derived: 1 sample = 2 bytes = 2 ints ⇒ 3M ints for 1.5M samples | Reported (list arithmetic re-confirmed) |
| V4 | The proposed `wavInfo` compiles and parses a canonical `wavFromPcm` WAV: `rate=24000 ch=1 bits=16 off=44 frames=4` | wrote the doc's exact code as a scratch module; `ailang check` clean; `ailang run` fixture | Confirmed (transcript in doc discussion) |
| V5 | Adversarial layout parses: LIST(odd size 5)+pad before data, fmt AFTER data, stereo 48 kHz → `rate=48000 ch=2 bits=16 off=34 frames=2` | built the bytes with `fromInts`/`le32`; ran `wavInfo` | Confirmed |
| V6 | Truncated file reports frames actually present: 44+8-byte WAV cut to 48 bytes → `off=44 frames=2` | `slice` + `wavInfo` | Confirmed |
| V7 | The reference decode is bit-exact `s16/32768` and identical on both engines: `[-1, 0.999969482421875, 3.0517578125e-05, 6.103515625e-05]` | pure-AILANG oracle (`map(\i. intToFloat(s16At(...))/32768.0, range(0, samples))`) run on evaluator **and** `--bytecode` — byte-identical | Confirmed |
| V8 | `bytes,int,int -> Result[[float],string]` auto-adapts to the VM (same Go `Impl` both engines) | read `internal/bytecode/builtin_adapt.go:45–80, 166–191` (scalar params; `Result` converts via `StdADTTag`); `internal/vm/builtin_coverage_test.go:31–46` ratchet | Confirmed |
| V9 | `floatsValue` / `resultOk` / `resultErr` / `floatListType` exist for the Go impl sketch | read `internal/builtins/float_codec.go:44–53, 85–105` | Confirmed |
| V10 | The names `WavInfo`, `wavInfo`, `pcm16ToFloats`, `_audio_pcm16_to_floats` are unallocated repo-wide | `grep` over `std/ internal/ cmd/ examples/ docs/` → empty | Confirmed |
| V11 | `Eq[bytes]` does not exist ⇒ chunk ids must be compared as u32 magic constants | `ailang check` on a version comparing `slice(...)==fromString("fmt ")`: `No instance for Eq[bytes] in scope` | Confirmed (failure was the finding) |
| V12 | Existing WAV test asserts what this doc says it asserts | read `tests/stdlib/audio_wav_test.ail` (golden headers, round-trip, durationMs cases) | Confirmed |
| V13 | `encode` ships without a `GoCodegenSpec`; `std/audio` has no row in `stdlibBuiltinFamilyPrefix`, so a new spec would require adding one | `grep` GoCodegen in `internal/builtins/audio.go` (none); read `stdlib_codegen_resolution_test.go:28–47, 100–115` | Confirmed |
| V14 | The fmt fields sit at chunk-relative `+8/+10/+12/+22` (bits at `+22`, NOT `+14`) | first scratch version read bits at `+14` and the fixture caught it (`bad rate/channels/bitsPerSample` — `+14` is the *rate field's high half*); fixed and re-run | Confirmed (bug found by fixture, fixed) |
| V15 | Module precedent states bulk audio work belongs in Go ("a single map pass over 50 s of PCM already costs 3.5 s") | read `std/audio.ail:1–18` and `internal/builtins/audio.go:1–12` | Confirmed |
| V16 | No new error code is proposed (Errs are `Result[..., string]` messages) | — | n/a |
| V17 | `ailang verify std/audio.ail` proves `durationMs`'s `ensures` today (regression baseline) | ran it: `✓ VERIFIED durationMs` — `4 exported functions: 1 verified, 2 skipped, 1 without contracts` | Confirmed |

## Non-Goals

- **Encoding from `[float]`** (`floatsToPcm16`) — sunholo-audio measures; when
  a synth pipeline needs it, it is a one-pass inverse with the same shape.
- **Non-16-bit decode** (8/24/32-bit PCM) — `wavInfo` parses and *reports*
  any `bitsPerSample` multiple of 8; only the decode is 16-bit. 24-bit decode
  is a follow-up with real demand.
- **WAVE_FORMAT_EXTENSIBLE (0xFFFE) and float WAVs (tag 3)** — `Err` naming
  the tag; extension is additive (the walk already carries the fields).
- **Resampling, mixing, streaming/chunked decode, Array[float] decode** —
  the `[float]` list matches the `_vec_decode_*` convention and the
  fold-based measure pattern; an `Array[float]` variant can ride the same
  builtin later if memory demands it.
- **Making `std/audio` transpilable to Go** (`GoCodegenSpec` for encode and
  decode together) — see V13; done once, with parity tests, not piecemeal.

## Timeline

**Day 1** (6h): Phase 1 `std/audio.ail` (2h) + Phase 2 Go builtin (4h).
**Day 2** (6h): Phase 2 tests (2h) + Phase 3 fixtures/example/docs (4h).
**Total: ~12 hours across 2–3 days** (estimate already 2x'd).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `[float]` list memory for very large files (50M boxed floats for 100 MB PCM) | Med | Decode windows are caller-chosen; document `Array[float]` variant as the escape hatch; the measure pattern folds without retaining |
| Adversarial chunk walks (deep chains of 8-byte headers) | Low | Walk is `O(fileSize/8)` worst case, pure, bounded by `length(b)`; each Err names the invariant |
| Sentinel confusion if a future fmt variant allows `channels=0` | Low | fmt validation rejects `ch <= 0` at parse time; comment pins the invariant in `walk` |
| Header lies (declared dataLen ≫ file) | Low | `frames` uses bytes present (min); documented, deliberate |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure parse + bit-exact decode; no ambient input, no clocks, no RNG |
| A2: Replayability | 0 | No trace/effect changes |
| A3: Effect Legibility | 0 | Everything stays `pure`; no new effects |
| A4: Explicit Authority | 0 | No capabilities touched (read side takes bytes the caller already holds) |
| A5: Bounded Verification | +1 | `requires` guards on the readers are locally checkable; fixtures pin byte-level output |
| A6: Safe Concurrency | 0 | No concurrency |
| A7: Machines First | +1 | Header facts become machine-decidable facts instead of per-consumer re-implementation drift |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | No resource accounting changes |
| A10: Composability | +1 | `wavInfo` → `pcm16ToFloats` → `std/list` folds compose with `std/fs` reads |
| A11: Structured Failure | +1 | Every failure is `Err` naming the field/invariant; no clamps, no silent zeros |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): pure functions of their bytes input
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): closing a std gap, not a human convenience

## Related Documents

**Implemented (may inform design):**
- [M-STD-AUDIO](../implemented/v0_44_0/m-std-audio.md) — the write side this doc completes (v0.44.0)

**Planned (check for overlap):**
- [M-STD-STRING-FORMAT-FIXED](./m-std-string-format-fixed.md) — same DX report (sunholo-audio); the example program prints its measurements through it

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- [std/audio reference](../../../docs/docs/reference/std-audio.md) — module docs to update
- [M-STD-AUDIO changelog](../../../changelogs/v0.32-current.md) — v0.44.0 entry
- DX report: sunholo-audio (stapledons-godot `design/sunholo-audio`), 2026-10

## Future Work

- `floatsToPcm16` — encode from `[float]` (one-pass inverse, same registry shape).
- 24/32-bit and float32 WAV decode; WAVE_FORMAT_EXTENSIBLE acceptance.
- `GoCodegenSpec` for both `std/audio` directions (transpilability, done once with parity tests).
- `Array[float]` decode variant for memory-bound consumers.

---

**Document created**: 2026-10-22
**Last updated**: 2026-10-22