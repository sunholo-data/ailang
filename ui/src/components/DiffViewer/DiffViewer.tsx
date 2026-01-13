/**
 * DiffViewer.tsx - Git diff viewer with syntax highlighting
 *
 * Supports unified and split (side-by-side) view modes.
 */
/* eslint-disable @typescript-eslint/no-use-before-define */

import React, { useMemo, useRef, useEffect, useCallback } from 'react';
import hljs from 'highlight.js/lib/core';
// Import common languages
import go from 'highlight.js/lib/languages/go';
import typescript from 'highlight.js/lib/languages/typescript';
import javascript from 'highlight.js/lib/languages/javascript';
import json from 'highlight.js/lib/languages/json';
import yaml from 'highlight.js/lib/languages/yaml';
import markdown from 'highlight.js/lib/languages/markdown';
import bash from 'highlight.js/lib/languages/bash';
import python from 'highlight.js/lib/languages/python';
import sql from 'highlight.js/lib/languages/sql';
import css from 'highlight.js/lib/languages/css';

import { parseDiff, FileDiff, DiffHunk, DiffLine } from './diffParser';
import styles from './DiffViewer.module.css';

// Register languages
hljs.registerLanguage('go', go);
hljs.registerLanguage('typescript', typescript);
hljs.registerLanguage('javascript', javascript);
hljs.registerLanguage('json', json);
hljs.registerLanguage('yaml', yaml);
hljs.registerLanguage('markdown', markdown);
hljs.registerLanguage('bash', bash);
hljs.registerLanguage('python', python);
hljs.registerLanguage('sql', sql);
hljs.registerLanguage('css', css);

export type ViewMode = 'unified' | 'split';

interface DiffViewerProps {
  diff: string;
  viewMode?: ViewMode;
  compact?: boolean;
  selectedFile?: string;
  onFileClick?: (path: string) => void;
  maxHeight?: string;
}

