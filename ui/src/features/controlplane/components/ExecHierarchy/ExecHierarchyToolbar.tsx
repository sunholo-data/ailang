/**
 * ExecHierarchyToolbar - Header controls for the exec hierarchy view
 * Contains view mode toggles, filters, and expand/collapse controls
 *
 * Extracted from ExecHierarchy.tsx (PR 3 - M-DASHBOARD-SIMPLIFICATION)
 */

import React from 'react';
import type { ViewMode, CoordinatorViewMode } from './types';
import styles from './ExecHierarchy.module.css';

// ============================================================================
// Types
// ============================================================================

export interface ExecHierarchyToolbarProps {
  // Counts
  isEmpty: boolean;
  totalNodeCount: number;
  topLevelNodeCount: number;

  // View state
  viewMode: ViewMode;
  coordViewMode: CoordinatorViewMode;
  reverseOrder: boolean;
  isExpanded: boolean;

  // View callbacks
  onViewModeChange: (mode: ViewMode) => void;
  onCoordViewModeChange: (mode: CoordinatorViewMode) => void;
  onToggleReverseOrder: () => void;
  onToggleExpand: () => void;

  // Filter state
  uniqueSpanTypes: string[];
  hiddenSpanTypes: Set<string>;
  effectiveHiddenCount: number;
  showSpanTypeFilter: boolean;

  // Filter callbacks
  onToggleSpanType: (spanType: string) => void;
  onToggleSpanTypeFilter: () => void;
  onShowAllSpanTypes: () => void;
  onHideAllSpanTypes: () => void;

  // Expand/collapse callbacks
  onExpandAll: () => void;
  onCollapseAll: () => void;
  onExpandOneLevel: () => void;
  onCollapseOneLevel: () => void;
}

// ============================================================================
// Component
// ============================================================================

