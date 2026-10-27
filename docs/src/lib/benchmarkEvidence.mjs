// Pure read model shared by the production comparison and its tests.
const finite = Number.isFinite;
const measured = s => s && s.runs > 0 && finite(s.rate);

export function scopeStats(row, scope = 'all', lang = 'ailang') {
  const raw = row.scopes?.[scope]?.[lang];
  if (!raw || !finite(raw.runs) || raw.runs <= 0) return null;
  const rate = finite(raw.rate) ? raw.rate : finite(raw.passes) ? raw.passes / raw.runs : null;
  return finite(rate) && rate >= 0 && rate <= 1 ? { ...raw, rate } : null;
}

export function evidenceRows(latest) {
  if (latest.benchmarkEvidence && latest.benchmarkEvidence.schemaVersion !== 1) {
    throw new Error(`Unsupported benchmark evidence schema: ${latest.benchmarkEvidence.schemaVersion}`);
  }
  const required = latest.benchmarkEvidence?.requiredCoreIds ?? Object.entries(latest.benchmarks || {})
    .filter(([, b]) => b.tier === 'core').map(([id]) => id);
  const rows = latest.benchmarkEvidence?.rows || legacyRows(latest, required);
  const counts = new Map();
  for (const r of rows) counts.set(`${r.mode}/${r.model}`, (counts.get(`${r.mode}/${r.model}`) || 0) + 1);
  return rows.map(r => {
    const ratings = {};
    // Existing ELO fits are model+mode+language, not harness-specific. Refuse
    // to attach a pooled fit to multiple distinct harness rows of the same ID.
    if (counts.get(`${r.mode}/${r.model}`) === 1) {
      for (const [lang, block] of Object.entries(latest.ratings?.[r.mode]?.byLang || {})) {
        const rating = block.models?.find(m => m.id === r.model);
        if (rating && finite(rating.elo)) ratings[lang] = rating.elo;
      }
    }
    return { ...r, ratings, requiredCoreIds: required };
  });
}

export function undefeated(row, lang) {
  const s = scopeStats(row, 'all', lang);
  return measured(s) && s.rate === 1;
}

export function isProvisional(row, lang) {
  const s = scopeStats(row, 'all', lang);
  const required = row.requiredCoreIds || [];
  return !s || required.length === 0 || !required.every(id => s.coreCompleted?.includes(id));
}

export function capabilityGroup(row, lang) {
  if (row.ratings[lang] == null && undefeated(row, lang)) return 0;
  return row.ratings[lang] != null ? 1 : 2;
}

export function costMetric(stats) {
  const entries = Object.entries(stats?.costs || {});
  if (entries.length !== 1) return { value: null, kind: entries.length > 1 ? 'mixed' : 'unavailable' };
  const [kind, c] = entries[0];
  return { value: c.complete && c.runs === stats.runs && finite(c.perSuccessUsd) ? c.perSuccessUsd : null, kind };
}

export function timingMetric(stats) {
  for (const kind of ['agent_success', 'generation_wall', 'recorded_duration']) {
    const t = stats?.timing?.[kind];
    if (t?.samples > 0 && finite(t.medianMs) && t.medianMs > 0) return { value: t.medianMs, kind, samples: t.samples };
  }
  return { value: null, kind: 'unavailable' };
}

export function sortRows(rows, { sort = 'elo', lang = 'ailang', scope = 'all' } = {}) {
  return [...rows].sort((a, b) => {
    if (sort === 'elo') {
      const ag = capabilityGroup(a, lang), bg = capabilityGroup(b, lang);
      if (ag !== bg) return ag - bg;
      if (ag === 0) {
        const ap = isProvisional(a, lang), bp = isProvisional(b, lang);
        if (ap !== bp) return ap ? 1 : -1;
        const difference = (scopeStats(b, 'all', lang)?.benchmarks?.length || 0) - (scopeStats(a, 'all', lang)?.benchmarks?.length || 0);
        if (difference) return difference;
      }
    }
    if (sort !== 'name') {
      const value = r => sort === 'cost' ? costMetric(scopeStats(r, scope, lang)).value
        : sort === 'rate' ? scopeStats(r, scope, lang)?.rate : r.ratings[lang];
      const av = value(a), bv = value(b);
      if (av == null && bv != null) return 1;
      if (bv == null && av != null) return -1;
      if (av != null && bv != null && av !== bv) return sort === 'cost' ? av - bv : bv - av;
    }
    return a.model.localeCompare(b.model) || a.id.localeCompare(b.id);
  });
}

