import React, { useMemo } from 'react';
import styles from './styles.module.css';

// Color tokens for the 4 outcome families live in styles.module.css
// (.failureSuccess / .failureBudget / .failureCapability / .failureProvider)
// so dark-mode CSS vars work and JSX stays free of inline color strings.

function formatModelName(name) {
  let s = name;
  let suffix = '';
  if (s.startsWith('motoko-or-'))   { suffix = ' (motoko · OR)'; s = s.slice('motoko-or-'.length); }
  else if (s.startsWith('motoko-')) { suffix = ' (motoko)';      s = s.slice('motoko-'.length); }
  else if (s.startsWith('opencode-or-')) { suffix = ' (agent · OR)'; s = s.slice('opencode-or-'.length); }
  else if (s.startsWith('opencode-'))    { suffix = ' (agent)';     s = s.slice('opencode-'.length); }
  else if (s.startsWith('pi-'))     { suffix = ' (Pi)'; s = s.slice('pi-'.length); }
  else if (s.startsWith('or-'))     { suffix = ' (OR)'; s = s.slice('or-'.length); }
  s = s
    .replace(/^claude-/, 'Claude ')
    .replace(/^gemini-/, 'Gemini ')
    .replace(/^gpt5/, 'GPT-5')
    .replace(/^glm-/, 'GLM ')
    .replace(/^kimi-/, 'Kimi ')
    .replace(/^qwen3-/, 'Qwen3 ')
    .replace(/^gemma-/, 'Gemma ')
    .replace(/^gemma4-/, 'Gemma4 ')
    .replace(/^deepseek-/, 'DeepSeek ')
    .replace(/-/g, ' ');
  return s + suffix;
}

/**
 * FailureCategoryBars — stacked horizontal bars per model showing the
 * outcome-family breakdown.
 *
 * Reads `models[name].sweet_spot.buckets` (computed at export time by
 * BuildSweetSpot — each (model × benchmark) gets exactly one bucket).
 *
 * Bar segments, left to right:
 *   - Success (green): fast_pass + slow_pass
 *   - Budget (orange): budget_blocked (cost_killed + step_exhausted)
 *   - Capability (red): capability_blocked (compile/runtime/logic/timeout)
 *   - Provider (gray): provider_blocked (quota + rate_limit + api_error)
 *
 * Operator question this answers: "Did this model fail because it can't
 * code, because we gave it too little budget, or because the provider
 * threw 429s?" — answered at a glance.
 */
export default function FailureCategoryBars({ models }) {
  const rows = useMemo(() => {
    const data = [];
    for (const [name, stats] of Object.entries(models || {})) {
      const ss = stats.sweet_spot;
      if (!ss || !ss.buckets) continue;
      const b = ss.buckets;
      const success = (b.fast_pass || 0) + (b.slow_pass || 0);
      const total = success + (b.budget_blocked || 0) + (b.capability_blocked || 0) + (b.refused || 0) + (b.provider_blocked || 0);
      if (total === 0) continue;
      data.push({
        name,
        displayName: formatModelName(name),
        total,
        success,
        budget:     b.budget_blocked || 0,
        capability: b.capability_blocked || 0,
        refused:    b.refused || 0,
        provider:   b.provider_blocked || 0,
      });
    }
    // Sort by success rate desc — best models on top.
    data.sort((a, b) => (b.success / b.total) - (a.success / a.total));
    return data;
  }, [models]);

  if (rows.length === 0) {
    return (
      <div className={styles.chartContainer}>
        <p style={{ color: 'var(--ifm-color-emphasis-600)' }}>
          No models with sweet_spot.buckets data yet. Re-run <code>ailang eval-report</code> with v0.19.0+ to populate.
        </p>
      </div>
    );
  }

  return (
    <div className={styles.chartContainer}>
      <h3 style={{ margin: '0 0 8px 0' }}>Failure Modes by Outcome Family</h3>
      <p className={styles.sweetSpotHeadlineNote}>
        Each (model × benchmark) lands in exactly one bucket. Capability =
        broken code; Budget = ran out of $ or turns (operator could raise);
        Refused = the model&apos;s safety layer declined the prompt (model behavior,
        not a coding failure); Provider = quota/rate-limit (provider noise). Refused
        and Provider are both excluded from capability scoring.
      </p>

      <div role="img" aria-label="Failure mode breakdown per model">
        {rows.map(r => {
          const segments = [
            { key: 'success',    label: 'Success',    count: r.success,    cls: styles.failureSuccess },
            { key: 'budget',     label: 'Budget',     count: r.budget,     cls: styles.failureBudget },
            { key: 'capability', label: 'Capability', count: r.capability, cls: styles.failureCapability },
            { key: 'refused',    label: 'Refused',    count: r.refused,    cls: styles.failureRefused },
            { key: 'provider',   label: 'Provider',   count: r.provider,   cls: styles.failureProvider },
          ];
          return (
            <div key={r.name} className={styles.failureBarRow}>
              <div className={styles.failureBarLabel}>{r.displayName}</div>
              <div className={styles.failureBar}>
                {segments.map(seg => {
                  const pct = (seg.count / r.total) * 100;
                  if (pct === 0) return null;
                  return (
                    <div
                      key={seg.key}
                      className={`${styles.failureBarSegment} ${seg.cls}`}
                      title={`${seg.label}: ${seg.count} of ${r.total} (${pct.toFixed(0)}%)`}
                      style={{ width: `${pct}%` }}
                    >
                      {pct > 8 ? seg.count : ''}
                    </div>
                  );
                })}
              </div>
              <div className={styles.failureBarTotal}>{r.total}</div>
            </div>
          );
        })}
      </div>

      <div className={styles.failureLegend}>
        {[
          { key: 'success',    label: 'Success (fast + slow passes)',                          cls: styles.failureSuccess },
          { key: 'budget',     label: 'Budget-blocked (cost_killed, step_exhausted)',          cls: styles.failureBudget },
          { key: 'capability', label: 'Capability-blocked (compile/runtime/logic/timeout)',    cls: styles.failureCapability },
          { key: 'refused',    label: 'Refused (safety-layer decline — not a coding failure)', cls: styles.failureRefused },
          { key: 'provider',   label: 'Provider-blocked (quota / rate_limit / api_error)',     cls: styles.failureProvider },
        ].map(item => (
          <div key={item.key} className={styles.failureLegendItem}>
            <span className={`${styles.failureLegendSwatch} ${item.cls}`} />
            <span>{item.label}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
