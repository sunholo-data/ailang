import React, { useState, useEffect, useMemo } from 'react';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend,
  ResponsiveContainer, ReferenceLine,
} from 'recharts';
import { benchmarkFetch } from '@site/src/lib/benchmarkFetch';

// OSReleaseTrend — release-over-release evolution of LOCAL model performance,
// rendered from /benchmarks/os/history.json (one entry per AILANG release,
// appended by tools/os-release-snapshot.sh / os-rotation-filler.sh).
//
// Design constraints this component encodes (learned from the v0.26→v0.30 data):
//  - "Overall" pass rate is NOT comparable across releases: rotations differ in
//    tier coverage (v0.28.0/v0.29.0 never ran frontier/vision) and trial count,
//    so the default view is a single stable tier, with the coverage caveat
//    rendered per release instead of hidden.
//  - The smoke tier is saturated (~100% for every model since v0.26.0) — it
//    carries no discriminative signal, mirroring the `saturated` flag on the
//    ELO benchmark ratings. It stays selectable but is labeled as such.
//  - The "AILANG − Python gap" metric is the cleanest read on whether LANGUAGE
//    improvements land: Python runs as a control arm on the same model, harness
//    and benchmarks, so model/harness noise subtracts out.

const TIER_OPTIONS = [
  { id: 'core',     label: 'Core' },
  { id: 'stretch',  label: 'Stretch' },
  { id: 'frontier', label: 'Frontier' },
  { id: 'vision',   label: 'Vision' },
  { id: 'smoke',    label: 'Smoke (saturated)' },
  { id: 'overall',  label: 'Overall (mixed coverage)' },
];

const METRIC_OPTIONS = [
  { id: 'ailang', label: 'AILANG pass rate' },
  { id: 'gap',    label: 'AILANG − Python gap' },
];

// Fixed colors by harness; model generation (qwen3-5 vs qwen3-6) becomes the
// dash pattern so five models stay readable as three color families.
const HARNESS_COLOR = { motoko: '#0891b2', opencode: '#8b5cf6', pi: '#16a34a' };

function harnessOf(model) {
  if (model.startsWith('motoko')) return 'motoko';
  if (model.startsWith('opencode')) return 'opencode';
  if (model.startsWith('pi')) return 'pi';
  return 'other';
}

function shortModel(model) {
  // motoko-local-qwen3-6-35b-a3b-mxfp8 → "motoko · qwen3-6"
  const h = harnessOf(model);
  const m = model.match(/qwen3-(\d+)/);
  return m ? `${h} · qwen3-${m[1]}` : model;
}

function semverKey(v) {
  const p = String(v || '').replace(/^v/, '').split('.').map((n) => parseInt(n, 10) || 0);
  return p[0] * 1e6 + (p[1] || 0) * 1e3 + (p[2] || 0);
}

// rate for (row, tier, metric) — null when the rotation didn't run that tier.
function metricValue(row, tier, metric) {
  const langMap = tier === 'overall' ? row.lang : (row.tiers || {})[tier];
  if (!langMap) return null;
  const a = langMap.ailang;
  if (metric === 'ailang') return typeof a === 'number' ? a * 100 : null;
  const p = langMap.python;
  if (typeof a !== 'number' || typeof p !== 'number') return null;
  return (a - p) * 100;
}

