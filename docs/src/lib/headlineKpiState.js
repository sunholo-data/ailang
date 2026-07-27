// Pure state-derivation for the Cost-Per-Verified-Success headline KPI card
// (M-COST-PER-SUCCESS-KPI, M3).
//
// This is deliberately free of React so it can be unit-tested (Babel + data
// fixtures) WITHOUT a headless browser. It maps the additive latest.json object
// `headlineKpis.costPerVerifiedSuccess` (the exact struct the observatory
// rollup / CLI / HTTP emit) plus the benchmarkFetch source signal into one of
// four display states. It NEVER fabricates a value: absent / incomplete /
// zero-denominator all degrade visibly rather than showing $0 or a stale prior.

export const KPI_STATE_ABSENT = 'absent';          // no headline object at all -> "baseline unavailable"
export const KPI_STATE_INCOMPLETE = 'incomplete';  // available=false (unknown cost / missing verification)
export const KPI_STATE_ZERO_DENOM = 'zero_denominator'; // verified_successes == 0
export const KPI_STATE_AVAILABLE = 'available';    // a real, complete number

// deriveHeadlineKpiState(latest, source) -> a plain object describing what to render.
//
//   latest : the parsed latest.json (or null)
//   source : BENCHMARK_SOURCE_GCS | BENCHMARK_SOURCE_FALLBACK | undefined
//
// Returns:
//   {
//     state,                 // one of KPI_STATE_*
//     isFallback,            // true when served from the static in-build copy
//     kpi,                   // the raw headline object (or null)
//     valueUSD,              // number|null  (only meaningful when available)
//     verifiedSuccesses,     // number|null
//     totalRuns,             // number|null
//     knownCostUSD,          // number|null
//     quotaStages,           // number|null
//     unknownStages,         // number|null
//     reason,                // machine reason string ('' when available/absent)
//     baselineId,            // string|null
//   }
export function deriveHeadlineKpiState(latest, source) {
  const isFallback = source === 'fallback';
  const kpi =
    latest && latest.headlineKpis && latest.headlineKpis.costPerVerifiedSuccess
      ? latest.headlineKpis.costPerVerifiedSuccess
      : null;

  const base = {
    isFallback,
    kpi,
    valueUSD: null,
    verifiedSuccesses: null,
    totalRuns: null,
    knownCostUSD: null,
    quotaStages: null,
    unknownStages: null,
    reason: '',
    baselineId: null,
  };

  if (!kpi) {
    return { ...base, state: KPI_STATE_ABSENT };
  }

  const enriched = {
    ...base,
    verifiedSuccesses: numOrNull(kpi.verified_successes),
    totalRuns: numOrNull(kpi.total_runs),
    knownCostUSD: numOrNull(kpi.known_cost_usd),
    quotaStages: numOrNull(kpi.quota_stages),
    unknownStages: numOrNull(kpi.unknown_stages),
    reason: typeof kpi.reason === 'string' ? kpi.reason : '',
    baselineId: typeof kpi.baseline_id === 'string' ? kpi.baseline_id : null,
  };

  // Available is the single source of truth from the backend. We never
  // second-guess it into showing a number; we only pick which unavailable
  // sub-state to render for a clearer message.
  if (kpi.available === true) {
    return {
      ...enriched,
      state: KPI_STATE_AVAILABLE,
      valueUSD: numOrNull(kpi.cost_per_verified_success_usd),
    };
  }

  if (enriched.verifiedSuccesses === 0 || kpi.reason === 'zero_denominator') {
    return { ...enriched, state: KPI_STATE_ZERO_DENOM };
  }
  return { ...enriched, state: KPI_STATE_INCOMPLETE };
}

// formatUSD4 renders a dollar amount at 4 decimals, or '—' for null/NaN.
export function formatUSD4(v) {
  if (typeof v !== 'number' || Number.isNaN(v)) return '—';
  return `$${v.toFixed(4)}`;
}

function numOrNull(v) {
  return typeof v === 'number' && !Number.isNaN(v) ? v : null;
}
