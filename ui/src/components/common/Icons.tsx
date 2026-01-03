/**
 * Consolidated SVG icons for the UI
 * All icons are React elements ready to use
 */
import React from 'react';

// Standard icon sizes
const SIZE_SM = 14;
const SIZE_MD = 16;
const SIZE_LG = 18;

// Common SVG props
const svgProps = (size: number) => ({
  width: size,
  height: size,
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 2,
  strokeLinecap: 'round' as const,
  strokeLinejoin: 'round' as const,
});

export const Icons = {
  // Message types
  send: (
    <svg {...svgProps(SIZE_LG)}>
      <line x1="22" y1="2" x2="11" y2="13" />
      <polygon points="22 2 15 22 11 13 2 9 22 2" />
    </svg>
  ),
  directive: (
    <svg {...svgProps(SIZE_SM)}>
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <polyline points="14 2 14 8 20 8" />
      <line x1="16" y1="13" x2="8" y2="13" />
      <line x1="16" y1="17" x2="8" y2="17" />
    </svg>
  ),
  question: (
    <svg {...svgProps(SIZE_SM)}>
      <circle cx="12" cy="12" r="10" />
      <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
      <line x1="12" y1="17" x2="12.01" y2="17" />
    </svg>
  ),
  status: (
    <svg {...svgProps(SIZE_SM)}>
      <path d="M22 12h-4l-3 9L9 3l-3 9H2" />
    </svg>
  ),
  result: (
    <svg {...svgProps(SIZE_SM)}>
      <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
      <polyline points="22 4 12 14.01 9 11.01" />
    </svg>
  ),
  message: (
    <svg {...svgProps(SIZE_MD)}>
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
    </svg>
  ),

  // User/Agent
  user: (
    <svg {...svgProps(SIZE_MD)}>
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
      <circle cx="12" cy="7" r="4" />
    </svg>
  ),
  bot: (
    <svg {...svgProps(SIZE_MD)}>
      <rect x="3" y="11" width="18" height="10" rx="2" />
      <circle cx="12" cy="5" r="2" />
      <path d="M12 7v4" />
    </svg>
  ),

  // Actions
  check: (
    <svg {...svgProps(SIZE_MD)}>
      <polyline points="20 6 9 17 4 12" />
    </svg>
  ),
  x: (
    <svg {...svgProps(SIZE_MD)}>
      <line x1="18" y1="6" x2="6" y2="18" />
      <line x1="6" y1="6" x2="18" y2="18" />
    </svg>
  ),
  stop: (
    <svg {...svgProps(SIZE_SM)}>
      <rect x="3" y="3" width="18" height="18" rx="2" />
    </svg>
  ),

  // Files/Folders
  file: (
    <svg {...svgProps(SIZE_SM)}>
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <polyline points="14 2 14 8 20 8" />
    </svg>
  ),
  folder: (
    <svg {...svgProps(SIZE_SM)}>
      <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
    </svg>
  ),
  tokens: (
    <svg {...svgProps(SIZE_MD)}>
      <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
      <polyline points="14 2 14 8 20 8" />
      <line x1="16" y1="13" x2="8" y2="13" />
      <line x1="16" y1="17" x2="8" y2="17" />
      <line x1="10" y1="9" x2="8" y2="9" />
    </svg>
  ),

  // Security
  lock: (
    <svg {...svgProps(SIZE_SM)}>
      <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
      <path d="M7 11V7a5 5 0 0 1 10 0v4" />
    </svg>
  ),

  // Metrics/Stats
  cpu: (
    <svg {...svgProps(SIZE_MD)}>
      <rect x="4" y="4" width="16" height="16" rx="2" />
      <rect x="9" y="9" width="6" height="6" />
      <line x1="9" y1="1" x2="9" y2="4" />
      <line x1="15" y1="1" x2="15" y2="4" />
      <line x1="9" y1="20" x2="9" y2="23" />
      <line x1="15" y1="20" x2="15" y2="23" />
      <line x1="20" y1="9" x2="23" y2="9" />
      <line x1="20" y1="14" x2="23" y2="14" />
      <line x1="1" y1="9" x2="4" y2="9" />
      <line x1="1" y1="14" x2="4" y2="14" />
    </svg>
  ),
  memory: (
    <svg {...svgProps(SIZE_MD)}>
      <rect x="2" y="6" width="20" height="12" rx="2" />
      <line x1="6" y1="10" x2="6" y2="14" />
      <line x1="10" y1="10" x2="10" y2="14" />
      <line x1="14" y1="10" x2="14" y2="14" />
      <line x1="18" y1="10" x2="18" y2="14" />
    </svg>
  ),
  clock: (
    <svg {...svgProps(SIZE_MD)}>
      <circle cx="12" cy="12" r="10" />
      <polyline points="12 6 12 12 16 14" />
    </svg>
  ),
  dollar: (
    <svg {...svgProps(SIZE_MD)}>
      <line x1="12" y1="1" x2="12" y2="23" />
      <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
    </svg>
  ),
  activity: (
    <svg {...svgProps(SIZE_MD)}>
      <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
    </svg>
  ),

  // Navigation
  chevronDown: (
    <svg {...svgProps(SIZE_MD)}>
      <polyline points="6 9 12 15 18 9" />
    </svg>
  ),
  chevronUp: (
    <svg {...svgProps(SIZE_MD)}>
      <polyline points="18 15 12 9 6 15" />
    </svg>
  ),
  chevronRight: (
    <svg {...svgProps(SIZE_MD)}>
      <polyline points="9 18 15 12 9 6" />
    </svg>
  ),
  chevronLeft: (
    <svg {...svgProps(SIZE_MD)}>
      <polyline points="15 18 9 12 15 6" />
    </svg>
  ),

  // Status
  warning: (
    <svg {...svgProps(SIZE_MD)}>
      <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
      <line x1="12" y1="9" x2="12" y2="13" />
      <line x1="12" y1="17" x2="12.01" y2="17" />
    </svg>
  ),
  spinner: (
    <svg className="spinner-icon" {...svgProps(SIZE_MD)}>
      <path d="M21 12a9 9 0 1 1-6.219-8.56" />
    </svg>
  ),

  // Decorative
  sparkles: (
    <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z" />
      <path d="M5 19l.5 1.5L7 21l-1.5.5L5 23l-.5-1.5L3 21l1.5-.5L5 19z" />
      <path d="M19 13l.5 1.5L21 15l-1.5.5L19 17l-.5-1.5L17 15l1.5-.5L19 13z" />
    </svg>
  ),
};

/**
 * Helper to get kind icon for messages
 */
export const getKindIcon = (kind: string): React.ReactNode => {
  switch (kind) {
    case 'directive': return Icons.directive;
    case 'question': return Icons.question;
    case 'status': return Icons.status;
    case 'result': return Icons.result;
    case 'approval_request': return Icons.lock;
    default: return Icons.directive;
  }
};

/**
 * Helper to get status color
 */
export const getStatusColor = (status: string): string => {
  switch (status) {
    case 'running': return 'var(--color-success)';
    case 'completed': return 'var(--color-primary)';
    case 'failed': return 'var(--color-danger)';
    case 'orphan': return 'var(--color-warning)';
    default: return 'var(--text-tertiary)';
  }
};

export default Icons;
