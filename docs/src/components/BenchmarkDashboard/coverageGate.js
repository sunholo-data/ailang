// M-EVAL-VALIDITY-DISCIPLINE (W2): shared coverage-gating helper.
//
// A model's ELO / pass-rate is only comparable to another's if it was measured on
// a similar benchmark set. A local model that ran 6 benchmarks must not sit at #1
// next to a 55-benchmark cloud model as if comparable. This builds one lookup from
// the ratings block (which already carries per-model `benchmarks` + `maxCoverage`,
// see ratings_export.go) so every dashboard component can mark under-covered models
// "provisional" the same way EloLeaderboard does — dimmed / no medal / badged,
// never hidden.
//
// Coverage is merged across standard + agent modes: a model's coverage is its best
// (max) across the modes it appears in, and maxCoverage is the max any model ran.

// Two thresholds, both deliberate — named here so they are a documented policy
// rather than two magic numbers that drifted apart in separate components.
//
// RATE: a pass rate degrades gracefully with coverage — a model on half the set
// still reports a meaningful (if noisier) rate.
// ELO: a *rating* does not. Missing runs are rarely random — an API-quota death
// mid-run skips the alphabetical tail, where the hardest frontier benchmarks live,
// inflating exactly the models with holes (v0.30.0: claude-sonnet-5 topped the
// board on 44/56 coverage that excluded gauntlet_10/quine/ssa_constant_fold/...).
// So ELO demands near-identical benchmark sets before it will call two models
// comparable.
export const RATE_COVERAGE_FRACTION = 0.5;
export const ELO_COVERAGE_FRACTION = 0.9;

export function buildCoverage(ratings) {
  const byId = {};
  let maxCoverage = 0;
  for (const mode of ['standard', 'agent']) {
    const block = ratings && ratings[mode];
    if (!block) continue;
    if (block.maxCoverage > maxCoverage) maxCoverage = block.maxCoverage;
    for (const m of block.models || []) {
      const n = m.benchmarks || 0;
      if (!(m.id in byId) || n > byId[m.id]) byId[m.id] = n;
      if (n > maxCoverage) maxCoverage = n;
    }
  }
  const threshold = Math.max(1, Math.round(maxCoverage * RATE_COVERAGE_FRACTION));
  const eloThreshold = Math.max(1, Math.ceil(maxCoverage * ELO_COVERAGE_FRACTION));
  return {
    maxCoverage,
    threshold,
    eloThreshold,
    // Provisional under the stricter ELO policy. Same lookup, same maxCoverage —
    // only the fraction differs, so the two views can never disagree on the inputs.
    isProvisionalForElo: (id) => id in byId && byId[id] < eloThreshold,
    // distinct benchmarks a model ran (best across modes), or null if unknown.
    benchmarksFor: (id) => (id in byId ? byId[id] : null),
    // provisional = we KNOW its coverage and it's below half the max. A model with
    // unknown coverage (not in the ratings block) is left unflagged — we can't
    // assert it's sparse, so we don't dim it.
    isProvisional: (id) => id in byId && byId[id] < threshold,
  };
}
