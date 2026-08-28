import React from 'react';

// DataProvenance — "which release and when, and is this live?" on every
// benchmark surface (M-EVAL-ROLLING-ELO M4).
//
// Before this, 5 of the benchmark pages (Model Leaderboard, ELO, Explorer,
// Value, Gallery) rendered numbers with NO version or date, and 12 of 13
// component fetch sites degraded silently to the in-build static copy — which
// can be several releases behind — with no indicator. A reader could not tell
// a fresh number from a stale one.
//
// `source` is optional: pass the value from benchmarkFetchWithSource where the
// caller has it. Omitted, the badge simply isn't shown (no fabricated claim of
// freshness).
export default function DataProvenance({ version, timestamp, source, note }) {
  if (!version && !timestamp) return null;

  const when = timestamp ? new Date(timestamp) : null;
  const whenValid = when && when.getFullYear() > 2000;
  const isFallback = source === 'fallback';

  return (
    <div
      style={{
        display: 'flex',
        flexWrap: 'wrap',
        alignItems: 'center',
        gap: '0.5rem',
        fontSize: '0.85rem',
        opacity: 0.85,
        margin: '0.25rem 0 1rem',
      }}
    >
      {version && (
        <span>
          Data: <strong>{version}</strong>
        </span>
      )}
      {whenValid && <span>· measured {when.toISOString().slice(0, 10)}</span>}
      {isFallback && (
        <span
          title="Served from the in-build copy because the live data source was unreachable — these numbers may be several releases behind."
          style={{
            padding: '0.1rem 0.45rem',
            borderRadius: '0.75rem',
            border: '1px solid currentColor',
            fontWeight: 600,
          }}
        >
          ⚠ stale (fallback copy)
        </span>
      )}
      {note && <span>· {note}</span>}
    </div>
  );
}
