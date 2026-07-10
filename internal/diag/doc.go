// Package diag is the home of the footgun coverage table and its CI enforcement.
//
// AILANG's edge over general-purpose languages for AI code synthesis is
// error-time teaching: when a model hits a footgun, the diagnostic should state
// the rule and carry the concrete fix, so a single self-repair round can act on
// it. footguns.md inventories every known footgun (sourced from the teaching
// prompt's Common Mistakes + docs/LIMITATIONS.md) and maps it to its current
// diagnostic, its target diagnostic, a fixture, the prompt lines it would let us
// delete, and a coverage status.
//
// The CI contract lives in footgun_fixtures_test.go: every "covered" or
// "shipped-this-sprint" entry has a fixture asserting both the diagnostic code
// and that the message/suggestion carries the fix substring. This converts a
// per-run prompt tax into an error-time diagnostic that is a tested contract.
//
// See design_docs/planned/v0_29_0/m-diagnostic-coverage.md (R1.1).
package diag