function legacyStats(s, coreCompleted = null) {
  if (!s?.totalRuns || !finite(s.successRate)) return null;
  return { runs: s.totalRuns, rate: s.successRate,
    passes: s.successRate === 1 ? s.totalRuns : s.successRate === 0 ? 0 : null,
    firstAttemptPasses: null, firstAttemptUnknown: s.totalRuns, repairs: null,
    benchmarks: [], coreCompleted, versions: {}, costs: {}, timing: {}, outcomes: {} };
}

function legacyRows(latest, required) {
  const rows = [];
  for (const [model, stats] of Object.entries(latest.models || {})) {
    const scopes = { all: {} };
    for (const [lang, s] of Object.entries(stats.languages || {})) {
      if (s.agentOnly) continue;
      // The legacy gallery pools modes. When an agent sample exists it cannot
      // establish which Core IDs the direct-model language sample covered.
      const mixedModes = stats.agentStats?.runs > 0 || latest.agentModels?.[model]?.agentStats?.runs > 0;
      const ids = mixedModes ? null : required.filter(id => latest.benchmarks?.[id]?.modelStats?.[model]?.[lang]?.runs > 0);
      const summary = legacyStats(s, ids);
      if (summary) scopes.all[lang] = summary;
    }
    for (const [tier, t] of Object.entries(latest.tiers || {})) {
      scopes[tier] = {};
      for (const [lang, s] of Object.entries(t.model_stats?.[model] || {})) {
        const summary = legacyStats(s);
        if (summary) scopes[tier][lang] = summary;
      }
    }
    if (Object.keys(scopes.all).length) rows.push({ id: JSON.stringify([model, 'standard', 'direct_api']), model, mode: 'standard', harness: 'direct_api', scopes, legacy: true });
  }
  for (const [model, s] of Object.entries({ ...(latest.models || {}), ...(latest.agentModels || {}) })) {
    if (!s.agentStats?.runs) continue;
    const all = {};
    for (const [lang, ls] of Object.entries(s.languages || {})) {
      const rate = ls.agentSuccessRate ?? (ls.agentOnly ? ls.successRate : null);
      const runs = ls.agentRuns ?? (ls.agentOnly ? ls.totalRuns : null);
      if (runs > 0 && finite(rate)) all[lang] = legacyStats({ totalRuns: runs, successRate: rate });
    }
    if (Object.keys(all).length) rows.push({ id: JSON.stringify([model, 'agent', s.agent_cli || 'unknown']), model,
      mode: 'agent', harness: s.agent_cli || 'unknown', scopes: { all }, legacy: true });
  }
  return rows;
}

export function modelName(model) {
  if (model === 'gemini-agents-argon') return 'Gemini 4 Argon';
  return model.replace(/^claude-/, 'Claude ').replace(/^gemini-/, 'Gemini ')
    .replace(/^gpt6-1-/, 'GPT-6.1 ').replace(/^gpt6-/, 'GPT-6 ')
    .replace(/^gpt5-6-/, 'GPT-5.6 ').replace(/^or-/, '')
    .replace(/(\d)-(\d)(?=-|$)/g, '$1.$2').replace(/-/g, ' ')
    .replace(/\b(argon|astra|sol|luna|sonnet|opus|haiku|fable|flash|pro|mini|terra)\b/g, word => word[0].toUpperCase() + word.slice(1));
}

export const percent = v => finite(v) ? `${(v * 100).toFixed(1)}%` : '—';
export const dollars = v => finite(v) ? v === 0 ? '$0 recorded' : `$${v.toFixed(v < .01 ? 4 : 3)}` : '—';
export const milliseconds = v => !finite(v) ? '—' : v < 1000 ? `${Math.round(v)} ms` : v < 60000 ? `${(v / 1000).toFixed(1)} s` : `${(v / 60000).toFixed(1)} min`;
export const costLabels = { metered: 'Metered', 'list-price-equivalent': 'List-price equivalent', 'free-local': 'Free local', unknown: 'Billing provenance unknown', mixed: 'Mixed billing categories', unavailable: 'Not retained' };
export const timingLabels = { agent_success: 'Agent time to success', generation_wall: 'Generation wall time', recorded_duration: 'Recorded execution duration', unavailable: 'Not retained' };
