import React from 'react';
import {
  deriveHeadlineKpiState,
  formatUSD4,
  KPI_STATE_ABSENT,
  KPI_STATE_INCOMPLETE,
  KPI_STATE_ZERO_DENOM,
  KPI_STATE_AVAILABLE,
} from '@site/src/lib/headlineKpiState';

/**
 * HeadlineKpiCard — the v1.0 top-line metric: Cost per verified success.
 *
 * Renders the additive latest.json `headlineKpis.costPerVerifiedSuccess` object
 * (the exact struct the observatory rollup / CLI / HTTP emit). All display logic
 * is derived by the PURE deriveHeadlineKpiState() so it is unit-tested without a
 * browser. The card handles four states — available, zero-denominator,
 * incomplete, absent — and shows a visible Fallback / stale-data badge when the
 * data came from the in-build static copy instead of the runtime GCS route.
 *
 * A verified success = compile + runtime + stdout pass AND affirmative
 * ai-check/Z3 proof (>=1 verified obligation, zero counterexamples/skipped/errors).
 * This is distinct from the per-model "$/success" (cost per benchmark pass) shown
 * below in the Value Score table — that value is intentionally NOT redefined here.
 */
export default function HeadlineKpiCard({ latest, source }) {
  const s = deriveHeadlineKpiState(latest, source);

  const wrap = {
    border: '1px solid var(--ifm-color-emphasis-300)',
    borderRadius: 10,
    padding: '18px 20px',
    marginBottom: 20,
    background: 'var(--ifm-background-surface-color)',
  };
  const label = {
    fontSize: '0.8em',
    textTransform: 'uppercase',
    letterSpacing: '0.06em',
    color: 'var(--ifm-color-emphasis-600)',
    fontWeight: 700,
  };
  const big = { fontSize: '2.2em', fontWeight: 800, lineHeight: 1.1, margin: '4px 0' };
  const sub = { fontSize: '0.9em', color: 'var(--ifm-color-emphasis-700)' };

  const fallbackBadge = s.isFallback ? (
    <span
      title="Served from the in-build static copy, not the live GCS/Cloud Run route. This data may be stale."
      style={{
        marginLeft: 10,
        fontSize: '0.7em',
        fontWeight: 700,
        color: '#8a5300',
        background: '#fff3cd',
        border: '1px solid #ffe08a',
        borderRadius: 6,
        padding: '2px 8px',
        verticalAlign: 'middle',
      }}
    >
      ⚠ Fallback / stale data
    </span>
  ) : null;

  const header = (
    <div style={label}>
      Cost per verified success{fallbackBadge}
    </div>
  );

  // ABSENT — no headline object banked yet (pre-M4). Degrade, never break.
  if (s.state === KPI_STATE_ABSENT) {
    return (
      <div style={wrap}>
        {header}
        <div style={{ ...big, color: 'var(--ifm-color-emphasis-500)' }}>—</div>
        <div style={sub}>
          Baseline unavailable — no verified cohort has been published yet. The v1.0
          measured baseline is pending ratification.
        </div>
      </div>
    );
  }

  // INCOMPLETE — backend said available=false for a reason other than an empty
  // denominator (e.g. an unknown-cost stage). Never show $0.
  if (s.state === KPI_STATE_INCOMPLETE) {
    return (
      <div style={wrap}>
        {header}
        <div style={{ ...big, color: 'var(--ifm-color-warning-dark, #b26a00)' }}>Incomplete</div>
        <div style={sub}>
          The KPI cannot be published for baseline{' '}
          <code>{s.baselineId || 'v1.0'}</code>: {reasonText(s.reason)}
          {s.unknownStages ? ` (${s.unknownStages} unattributed-cost stage(s))` : ''}. Refusing
          to display $0 or a stale value.
        </div>
      </div>
    );
  }

  // ZERO DENOMINATOR — runs banked but no verified success. Distinct message.
  if (s.state === KPI_STATE_ZERO_DENOM) {
    return (
      <div style={wrap}>
        {header}
        <div style={{ ...big, color: 'var(--ifm-color-warning-dark, #b26a00)' }}>No verified successes</div>
        <div style={sub}>
          {fmtInt(s.totalRuns)} run(s) in baseline <code>{s.baselineId || 'v1.0'}</code>, but
          0 completed contract verification — the denominator is zero, so no cost-per-verified-success
          can be computed.
        </div>
      </div>
    );
  }

  // AVAILABLE — the real number.
  return (
    <div style={wrap}>
      {header}
      <div style={{ ...big, color: 'var(--ifm-color-primary)' }}>{formatUSD4(s.valueUSD)}</div>
      <div style={sub}>
        <strong>{fmtInt(s.verifiedSuccesses)}</strong> verified successes /{' '}
        <strong>{fmtInt(s.totalRuns)}</strong> total runs ·{' '}
        <strong>{formatUSD4(s.knownCostUSD)}</strong> known metered cost (Reported + Estimated,
        includes failed runs)
        {s.quotaStages ? ` · ${s.quotaStages} quota stage(s) at $0` : ''}
        {s.baselineId ? ` · baseline ${s.baselineId}` : ''}
      </div>
      <div style={{ ...sub, marginTop: 4, fontSize: '0.8em', color: 'var(--ifm-color-emphasis-500)' }}>
        Verified = compile + runtime + stdout pass with an affirmative ai-check/Z3 proof (≥1 obligation,
        no counterexamples, no skipped obligations, no verifier errors). Distinct from the per-model
        “$/success” below, which is cost per benchmark pass.
      </div>
    </div>
  );
}

function reasonText(reason) {
  switch (reason) {
    case 'unknown_cost':
      return 'one or more cohort stages have unattributable (unknown) cost';
    case 'zero_denominator':
      return 'there are no verified successes';
    case 'empty_cohort':
      return 'the cohort is empty';
    default:
      return 'the cohort data is incomplete';
  }
}

function fmtInt(v) {
  return typeof v === 'number' && !Number.isNaN(v) ? String(v) : '—';
}
