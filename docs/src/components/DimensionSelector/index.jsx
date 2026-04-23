import React from 'react';

const containerStyle = {
  display: 'flex',
  flexWrap: 'wrap',
  gap: '12px',
  alignItems: 'center',
  marginBottom: '16px',
  padding: '10px 14px',
  background: 'var(--ifm-color-emphasis-100)',
  borderRadius: '8px',
  fontSize: '0.85rem',
};

const groupStyle = {
  display: 'flex',
  alignItems: 'center',
  gap: '6px',
  flexWrap: 'wrap',
};

const labelStyle = {
  fontWeight: 600,
  color: 'var(--ifm-color-emphasis-700)',
  marginRight: '2px',
};

function Chip({ label, active, onClick }) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: '3px 10px',
        borderRadius: '14px',
        border: active
          ? '2px solid var(--ifm-color-primary)'
          : '1px solid var(--ifm-color-emphasis-300)',
        background: active ? 'var(--ifm-color-primary)' : 'transparent',
        color: active ? '#fff' : 'var(--ifm-color-emphasis-700)',
        cursor: 'pointer',
        fontWeight: active ? 600 : 400,
        fontSize: '0.8rem',
        transition: 'all 0.15s',
      }}
    >
      {label}
    </button>
  );
}

/**
 * Shared filter bar for benchmark pages.
 *
 * Props:
 *   tiers       - string[] of available tier keys
 *   selectedTier - string | null
 *   onTierChange - (tier: string | null) => void
 *
 *   languages       - string[] of available language keys
 *   selectedLangs   - Set<string> | null (null = all)
 *   onLangsChange   - (Set<string>) => void
 *
 *   harnesses       - string[] of available harness keys (optional)
 *   selectedHarnesses - Set<string> | null
 *   onHarnessesChange - (Set<string>) => void
 */
export default function DimensionSelector({
  tiers = [],
  selectedTier,
  onTierChange,
  languages = [],
  selectedLangs,
  onLangsChange,
  harnesses = [],
  selectedHarnesses,
  onHarnessesChange,
}) {
  function toggleSet(set, key, onChange) {
    const next = new Set(set || []);
    if (next.has(key)) {
      next.delete(key);
    } else {
      next.add(key);
    }
    onChange(next.size === 0 ? null : next);
  }

  const TIER_LABELS = { smoke: 'Smoke', core: 'Core', stretch: 'Stretch', vision: 'Vision' };

  return (
    <div style={containerStyle}>
      {tiers.length > 0 && (
        <div style={groupStyle}>
          <span style={labelStyle}>Tier:</span>
          <Chip label="All" active={!selectedTier} onClick={() => onTierChange(null)} />
          {tiers.map((t) => (
            <Chip
              key={t}
              label={TIER_LABELS[t] || t}
              active={selectedTier === t}
              onClick={() => onTierChange(selectedTier === t ? null : t)}
            />
          ))}
        </div>
      )}

      {languages.length > 0 && (
        <div style={groupStyle}>
          <span style={labelStyle}>Language:</span>
          <Chip
            label="All"
            active={!selectedLangs}
            onClick={() => onLangsChange(null)}
          />
          {languages.map((l) => (
            <Chip
              key={l}
              label={l}
              active={selectedLangs ? selectedLangs.has(l) : false}
              onClick={() => toggleSet(selectedLangs, l, onLangsChange)}
            />
          ))}
        </div>
      )}

      {harnesses.length > 0 && (
        <div style={groupStyle}>
          <span style={labelStyle}>Harness:</span>
          <Chip
            label="All"
            active={!selectedHarnesses}
            onClick={() => onHarnessesChange(null)}
          />
          {harnesses.map((h) => (
            <Chip
              key={h}
              label={h}
              active={selectedHarnesses ? selectedHarnesses.has(h) : false}
              onClick={() => toggleSet(selectedHarnesses, h, onHarnessesChange)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
