import React, { useMemo } from 'react';
import styles from './styles.module.css';

// Color tokens for the 4 outcome families. Match the CLI sweet-spot text
// output's mental model (capability=red, budget=orange, provider=gray,
// success=green). Falls back to CSS vars so dark mode picks up correctly.
const COLORS = {
  success:    'var(--ifm-color-success, #16a34a)',
  capability: 'var(--ifm-color-danger,  #dc2626)',
  budget:     'var(--ifm-color-warning, #d97706)',
  provider:   'var(--ifm-color-emphasis-500, #6b7280)',
};

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
      const total = success + (b.budget_blocked || 0) + (b.capability_blocked || 0) + (b.provider_blocked || 0);
      if (total === 0) continue;
      data.push({
        name,
        displayName: formatModelName(name),
        total,
        success,
        budget:     b.budget_blocked || 0,
        capability: b.capability_blocked || 0,
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

  // Longest label width drives the label column.
  const labelWidth = 220;

  return (
    <div className={styles.chartContainer}>
      <h3 style={{ margin: '0 0 8px 0' }}>Failure Modes by Outcome Family</h3>
      <p style={{ fontSize: '0.9rem', color: 'var(--ifm-color-emphasis-700)', marginBottom: 12 }}>
        Each (model × benchmark) lands in exactly one bucket. Capability =
        broken code; Budget = ran out of $ or turns (operator could raise);
        Provider = quota/rate-limit (provider noise, excluded from capability scoring).
      </p>

      <div role="img" aria-label="Failure mode breakdown per model">
        {rows.map(r => {
          const segments = [
            { key: 'success',    label: 'Success',    count: r.success,    color: COLORS.success },
            { key: 'budget',     label: 'Budget',     count: r.budget,     color: COLORS.budget },
            { key: 'capability', label: 'Capability', count: r.capability, color: COLORS.capability },
            { key: 'provider',   label: 'Provider',   count: r.provider,   color: COLORS.provider },
          ];
          return (
            <div key={r.name} style={{ display: 'flex', alignItems: 'center', marginBottom: 6, fontSize: '0.85rem' }}>
              <div style={{
                width: labelWidth,
                paddingRight: 10,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}>
                {r.displayName}
              </div>
              <div style={{
                flex: 1,
                display: 'flex',
                height: 22,
                border: '1px solid var(--ifm-color-emphasis-300)',
                borderRadius: 3,
                overflow: 'hidden',
              }}>
                {segments.map(seg => {
                  const pct = (seg.count / r.total) * 100;
                  if (pct === 0) return null;
                  return (
                    <div
                      key={seg.key}
                      title={`${seg.label}: ${seg.count} of ${r.total} (${pct.toFixed(0)}%)`}
                      style={{
                        width: `${pct}%`,
                        background: seg.color,
                        color: '#fff',
                        fontSize: '0.75rem',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        overflow: 'hidden',
                      }}
                    >
                      {pct > 8 ? seg.count : ''}
                    </div>
                  );
                })}
              </div>
              <div style={{
                width: 50,
                paddingLeft: 10,
                fontFamily: 'monospace',
                color: 'var(--ifm-color-emphasis-700)',
              }}>
                {r.total}
              </div>
            </div>
          );
        })}
      </div>

      <div style={{ display: 'flex', gap: 16, marginTop: 12, fontSize: '0.8rem', flexWrap: 'wrap' }}>
        {[
          { key: 'success', label: 'Success (fast + slow passes)' },
          { key: 'budget',  label: 'Budget-blocked (cost_killed, step_exhausted)' },
          { key: 'capability', label: 'Capability-blocked (compile/runtime/logic/timeout)' },
          { key: 'provider', label: 'Provider-blocked (quota / rate_limit / api_error)' },
        ].map(item => (
          <div key={item.key} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <span style={{
              width: 12,
              height: 12,
              background: COLORS[item.key],
              borderRadius: 2,
              display: 'inline-block',
            }} />
            <span>{item.label}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
