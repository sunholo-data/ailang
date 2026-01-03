import styles from './WorkspaceBadge.module.css';

interface WorkspaceBadgeProps {
  workspace: string;
  size?: 'small' | 'medium';
}

// Map workspace IDs to display names
const workspaceConfig: Record<string, { name: string; colorClass: string }> = {
  'ailang': { name: 'AILANG', colorClass: 'ailang' },
  'claude-code': { name: 'Claude', colorClass: 'claude' },
  'stapledons_voyage': { name: 'Stapledon', colorClass: 'stapledon' },
  'coordinator': { name: 'Coordinator', colorClass: 'coordinator' },
  'unknown': { name: 'Unknown', colorClass: 'unknown' },
};

export function WorkspaceBadge({ workspace, size = 'small' }: WorkspaceBadgeProps) {
  if (!workspace) {
    return null;
  }

  const normalizedWorkspace = workspace.toLowerCase();
  const config = workspaceConfig[normalizedWorkspace] || {
    name: workspace.length > 12 ? workspace.slice(0, 12) + '...' : workspace,
    colorClass: 'default'
  };

  return (
    <span
      className={`${styles.badge} ${styles[config.colorClass]} ${styles[size]}`}
      title={`Workspace: ${workspace}`}
    >
      {config.name}
    </span>
  );
}
