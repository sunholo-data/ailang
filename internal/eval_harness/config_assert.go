package eval_harness

import "fmt"

// DefaultMotokoProfile mirrors the executor's fallback when models.yml sets no
// motoko_profile (see internal/executor/motoko/motoko.go). Kept here so the
// assertion can tell "claimed nothing, got the default" (fine) apart from
// "claimed X, got Y" (a real contradiction).
const DefaultMotokoProfile = "dogfood"

// AssertResolvedProfile compares what models.yml CLAIMED a run would use
// against what the subject reports it ACTUALLY loaded, and returns an invalid
// marker when they contradict each other. Returns nil when there is nothing to
// flag.
//
// # WHY THIS EXISTS
//
// All ten cloud motoko entries set no motoko_profile, so they silently fell
// through to `dogfood` — motoko's own self-hosting profile. They ran with
// neither ailang_docs nor microrag, and with a verify gate (`make check_core`)
// that only works inside the motoko repo and is meaningless in a benchmark
// workspace. Meanwhile their models.yml descriptions advertised "DP7 verifier +
// microRAG context + extensions". The rows were real measurements of something,
// just not of the thing they claimed. Nothing caught it for weeks because the
// claim was never compared against reality.
//
// Reading the resolved value from the subject's own step-0 broadcast — rather
// than from the config we passed in — is what makes this a check rather than a
// tautology.
//
// Both "" cases are deliberately permissive:
//   - resolved == "" means the executor does not broadcast a profile, so there
//     is nothing to compare. M4 is opt-in per executor, like the canary.
//   - claimed == "" means models.yml asserted nothing, so nothing is
//     contradicted whatever the subject loaded.
func AssertResolvedProfile(claimed, resolved string) *Validity {
	if claimed == "" || resolved == "" {
		return nil
	}
	if claimed == resolved {
		return nil
	}
	v := MarkInvalid(ReasonConfigMismatch)
	v.Detail = fmt.Sprintf(
		"models.yml claimed motoko_profile %q but the run resolved profile %q — this row does not measure what it says it does",
		claimed, resolved)
	return v
}