export default function OSReleaseTrend() {
  const [history, setHistory] = useState(undefined); // undefined=loading, null/[]=absent
  const [tier, setTier] = useState('core');
  const [metric, setMetric] = useState('ailang');

  useEffect(() => {
    benchmarkFetch('os/history.json')
      .then((r) => (r.ok ? r.json() : []))
      .then((h) => setHistory(Array.isArray(h) ? h : []))
      .catch(() => setHistory([]));
  }, []);

  const { chartData, models, coverage } = useMemo(() => {
    if (!Array.isArray(history) || history.length === 0) {
      return { chartData: [], models: [], coverage: [] };
    }
    const entries = history
      .filter((e) => e && e.ailang_version && Array.isArray(e.rows))
      .slice()
      .sort((a, b) => semverKey(a.ailang_version) - semverKey(b.ailang_version));

    // NOTE: Array.from, never [...new Set(...)] — Docusaurus prod Babel lowers
    // array-spread to [].concat which does not spread a Set.
    const models = Array.from(new Set(
      entries.flatMap((e) => e.rows.map((r) => r.model).filter(Boolean))
    ));

    const allTiers = ['core', 'stretch', 'frontier', 'vision', 'smoke'];
    const chartData = entries.map((e) => {
      const point = { version: e.ailang_version };
      e.rows.forEach((r) => { if (r && r.model) point[r.model] = metricValue(r, tier, metric); });
      return point;
    });

    const coverage = entries.map((e) => {
      const ran = new Set();
      e.rows.forEach((r) => Object.keys(r.tiers || {}).forEach((t) => ran.add(t)));
      const missing = allTiers.filter((t) => !ran.has(t));
      return { version: e.ailang_version, trials: e.trials, generated: e.generated, missing };
    });

    return { chartData, models, coverage };
  }, [history, tier, metric]);

  if (history === undefined) return <p>Loading release history…</p>;
  if (!chartData.length) {
    return (
      <div style={{
        padding: '16px 18px', border: '1px dashed var(--ifm-color-emphasis-300)',
        borderRadius: 8, color: 'var(--ifm-color-emphasis-700)',
      }}>
        <strong>No release history published yet.</strong> Per-release entries appear in{' '}
        <code>/benchmarks/os/history.json</code> once the rotation filler snapshots a release.
      </div>
    );
  }

  const isGap = metric === 'gap';
  const selBtn = (active) => ({
    padding: '3px 10px', borderRadius: 6, cursor: 'pointer', fontSize: '0.85em',
    border: `1px solid ${active ? 'var(--ifm-color-primary)' : 'var(--ifm-color-emphasis-300)'}`,
    background: active ? 'var(--ifm-color-primary)' : 'transparent',
    color: active ? '#fff' : 'var(--ifm-color-emphasis-700)',
  });

  return (
    <div>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 6 }}>
        {METRIC_OPTIONS.map((m) => (
          <button key={m.id} style={selBtn(metric === m.id)} onClick={() => setMetric(m.id)}>
            {m.label}
          </button>
        ))}
        <span style={{ width: 12 }} />
        {TIER_OPTIONS.map((t) => (
          <button key={t.id} style={selBtn(tier === t.id)} onClick={() => setTier(t.id)}>
            {t.label}
          </button>
        ))}
      </div>

      <ResponsiveContainer width="100%" height={320}>
        <LineChart data={chartData} margin={{ top: 8, right: 16, bottom: 4, left: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="var(--ifm-color-emphasis-200)" />
          <XAxis dataKey="version" tick={{ fontSize: 12 }} />
          <YAxis
            domain={isGap ? [-40, 20] : [0, 100]}
            tickFormatter={(v) => `${v > 0 && isGap ? '+' : ''}${v}${isGap ? 'pp' : '%'}`}
            tick={{ fontSize: 12 }}
            width={48}
          />
          <Tooltip
            formatter={(v, name) => [
              v == null ? '—' : `${v > 0 && isGap ? '+' : ''}${Math.round(v)}${isGap ? 'pp' : '%'}`,
              shortModel(name),
            ]}
            contentStyle={{
              background: 'var(--ifm-background-surface-color)',
              border: '1px solid var(--ifm-color-emphasis-300)', borderRadius: 6, fontSize: '0.85em',
            }}
          />
          <Legend formatter={(name) => shortModel(name)} wrapperStyle={{ fontSize: '0.82em' }} />
          {isGap && <ReferenceLine y={0} stroke="var(--ifm-color-emphasis-500)" strokeDasharray="4 4" />}
          {models.map((m) => (
            <Line
              key={m} type="monotone" dataKey={m}
              stroke={HARNESS_COLOR[harnessOf(m)] || '#64748b'}
              strokeDasharray={/qwen3-5/.test(m) ? '5 4' : undefined}
              strokeWidth={2} dot={{ r: 3 }} connectNulls={false}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>

      {/* Coverage caveats: WHY overall numbers jump between releases. */}
      <div style={{ marginTop: 6, fontSize: '0.8em', color: 'var(--ifm-color-emphasis-600)' }}>
        {coverage.map((c) => (
          <span key={c.version} style={{ marginRight: 14, whiteSpace: 'nowrap', display: 'inline-block' }}>
            <strong>{c.version}</strong>
            {c.trials != null && <> · N={c.trials}</>}
            {c.missing.length > 0 && (
              <span style={{ color: '#b45309' }}> · no {c.missing.join('/')} data</span>
            )}
          </span>
        ))}
      </div>
      {isGap ? (
        <p style={{ fontSize: '0.85em', color: 'var(--ifm-color-emphasis-700)', marginTop: 8 }}>
          <strong>0pp = parity with Python</strong> on the same model, harness and benchmarks —
          Python acts as the control arm, so this gap isolates AILANG-specific friction from
          model/harness noise. A gap trending toward 0 across releases means language, stdlib
          and prompt fixes are landing.
        </p>
      ) : (
        <p style={{ fontSize: '0.85em', color: 'var(--ifm-color-emphasis-700)', marginTop: 8 }}>
          Per-tier views compare like against like across releases. <strong>Overall</strong> mixes
          whatever tiers each rotation covered (see the coverage line above) and{' '}
          <strong>smoke</strong> has been saturated (~100%) since v0.26.0, so neither tracks
          release-over-release progress — <strong>core / stretch / frontier</strong> are the
          benchmarks with signal, matching the non-saturated set on the{' '}
          <a href="/docs/benchmarks/elo">ELO ratings</a> page.
        </p>
      )}
    </div>
  );
}
