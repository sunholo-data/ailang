import { useState, useEffect } from 'react';
import styles from './FilterBar.module.css';

interface FilterBarProps {
  providers: string[];
  workspaces: string[];
  selectedProvider: string;
  selectedWorkspace: string;
  onProviderChange: (provider: string) => void;
  onWorkspaceChange: (workspace: string) => void;
  onClearFilters: () => void;
}

export function FilterBar({
  providers,
  workspaces,
  selectedProvider,
  selectedWorkspace,
  onProviderChange,
  onWorkspaceChange,
  onClearFilters,
}: FilterBarProps) {
  const hasFilters = selectedProvider !== '' || selectedWorkspace !== '';

  return (
    <div className={styles.filterBar}>
      <div className={styles.filters}>
        {/* Provider Filter */}
        <div className={styles.filterGroup}>
          <label className={styles.filterLabel}>Provider</label>
          <select
            className={styles.filterSelect}
            value={selectedProvider}
            onChange={(e) => onProviderChange(e.target.value)}
          >
            <option value="">All Providers</option>
            {providers.map((provider) => (
              <option key={provider} value={provider}>
                {formatProviderName(provider)}
              </option>
            ))}
          </select>
        </div>

        {/* Workspace Filter */}
        <div className={styles.filterGroup}>
          <label className={styles.filterLabel}>Workspace</label>
          <select
            className={styles.filterSelect}
            value={selectedWorkspace}
            onChange={(e) => onWorkspaceChange(e.target.value)}
          >
            <option value="">All Workspaces</option>
            {workspaces.map((workspace) => (
              <option key={workspace} value={workspace}>
                {formatWorkspaceName(workspace)}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Clear Filters Button */}
      {hasFilters && (
        <button className={styles.clearButton} onClick={onClearFilters}>
          Clear Filters
        </button>
      )}

      {/* Active Filters Summary */}
      {hasFilters && (
        <div className={styles.activeFilters}>
          {selectedProvider && (
            <span className={styles.activeFilter}>
              Provider: {formatProviderName(selectedProvider)}
              <button
                className={styles.removeFilter}
                onClick={() => onProviderChange('')}
                title="Remove filter"
              >
                x
              </button>
            </span>
          )}
          {selectedWorkspace && (
            <span className={styles.activeFilter}>
              Workspace: {formatWorkspaceName(selectedWorkspace)}
              <button
                className={styles.removeFilter}
                onClick={() => onWorkspaceChange('')}
                title="Remove filter"
              >
                x
              </button>
            </span>
          )}
        </div>
      )}
    </div>
  );
}

// Helper functions to format display names
function formatProviderName(provider: string): string {
  const names: Record<string, string> = {
    'claude': 'Claude',
    'claude-code': 'Claude Code',
    'gemini': 'Gemini',
    'gemini-cli': 'Gemini CLI',
    'openai': 'OpenAI',
    'gpt': 'GPT',
    'ollama': 'Ollama',
  };
  return names[provider.toLowerCase()] || provider;
}

function formatWorkspaceName(workspace: string): string {
  const names: Record<string, string> = {
    'ailang': 'AILANG',
    'stapledons_voyage': 'Stapledon',
    'coordinator': 'Coordinator',
  };
  return names[workspace.toLowerCase()] || workspace;
}
