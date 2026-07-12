// M-EVAL-OS-LONGITUDINAL: shared helper to fold the on-device rig's version-trend
// (os/history.json) into the cloud baseline `history` so local model×harness combos
// (motoko/opencode/pi on qwen) appear on the trend charts as a "Local agent"
// provider (see getProvider in modelColors.js). Reused by PerModelTrend,
// ModelDeltaTrend, and SuccessTrend so all three tell the same on-device story.

import { useState, useEffect } from 'react';
import { benchmarkFetch } from '@site/src/lib/benchmarkFetch';

// Runtime fetch of the local rig history. Returns null while loading, then an array
// ([] if missing/malformed — a no-op merge).
export function useOSHistory() {
  const [os, setOS] = useState(null);
  useEffect(() => {
    benchmarkFetch('os/history.json')
      .then((r) => (r.ok ? r.json() : []))
      .then((h) => setOS(Array.isArray(h) ? h : []))
      .catch(() => setOS([]));
  }, []);
  return os;
}

// langMap { ailang: rate, … } → modelStats fragment { ailang: {successRate, totalRuns}, … }.
function toStats(langMap) {
  const ms = {};
  for (const [l, rate] of Object.entries(langMap || {})) {
    if (typeof rate === 'number') ms[l] = { successRate: rate, totalRuns: 1 };
  }
  return ms;
}

// Split one version's os rows into an overall fragment + per-tier fragments (from
// the publisher's rows[].tiers block, so Core/Stretch/Frontier are tier-accurate).
function buildFragments(rows) {
  const overall = {};
  const byTier = {};
  (rows || []).forEach((r) => {
    if (!r || !r.model || !r.lang) return;
    overall[r.model] = toStats(r.lang);
    if (r.tiers && typeof r.tiers === 'object') {
      for (const [tier, langMap] of Object.entries(r.tiers)) {
        (byTier[tier] = byTier[tier] || {})[r.model] = toStats(langMap);
      }
    }
  });
  return { overall, byTier };
}

export function mergeOSHistory(history, osHistory) {
  if (!Array.isArray(history)) return history || [];
  if (!osHistory || osHistory.length === 0) return history;
  const osByVer = {};
  osHistory.forEach((e) => { if (e && e.ailang_version) osByVer[e.ailang_version] = e; });

  const matched = new Set();
  const mapped = history.map((entry) => {
    const base = (entry.version || '').split('-')[0];
    const osEntry = osByVer[entry.version] || osByVer[base];
    if (!osEntry) return entry;
    matched.add(osEntry.ailang_version);
    const { overall, byTier } = buildFragments(osEntry.rows);
    if (Object.keys(overall).length === 0) return entry;
    const merged = { ...entry, modelStats: { ...(entry.modelStats || {}), ...overall } };
    if (entry.tiers && typeof entry.tiers === 'object') {
      const tiers = {};
      for (const [t, tv] of Object.entries(entry.tiers)) {
        // Prefer the tier-specific local rate; a tier the rig hasn't run yet gets no
        // local row (line is absent there, honestly, until coverage reaches it).
        // Pre-tiers os data (no `tiers`) falls back to the overall fragment.
        const lf = byTier[t] || (Object.keys(byTier).length === 0 ? overall : null);
        tiers[t] = (tv && typeof tv === 'object' && lf)
          ? { ...tv, modelStats: { ...(tv.modelStats || {}), ...lf } }
          : tv;
      }
      merged.tiers = tiers;
    }
    return merged;
  });

  // Versions the rig ran that the cloud history LACKS get their own local-only entry
  // so the "Local agent" trend spans ALL rig releases, not just those the cloud also
  // happened to run. Cloud lines are null there (localOnly flag), the local line shows.
  const extra = [];
  Object.values(osByVer).forEach((osEntry) => {
    if (matched.has(osEntry.ailang_version)) return;
    const { overall, byTier } = buildFragments(osEntry.rows);
    if (Object.keys(overall).length === 0) return;
    const tiers = {};
    for (const [t, frag] of Object.entries(byTier)) tiers[t] = { modelStats: frag };
    extra.push({
      version: osEntry.ailang_version,
      timestamp: osEntry.generated ? `${osEntry.generated}T00:00:00Z` : new Date().toISOString(),
      modelStats: overall,
      tiers,
      localOnly: true,
    });
  });
  return extra.length ? [...mapped, ...extra] : mapped;
}
