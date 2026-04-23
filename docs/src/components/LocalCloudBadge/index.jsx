import React from 'react';

const badgeStyle = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: '3px',
  fontSize: '0.7rem',
  fontWeight: 600,
  padding: '1px 6px',
  borderRadius: '10px',
  whiteSpace: 'nowrap',
};

const cloudStyle = {
  ...badgeStyle,
  background: 'rgba(59,130,246,0.12)',
  color: '#2563eb',
  border: '1px solid rgba(59,130,246,0.25)',
};

const localStyle = {
  ...badgeStyle,
  background: 'rgba(234,88,12,0.1)',
  color: '#c2410c',
  border: '1px solid rgba(234,88,12,0.25)',
};

/**
 * Renders a small cloud/local badge based on provider_type and timeout_scale.
 * Used in harness and model tables to distinguish Ollama (local) from cloud models.
 */
export default function LocalCloudBadge({ providerType, timeoutScale }) {
  if (providerType === 'local') {
    const scale = timeoutScale && timeoutScale > 1 ? `×${Math.round(timeoutScale)}` : '';
    return (
      <span style={localStyle} title={scale ? `Timeout scaled ${scale} vs cloud baseline` : 'Local model'}>
        🖥 Local{scale ? ` (${scale})` : ''}
      </span>
    );
  }
  return (
    <span style={cloudStyle} title="Cloud API model">
      ☁ Cloud
    </span>
  );
}
