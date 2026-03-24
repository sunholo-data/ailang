import React, { useState } from 'react';
import { usePackageDetail } from '@site/src/hooks/useRegistryData';
import styles from './styles.module.css';

/**
 * PackageDetail — renders on static package pages.
 * Shows static data immediately (no spinner), hydrates with live version timeline + provenance.
 */
export default function PackageDetail({ packageName, staticData }) {
  const [vendor, name] = packageName.split('/');
  const { data: liveDetail, stale } = usePackageDetail(vendor, name);

  const exports = staticData?.exports || [];
  const deps = staticData?.dependencies || [];
  const dependents = liveDetail?.dependents || [];
  const versions = liveDetail?.versions || [];

  return (
    <div className={styles.detail}>
      {stale && (
        <div className={styles.staleBanner}>
          Live data unavailable — showing cached information.
        </div>
      )}

      {/* Source / Docs / License links */}
      <PackageLinks detail={liveDetail} staticData={staticData} />

      <div className={styles.detailGrid}>
        {/* Exports */}
        <div className={styles.detailSection}>
          <div className={styles.sectionTitle}>Exported Modules</div>
          {exports.length > 0 ? (
            <ul className={styles.exportList}>
              {exports.map((exp) => <li key={exp}>{exp}</li>)}
            </ul>
          ) : (
            <p style={{ fontSize: '0.85rem', color: 'var(--ifm-color-emphasis-500)' }}>
              No exports listed.
            </p>
          )}
        </div>

        {/* Dependencies & Dependents */}
        <div className={styles.detailSection}>
          <div className={styles.sectionTitle}>Dependencies</div>
          {deps.length > 0 ? (
            deps.map((dep) => (
              <a key={dep} href={`/docs/packages/${dep.replace('/', '/')}`} className={styles.depLink}>
                {dep}
              </a>
            ))
          ) : (
            <p style={{ fontSize: '0.85rem', color: 'var(--ifm-color-emphasis-500)' }}>
              No dependencies (pure standalone package).
            </p>
          )}

          {dependents.length > 0 && (
            <>
              <div className={styles.sectionTitle} style={{ marginTop: '1rem' }}>Used By</div>
              {dependents.map((dep) => (
                <a key={dep} href={`/docs/packages/${dep.replace('/', '/')}`} className={styles.depLink}>
                  {dep}
                </a>
              ))}
            </>
          )}
        </div>
      </div>

      {/* Version Timeline */}
      {versions.length > 0 && (
        <>
          <h3>Version History</h3>
          <VersionTimeline versions={versions} packageName={packageName} />
        </>
      )}

      {versions.length === 0 && !stale && (
        <p style={{ fontSize: '0.85rem', color: 'var(--ifm-color-emphasis-500)', fontStyle: 'italic' }}>
          Loading version history from registry...
        </p>
      )}
    </div>
  );
}