export const ExecHierarchyToolbar: React.FC<ExecHierarchyToolbarProps> = ({
  isEmpty,
  totalNodeCount,
  topLevelNodeCount,
  viewMode,
  coordViewMode,
  reverseOrder,
  isExpanded,
  onViewModeChange,
  onCoordViewModeChange,
  onToggleReverseOrder,
  onToggleExpand,
  uniqueSpanTypes,
  hiddenSpanTypes,
  effectiveHiddenCount,
  showSpanTypeFilter,
  onToggleSpanType,
  onToggleSpanTypeFilter,
  onShowAllSpanTypes,
  onHideAllSpanTypes,
  onExpandAll,
  onCollapseAll,
  onExpandOneLevel,
  onCollapseOneLevel,
}) => {
  return (
    <div className={styles.header}>
      <div className={styles.headerLeft}>
        <div className={styles.headerTitle}>
          <span className={styles.headerIcon}>◎</span>
          <span className={styles.headerLabel}>Execution Spans</span>
        </div>

        {/* Span Count Readout */}
        {!isEmpty && totalNodeCount > 0 && (
          <div
            className={styles.telemetryItem}
            title={`Total execution spans in view (${topLevelNodeCount} top-level, ${totalNodeCount} total including nested)`}
          >
            <span className={styles.telemetryValue}>{totalNodeCount}</span>
            <span className={styles.telemetryLabel}>spans</span>
          </div>
        )}
      </div>

      <div className={styles.headerControls}>
        {/* Span Type Filter */}
        {uniqueSpanTypes.length > 0 && (
          <div className={styles.filterDropdown}>
            <button
              className={`${styles.filterBtn} ${effectiveHiddenCount > 0 ? styles.filterBtnActive : ''}`}
              onClick={onToggleSpanTypeFilter}
              title="Filter which span types to display"
            >
              <span className={styles.filterIcon}>⚙</span>
              <span className={styles.filterText}>
                {effectiveHiddenCount > 0
                  ? `${uniqueSpanTypes.length - effectiveHiddenCount}/${uniqueSpanTypes.length}`
                  : 'Types'}
              </span>
              <span className={styles.filterChevron}>{showSpanTypeFilter ? '▴' : '▾'}</span>
            </button>

            {showSpanTypeFilter && (
              <div className={styles.filterMenu}>
                <div className={styles.filterMenuHeader}>
                  <span className={styles.filterMenuTitle}>Span Types</span>
                  <div className={styles.filterMenuActions}>
                    <button onClick={onShowAllSpanTypes} className={styles.filterMenuAction}>
                      All
                    </button>
                    <button onClick={onHideAllSpanTypes} className={styles.filterMenuAction}>
                      None
                    </button>
                  </div>
                </div>
                <div className={styles.filterMenuList}>
                  {uniqueSpanTypes.map(spanType => (
                    <label key={spanType} className={styles.filterOption}>
                      <input
                        type="checkbox"
                        checked={!hiddenSpanTypes.has(spanType)}
                        onChange={() => onToggleSpanType(spanType)}
                      />
                      <span className={styles.filterOptionCheck}>
                        {!hiddenSpanTypes.has(spanType) ? '✓' : ''}
                      </span>
                      <span className={styles.filterOptionLabel}>{spanType}</span>
                    </label>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Coordinator View Mode Toggle */}
        {!isEmpty && (
          <div className={styles.viewToggle}>
            <button
              className={`${styles.viewToggleBtn} ${coordViewMode === 'nested' ? styles.viewToggleBtnActive : ''}`}
              onClick={() => onCoordViewModeChange('nested')}
              title="Nested View (child tasks under parents)"
            >
              ⊏
            </button>
            <button
              className={`${styles.viewToggleBtn} ${coordViewMode === 'breakout' ? styles.viewToggleBtnActive : ''}`}
              onClick={() => onCoordViewModeChange('breakout')}
              title="Breakout View (each task as separate root)"
            >
              ⊔
            </button>
          </div>
        )}

        {/* Collapse/Expand Controls */}
        {!isEmpty && (
          <div className={styles.collapseControls}>
            <button
              className={styles.collapseBtn}
              onClick={onCollapseAll}
              title="Collapse All"
            >
              ⊟
            </button>
            <button
              className={styles.collapseBtn}
              onClick={onCollapseOneLevel}
              title="Collapse One Level"
            >
              −
            </button>
            <button
              className={styles.collapseBtn}
              onClick={onExpandOneLevel}
              title="Expand One Level"
            >
              +
            </button>
            <button
              className={styles.collapseBtn}
              onClick={onExpandAll}
              title="Expand All"
            >
              ⊞
            </button>
          </div>
        )}

        {/* View Toggle */}
        <div className={styles.viewToggle}>
          <button
            className={`${styles.viewToggleBtn} ${viewMode === 'tree' ? styles.viewToggleBtnActive : ''}`}
            onClick={() => onViewModeChange('tree')}
            title="Tree View"
          >
            ≡
          </button>
          <button
            className={`${styles.viewToggleBtn} ${viewMode === 'graph' ? styles.viewToggleBtnActive : ''}`}
            onClick={() => onViewModeChange('graph')}
            title="Graph View"
          >
            ⬡
          </button>
          <button
            className={`${styles.viewToggleBtn} ${viewMode === 'timeline' ? styles.viewToggleBtnActive : ''}`}
            onClick={() => onViewModeChange('timeline')}
            title="Timeline View"
          >
            ▥
          </button>
          <button
            className={`${styles.viewToggleBtn} ${viewMode === 'chat' ? styles.viewToggleBtnActive : ''}`}
            onClick={() => onViewModeChange('chat')}
            title="Chat History"
          >
            💬
          </button>
          <button
            className={`${styles.viewToggleBtn} ${viewMode === 'evolution' ? styles.viewToggleBtnActive : ''}`}
            onClick={() => onViewModeChange('evolution')}
            title="Evolution Tree"
          >
            🌳
          </button>
        </div>

        {/* Reverse Order Toggle */}
        <button
          className={`${styles.viewToggleBtn} ${reverseOrder ? styles.viewToggleBtnActive : ''}`}
          onClick={onToggleReverseOrder}
          title={reverseOrder ? "Show oldest first" : "Show newest first"}
        >
          {reverseOrder ? "↓" : "↑"}
        </button>

        {/* Expand Button */}
        <button
          className={styles.expandBtn}
          onClick={onToggleExpand}
          title={isExpanded ? 'Collapse' : 'Expand'}
        >
          {isExpanded ? '⤢' : '⤡'}
        </button>
      </div>
    </div>
  );
};

export default ExecHierarchyToolbar;
