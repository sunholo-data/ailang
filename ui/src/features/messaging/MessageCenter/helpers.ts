/**
 * MessageCenter helper functions
 */
import { Message, ExecutionMetadata } from '../../../types';

// Maximum characters to display before truncating (10KB)
export const MAX_DISPLAY_LENGTH = 10 * 1024;
// Maximum lines to show before truncating
export const MAX_DISPLAY_LINES = 200;

/**
 * Parse execution metadata from a message's metadata_json field
 */
export const parseExecutionMetadata = (metadataJson?: string): ExecutionMetadata | null => {
  if (!metadataJson) return null;
  try {
    const metadata = JSON.parse(metadataJson);
    return metadata.execution_stats || null;
  } catch {
    return null;
  }
};

/**
 * Check if a status message indicates "running" state
 */
export const isRunningStatus = (message: Message): boolean => {
  if (message.kind !== 'status') return false;
  const content = message.content.toLowerCase();
  return content.includes('running') || content.includes('thinking') || content.includes('executing') || content.includes('processing');
};

/**
 * Get truncated content for display with metadata about truncation
 */
export interface TruncatedContent {
  needsTruncation: boolean;
  truncated: string;
  fullLength: number;
  lineCount: number;
}

export const getTruncatedContent = (content: string): TruncatedContent => {
  const lineCount = (content.match(/\n/g) || []).length + 1;
  const needsTruncation = content.length > MAX_DISPLAY_LENGTH || lineCount > MAX_DISPLAY_LINES;

  if (!needsTruncation) {
    return { needsTruncation: false, truncated: content, fullLength: content.length, lineCount };
  }

  // Truncate by character limit first
  let truncated = content.slice(0, MAX_DISPLAY_LENGTH);

  // Then truncate by line limit if still too many lines
  const lines = truncated.split('\n');
  if (lines.length > MAX_DISPLAY_LINES) {
    truncated = lines.slice(0, MAX_DISPLAY_LINES).join('\n');
  }

  // Try to end at a newline for cleaner display
  const lastNewline = truncated.lastIndexOf('\n');
  if (lastNewline > truncated.length * 0.8) {
    truncated = truncated.slice(0, lastNewline);
  }

  return { needsTruncation: true, truncated, fullLength: content.length, lineCount };
};

/**
 * Parse approval ID from message metadata
 */
export const getApprovalId = (message: Message): string | null => {
  if (!message.metadata_json) return null;
  try {
    const metadata = JSON.parse(message.metadata_json);
    return metadata.approval_id || null;
  } catch {
    return null;
  }
};