function PackageLinks({ detail, staticData }) {
  const repo = detail?.index?.repository || staticData?.repository;
  const homepage = detail?.index?.homepage || staticData?.homepage;
  const licenseUrl = detail?.index?.license_url || staticData?.license_url;

  if (!repo && !homepage && !licenseUrl) return null;

  return (
    <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap', marginBottom: '1rem' }}>
      {repo && (
        <a href={repo} target="_blank" rel="noopener noreferrer" className={styles.depLink} style={{ display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}>
          Source Code &#8599;
        </a>
      )}
      {homepage && (
        <a href={homepage} target="_blank" rel="noopener noreferrer" className={styles.depLink} style={{ display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}>
          Documentation &#8599;
        </a>
      )}
      {licenseUrl && (
        <a href={licenseUrl} target="_blank" rel="noopener noreferrer" className={styles.depLink} style={{ display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}>
          License &#8599;
        </a>
      )}
    </div>
  );
}

function VersionTimeline({ versions, packageName }) {
  return (
    <div className={styles.timeline}>
      {[...versions].reverse().map((v, i) => (
        <TimelineEntry key={v.version} entry={v} isLatest={i === 0} />
      ))}
    </div>
  );
}

function TimelineEntry({ entry, isLatest }) {
  const [showProvenance, setShowProvenance] = useState(false);
  const meta = entry.metadata;
  const history = entry.history;
  const provenance = meta?.provenance;

  const publishDate = meta?.published_at
    ? new Date(meta.published_at).toLocaleDateString('en-US', {
        year: 'numeric', month: 'short', day: 'numeric',
      })
    : 'Unknown date';

  const sizeKB = meta?.tarball_size_bytes
    ? `${(meta.tarball_size_bytes / 1024).toFixed(1)} KB`
    : null;

  return (
    <div className={styles.timelineEntry}>
      <div className={`${styles.timelineDot} ${isLatest ? '' : styles.timelineDotOld}`} />

      <div className={styles.timelineHeader}>
        <span className={styles.timelineVersion}>v{entry.version}</span>
        <span className={styles.timelineDate}>{publishDate}</span>
        {meta?.published_by && (
          <span className={styles.timelineAuthor}>by {meta.published_by}</span>
        )}
      </div>

      {(history?.summary || meta?.manifest?.ai_summary) && (
        <div className={styles.timelineSummary}>
          &ldquo;{history?.summary || meta?.manifest?.ai_summary}&rdquo;
        </div>
      )}

      <div className={styles.timelineMeta}>
        {provenance?.change_class && (
          <span>Class {provenance.change_class}</span>
        )}
        {provenance?.auto_approved && <span>auto-approved</span>}
        {sizeKB && <span>{sizeKB}</span>}
      </div>

      {(provenance || meta?.content_hash) && (
        <button
          className={styles.provenanceToggle}
          onClick={() => setShowProvenance(!showProvenance)}
        >
          {showProvenance ? '▾ Hide Provenance' : '▸ View Provenance'}
        </button>
      )}

      {showProvenance && <ProvenancePanel metadata={meta} history={history} />}
    </div>
  );
}

function ProvenancePanel({ metadata, history }) {
  const p = metadata?.provenance;
  const v = metadata?.validation;

  return (
    <div className={styles.provenance}>
      <div className={styles.provenanceGrid}>
        {/* Validation */}
        {v && (
          <div>
            <div className={styles.sectionTitle}>Validation</div>
            <ul className={styles.validationList}>
              <li className={v.compiles ? styles.checkPass : styles.checkFail}>
                {v.compiles ? '✓' : '✗'} Compiles
              </li>
              <li className={v.effects_valid ? styles.checkPass : styles.checkFail}>
                {v.effects_valid ? '✓' : '✗'} Effects valid
              </li>
              <li>
                {v.contracts_verified}/{v.contracts_total} contracts verified
                {v.contracts_skipped > 0 && ` (${v.contracts_skipped} skipped)`}
              </li>
              {v.ailang_version && <li>AILANG {v.ailang_version}</li>}
            </ul>
          </div>
        )}

        {/* Hashes */}
        <div>
          <div className={styles.sectionTitle}>Hashes</div>
          {metadata?.content_hash && (
            <div className={styles.provenanceField}>
              <span className={styles.provenanceLabel}>Content</span>
              <span className={styles.hashValue}>{metadata.content_hash}</span>
            </div>
          )}
          {metadata?.interface_hash && (
            <div className={styles.provenanceField} style={{ marginTop: '0.4rem' }}>
              <span className={styles.provenanceLabel}>Interface</span>
              <span className={styles.hashValue}>{metadata.interface_hash}</span>
            </div>
          )}
          {metadata?.tarball_hash && (
            <div className={styles.provenanceField} style={{ marginTop: '0.4rem' }}>
              <span className={styles.provenanceLabel}>Tarball</span>
              <span className={styles.hashValue}>{metadata.tarball_hash}</span>
            </div>
          )}
        </div>
      </div>

      {/* Provenance Chain */}
      {p && (
        <div className={styles.provenanceGrid}>
          {p.trigger_message_id && (
            <div className={styles.provenanceField}>
              <span className={styles.provenanceLabel}>Trigger</span>
              <span className={styles.provenanceValue}>{p.trigger_message_id}</span>
            </div>
          )}
          {p.change_class && (
            <div className={styles.provenanceField}>
              <span className={styles.provenanceLabel}>Change Class</span>
              <span className={styles.provenanceValue}>
                {p.change_class} {p.auto_approved ? '(auto-approved)' : `(approved by ${p.approved_by || 'unknown'})`}
              </span>
            </div>
          )}
          {p.previous_version && (
            <div className={styles.provenanceField}>
              <span className={styles.provenanceLabel}>Previous</span>
              <span className={styles.provenanceValue}>v{p.previous_version}</span>
            </div>
          )}
          {p.agent_trace_id && (
            <div className={styles.provenanceField}>
              <span className={styles.provenanceLabel}>Trace</span>
              <span className={styles.hashValue}>{p.agent_trace_id}</span>
            </div>
          )}
        </div>
      )}

      {/* Message Trail */}
      {history?.messages && history.messages.length > 0 && (
        <div className={styles.messageTrail}>
          <div className={styles.sectionTitle}>Message Trail</div>
          {history.messages.map((msg, i) => (
            <div key={i} className={styles.messageEntry}>
              <span className={styles.messageTime}>
                {msg.timestamp ? new Date(msg.timestamp).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' }) : ''}
              </span>
              <span className={styles.messageKind}>[{msg.kind}]</span>
              <span className={styles.messageDetail}>
                {msg.title || msg.detail}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
