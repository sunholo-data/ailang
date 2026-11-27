// Package oggopus implements the RFC 7845 Ogg Opus container: an io.Writer-based
// encoder and an io.Reader-based decoder wrapping the raw opus codec. This is
// the BirdNET-Go deliverable. See docs/api-design.md.
//
// Decode output is always 48 kHz interleaved little-endian int16 PCM
// (OutputSampleRate), independent of the encoded stream's original input rate;
// Info.InputSampleRate is informational only.
package oggopus
