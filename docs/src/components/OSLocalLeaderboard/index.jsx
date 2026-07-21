import React, { useState, useEffect } from 'react';
import { benchmarkFetch } from '@site/src/lib/benchmarkFetch';
import { buildCoverage } from '@site/src/components/BenchmarkDashboard/coverageGate';
import { localRate, formatLocalName } from '@site/src/lib/localModel';

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

// Tier breakdown is carried per-row in os/latest.json but was never surfaced: a
// single blended pass rate hides that (e.g.) frontier is 0% while core is ~89%.
const TIER_ORDER = ['core', 'stretch', 'frontier', 'vision', 'smoke'];

function tierColumns(rows) {
  const seen = new Set();
  for (const r of rows) {
    for (const t of Object.keys((r && r.tiers) || {})) seen.add(t);
  }
  // NOTE: Array.from, never [...new Set()] — the prod Docusaurus Babel config
  // lowers array-spread to [].concat(), which does NOT spread a Set (dev/SSR use
  // native spread and never catch it).
  return Array.from(seen).sort((a, b) => {
    const ia = TIER_ORDER.indexOf(a), ib = TIER_ORDER.indexOf(b);
    if (ia < 0 && ib < 0) return a.localeCompare(b);
    if (ia < 0) return 1; if (ib < 0) return -1;
    return ia - ib;
  });
}

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
  const [coverage, setCoverage] = useState(null); // null => coverage unknown
  const [main, setMain] = useState(null);         // latest.json — canonical rate source

  useEffect(() => {
    benchmarkFetch('os/latest.json')
      .then((r) => (r.ok ? r.json() : null))
      .then(setData)
      .catch(() => setData(null));

    // os/latest.json carries no denominator, so a partly-filled rotation used to
    // render as a bare pass rate. The main leaderboard's ratings block already
    // tracks per-model `benchmarks` + `maxCoverage` for these same model ids —
    // reuse it so this page gates coverage the same way ELO/performance do.
    // Failure is non-fatal: coverage stays unknown and rows render unbadged.
    benchmarkFetch('latest.json')
      .then((r) => (r.ok ? r.json() : null))
      .then((j) => {
        setMain(j || null);
        setCoverage(j && j.ratings ? buildCoverage(j.ratings) : null);
      })
      .catch(() => { setMain(null); setCoverage(null); });
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
  const tiers = tierColumns(data.rows);
  // Tier cells are AILANG-only: the tier breakdown exists to show where the
  // headline AILANG number comes from, and breaking out every tier x language
  // would be a 12+ column table.
  const tierLang = langs.includes('ailang') ? 'ailang' : langs[0];
  const maxCov = coverage ? coverage.maxCoverage : null;
  const anyProvisional = coverage && data.rows.some((r) => coverage.isProvisional(r.model));

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
            {tiers.map((t) => (
              <th key={`t-${t}`} style={{ padding: '6px 10px', textAlign: 'center', fontWeight: 400, color: 'var(--ifm-color-emphasis-600)' }}
                  title={`${LANG_LABEL[tierLang] || tierLang} pass rate on the ${t} tier`}>
                {t}
              </th>
            ))}
            {coverage && (
              <th style={{ padding: '6px 8px', textAlign: 'right', fontWeight: 400, color: 'var(--ifm-color-emphasis-500)' }}
                  title="distinct benchmarks run so far (of the max any model ran)">
                cov
              </th>
            )}
          </tr>
        </thead>
        <tbody>
          {data.rows.map((row, i) => {
            const prov = coverage ? coverage.isProvisional(row.model) : false;
            const nBench = coverage ? coverage.benchmarksFor(row.model) : null;
            return (
              <tr key={i} style={{ borderBottom: '1px solid var(--ifm-color-emphasis-200)', opacity: prov ? 0.72 : 1 }}>
                <td style={{ padding: '6px 10px', fontWeight: 600, fontStyle: prov ? 'italic' : 'normal' }}
                    title={row.model}>
                  {formatLocalName(row.model)}
                </td>
                <td style={{ padding: '6px 10px', color: 'var(--ifm-color-emphasis-700)' }}>{row.harness || '—'}</td>
                {langs.map((l) => {
                  // CANONICAL rate: latest.json (runs-based), the same accumulator the
                  // cloud tables use. os/latest.json's row.lang divides by TRIALS and
                  // publishes on its own cadence, so it drifts — it produced 88.9% here
                  // while the performance page showed 92.3% for the same model/release.
                  // Fall back to the os value only when latest.json has no agent entry.
                  const canon = main ? localRate(main, row.model, l) : null;
                  const rate = canon ? canon.rate : (row.lang ? row.lang[l] : null);
                  return (
                    <td key={l} style={{
                      padding: '6px 10px', textAlign: 'center', background: heat(rate),
                      fontWeight: rate >= 0.85 ? 700 : 400,
                    }}
                        title={canon ? `${Math.round(canon.rate * canon.runs)}/${canon.runs} runs` : 'from os/latest.json (trials-based) — no agent entry in latest.json'}>
                      {rate == null ? '—' : `${Math.round(rate * 100)}%`}
                    </td>
                  );
                })}
                {tiers.map((t) => {
                  // A tier absent from this row was not run; a tier present with 0
                  // genuinely scored 0 — these must not render the same way.
                  const cell = row.tiers && row.tiers[t];
                  const rate = cell ? cell[tierLang] : null;
                  return (
                    <td key={`t-${t}`} style={{
                      padding: '6px 10px', textAlign: 'center', background: heat(rate),
                      fontSize: '0.92em', color: rate == null ? 'var(--ifm-color-emphasis-500)' : undefined,
                    }}
                        title={rate == null ? `${t}: not run in this rotation` : undefined}>
                      {rate == null ? '—' : `${Math.round(rate * 100)}%`}
                    </td>
                  );
                })}
                {coverage && (
                  <td style={{
                    padding: '6px 8px', textAlign: 'right', fontVariantNumeric: 'tabular-nums',
                    fontSize: '0.85em', color: prov ? '#b45309' : 'var(--ifm-color-emphasis-500)',
                    fontWeight: prov ? 700 : 400,
                  }}
                      title={nBench == null
                        ? 'coverage unknown for this model'
                        : (prov
                          ? `provisional — only ${nBench} of ${maxCov} benchmarks run so far; this rate will move as the rotation fills in`
                          : `${nBench} of ${maxCov} benchmarks run`)}>
                    {nBench == null ? '—' : `${nBench}/${maxCov}`}
                  </td>
                )}
              </tr>
            );
          })}
        </tbody>
      </table>
      {anyProvisional && (
        <p style={{ fontSize: '0.8em', color: 'var(--ifm-color-emphasis-600)', marginTop: 6 }}>
          <span style={{ color: '#b45309', fontWeight: 700 }}>Provisional</span> rows (italic, low{' '}
          <strong>cov</strong>) have run only a fraction of the {maxCov} benchmarks — these rates are
          a partial sample of the release and will move as the rotation fills coverage in. The rig
          runs AILANG to full coverage first, so cross-language columns appear only once that lap
          completes.
        </p>
      )}
    </div>
  );
}
