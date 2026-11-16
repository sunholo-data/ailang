// Package std provides embedded access to AILANG standard library files.
// This enables WASM builds to include stdlib without filesystem access.
package std

import "embed"

// FS contains all .ail files from the stdlib directory.
// These are embedded at compile time using Go's embed package.
//
//go:embed *.ail ai/*.ail stream/*.ail
var FS embed.FS

// Stability is std/STABILITY.toml: the tier (stable|experimental|internal) of
// every std module. internal/iface reads it to stamp `"tier"` on interfaces.
//
//go:embed STABILITY.toml
var Stability []byte
