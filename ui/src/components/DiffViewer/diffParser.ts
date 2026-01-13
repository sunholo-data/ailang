/**
 * diffParser.ts - Parse unified diff format into structured data
 *
 * Parses git unified diff output into a structured format for rendering.
 */

export interface DiffLine {
  type: 'add' | 'delete' | 'context' | 'header' | 'hunk';
  content: string;
  oldLineNumber?: number;
  newLineNumber?: number;
}

export interface DiffHunk {
  oldStart: number;
  oldCount: number;
  newStart: number;
  newCount: number;
  header: string;
  lines: DiffLine[];
}

export interface FileDiff {
  oldPath: string;
  newPath: string;
  status: 'added' | 'deleted' | 'modified' | 'renamed';
  additions: number;
  deletions: number;
  hunks: DiffHunk[];
  language: string;
}

export interface ParsedDiff {
  files: FileDiff[];
  totalAdditions: number;
  totalDeletions: number;
}

/**
 * Detect language from file extension for syntax highlighting
 */
export function detectLanguage(filename: string): string {
  const ext = filename.split('.').pop()?.toLowerCase() || '';
  const languageMap: Record<string, string> = {
    // Go
    go: 'go',
    // JavaScript/TypeScript
    js: 'javascript',
    jsx: 'javascript',
    ts: 'typescript',
    tsx: 'typescript',
    // Web
    html: 'html',
    css: 'css',
    scss: 'scss',
    // Data
    json: 'json',
    yaml: 'yaml',
    yml: 'yaml',
    toml: 'toml',
    // Documentation
    md: 'markdown',
    mdx: 'markdown',
    // Shell
    sh: 'bash',
    bash: 'bash',
    zsh: 'bash',
    // Python
    py: 'python',
    // SQL
    sql: 'sql',
    // Other
    makefile: 'makefile',
    dockerfile: 'dockerfile',
  };

  // Handle special filenames
  const basename = filename.split('/').pop()?.toLowerCase() || '';
  if (basename === 'makefile') return 'makefile';
  if (basename === 'dockerfile') return 'dockerfile';
  if (basename.startsWith('.') && !ext) return 'plaintext';

  return languageMap[ext] || 'plaintext';
}

/**
 * Parse a unified diff string into structured data
 */
export function parseDiff(diffText: string): ParsedDiff {
  const files: FileDiff[] = [];
  let totalAdditions = 0;
  let totalDeletions = 0;

  if (!diffText || !diffText.trim()) {
    return { files, totalAdditions, totalDeletions };
  }

  // Split into file sections
  const fileSections = diffText.split(/(?=^diff --git )/m).filter(Boolean);

  for (const section of fileSections) {
    const fileDiff = parseFileDiff(section);
    if (fileDiff) {
      files.push(fileDiff);
      totalAdditions += fileDiff.additions;
      totalDeletions += fileDiff.deletions;
    }
  }

  return { files, totalAdditions, totalDeletions };
}

/**
 * Parse a single file's diff section
 */
