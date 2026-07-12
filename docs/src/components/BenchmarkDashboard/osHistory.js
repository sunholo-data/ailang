// M-EVAL-OS-LONGITUDINAL: shared helper to fold the on-device rig's version-trend
// (os/history.json) into the cloud baseline `history` so local model×harness combos
// (motoko/opencode/pi on qwen) appear on the trend charts as a "Local agent"
// provider (see getProvider in modelColors.js). Reused by PerModelTrend,
// ModelDeltaTrend, and SuccessTrend so all three tell the same on-device story.

import { useState, useEffect } from 'react';

// Runtime fetch of the local rig history. Returns null while loading, then an array
// ([] if missing/malformed — a no-op merge).
export function useOSHistory() {
  const [os, setOS] = useState(null);
  useEffect(() => {
    fetch('/benchmarks/os/history.json')
      .then((r) => (r.ok ? r.json() : []))
      .then((h) => setOS(Array.isArray(h) ? h : []))
      .catch(() => setOS([]));
  }, []);
  return os;
}

// Merge each os/history version's per-(model,harness,lang) rate into the matching
// baseline entry's `modelStats` (all-tier). Matched by base version string so a
// git-described baseline (v0.29.2-29-g…) still lines up with a release snapshot
// (v0.29.2). Additive + guarded: unknown versions and malformed rows are skipped.
export function mergeOSHistory(history, osHistory) {
  if (!Array.isArray(history)) return history || [];
  if (!osHistory || osHistory.length === 0) return history;
  const osByVer = {};
  osHistory.forEach((e) => {
    if (e && e.ailang_version) osByVer[e.ailang_version] = e.rows || [];
  });
  // langMap { ailang: rate, … } → modelStats fragment { ailang: {successRate, totalRuns}, … }.
  const toStats = (langMap) => {
    const ms = {};
    for (const [l, rate] of Object.entries(langMap || {})) {
      if (typeof rate === 'number') ms[l] = { successRate: rate, totalRuns: 1 };
    }
    return ms;
  };

  return history.map((entry) => {
    const base = (entry.version || '').split('-')[0];
    const rows = osByVer[entry.version] || osByVer[base];
    if (!rows || !rows.length) return entry;
    // Overall (all-tier) local fragment + per-tier fragments (tier → model → stats)
    // from the publisher's `tiers` block, so the Core/Stretch/Frontier lines are
    // tier-ACCURATE, not a blend. Falls back to `lang` for pre-tiers os data.
    const overall = {};
    const byTier = {};
    rows.forEach((r) => {
      if (!r || !r.model || !r.lang) return;
      overall[r.model] = toStats(r.lang);
      if (r.tiers && typeof r.tiers === 'object') {
        for (const [tier, langMap] of Object.entries(r.tiers)) {
          (byTier[tier] = byTier[tier] || {})[r.model] = toStats(langMap);
        }
      }
    });
    if (Object.keys(overall).length === 0) return entry;
    const merged = { ...entry, modelStats: { ...(entry.modelStats || {}), ...overall } };
    if (entry.tiers && typeof entry.tiers === 'object') {
      const tiers = {};
      for (const [t, tv] of Object.entries(entry.tiers)) {
        // Prefer the tier-specific local rate; a tier the rig hasn't run yet gets no
        // local row (line is absent there, honestly, until coverage reaches it).
        // Pre-tiers os data (no `tiers` block) falls back to the overall fragment.
        const localForTier = byTier[t] || (Object.keys(byTier).length === 0 ? overall : null);
        tiers[t] = (tv && typeof tv === 'object' && localForTier)
          ? { ...tv, modelStats: { ...(tv.modelStats || {}), ...localForTier } }
          : tv;
      }
      merged.tiers = tiers;
    }
    return merged;
  });
}
