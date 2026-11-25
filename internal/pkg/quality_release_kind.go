package pkg

import "fmt"

// ReleaseKindGateHardFrom is the first ailang version at which PUB025 blocks a
// publish. Below it PUB025 is a badge: the same one-minor grace window as
// PUB001/PUB002 (ReleaseGateHardFrom), per M-PKG-PARALLEL-AGENTS D5.
const ReleaseKindGateHardFrom = "0.64.0"

// ReleaseKindGatesHard reports whether PUB025 blocks at the given ailang
// version string (same parsing as ReleaseGatesHard; "dev" is in grace).
func ReleaseKindGatesHard(ailangVersion string) bool {
	return gatesHardFrom(ailangVersion, ReleaseKindGateHardFrom)
}

// releaseKindVsMeasured raises PUB025 when the declared [release] kind
// contradicts the measured change class. A non-breaking claim over a measured
// C (an export or contract removed or changed) is the dangerous direction: it
// warns, then gates. "breaking" over a measured A only over-warns dependents,
// so it stays informational. U and "" (not measurable) raise nothing.
func releaseKindVsMeasured(r *QualityReport, kind string, in QualityInputs) {
	against := in.MeasuredAgainst
	if against == "" {
		against = "the previous version"
	}
	switch {
	case in.MeasuredChangeClass == "C" && (kind == "fix" || kind == "security" || kind == "feature"):
		r.finding("PUB025", in.ReleaseKindGatesHard, "warn", fmt.Sprintf(
			"[release] kind = %q, but the measured change against %s is class C (an export or contract was removed or changed): declare kind = \"breaking\" — dependents' cascade routes it to review either way", kind, against))
	case in.MeasuredChangeClass == "A" && kind == "breaking":
		r.badge("PUB025", "info", fmt.Sprintf(
			"[release] kind = \"breaking\", but nothing exported changed against %s (class A)", against))
	}
}