function parseFileDiff(section: string): FileDiff | null {
  const lines = section.split('\n');

  // Parse the diff header: diff --git a/path b/path
  const diffHeader = lines[0];
  const headerMatch = diffHeader?.match(/^diff --git a\/(.+) b\/(.+)/);
  if (!headerMatch) return null;

  const oldPath = headerMatch[1];
  const newPath = headerMatch[2];

  // Determine status from subsequent lines
  let status: FileDiff['status'] = 'modified';
  let additions = 0;
  let deletions = 0;

  // Look for new file / deleted file indicators
  for (const line of lines.slice(1, 10)) {
    if (line.startsWith('new file mode')) {
      status = 'added';
    } else if (line.startsWith('deleted file mode')) {
      status = 'deleted';
    } else if (line.startsWith('rename from')) {
      status = 'renamed';
    }
  }

  // Parse hunks
  const hunks: DiffHunk[] = [];
  let currentHunk: DiffHunk | null = null;
  let oldLineNum = 0;
  let newLineNum = 0;

  for (const line of lines) {
    // Hunk header: @@ -10,6 +10,8 @@ optional context
    const hunkMatch = line.match(/^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$/);
    if (hunkMatch) {
      if (currentHunk) {
        hunks.push(currentHunk);
      }
      oldLineNum = parseInt(hunkMatch[1], 10);
      newLineNum = parseInt(hunkMatch[3], 10);
      currentHunk = {
        oldStart: oldLineNum,
        oldCount: parseInt(hunkMatch[2] || '1', 10),
        newStart: newLineNum,
        newCount: parseInt(hunkMatch[4] || '1', 10),
        header: line,
        lines: [{ type: 'hunk', content: line }],
      };
      continue;
    }

    if (!currentHunk) continue;

    // Process diff lines
    if (line.startsWith('+') && !line.startsWith('+++')) {
      currentHunk.lines.push({
        type: 'add',
        content: line.substring(1),
        newLineNumber: newLineNum++,
      });
      additions++;
    } else if (line.startsWith('-') && !line.startsWith('---')) {
      currentHunk.lines.push({
        type: 'delete',
        content: line.substring(1),
        oldLineNumber: oldLineNum++,
      });
      deletions++;
    } else if (line.startsWith(' ')) {
      currentHunk.lines.push({
        type: 'context',
        content: line.substring(1),
        oldLineNumber: oldLineNum++,
        newLineNumber: newLineNum++,
      });
    }
  }

  if (currentHunk) {
    hunks.push(currentHunk);
  }

  return {
    oldPath,
    newPath,
    status,
    additions,
    deletions,
    hunks,
    language: detectLanguage(newPath),
  };
}

/**
 * Extract file changes from a diff for the file tree
 */
export interface FileChange {
  path: string;
  additions: number;
  deletions: number;
  status: 'added' | 'deleted' | 'modified' | 'renamed';
}

export function extractFileChanges(diff: ParsedDiff): FileChange[] {
  return diff.files.map((f) => ({
    path: f.newPath,
    additions: f.additions,
    deletions: f.deletions,
    status: f.status,
  }));
}

/**
 * Build a tree structure from flat file paths
 */
export interface TreeNode {
  name: string;
  path: string;
  type: 'file' | 'directory';
  additions: number;
  deletions: number;
  status?: FileChange['status'];
  children: TreeNode[];
  expanded: boolean;
}

export function buildFileTree(files: FileChange[]): TreeNode {
  const root: TreeNode = {
    name: '',
    path: '',
    type: 'directory',
    additions: 0,
    deletions: 0,
    children: [],
    expanded: true,
  };

  for (const file of files) {
    const parts = file.path.split('/');
    let current = root;

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      const isFile = i === parts.length - 1;
      const currentPath = parts.slice(0, i + 1).join('/');

      let child = current.children.find((c) => c.name === part);

      if (!child) {
        child = {
          name: part,
          path: currentPath,
          type: isFile ? 'file' : 'directory',
          additions: isFile ? file.additions : 0,
          deletions: isFile ? file.deletions : 0,
          status: isFile ? file.status : undefined,
          children: [],
          expanded: true,
        };
        current.children.push(child);
      }

      if (!isFile) {
        // Accumulate counts for directories
        child.additions += file.additions;
        child.deletions += file.deletions;
      }

      current = child;
    }

    // Update root totals
    root.additions += file.additions;
    root.deletions += file.deletions;
  }

  // Sort children: directories first, then alphabetically
  const sortChildren = (node: TreeNode) => {
    node.children.sort((a, b) => {
      if (a.type !== b.type) {
        return a.type === 'directory' ? -1 : 1;
      }
      return a.name.localeCompare(b.name);
    });
    node.children.forEach(sortChildren);
  };

  sortChildren(root);

  return root;
}
