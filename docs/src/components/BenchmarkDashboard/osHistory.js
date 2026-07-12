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
  return history.map((entry) => {
    const base = (entry.version || '').split('-')[0];
    const rows = osByVer[entry.version] || osByVer[base];
    if (!rows || !rows.length) return entry;
    // Build the local per-model fragment for this version.
    const localStats = {};
    rows.forEach((r) => {
      if (!r || !r.model || !r.lang) return;
      const ms = {};
      for (const [l, rate] of Object.entries(r.lang)) {
        if (typeof rate === 'number') ms[l] = { successRate: rate, totalRuns: 1 };
      }
      localStats[r.model] = ms;
    });
    if (Object.keys(localStats).length === 0) return entry;
    const merged = { ...entry, modelStats: { ...(entry.modelStats || {}), ...localStats } };
    // The rig's rate is NOT tier-split (the rotation runs AILANG across tiers), so
    // inject the same fragment into every tier's modelStats too — otherwise the
    // "Local agent" line vanishes in tier views (and Core is the default view). The
    // asterisk flags it as a blended/incomplete figure. Once os/history carries
    // per-tier local rates, replace this with a tier-scoped merge.
    if (entry.tiers && typeof entry.tiers === 'object') {
      const tiers = {};
      for (const [t, tv] of Object.entries(entry.tiers)) {
        tiers[t] = tv && typeof tv === 'object'
          ? { ...tv, modelStats: { ...(tv.modelStats || {}), ...localStats } }
          : tv;
      }
      merged.tiers = tiers;
    }
    return merged;
  });
}
