import React from 'react';
import { scopeStats, costMetric, timingMetric, isProvisional, modelName, dollars, milliseconds, costLabels, timingLabels } from '../../lib/benchmarkEvidence.mjs';
import styles from './styles.module.css';

const count = (n, runs) => n == null ? 'Not retained' : `${n} of ${runs}`;
const tokens = (total, samples, runs) => !samples ? 'Not retained' : samples < runs ? `${total} (${samples}/${runs} runs retained)` : total;

export default function EvidenceDetail({ row, scope, lang, onClose }) {
  const s = scopeStats(row, scope, lang), all = scopeStats(row, 'all', lang);
  const cost = costMetric(s), timing = timingMetric(s);
  return <section className={styles.detail} aria-label={`Evidence for ${modelName(row.model)}`}>
    <div className={styles.detailHead}><div><h3>{modelName(row.model)}</h3><p className={styles.note}>{row.mode === 'agent' ? row.harness : 'Direct API'} · {scope} · {lang}</p><code>{row.model}</code></div>
      <button type="button" className={styles.close} onClick={onClose}>Close evidence</button></div>
    {!s ? <p>No measurements retained for this model in the selected task group and language.</p> : <>
      <div className={styles.detailGrid}>
        <p>Final passes<strong>{count(s.passes, s.runs)}</strong></p>
        <p>First-attempt passes<strong>{s.firstAttemptUnknown > 0 ? 'Incomplete record' : count(s.firstAttemptPasses, s.runs)}</strong></p>
        <p>Cost per solved task<strong>{dollars(cost.value)}</strong><span>{costLabels[cost.kind]}</span></p>
        <p>Median timing<strong>{milliseconds(timing.value)}</strong><span>{timingLabels[timing.kind]}</span></p>
        <p>Repairs used<strong>{s.repairs ?? 'Not retained'}</strong></p>
        <p>Output tokens<strong>{tokens(s.outputTokens, s.outputTokenSamples, s.runs)}</strong></p>
        <p>Total recorded tokens<strong>{tokens(s.totalTokens, s.totalTokenSamples, s.runs)}</strong></p>
        <p>Core coverage, all tasks<strong>{all?.coreCompleted?.length ?? 'Unknown'} / {row.requiredCoreIds.length || '?'}</strong><span>{isProvisional(row, lang) ? 'Provisional' : 'All required Core IDs covered'}</span></p>
      </div>
      <p className={styles.note}>Measurement dates: {s.firstRun?.slice(0,10) || 'not retained'} to {s.lastRun?.slice(0,10) || 'not retained'}.
        {' '}Latest reported version: {s.latestVersion || 'not retained'}. A latest version does not apply to every row in a mixed-version rollup.</p>
      <details><summary>Contributing versions and recorded outcomes</summary>
        <div className={styles.outcomes}>{Object.entries(s.versions || {}).map(([v,n]) => <span key={v}>{v}: <strong>{n} runs</strong></span>)}
          {s.unknownVersionRuns > 0 && <span>Version unavailable: {s.unknownVersionRuns} runs</span>}</div>
        {!Object.keys(s.versions || {}).length && !s.unknownVersionRuns && <p className={styles.note}>Version provenance was not retained.</p>}
        <p className={styles.note}>Prompt versions: {Object.entries(s.promptVersions || {}).map(([v,n]) => `${v} (${n} runs)`).join(', ') || 'Not retained'}.</p>
        <div className={styles.outcomes}>{Object.entries(s.outcomes || {}).map(([name,n]) => <span key={name}>{name.replaceAll('_',' ')}: <strong>{n}</strong></span>)}</div>
      </details>
      <details className={styles.definitions}><summary>Cost categories and timing boundaries</summary>
        <div className={styles.tableWrap}><table className={styles.table}><thead><tr><th>Cost provenance</th><th>Runs / passes</th><th>Recorded total</th><th>Cost / success</th></tr></thead>
          <tbody>{Object.entries(s.costs || {}).map(([kind,c]) => <tr key={kind}><td>{costLabels[kind] || kind}</td><td>{c.runs} / {c.passes}</td><td>{dollars(c.totalUsd)}{!c.complete && <small>Incomplete subtotal</small>}</td><td>{dollars(c.perSuccessUsd)}</td></tr>)}</tbody></table></div>
        {!Object.keys(s.costs || {}).length && <p>Scoped cost was not retained.</p>}
        <p className={styles.note}>Failed-attempt costs are included within each category. Subscription-equivalent cost is not a metered bill; unknown billing provenance is not assumed metered. A zero-success denominator is unavailable.</p>
        <div className={styles.outcomes}>{Object.entries(s.timing || {}).map(([kind,t]) => <span key={kind}>{timingLabels[kind] || kind}: <strong>{milliseconds(t.medianMs)}</strong> ({t.samples} samples)</span>)}</div>
        <p className={styles.note}>Generation wall time measures recorded generation calls. Agent time to success uses recorded first passing-solution time. Recorded execution duration is a different boundary and is not claimed as full generation latency.</p>
      </details>
      <details className={styles.definitions}><summary>Tasks behind this measurement ({s.benchmarks?.length || 'IDs not retained'})</summary>
        <ul className={styles.taskLinks}>{(s.benchmarks || []).map(id => <li key={id}><a href={`/docs/benchmarks/gallery#bench-${encodeURIComponent(id)}`}>{id.replaceAll('_',' ')}</a></li>)}</ul>
        {!s.benchmarks?.length && <p>Task identities were not retained in this legacy projection.</p>}
      </details>
    </>}
  </section>;
}
