package main

import (
	"github.com/sunholo-data/ailang/internal/platform/sharedmem"
	"github.com/sunholo-data/ailang/internal/platform/streamws"
)

// registerPlatform wires the platform backends into the language core's
// registration seams. The core (internal/effects) defines the contracts and
// links neither cgo sqlite nor a websocket library; this binary supplies both.
// Called once at the top of main, before any command dispatch — and from the
// cmd test binary's TestMain, which never runs main. Without it, brain
// commands and ws:// Stream.connect fail with effects.ErrBackendNotRegistered.
//
// M-V1-SIMPLIFICATION-PROGRAM Phase 1.3: one place to look for what the
// binary adds to the core.
func registerPlatform() {
	sharedmem.Register() // persistent SharedCache / BrainStore backend (SQLite)
	streamws.Register()  // ws:// and wss:// transport for the Stream effect
}
