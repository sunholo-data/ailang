import React, { useEffect, useMemo, useState } from 'react';
import { benchmarkFetchWithSource } from '../../lib/benchmarkFetch';
import { evidenceRows, scopeStats, sortRows } from '../../lib/benchmarkEvidence.mjs';
import ComparisonTable from './ComparisonTable';
import OverallTrends from './OverallTrends';
import EvidenceDetail from './EvidenceDetail';
import HistoryAccess from './HistoryAccess';
import styles from './styles.module.css';

export default function BenchmarkPerformance() {
  const [snapshot, setSnapshot] = useState(null), [error, setError] = useState(null);
  useEffect(() => {
    let active = true;
    benchmarkFetchWithSource('latest.json').then(async ({ response, source }) => {
      if (!response.ok) throw new Error(`Benchmark data request failed (${response.status})`);
      const latest = await response.json();
      const rows = evidenceRows(latest);
      if (active) setSnapshot({ latest, rows, source });
    }).catch(e => { if (active) setError(e.message); });
    return () => { active = false; };
  }, []);
  if (error) return <div role="alert" className={styles.notice}>Could not load benchmark evidence: {error}. Reload this page to retry.</div>;
  if (!snapshot) return <p role="status">Loading saved benchmark evidence…</p>;
  return <PerformanceView {...snapshot} />;
}

export function PerformanceView({ latest, rows, source }) {
  const [mode, setMode] = useState('standard'), [lang, setLang] = useState('ailang');
  const [scope, setScope] = useState('all'), [search, setSearch] = useState(''), [sort, setSort] = useState('elo');
  const [showPython, setShowPython] = useState(true), [selected, setSelected] = useState(null);
  const modeRows = useMemo(() => rows.filter(r => r.mode === mode), [rows, mode]);
  const scopes = Array.from(new Set(modeRows.flatMap(r => Object.keys(r.scopes))));
  const visible = useMemo(() => sortRows(modeRows.filter(r =>
    (scopeStats(r, scope, lang) || (lang === 'ailang' && showPython && scopeStats(r, scope, 'python'))) &&
    `${r.model} ${r.harness}`.toLowerCase().includes(search.toLowerCase())), { scope, lang, sort }),
  [modeRows, scope, lang, showPython, search, sort]);
  const selectedRow = modeRows.find(r => r.id === selected);
  const hasLegacy = modeRows.some(r => r.legacy);
  function download() {
    const url = URL.createObjectURL(new Blob([JSON.stringify(latest, null, 2)], { type: 'application/json' }));
    const a = document.createElement('a'); a.href = url; a.download = 'ailang-benchmark-snapshot.json'; a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
  return <div className={styles.page}>
    <div className={styles.provenance}><strong>Latest available evidence</strong><span>Published {latest.timestamp?.slice(0, 10) || 'date unavailable'}</span>
      <span>Snapshot {latest.version || 'version unavailable'}</span>{source === 'fallback' && <span className={styles.badge}>Static fallback copy</span>}</div>
    <p className={styles.intro}>Compare model capability, success rates and cost using saved results. Each measurement keeps its original AILANG version.</p>
    <nav className={styles.nav} aria-label="Benchmark views"><a href="#overall-trends">Overall trends</a><a href="#compare-models">Model comparison</a>
      <a href="#saved-history">Saved history</a><a href="/docs/benchmarks/gallery">Tasks</a><a href="/docs/benchmarks/elo">ELO & difficulty</a><a href="/docs/benchmarks/value">Value & economics</a></nav>
    <OverallTrends latest={latest} />
    <section id="compare-models" className={styles.section}>
      <div className={styles.controls}>
        <div className={styles.mode} role="group" aria-label="Evaluation mode">{[['standard', 'Direct model'], ['agent', 'Coding agent']].map(([value, label]) =>
          <button type="button" key={value} aria-pressed={mode === value} onClick={() => { setMode(value); setScope('all'); setSelected(null); }}>{label}</button>)}</div>
        <label>Language<select value={lang} onChange={e => setLang(e.target.value)}><option value="ailang">AILANG</option><option value="python">Python</option></select></label>
        <label>Tasks<select value={scope} onChange={e => setScope(e.target.value)}>{['all', 'smoke', 'core', 'stretch', 'frontier', 'long-frontier', 'vision', 'experimental', 'unclassified'].filter(s => scopes.includes(s)).map(s => <option key={s} value={s}>{s === 'all' ? 'All measured tasks' : s}</option>)}</select></label>
        {lang === 'ailang' && <label className={styles.check}><input type="checkbox" checked={showPython} onChange={e => setShowPython(e.target.checked)} />Python reference</label>}
      </div>
      <p className={styles.note}>{mode === 'standard' ? 'Direct API results, after the recorded self-repair policy.' : 'Multi-turn results for each model and its named coding harness.'}
        {' '}ELO uses all tasks in this mode and language; pass rates, cost and timing use the selected tasks.</p>
      {hasLegacy && <p className={styles.notice}>Legacy snapshot: scoped cost, timing or provenance may not have been retained. Missing values are shown explicitly.</p>}
      <div className={styles.tableTools}><div><h2>Compare models</h2><span>{visible.length} {mode === 'agent' ? 'model + harness configurations' : 'models'}</span></div>
        <div className={styles.controls}><label>Find a model<input type="search" value={search} onChange={e => setSearch(e.target.value)} placeholder="Model or harness…" /></label>
          <label>Order by<select value={sort} onChange={e => setSort(e.target.value)}><option value="elo">ELO, undefeated first</option><option value="rate">Pass rate</option><option value="cost">Cost per solved task</option><option value="name">Model name</option></select></label></div></div>
      <p className={styles.note}>Different model/task samples are descriptive results, not a matched-task language delta. Undefeated groups are not a fitted joint rank.</p>
      {visible.length ? <ComparisonTable rows={visible} scope={scope} lang={lang} sort={sort} showPython={showPython} onSelect={setSelected} selected={selected} />
        : <p role="status" className={styles.notice}>No saved measurements match this mode, language, task group and search. Choose another scope or clear the search.</p>}
      {selectedRow && <EvidenceDetail row={selectedRow} scope={scope} lang={lang} onClose={() => setSelected(null)} />}
      <p className={styles.note}>— means not measured or not retained. Measured zero is shown as zero. Select a model for its recorded evidence.</p>
    </section>
    <HistoryAccess latest={latest} onDownload={download} />
    <details className={styles.definitions}><summary>How to read this evidence</summary><p>ELO estimates capability from task difficulty. Undefeated models have no observed loss to calibrate a finite anchored rating. Smaller or incomplete Core samples remain provisional.</p>
      <p>Cost per solved task includes failed-attempt spending in the same scope. Metered dollars, subscription list-price equivalents and free-local cost remain separate. Timing states its measurement boundary.</p>
      <p>Snapshots may reuse older measurements. A new language release does not require a new model run. Historical model/task mixes change; only a fixed-panel comparison supports a claim about language progress alone.</p>
      <a href="/docs/benchmarks/overview">Benchmark methodology and the other research views</a></details>
  </div>;
}
