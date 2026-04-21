import React from 'react';
import styles from './styles.module.css';

// Canonical order from internal/eval_harness/spec.go ValidTagTaxonomy —
// keep in sync or the filter drops tags silently.
const TAG_ORDER = [
  'adt_pattern_match',
  'recursion',
  'effects_io',
  'contracts',
  'data_transform',
  'records',
  'functional',
  'type_safety',
  'string_algo',
  'state_machine',
  'algorithmic',
  'error_handling',
];

const TAG_LABELS = {
  adt_pattern_match: 'ADT + Match',
  recursion: 'Recursion',
  effects_io: 'Effects/IO',
  contracts: 'Contracts',
  data_transform: 'Data Transform',
  records: 'Records',
  functional: 'Functional',
  type_safety: 'Type Safety',
  string_algo: 'Strings',
  state_machine: 'State Machine',
  algorithmic: 'Algorithms',
  error_handling: 'Error Handling',
};

// TagFilter mirrors TierToggle but operates on the 12-tag taxonomy.
// Disabled (greyed-out) when a tier is selected — per-tier × per-tag
// aggregates would need tiers[t].tags[tag] backend-side.
export default function TagFilter({ tags, selected, onSelect, disabled }) {
  if (!tags || Object.keys(tags).length === 0) return null;

  return (
    <div>
      <div className={`${styles.tierToggle} ${disabled ? styles.tierToggleDisabled : ''}`}>
        <span className={styles.tierToggleLabel}>Tag:</span>
        <button
          type="button"
          className={`${styles.tierButton} ${selected === null ? styles.tierButtonActive : ''}`}
          onClick={() => onSelect(null)}
          disabled={disabled}
        >
          All
        </button>
        {TAG_ORDER.filter((t) => tags[t]).map((t) => (
          <button
            key={t}
            type="button"
            className={`${styles.tierButton} ${selected === t ? styles.tierButtonActive : ''}`}
            onClick={() => onSelect(t)}
            disabled={disabled}
            title={
              tags[t].delta !== undefined
                ? `AILANG vs Python delta: ${(tags[t].delta * 100).toFixed(1)}%`
                : ''
            }
          >
            {TAG_LABELS[t] || t}
            <span className={styles.tierButtonCount}>({tags[t].benchmark_count || 0})</span>
          </button>
        ))}
      </div>
      {disabled ? (
        <div className={styles.tierHeadline}>
          Tag filter disabled while a tier is selected — pick <code>All</code> tier above to narrow by tag.
        </div>
      ) : selected ? (
        <div className={styles.tierHeadline}>
          Showing benchmarks tagged <code>{selected}</code>
          {tags[selected]?.delta !== undefined && (
            <> · AILANG vs Python gap: {(tags[selected].delta * 100).toFixed(1)}%</>
          )}
        </div>
      ) : (
        <div className={styles.tierHeadline}>Filter charts by feature area tag</div>
      )}
    </div>
  );
}

export { TAG_ORDER, TAG_LABELS };
