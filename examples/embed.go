// Package examples embeds the searchable examples corpus — manifest.json and
// the gated runnable/ tree — so `ailang examples` (and policy-tool's
// examples_* ops) work from a release binary with no corpus on disk (#1552).
// It follows std/embed.go: the files are embedded from where they live, so
// there is no copy to sync and nothing to drift.
package examples

import "embed"

// Corpus holds manifest.json and runnable/**. Its root mirrors the layout
// `ailang examples download` extracts to ~/.ailang/examples.
//
//go:embed manifest.json runnable
var Corpus embed.FS
