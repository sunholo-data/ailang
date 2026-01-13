export { DiffViewer, default } from './DiffViewer';
export type { ViewMode } from './DiffViewer';
export {
  parseDiff,
  extractFileChanges,
  buildFileTree,
  detectLanguage,
} from './diffParser';
export type {
  ParsedDiff,
  FileDiff,
  DiffHunk,
  DiffLine,
  FileChange,
  TreeNode,
} from './diffParser';
