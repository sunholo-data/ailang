package eval_analysis

// ShouldExcludeFromCapability returns true when a result with the given
// error_category should be excluded from per-model *capability* statistics
// (success rate, p90 cost-per-success, etc.).
//
// The intent: a model that ran out of provider-side quota or got 429'd tells
// us nothing about whether it could solve the benchmark — those failures are
// provider noise, not capability signal. Conversely, a `timeout`, `cost_killed`
// or `step_exhausted` IS capability signal: it means the model couldn't solve
// the task within the budget allotted, which is information operators need.
//
// Before M-EVAL-SWEET-SPOT (v0.19.0) the eval_analysis package indiscriminately
// excluded the catch-all `api_error` bucket — which swallowed legitimate
// capability failures (timeouts, executor crashes) alongside provider noise.
// This helper restores the distinction.
//
// Excluded categories:
//   - quota_exhausted:  provider account/key cap reached
//   - rate_limit:       429 / transient throttling
//   - api_error:        catch-all — historically the only exclusion bucket
//     (kept for legacy result JSONs that pre-date M1/M2's
//     typed categorization, where genuine quota kills are
//     still labelled api_error)
//
// Included categories (capability signal):
//   - timeout, cost_killed, step_exhausted: model couldn't solve in budget
//     — operators should see these in stats so they can decide to extend the
//     budget instead of hiding the model as "broken"
//   - compile_error, runtime_error, logic_error, verify_error: real failures
//   - none: success
//
// Future direction: once enough time has passed that legacy result JSONs
// have been refreshed (or backfilled via offline categorization), `api_error`
// can be removed from the exclusion set so the only excluded bucket is
// genuine provider noise.
func ShouldExcludeFromCapability(category string) bool {
	switch category {
	case "quota_exhausted", "rate_limit", "api_error":
		return true
	default:
		// NOTE: "refused" (safety-layer declines) deliberately stays INCLUDED
		// here — a refusal still counts as a non-pass in the rates ("keep
		// counting" product decision). It gets its own visibility bucket in the
		// failure-mode breakdown + a per-language refusal-rate annotation, but is
		// NOT excluded from capability/reliability scoring (which would also wrongly
		// pull it into the api_error reliability counters that share this predicate).
		return false
	}
}
