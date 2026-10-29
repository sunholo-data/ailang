import React from 'react';
import styles from './styles.module.css';

export default function HistoryAccess({ latest, onDownload }) {
  return <section id="saved-history" className={styles.history}>
    <details><summary>Browse all {latest.history?.length || 0} saved release snapshots</summary>
      <p className={styles.note}>These are published snapshots of saved evidence, not a promise that each model was rerun at every release. Old fields and values remain available in the full download.</p>
      <div className={styles.tableWrap}><table className={styles.table}><thead><tr><th>Snapshot</th><th>Published</th><th>Saved runs</th><th>Model ELO retained</th></tr></thead>
        <tbody>{(latest.history || []).map((h,i) => <tr key={`${h.version}/${i}`}><td>{h.version}</td><td>{h.timestamp?.slice(0,10) || 'Not retained'}</td><td>{h.totalRuns ?? 'Not retained'}</td><td>{Object.keys(h.ratings?.models || {}).length || 'Not retained'}</td></tr>)}</tbody></table></div>
    </details>
    <button className={styles.download} type="button" onClick={onDownload}>Download the full published snapshot</button>
    <p className={styles.note}>For more detail: <a href="/docs/benchmarks/explorer">agent harness comparisons</a>, <a href="/docs/benchmarks/os-model-leaderboard">local models</a>, <a href="/docs/benchmarks/gallery">task specs and solutions</a>, and <a href="/docs/benchmarks/value">cost and value metrics</a>.</p>
  </section>;
}
