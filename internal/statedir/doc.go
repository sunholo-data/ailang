// Package statedir resolves where AILANG keeps its per-user state: the
// coordinator, collaboration and observatory SQLite databases, mission pid
// files, quota ledgers, heartbeats and the rest of what lives under
// ~/.ailang/state today.
//
// Resolution, in order:
//
//  1. AILANG_STATE_DIR, when set — used verbatim.
//  2. $HOME/.ailang/state, with $HOME taken from os.UserHomeDir (USERPROFILE on
//     Windows, so tests must set the home through testutil.SetHomeDir).
//
// When neither resolves, Dir returns an error. It never returns "" or "." —
// a relative fallback silently creates a second, private state tree in
// whatever directory the process happens to be in, which is how `ailang
// coordinator status` and `ailang chains list` came to disagree about which
// coordinator.db was the real one (M-V1-SIMPLIFY-S2 M4, 2026-09-15).
//
// This package is the ONLY place that reads AILANG_STATE_DIR and the only
// place that spells out the ".ailang/state" suffix. It is a leaf: standard
// library only, so every store implementation (internal/storage,
// internal/coordinator, internal/messaging, internal/observatory) can import
// it without a cycle. leaf_test.go enforces that.
//
// Project-scoped state — the .ailang/state directory INSIDE a repository,
// holding tracked sprint JSON, evaluations and the project brain — is a
// different tree with a different lifetime. Project addresses it explicitly
// by root so the two are never confused.
package statedir