export const DiffViewer: React.FC<DiffViewerProps> = ({
  diff,
  viewMode = 'unified',
  compact = false,
  selectedFile,
  onFileClick,
  maxHeight = '100%',
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const fileRefs = useRef<Map<string, HTMLDivElement>>(new Map());

  const parsedDiff = useMemo(() => parseDiff(diff), [diff]);

  // Scroll to selected file
  useEffect(() => {
    if (selectedFile && fileRefs.current.has(selectedFile)) {
      const element = fileRefs.current.get(selectedFile);
      element?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  }, [selectedFile]);

  const setFileRef = useCallback((path: string, el: HTMLDivElement | null) => {
    if (el) {
      fileRefs.current.set(path, el);
    } else {
      fileRefs.current.delete(path);
    }
  }, []);

  if (!diff || !diff.trim()) {
    return (
      <div className={styles.empty}>
        <span>No changes to display</span>
      </div>
    );
  }

  if (parsedDiff.files.length === 0) {
    return (
      <div className={styles.empty}>
        <span>Unable to parse diff</span>
        <pre className={styles.rawDiff}>{diff}</pre>
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      className={`${styles.diffViewer} ${compact ? styles.compact : ''}`}
      style={{ maxHeight }}
    >
      {/* Summary */}
      {!compact && (
        <div className={styles.summary}>
          <span className={styles.fileCount}>
            {parsedDiff.files.length} file{parsedDiff.files.length !== 1 ? 's' : ''} changed
          </span>
          <span className={styles.additions}>+{parsedDiff.totalAdditions}</span>
          <span className={styles.deletions}>-{parsedDiff.totalDeletions}</span>
        </div>
      )}

      {/* Files */}
      {parsedDiff.files.map((file) => (
        <FileDiffView
          key={file.newPath}
          file={file}
          viewMode={viewMode}
          compact={compact}
          isSelected={selectedFile === file.newPath}
          onFileClick={onFileClick}
          ref={(el) => setFileRef(file.newPath, el)}
        />
      ))}
    </div>
  );
};

interface FileDiffViewProps {
  file: FileDiff;
  viewMode: ViewMode;
  compact: boolean;
  isSelected: boolean;
  onFileClick?: (path: string) => void;
}

const FileDiffView = React.forwardRef<HTMLDivElement, FileDiffViewProps>(
  ({ file, viewMode, compact, isSelected, onFileClick }, ref) => {
    const [collapsed, setCollapsed] = React.useState(false);

    const handleHeaderClick = () => {
      if (onFileClick) {
        onFileClick(file.newPath);
      }
    };

    const statusIcon = {
      added: '+',
      deleted: '-',
      modified: 'M',
      renamed: 'R',
    }[file.status];

    const statusClass = styles[`status${file.status.charAt(0).toUpperCase()}${file.status.slice(1)}`];

    return (
      <div
        ref={ref}
        className={`${styles.fileDiff} ${isSelected ? styles.selected : ''}`}
      >
        {/* File Header */}
        <div className={styles.fileHeader} onClick={handleHeaderClick}>
          <button
            className={styles.collapseButton}
            onClick={(e) => {
              e.stopPropagation();
              setCollapsed(!collapsed);
            }}
          >
            {collapsed ? '+' : '-'}
          </button>
          <span className={`${styles.statusBadge} ${statusClass}`}>{statusIcon}</span>
          <span className={styles.filePath}>{file.newPath}</span>
          <span className={styles.fileStats}>
            <span className={styles.additions}>+{file.additions}</span>
            <span className={styles.deletions}>-{file.deletions}</span>
          </span>
        </div>

        {/* Diff Content */}
        {!collapsed && (
          <div className={styles.fileContent}>
            {viewMode === 'unified' ? (
              <UnifiedDiffView file={file} compact={compact} />
            ) : (
              <SplitDiffView file={file} compact={compact} />
            )}
          </div>
        )}
      </div>
    );
  }
);

FileDiffView.displayName = 'FileDiffView';

interface DiffContentProps {
  file: FileDiff;
  compact: boolean;
}

/**
 * Unified diff view - single column with +/- lines
 */
const UnifiedDiffView: React.FC<DiffContentProps> = ({ file, compact }) => {
  return (
    <table className={styles.diffTable}>
      <tbody>
        {file.hunks.map((hunk, hunkIndex) => (
          <HunkView key={hunkIndex} hunk={hunk} file={file} compact={compact} />
        ))}
      </tbody>
    </table>
  );
};

interface HunkViewProps {
  hunk: DiffHunk;
  file: FileDiff;
  compact: boolean;
}

const HunkView: React.FC<HunkViewProps> = ({ hunk, file, compact }) => {
  return (
    <>
      {/* Hunk header */}
      <tr className={styles.hunkHeader}>
        <td className={styles.lineNumber}></td>
        <td className={styles.lineNumber}></td>
        <td className={styles.hunkHeaderContent}>{hunk.header}</td>
      </tr>
      {/* Diff lines */}
      {hunk.lines
        .filter((line) => line.type !== 'hunk')
        .map((line, lineIndex) => (
          <DiffLineView
            key={lineIndex}
            line={line}
            language={file.language}
            compact={compact}
          />
        ))}
    </>
  );
};

interface DiffLineViewProps {
  line: DiffLine;
  language: string;
  compact: boolean;
}

const DiffLineView: React.FC<DiffLineViewProps> = ({ line, language, compact }) => {
  const lineClass = styles[`line${line.type.charAt(0).toUpperCase()}${line.type.slice(1)}`];

  // Syntax highlight the content
  const highlightedContent = useMemo(() => {
    if (!line.content || line.type === 'header' || line.type === 'hunk') {
      return line.content;
    }
    try {
      const result = hljs.highlight(line.content, { language, ignoreIllegals: true });
      return result.value;
    } catch {
      return line.content;
    }
  }, [line.content, language, line.type]);

  return (
    <tr className={lineClass}>
      <td className={styles.lineNumber}>
        {line.oldLineNumber ?? ''}
      </td>
      <td className={styles.lineNumber}>
        {line.newLineNumber ?? ''}
      </td>
      <td className={styles.lineContent}>
        <span className={styles.linePrefix}>
          {line.type === 'add' ? '+' : line.type === 'delete' ? '-' : ' '}
        </span>
        <code
          className={styles.code}
          dangerouslySetInnerHTML={{ __html: highlightedContent }}
        />
      </td>
    </tr>
  );
};

/**
 * Split diff view - side-by-side columns
 */
const SplitDiffView: React.FC<DiffContentProps> = ({ file, compact }) => {
  // Build paired lines for side-by-side display
  const pairedLines = useMemo(() => {
    const pairs: Array<{
      left: DiffLine | null;
      right: DiffLine | null;
      isHunkHeader: boolean;
      header?: string;
    }> = [];

    for (const hunk of file.hunks) {
      // Add hunk header
      pairs.push({
        left: null,
        right: null,
        isHunkHeader: true,
        header: hunk.header,
      });

      // Process lines - pair deletions with additions
      let i = 0;
      const lines = hunk.lines.filter((l) => l.type !== 'hunk');

      while (i < lines.length) {
        const line = lines[i];

        if (line.type === 'context') {
          pairs.push({ left: line, right: line, isHunkHeader: false });
          i++;
        } else if (line.type === 'delete') {
          // Look for matching addition
          const nextLine = lines[i + 1];
          if (nextLine?.type === 'add') {
            pairs.push({ left: line, right: nextLine, isHunkHeader: false });
            i += 2;
          } else {
            pairs.push({ left: line, right: null, isHunkHeader: false });
            i++;
          }
        } else if (line.type === 'add') {
          pairs.push({ left: null, right: line, isHunkHeader: false });
          i++;
        } else {
          i++;
        }
      }
    }

    return pairs;
  }, [file.hunks]);

  return (
    <table className={`${styles.diffTable} ${styles.splitTable}`}>
      <tbody>
        {pairedLines.map((pair, index) => {
          if (pair.isHunkHeader) {
            return (
              <tr key={index} className={styles.hunkHeader}>
                <td colSpan={4} className={styles.hunkHeaderContent}>
                  {pair.header}
                </td>
              </tr>
            );
          }

          return (
            <tr key={index} className={styles.splitRow}>
              {/* Left side (old) */}
              <td className={styles.lineNumber}>{pair.left?.oldLineNumber ?? ''}</td>
              <td
                className={`${styles.splitContent} ${
                  pair.left?.type === 'delete' ? styles.lineDelete : ''
                }`}
              >
                {pair.left && (
                  <SplitLineContent line={pair.left} language={file.language} />
                )}
              </td>
              {/* Right side (new) */}
              <td className={styles.lineNumber}>{pair.right?.newLineNumber ?? ''}</td>
              <td
                className={`${styles.splitContent} ${
                  pair.right?.type === 'add' ? styles.lineAdd : ''
                }`}
              >
                {pair.right && (
                  <SplitLineContent line={pair.right} language={file.language} />
                )}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
};

interface SplitLineContentProps {
  line: DiffLine;
  language: string;
}

const SplitLineContent: React.FC<SplitLineContentProps> = ({ line, language }) => {
  const highlightedContent = useMemo(() => {
    if (!line.content) return '';
    try {
      const result = hljs.highlight(line.content, { language, ignoreIllegals: true });
      return result.value;
    } catch {
      return line.content;
    }
  }, [line.content, language]);

  return (
    <code
      className={styles.code}
      dangerouslySetInnerHTML={{ __html: highlightedContent }}
    />
  );
};

export default DiffViewer;
