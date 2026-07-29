package eval_analysis

import (
	"fmt"
	"math"
)

// WHY PAIRED ANALYSIS
//
// The weekly A/B compared two aggregate pass RATES. That discards the pairing
// and leaves the between-benchmark variance — which dominates — in the error
// term. At n=84 and p~0.73 the standard error on one arm is 4.8pp and on an
// unpaired difference 6.8pp, so nothing below roughly +/-13pp can reach
// significance. The three usable microRAG deltas (-3.1, -4.8, +13.1) are all
// inside that band, with the "positive" sitting exactly at the threshold.
// Running the same design weekly for another year would not resolve it.
//
// Pairing on the benchmark cancels that variance: a benchmark that both arms
// pass, or both fail, carries no information about the treatment. Only the
// DISCORDANT pairs do. That is McNemar's test, and it costs no extra GPU time —
// it is a better reading of runs we already do.

// MinDiscordantForPValue is the floor below which no p-value is reported.
//
// With only a handful of discordant pairs any p-value is false precision. The
// counts are always reported so a reader can see the evidence base; the
// statistic is withheld until there is one.
const MinDiscordantForPValue = 10

// exactBinomialMaxN is the b+c below which the exact binomial test is used
// instead of the chi-square approximation. Chi-square relies on a normal
// approximation that is unreliable for small discordant counts.
const exactBinomialMaxN = 25

// Pair is one benchmark+trial observed under both arms.
type Pair struct {
	ID      string `json:"id"`
	Lang    string `json:"lang"`
	Trial   int    `json:"trial"`
	OnPass  bool   `json:"on_pass"`
	OffPass bool   `json:"off_pass"`
}

// PairedResult is the full comparison: the aggregate delta that has always been
// reported, PLUS the paired counts that make it interpretable.
type PairedResult struct {
	Pairs []Pair `json:"pairs"`

	// Aggregate figures, preserved unchanged so the existing *_ab.jsonl trend
	// stays comparable across this schema change. M3 is additive.
	OnPass   int     `json:"on_pass"`
	OnTotal  int     `json:"on_total"`
	OffPass  int     `json:"off_pass"`
	OffTotal int     `json:"off_total"`
	DeltaPP  float64 `json:"delta_pp"`

	// Discordant counts: b = only the ON arm passed, c = only OFF passed.
	OnlyOnPassed  int `json:"only_on_passed"`
	OnlyOffPassed int `json:"only_off_passed"`

	// Unpaired counts benchmarks present in one arm but not the other. Surfaced
	// rather than silently dropped: dropping them biases the comparison, and a
	// rising unpaired count is itself a signal that one arm is failing to run.
	Unpaired int `json:"unpaired"`

	McNemar McNemarResult `json:"mcnemar"`

	// Headroom flags a control arm with too little room to move for the
	// comparison to resolve anything. Advisory; never blocks.
	Headroom Headroom `json:"headroom"`
}

// McNemarResult reports the test over the discordant pairs.
type McNemarResult struct {
	// Reportable is false when there is too little discordance to say anything.
	Reportable bool `json:"reportable"`
	// Method is "exact_binomial" or "chi_square_continuity".
	Method string `json:"method,omitempty"`
	// Statistic is the chi-square value (chi-square method only).
	Statistic float64 `json:"statistic,omitempty"`
	// PValue is two-sided. Zero when not Reportable.
	PValue float64 `json:"p_value,omitempty"`
	// Note explains an unreportable result.
	Note string `json:"note,omitempty"`
}

// passed reports whether a row counts as a success, using the same three-way
// definition the rest of the pipeline uses.
func passed(r *BenchmarkResult) bool {
	return r.CompileOk && r.RuntimeOk && r.StdoutOk
}

type pairKey struct {
	ID    string
	Lang  string
	Trial int
}

