// Package check holds the static analyses `ailang check --package` and
// `ai-check` run on top of the type checker: unresolved module-level
// references (M-PKG-INTERREF), STRICT_FALLBACK_001 promotion at the publish
// boundary, package source discovery, and the per-file package check loop.
//
// It is not the type checker and it is not the CLI: compile phases live in
// internal/pipeline (the sibling to use for anything that parses, elaborates
// or type-checks), and flag parsing, colours and exit codes stay in cmd/ailang.
package check
