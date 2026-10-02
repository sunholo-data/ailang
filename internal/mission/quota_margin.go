package mission

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
)

// START MARGINS (Mark, attended 2026-10-02): "only start if we have some headroom ... if
// quota is 80% and we are on 79.5% there is not much point starting even though its under
// quota." A bare `used < allowance` test admitted a run with half a point to spare; the run
// then pushed the bucket over, every later fire fell through to the next bucket, and the
// pace line creeping back up re-admitted the next run with the same half point. The loops
// flapped on and off all night (2026-10-01/02).
//
// A bucket is admitted only while its headroom (allowance minus usage, in the bucket's own
// unit) is at least the margin. The defaults are one typical mission run, measured
// 2026-09-24..10-02 as the p75 per-run cost per bucket, rounded up:
//
//	codex      2pp  of the weekly window (p75 0.8-1.2pp; Sol 6.1 runs)
//	anthropic  1pp  of the weekly window (p75 0.3-0.6pp)
//	ollama     3pp  of the trailing-24h gauge ration (p75 1.3-3.3pp)
//	openrouter $0.60 of the daily dollar ration (p75 ~$0.47-0.56)
//
// AILANG_QUOTA_MARGIN_<BUCKET> overrides one bucket (a non-negative number; 0 restores the
// bare comparison). Attended sessions are never gated by this.
var defaultStartMargins = map[string]float64{
	"codex":      2,
	"anthropic":  1,
	"ollama":     3,
	"openrouter": 0.60,
}

// StartMargin returns the headroom a bucket must have before a new run is admitted.
func StartMargin(bucket string) float64 {
	def := defaultStartMargins[bucket]
	raw := strings.TrimSpace(config.Raw("AILANG_QUOTA_MARGIN_" + strings.ToUpper(bucket)))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return def
	}
	return v
}

// headroomShortReason is the operator line for a bucket that is under its allowance but
// inside the start margin, so the report says why a not-yet-over bucket is blocked.
func headroomShortReason(provider, unit string, headroom, margin float64) string {
	return fmt.Sprintf("%s headroom %s%s is below the %s%s start margin; new routing waits for headroom",
		provider, trimFloat(headroom), unit, trimFloat(margin), unit)
}
