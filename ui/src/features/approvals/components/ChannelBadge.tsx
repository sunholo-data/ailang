/**
 * ChannelBadge.tsx - Shows the source channel of an approval
 *
 * Displays visual indicator for where the approval came from:
 * - dashboard: Direct UI interaction
 * - cli: Command line (ailang coordinator approve/reject)
 * - github: GitHub label change
 *
 * Part of M-DASHBOARD-APPROVAL-INTEGRATION multi-channel workflow.
 */

import React from 'react';
import './ChannelBadge.css';

export type ApprovalChannel = 'dashboard' | 'cli' | 'github';

interface ChannelBadgeProps {
  channel?: string;
  className?: string;
}

const channelConfig: Record<ApprovalChannel, { icon: string; label: string; title: string }> = {
  dashboard: {
    icon: '🖥️',
    label: 'Dashboard',
    title: 'Approved/Rejected via Dashboard UI',
  },
  cli: {
    icon: '⌨️',
    label: 'CLI',
    title: 'Approved/Rejected via ailang coordinator command',
  },
  github: {
    icon: '🐙',
    label: 'GitHub',
    title: 'Approved/Rejected via GitHub label',
  },
};

export const ChannelBadge: React.FC<ChannelBadgeProps> = ({
  channel,
  className = '',
}) => {
  if (!channel) {
    return null;
  }

  const config = channelConfig[channel as ApprovalChannel];
  if (!config) {
    // Unknown channel - show generic badge
    return (
      <span className={`channel-badge unknown ${className}`} title={`Source: ${channel}`}>
        <span className="channel-icon">📍</span>
        <span className="channel-label">{channel}</span>
      </span>
    );
  }

  return (
    <span className={`channel-badge ${channel} ${className}`} title={config.title}>
      <span className="channel-icon">{config.icon}</span>
      <span className="channel-label">{config.label}</span>
    </span>
  );
};

export default ChannelBadge;
