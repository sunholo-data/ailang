import { useMemo } from 'react';

// Default when events[] is absent — preserves the pre-M-DASH-V2 annotation
// that the three trend charts hardcoded. Once v0.13.0 baselines have been
// regenerated with events.yml in the payload this fallback is a no-op.
const LEGACY_ANNOTATIONS = [
  { version: 'v0.9.1.1', label: '+5 contract benchmarks', kind: 'benchmark_add', color: '#888' },
];

// useEvents returns the suite-change annotations that should render on a
// given chart as dashed ReferenceLines.
//
// Filtering rules:
//   - `kinds`: if set, only keep events whose `kind` is in the list. E.g.
//     ModelDeltaTrend hides taxonomy events because they don't shift the
//     per-model delta — only benchmark add/remove events do.
//   - `selectedTier`: if non-null, hide events tagged `affects_tiers` that
//     don't include the selected tier. Lets the "stretch only" +2 vision
//     benchmark event disappear when the user selects Core.
//
// Events returned in input (chronological) order.
export function useEvents(events, { kinds, selectedTier } = {}) {
  return useMemo(() => {
    const source = Array.isArray(events) && events.length > 0 ? events : LEGACY_ANNOTATIONS;
    return source.filter((e) => {
      if (kinds && !kinds.includes(e.kind)) return false;
      if (selectedTier && Array.isArray(e.affects_tiers) && e.affects_tiers.length > 0) {
        return e.affects_tiers.includes(selectedTier);
      }
      return true;
    });
  }, [events, kinds, selectedTier]);
}

// annotationColor picks a reasonable color when the event doesn't carry one.
// Different kinds get different default hues so a taxonomy change is visually
// distinct from a benchmark add even without an explicit `color`.
export function annotationColor(event) {
  if (event.color) return event.color;
  switch (event.kind) {
    case 'taxonomy':
      return '#E67E22';
    case 'prompt':
      return '#3498DB';
    case 'benchmark_remove':
      return '#E74C3C';
    case 'benchmark_add':
    default:
      return '#888';
  }
}

// Group annotations by (formatted) version so ReferenceLine renders once per
// version tick even when multiple events share a release (v0.13.0 currently
// has two). Saves us from stacking labels that overlap and become unreadable.
export function groupByVersion(annotations, formatVersion) {
  const map = new Map();
  annotations.forEach((ann) => {
    const v = formatVersion(ann.version);
    if (!map.has(v)) map.set(v, []);
    map.get(v).push(ann);
  });
  return map;
}

// snapEventsToVersions re-maps a version→events map onto the versions that
// actually appear on the chart's x-axis. An annotation whose exact version has
// no baseline data-point — e.g. the frontier benchmarks added at v0.29.0 when
// the eval suite jumped v0.25.0 → v0.29.2 — would otherwise be silently dropped
// by the ReferenceLine's `exists` check. Instead we snap it to the nearest
// baseline so the annotation still renders. `chartVersions` is the ordered list
// of x-axis version labels (e.g. chartData.map(d => d.version)).
export function snapEventsToVersions(eventsByVersion, chartVersions) {
  const present = new Set(chartVersions);
  const key = (v) => (v || '').replace(/^v/, '').split(/[.-]/).map((n) => parseInt(n, 10) || 0);
  const cmp = (a, b) => {
    const A = key(a), B = key(b);
    for (let i = 0; i < Math.max(A.length, B.length); i++) {
      if ((A[i] || 0) !== (B[i] || 0)) return (A[i] || 0) - (B[i] || 0);
    }
    return 0;
  };
  const nearest = (v) => {
    if (present.has(v)) return v;
    let best = null;
    let bestD = Infinity;
    chartVersions.forEach((cv) => {
      const d = Math.abs(cmp(v, cv));
      if (d < bestD) { bestD = d; best = cv; }
    });
    return best;
  };
  const out = new Map();
  eventsByVersion.forEach((evs, ver) => {
    const target = nearest(ver);
    if (!target) return;
    const tagged = target === ver ? evs : evs.map((e) => ({ ...e, snappedFrom: ver }));
    out.set(target, [...(out.get(target) || []), ...tagged]);
  });
  return out;
}
