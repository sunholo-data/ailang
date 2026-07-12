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
  const threshold = Math.max(1, Math.round(maxCoverage * 0.5));
  return {
    maxCoverage,
    threshold,
    // distinct benchmarks a model ran (best across modes), or null if unknown.
    benchmarksFor: (id) => (id in byId ? byId[id] : null),
    // provisional = we KNOW its coverage and it's below half the max. A model with
    // unknown coverage (not in the ratings block) is left unflagged — we can't
    // assert it's sparse, so we don't dim it.
    isProvisional: (id) => id in byId && byId[id] < threshold,
  };
}
