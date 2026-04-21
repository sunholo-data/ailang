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
