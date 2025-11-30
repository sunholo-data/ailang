/**
 * Helper functions for displaying user-friendly agent/entity names
 */

// Map of internal IDs to human-friendly display names
const DISPLAY_NAMES: Record<string, string> = {
  'user': 'You',
  'human': 'You',
  'system': 'System',
};

// IDs that represent the human operator (not AI agents)
const HUMAN_IDS = new Set(['user', 'human']);

/**
 * Get a human-friendly display name for an agent/entity ID
 */
export function getDisplayName(id: string): string {
  if (!id) return 'Unknown';

  // Check for exact match first
  if (DISPLAY_NAMES[id.toLowerCase()]) {
    return DISPLAY_NAMES[id.toLowerCase()];
  }

  return id;
}

/**
 * Check if an ID represents the human user (not an AI agent)
 */
export function isHumanUser(id: string): boolean {
  if (!id) return false;
  return HUMAN_IDS.has(id.toLowerCase());
}

/**
 * Get the label to show in the agent list
 * Returns "You (Human)" for human IDs to distinguish from AI agents
 */
export function getAgentLabel(id: string): string {
  if (isHumanUser(id)) {
    return 'You (Human)';
  }
  return id;
}

/**
 * Get an appropriate icon type for an entity
 */
export function getEntityIconType(id: string, fromType?: string): 'human' | 'agent' | 'system' {
  if (fromType === 'human' || isHumanUser(id)) {
    return 'human';
  }
  if (id === 'system' || fromType === 'system') {
    return 'system';
  }
  return 'agent';
}
