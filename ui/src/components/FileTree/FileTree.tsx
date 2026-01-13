/**
 * FileTree.tsx - Hierarchical file browser for diff review
 *
 * Displays changed files in a tree structure with expand/collapse,
 * line count stats, and click-to-select functionality.
 */
/* eslint-disable @typescript-eslint/no-use-before-define */

import React, { useMemo, useState, useCallback } from 'react';
import { FileChange, TreeNode, buildFileTree } from '../DiffViewer/diffParser';
import styles from './FileTree.module.css';

interface FileTreeProps {
  files: FileChange[];
  selectedFile?: string;
  onSelectFile: (path: string) => void;
  compact?: boolean;
}

export const FileTree: React.FC<FileTreeProps> = ({
  files,
  selectedFile,
  onSelectFile,
  compact = false,
}) => {
  const tree = useMemo(() => buildFileTree(files), [files]);
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(() => {
    // Start with all directories expanded
    const paths = new Set<string>();
    const collectPaths = (node: TreeNode) => {
      if (node.type === 'directory') {
        paths.add(node.path);
        node.children.forEach(collectPaths);
      }
    };
    collectPaths(tree);
    return paths;
  });

  const toggleExpand = useCallback((path: string) => {
    setExpandedPaths((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  }, []);

  const collapseAll = useCallback(() => {
    setExpandedPaths(new Set());
  }, []);

  const expandAll = useCallback(() => {
    const paths = new Set<string>();
    const collectPaths = (node: TreeNode) => {
      if (node.type === 'directory') {
        paths.add(node.path);
        node.children.forEach(collectPaths);
      }
    };
    collectPaths(tree);
    setExpandedPaths(paths);
  }, [tree]);

  if (files.length === 0) {
    return (
      <div className={styles.empty}>
        <span>No files changed</span>
      </div>
    );
  }

  return (
    <div className={`${styles.fileTree} ${compact ? styles.compact : ''}`}>
      {/* Header */}
      <div className={styles.header}>
        <span className={styles.title}>
          {files.length} file{files.length !== 1 ? 's' : ''}
        </span>
        <div className={styles.controls}>
          <button
            className={styles.controlButton}
            onClick={expandAll}
            title="Expand all"
          >
            +
          </button>
          <button
            className={styles.controlButton}
            onClick={collapseAll}
            title="Collapse all"
          >
            -
          </button>
        </div>
      </div>

      {/* Summary */}
      <div className={styles.summary}>
        <span className={styles.additions}>+{tree.additions}</span>
        <span className={styles.deletions}>-{tree.deletions}</span>
      </div>

      {/* Tree */}
      <div className={styles.treeContent}>
        {tree.children.map((node) => (
          <TreeNodeView
            key={node.path}
            node={node}
            depth={0}
            expandedPaths={expandedPaths}
            selectedFile={selectedFile}
            onSelectFile={onSelectFile}
            onToggleExpand={toggleExpand}
          />
        ))}
      </div>
    </div>
  );
};

interface TreeNodeViewProps {
  node: TreeNode;
  depth: number;
  expandedPaths: Set<string>;
  selectedFile?: string;
  onSelectFile: (path: string) => void;
  onToggleExpand: (path: string) => void;
}

const TreeNodeView: React.FC<TreeNodeViewProps> = ({
  node,
  depth,
  expandedPaths,
  selectedFile,
  onSelectFile,
  onToggleExpand,
}) => {
  const isExpanded = expandedPaths.has(node.path);
  const isSelected = selectedFile === node.path;
  const isDirectory = node.type === 'directory';

  const handleClick = () => {
    if (isDirectory) {
      onToggleExpand(node.path);
    } else {
      onSelectFile(node.path);
    }
  };

  const icon = getFileIcon(node);
  const statusClass = node.status ? styles[`status${capitalize(node.status)}`] : '';

  return (
    <>
      <div
        className={`${styles.treeNode} ${isSelected ? styles.selected : ''} ${statusClass}`}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
        onClick={handleClick}
      >
        {/* Expand/collapse indicator for directories */}
        {isDirectory ? (
          <span className={styles.expandIcon}>
            {isExpanded ? '▾' : '▸'}
          </span>
        ) : (
          <span className={styles.expandIcon}></span>
        )}

        {/* Icon */}
        <span className={styles.icon}>{icon}</span>

        {/* Name */}
        <span className={styles.name}>{node.name}</span>

        {/* Stats */}
        <span className={styles.stats}>
          {node.additions > 0 && (
            <span className={styles.additions}>+{node.additions}</span>
          )}
          {node.deletions > 0 && (
            <span className={styles.deletions}>-{node.deletions}</span>
          )}
        </span>
      </div>

      {/* Children (if expanded) */}
      {isDirectory && isExpanded && (
        <>
          {node.children.map((child) => (
            <TreeNodeView
              key={child.path}
              node={child}
              depth={depth + 1}
              expandedPaths={expandedPaths}
              selectedFile={selectedFile}
              onSelectFile={onSelectFile}
              onToggleExpand={onToggleExpand}
            />
          ))}
        </>
      )}
    </>
  );
};

/**
 * Get an icon based on file type/extension
 */
function getFileIcon(node: TreeNode): string {
  if (node.type === 'directory') {
    return '📁';
  }

  const ext = node.name.split('.').pop()?.toLowerCase() || '';
  const iconMap: Record<string, string> = {
    // Go
    go: '🔷',
    // JavaScript/TypeScript
    js: '📜',
    jsx: '⚛️',
    ts: '📘',
    tsx: '⚛️',
    // Web
    html: '🌐',
    css: '🎨',
    scss: '🎨',
    // Data
    json: '📋',
    yaml: '📋',
    yml: '📋',
    toml: '📋',
    // Documentation
    md: '📝',
    mdx: '📝',
    // Shell
    sh: '🖥️',
    bash: '🖥️',
    // Python
    py: '🐍',
    // Images
    png: '🖼️',
    jpg: '🖼️',
    svg: '🖼️',
    // Config
    lock: '🔒',
  };

  // Special filenames
  const basename = node.name.toLowerCase();
  if (basename === 'makefile') return '⚙️';
  if (basename === 'dockerfile') return '🐳';
  if (basename.startsWith('.')) return '⚙️';

  return iconMap[ext] || '📄';
}

function capitalize(str: string): string {
  return str.charAt(0).toUpperCase() + str.slice(1);
}

export default FileTree;
