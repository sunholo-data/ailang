# std/audio — PCM to WAV and Ogg Opus

The `std/audio` module (v0.44.0+) turns raw PCM, the format text-to-speech APIs
such as Gemini TTS and Live return, into files a person can play. Every function
is **pure**: no `Process`, no `FS`. Write the result with `std/fs.writeFileBytes`.

| Function | What it does |
|---|---|
| `wavHeader(dataLen, rate, channels, bitsPerSample) -> bytes` | The canonical 44-byte RIFF/WAVE header (format tag 1, integer PCM) |
| `wavFromPcm(pcm, rate, channels, bitsPerSample) -> bytes` | `wavHeader` followed by the PCM, unchanged |
| `durationMs(pcmLen, rate, channels, bitsPerSample) -> int` | Clip length in whole milliseconds, rounded down |
| `encode(pcm, rate, channels, codec) -> Result[bytes, string]` | Compress 16-bit PCM; `codec` is `OggOpus(bitrate)` |

The WAV helpers are plain AILANG over `std/bytes`. `durationMs` carries an
`ensures { result >= 0 }` contract that `ailang verify std/audio.ail` proves.

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
  follow-up; see `design_docs/planned/v0_44_0/m-std-audio.md` M3.
- Decoding, MP3, Vorbis and streaming (incremental) encoding.
