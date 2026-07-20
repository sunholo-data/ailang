import React, { useState, useEffect } from 'react';
import { benchmarkFetch } from '@site/src/lib/benchmarkFetch';

// OS / Local-model leaderboard (M-EVAL-BENCHMARK-UI-CONSOLIDATION phase C).
// Renders cross-language × harness pass rates for open/locally-hosted models from
// STATIC data published by `ailang eval-publish` into /benchmarks/os/latest.json
// (zero server cost). Degrades to a placeholder when no rotation has been
// published yet — so the live site never shows fake numbers.
//
// Expected schema (/benchmarks/os/latest.json):
// {
//   "version": "v0.26.0",                  // rotation/release tag ("sample-*" => banner)
//   "generated": "2026-06-13",
//   "trials": 3,                           // N trials per (model, benchmark, lang)
//   "languages": ["ailang","python","javascript","go"],
//   "rows": [
//     { "model": "gemma4-26b", "harness": "opencode",
//       "lang": { "ailang": 0.55, "python": 0.82, "javascript": 0.71, "go": 0.63 } }
//   ]
// }

const LANG_LABEL = { ailang: 'AILANG', python: 'Python', javascript: 'JavaScript', go: 'Go' };
const LANG_ORDER = ['ailang', 'python', 'javascript', 'go'];

function heat(rate) {
  if (rate == null) return 'transparent';
  if (rate >= 0.85) return 'rgba(34,197,94,0.25)';
  if (rate >= 0.70) return 'rgba(34,197,94,0.12)';
  if (rate >= 0.50) return 'rgba(234,179,8,0.18)';
  if (rate >= 0.30) return 'rgba(249,115,22,0.18)';
  return 'rgba(239,68,68,0.20)';
}

export default function OSLocalLeaderboard() {
  const [data, setData] = useState(undefined); // undefined=loading, null=absent

  useEffect(() => {
    benchmarkFetch('os/latest.json')
      .then((r) => (r.ok ? r.json() : null))
      .then(setData)
      .catch(() => setData(null));
  }, []);

  if (data === undefined) return <p>Loading local-rig data…</p>;

  if (!data || !Array.isArray(data.rows) || data.rows.length === 0) {
    return (
      <div style={{
        padding: '16px 18px', border: '1px dashed var(--ifm-color-emphasis-300)',
        borderRadius: 8, color: 'var(--ifm-color-emphasis-700)',
      }}>
        <strong>No local-rig rotation published yet.</strong> Cross-language (incl. JS &amp; Go) and
        cross-harness numbers appear here once a rotation is published with{' '}
        <code>ailang eval-publish</code>. Cloud AILANG-vs-Python results are on the{' '}
        <a href="/docs/benchmarks/performance">Model Leaderboard</a> and{' '}
        <a href="/docs/benchmarks/elo">ELO</a> pages.
      </div>
    );
  }

  const langs = (data.languages && data.languages.length ? data.languages : LANG_ORDER).slice().sort(
    (a, b) => {
      const ia = LANG_ORDER.indexOf(a), ib = LANG_ORDER.indexOf(b);
      if (ia < 0 && ib < 0) return a.localeCompare(b);
      if (ia < 0) return 1; if (ib < 0) return -1;
      return ia - ib;
    }
  );
  const isSample = typeof data.version === 'string' && data.version.startsWith('sample');

  return (
    <div>
      <div style={{ marginBottom: 10, fontSize: '0.88em', color: 'var(--ifm-color-emphasis-700)' }}>
        {/* The AILANG release the runs executed against is the headline; the
            rotation snapshot tag (rolling-YYYYMMDD) is provenance detail. */}
        {data.ailang_version
          ? <span><strong>AILANG {data.ailang_version}</strong> · rotation {data.version}</span>
          : (data.version && <strong>{data.version}</strong>)}
        {data.trials != null && <span> · N={data.trials} trials</span>}
        {data.generated && <span> · {data.generated}</span>}
        {isSample && (
          <span style={{ marginLeft: 8, color: '#b45309', fontWeight: 700 }}>SAMPLE DATA</span>
        )}
      </div>
      <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.9em' }}>
        <thead>
          <tr style={{ borderBottom: '2px solid var(--ifm-color-emphasis-300)', textAlign: 'left' }}>
            <th style={{ padding: '6px 10px' }}>Model</th>
            <th style={{ padding: '6px 10px' }}>Harness</th>
            {langs.map((l) => (
              <th key={l} style={{ padding: '6px 10px', textAlign: 'center' }}>{LANG_LABEL[l] || l}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.rows.map((row, i) => (
            <tr key={i} style={{ borderBottom: '1px solid var(--ifm-color-emphasis-200)' }}>
              <td style={{ padding: '6px 10px', fontWeight: 600 }}>{row.model}</td>
              <td style={{ padding: '6px 10px', color: 'var(--ifm-color-emphasis-700)' }}>{row.harness || '—'}</td>
              {langs.map((l) => {
                const rate = row.lang ? row.lang[l] : null;
                return (
                  <td key={l} style={{
                    padding: '6px 10px', textAlign: 'center', background: heat(rate),
                    fontWeight: rate >= 0.85 ? 700 : 400,
                  }}>
                    {rate == null ? '—' : `${Math.round(rate * 100)}%`}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
