---
reviewBy: 2027-02-15
---

# std/audio — checked WAV, packed PCM and Ogg Opus

The `std/audio` module (v0.44.0+) turns raw PCM, the format text-to-speech APIs
such as Gemini TTS and Live return, into files a person can play. Every function
is **pure**: no `Process`, no `FS`. Write the result with `std/fs.writeFileBytes`.

| Function | What it does |
|---|---|
| `wavHeader(dataLen, rate, channels, bitsPerSample) -> bytes` | The canonical 44-byte RIFF/WAVE header (format tag 1, integer PCM) |
| `wavFromPcm(pcm, rate, channels, bitsPerSample) -> bytes` | Validated header and PCM, with a zero pad after odd-length data |
| `wavHeaderChecked(dataLen, rate, channels, bitsPerSample) -> Result[bytes, string]` | Recoverable header validation |
| `wavFromPcmChecked(pcm, rate, channels, bitsPerSample) -> Result[bytes, string]` | Recoverable complete-file validation |
| `wavInfo(wav) -> Result[WavInfo, string]` | Validated RIFF metadata and actual data extent |
| `pcm16ToFloats(pcm, offset, frames, channels) -> Result[Array[float], string]` | Interleaved s16le to a packed array |
| `floatsToPcm16(samples, channels, policy) -> Result[bytes, string]` | Explicit checked quantization to s16le |
| `durationMs(pcmLen, rate, channels, bitsPerSample) -> int` | Clip length in whole milliseconds, rounded down |
| `encode(pcm, rate, channels, codec) -> Result[bytes, string]` | Compress 16-bit PCM; `codec` is `OggOpus(bitrate)` |

The metadata reader is pure AILANG over small header slices. Checked native
writing and bulk PCM conversion enforce validation in ordinary execution,
including when `--verify-contracts` is absent. `durationMs` retains its existing
integer flooring behavior. Static contract results describe the verifier's
encoded Real/Int model, not machine-overflow or IEEE-754 equivalence.

## WAV validation and compatibility

The writers accept integer PCM with 8, 16, 24 or 32 bits per sample, positive
rate/channels, representable RIFF/header fields and whole interleaved frames.
Materialized PCM is limited to 100 MiB. Header-only calls validate RIFF size
without allocating the conceptual data buffer. Odd data gets one zero pad byte;
the data chunk size excludes the pad and the outer RIFF size includes it. A
manual `wavHeader` caller must append that pad when writing odd data.

The legacy `wavHeader`/`wavFromPcm` signatures remain bytes-returning. They now
fail loudly on partial frames, unsupported widths, overflow or oversized
materialized buffers; use the checked forms to recover with `Err`. Accepted
even-length PCM16 header bytes are unchanged.

## WAV metadata and PCM conversion

`WavInfo` contains encoding (`IntegerPcm`), layout (`Interleaved`), rate,
channels, containerBits, validBits, blockAlign, byteRate, dataOffset, dataBytes
and frames. Channel count does not imply speaker positions. The reader accepts
fmt/data in either order and skips unknown chunks, validates their extents and
odd padding, and rejects duplicate fmt/data, incomplete frames and inconsistent
format fields. It walks only the declared RIFF extent, ignoring trailing bytes,
and refuses more than 65,536 chunks. IEEE float and extensible WAV encodings
return `Err`; they are never reinterpreted as integer PCM.

Decode exactly `frames * channels` samples from the even byte offset. For a
WAV, first check containerBits == 16 and pass the metadata dataOffset/frames.
Conversion retains all channels in file order and uses `s16 / 32768`: -32768
becomes -1.0, and 32767 becomes 0.999969482421875. No multi-million-element byte
or sample list is needed. The selected PCM extent is capped at 100 MiB, so its
float64 result can occupy 400 MiB; this is not a process-wide memory guarantee.

Encoding requires whole frames and finite samples. `RejectOutOfRange` refuses
finite values outside [-1,1]; `ClipOutOfRange` clamps them. Both policies reject
NaN/infinities. Quantization multiplies by 32768, rounds nearest with ties to
even, then saturates to [-32768,32767]. +1 maps to 32767. Every decoded s16
sample round-trips exactly. The downstream package's legacy *32767 quantizer
remains a separate compatibility API.

[The checked stereo example](https://github.com/sunholo-data/ailang/blob/dev/examples/runnable/audio_checked.ail)
shows metadata, packed conversion and recoverable invalid-frame errors.

## encode

`encode(pcm, rate, channels, OggOpus(bitrate))` writes an Ogg Opus file
(RFC 7845). The input must be interleaved 16-bit little-endian PCM.

| Argument | Accepted values |
|---|---|
| `rate` | 8000, 12000, 16000, 24000 or 48000 Hz |
| `channels` | 1 or 2 |
| `bitrate` | 6000 to 510000 bits/s |
| `pcm` | a whole number of samples, at most 100 MB |

Anything else returns `Err` with a message naming the field. Nothing is clamped
or resampled silently: 44.1 kHz audio is rejected, not converted.

**Size.** At `OggOpus(24000)`, 51 s of 24 kHz mono speech (2.4 MB of PCM, the
same as a 2.4 MB WAV) becomes about 237 KB. A three-minute brief is under 1 MB.
Raise the bitrate for better quality; 16000 roughly halves the file.

**Determinism.** The same PCM, rate, channel count and bitrate always give the
same bytes. The Ogg stream serial number is a hash of the input rather than a
random number, and the codec is a fixed-point port of libopus, so the output
is also the same on amd64, arm64 and WebAssembly. A golden SHA-256 in the Go
tests pins it.

**Codec.** The encoder is [go-opus](https://github.com/tphakala/go-opus) v1.1.0
(BSD-3-Clause), vendored under `third_party/goopus`. It is pure Go, so
`std/audio` works in the WASM build. It codes in CELT mode only, which is less
efficient than libopus's SILK mode for low-bitrate speech; 24 kbps is a good
default for narration.

Decoding the output back to PCM is not part of `std/audio`. macOS reads the
files with `afinfo`, `afplay` and `afconvert`.

## Example

[`examples/runnable/std_audio_brief.ail`](https://github.com/sunholo-data/ailang/blob/dev/examples/runnable/std_audio_brief.ail)
has a `saveBrief(pcm, stem)` function that writes `<stem>.wav` and `<stem>.ogg`,
and a `convert` entry point for a raw PCM file:

```bash
ailang run --caps IO,FS,Env --entry convert examples/runnable/std_audio_brief.ail -- brief.pcm brief
```

## Not included

- AAC/m4a: the only pure-Go encoder is LGPL, and the alternative needs cgo.
- Resampling (for example 24 kHz to 16 kHz for Live API input). Planned as a
  follow-up; see `design_docs/implemented/v0_44_0/m-std-audio.md` M3.
- Compressed-audio decoding, MP3, Vorbis and streaming (incremental) encoding.
