import React, { useState } from 'react';
import { AlertTriangle, ShieldCheck } from 'lucide-react';
import styles from './styles.module.css';

function formatModelName(name) {
  if (name.includes('claude-sonnet')) return 'Claude Sonnet';
  if (name.includes('claude-haiku')) return 'Claude Haiku';
  if (name.includes('claude-opus')) return 'Claude Opus';
  if (name.includes('gemini-3-1-pro') || name.includes('gemini-3-pro')) return 'Gemini 3 Pro';
  if (name.includes('gemini-3-flash')) return 'Gemini 3 Flash';
  if (name.includes('gemini-2-5-pro')) return 'Gemini 2.5 Pro';
  if (name.includes('gemini-2-5-flash')) return 'Gemini 2.5 Flash';
  if (name.includes('gpt5-2-codex')) return 'GPT-5 Codex';
  if (name.includes('gpt5-4')) return 'GPT-5.4';
  if (name.includes('gpt5-1-instant')) return 'GPT-5.1 Instant';
  if (name.includes('gpt5-1')) return 'GPT-5.1';
  if (name.includes('gpt5-mini')) return 'GPT-5 Mini';
  if (name.includes('gpt-5-mini')) return 'GPT-5 Mini';
  if (name.includes('gpt-5')) return 'GPT-5';
  return name;
}

// ReliabilityCard surfaces API + refusal metrics as a first-class signal.
// Infra failures (api_error) are plotted identically to code-quality
// regressions elsewhere; this card separates them so readers see a gemini
// quota exhaustion for what it is.
export default function ReliabilityCard({ aggregates, models, activeTier }) {
  const [expanded, setExpanded] = useState(false);

  // Source-of-truth for the headline number: tier-scoped if a tier is
  // selected, otherwise global aggregates.
  const apiErrorCount = activeTier ? (activeTier.api_error_count || 0) : (aggregates.apiErrorCount || 0);
  const refusalCount = activeTier ? (activeTier.refusal_count || 0) : (aggregates.refusalCount || 0);
  const totalRuns = activeTier ? (activeTier.total_runs || 0) : ((aggregates.apiErrorCount && aggregates.apiErrorRate) ? Math.round(aggregates.apiErrorCount / aggregates.apiErrorRate) : 0);

  const reliabilityPct = totalRuns > 0
    ? ((1 - (apiErrorCount / totalRuns)) * 100).toFixed(1)
    : '100.0';

  const hasIssues = apiErrorCount > 0 || refusalCount > 0;

  // Per-model breakdown for the expandable panel.
  const modelRows = models
    ? Object.entries(models)
        .map(([name, m]) => ({
          name,
          reliability: m?.reliability || {},
        }))
        .filter((r) => r.reliability.apiErrorCount > 0 || r.reliability.refusalCount > 0)
        .sort((a, b) => (b.reliability.apiErrorCount || 0) - (a.reliability.apiErrorCount || 0))
    : [];

  return (
    <div className={`${styles.reliabilityCard} ${hasIssues ? styles.reliabilityCardWarn : ''}`}>
      <div className={styles.reliabilityHeader}>
        <div className={styles.reliabilityIcon}>
          {hasIssues ? <AlertTriangle size={20} /> : <ShieldCheck size={20} />}
        </div>
        <div className={styles.reliabilityBody}>
          <div className={styles.reliabilityTitle}>
            API Reliability{activeTier ? ` · ${activeTier.label || 'tier'}` : ''}
          </div>
          <div className={styles.reliabilityRow}>
            <span className={styles.reliabilityValue}>{reliabilityPct}%</span>
            <span className={styles.reliabilitySub}>
              {apiErrorCount > 0
                ? `${apiErrorCount} API errors across ${totalRuns} runs`
                : `No API failures across ${totalRuns} runs`}
              {refusalCount > 0 ? ` · ${refusalCount} refusals` : ''}
            </span>
          </div>
        </div>
        {modelRows.length > 0 && (
          <button
            type="button"
            className={styles.reliabilityToggle}
            onClick={() => setExpanded((v) => !v)}
          >
            {expanded ? 'Hide' : 'Per model'}
          </button>
        )}
      </div>

      {expanded && modelRows.length > 0 && (
        <div className={styles.reliabilityDetail}>
          <table className={styles.reliabilityTable}>
            <thead>
              <tr>
                <th>Model</th>
                <th>API errors</th>
                <th>AILANG / Python</th>
                <th>Refusals</th>
              </tr>
            </thead>
            <tbody>
              {modelRows.map((r) => (
                <tr key={r.name}>
                  <td>{formatModelName(r.name)}</td>
                  <td>
                    {r.reliability.apiErrorCount}
                    <span className={styles.reliabilityPct}>
                      {' '}({((r.reliability.apiErrorRate || 0) * 100).toFixed(1)}%)
                    </span>
                  </td>
                  <td>
                    {r.reliability.ailangApiError || 0} / {r.reliability.pythonApiError || 0}
                  </td>
                  <td>{r.reliability.refusalCount || 0}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <p className={styles.reliabilityHint}>
            API errors = provider quota, key revocation, 403s — NOT code quality.
            These runs are excluded from the "Success Rate by Model" chart when they dominate a release.
          </p>
        </div>
      )}
    </div>
  );
}
