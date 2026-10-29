function versionParts(version) {
  return String(version || '').replace(/^v/, '').split('.').map(p => Number.parseInt(p, 10) || 0);
}

function compareVersion(a, b) {
  const av = versionParts(a.version), bv = versionParts(b.version);
  for (let i = 0; i < Math.max(av.length, bv.length); i++) {
    const d = (av[i] || 0) - (bv[i] || 0);
    if (d) return d;
  }
  return String(a.timestamp || '').localeCompare(String(b.timestamp || ''));
}

export function historyPoints(latest, scope = 'core', range = 'all') {
  const sorted = [...(latest.history || [])].sort(compareVersion);
  const points = sorted.map(h => {
    const tier = scope === 'all' ? null : h.tiers?.[scope];
    const stat = lang => {
      const runs = tier ? tier[`${lang}_runs`] : scope === 'all' ? h.languageStats?.[lang]?.total_runs : null;
      const rate = tier ? tier[`${lang}_success_rate`] : scope === 'all' ? h.languageStats?.[lang]?.success_rate : null;
      return { runs: runs ?? null, rate: runs > 0 && Number.isFinite(rate) ? rate * 100 : null };
    };
    const a = stat('ailang'), p = stat('python');
    const index = h.ratings?.direction_index;
    return { version: h.version, published: h.timestamp, ailang: a.rate, python: p.rate,
      ailangRuns: a.runs, pythonRuns: p.runs, modelCount: Object.keys(h.modelStats || {}).length,
      taskCount: tier?.benchmark_count ?? null,
      events: (latest.events || []).filter(e => e.version === h.version &&
        (scope === 'all' || !e.affects_tiers?.length || e.affects_tiers.includes(scope))),
      elo: h.ratings?.models || {}, anchor: h.ratings?.anchor_version || 'not-retained',
      direction: Number.isFinite(index) ? index : null, panel: h.ratings?.panel_version || null,
      directionTrials: h.ratings?.trials ?? null, snapshot: h };
  });
  return range === 'recent' ? points.slice(-20) : points;
}

// Each contiguous anchor segment gets its own data key. Recharts cannot then
// connect across a changed anchor, even when adjacent snapshots have values.
export function eloSeries(points, models) {
  const data = points.map(p => ({ ...p })), series = [];
  for (const model of models) {
    let priorAnchor = null, current = null;
    points.forEach((p, i) => {
      const value = p.elo[model];
      if (!Number.isFinite(value)) { current = null; priorAnchor = null; return; }
      if (current == null || priorAnchor !== p.anchor) {
        current = `elo_${series.length}`;
        series.push({ key: current, model, anchor: p.anchor });
      }
      data[i][current] = value;
      priorAnchor = p.anchor;
    });
  }
  return { data, series };
}