// PairArms joins two arms on (benchmark, lang, trial).
//
// Trial MUST be in the key: both trials of a benchmark share (id, lang, seed),
// so a join without it would pair trial 1 of one arm against trial 2 of the
// other and report noise as signal.
func PairArms(on, off []*BenchmarkResult) *PairedResult {
	res := &PairedResult{}

	offByKey := make(map[pairKey]*BenchmarkResult, len(off))
	for _, r := range off {
		offByKey[pairKey{r.ID, r.Lang, r.Trial}] = r
	}

	matchedOff := make(map[pairKey]bool, len(off))

	for _, o := range on {
		res.OnTotal++
		onPassed := passed(o)
		if onPassed {
			res.OnPass++
		}

		k := pairKey{o.ID, o.Lang, o.Trial}
		counterpart, ok := offByKey[k]
		if !ok {
			res.Unpaired++
			continue
		}
		matchedOff[k] = true

		offPassed := passed(counterpart)
		res.Pairs = append(res.Pairs, Pair{
			ID: o.ID, Lang: o.Lang, Trial: o.Trial,
			OnPass: onPassed, OffPass: offPassed,
		})

		switch {
		case onPassed && !offPassed:
			res.OnlyOnPassed++
		case !onPassed && offPassed:
			res.OnlyOffPassed++
		}
	}

	// Aggregate the OFF arm independently, and count its orphans too.
	for _, r := range off {
		res.OffTotal++
		if passed(r) {
			res.OffPass++
		}
		if !matchedOff[pairKey{r.ID, r.Lang, r.Trial}] {
			res.Unpaired++
		}
	}

	if res.OnTotal > 0 && res.OffTotal > 0 {
		res.DeltaPP = 100 * (float64(res.OnPass)/float64(res.OnTotal) - float64(res.OffPass)/float64(res.OffTotal))
	}

	res.McNemar = McNemar(res.OnlyOnPassed, res.OnlyOffPassed)
	// The OFF arm is the control: it is the baseline the treatment is measured
	// against, so its ceiling is what bounds what this comparison can detect.
	res.Headroom = CheckHeadroom(res.OffPass, res.OffTotal, DefaultHeadroomCeiling)
	return res
}

// McNemar tests whether the discordant pairs are asymmetric.
//
// b = pairs only the ON arm passed, c = pairs only OFF passed. Concordant pairs
// carry no information about the treatment and are correctly ignored.
func McNemar(b, c int) McNemarResult {
	n := b + c

	if n < MinDiscordantForPValue {
		return McNemarResult{
			Reportable: false,
			Note: fmt.Sprintf(
				"only %d discordant pair(s) (b=%d, c=%d); below the floor of %d, so no p-value is reported — the evidence base is too small for the number to mean anything",
				n, b, c, MinDiscordantForPValue),
		}
	}

	if n < exactBinomialMaxN {
		return McNemarResult{
			Reportable: true,
			Method:     "exact_binomial",
			PValue:     exactBinomialTwoSided(b, n),
		}
	}

	// Chi-square with Yates' continuity correction.
	diff := math.Abs(float64(b-c)) - 1
	if diff < 0 {
		diff = 0
	}
	chi2 := (diff * diff) / float64(n)
	return McNemarResult{
		Reportable: true,
		Method:     "chi_square_continuity",
		Statistic:  chi2,
		PValue:     chiSquareOneDFPValue(chi2),
	}
}

// exactBinomialTwoSided computes the two-sided exact binomial p-value for b
// successes in n trials under p=0.5 (the McNemar null).
func exactBinomialTwoSided(b, n int) float64 {
	// Two-sided via the symmetric doubling of the smaller tail, capped at 1.
	k := b
	if n-b < k {
		k = n - b
	}
	var tail float64
	for i := 0; i <= k; i++ {
		tail += binomialPMF(i, n)
	}
	p := 2 * tail
	if p > 1 {
		p = 1
	}
	return p
}

// binomialPMF returns P(X=k) for X~Binomial(n, 0.5), computed in log space to
// avoid overflow on large n.
func binomialPMF(k, n int) float64 {
	logC, _ := math.Lgamma(float64(n) + 1)
	lk, _ := math.Lgamma(float64(k) + 1)
	lnk, _ := math.Lgamma(float64(n-k) + 1)
	return math.Exp(logC - lk - lnk - float64(n)*math.Ln2)
}

// chiSquareOneDFPValue returns the upper-tail probability for a chi-square
// statistic with 1 degree of freedom: P = erfc(sqrt(x/2)).
func chiSquareOneDFPValue(x float64) float64 {
	if x <= 0 {
		return 1
	}
	return math.Erfc(math.Sqrt(x / 2))
}
