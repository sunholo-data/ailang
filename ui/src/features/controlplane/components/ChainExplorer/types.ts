/**
 * Types for the ChainExplorer component family.
 * Reuses ChainData, ChainStageData, Span from ExecHierarchy/types.ts
 */

/** Summary data returned by GET /api/chains (mirrors Go observatory.ChainSummary) */
export interface ChainSummary {
  id: string;
  source_type: string;
  source_ref: string;
  github_repo?: string;
  github_issue_number?: number;
  status: 'active' | 'pending_approval' | 'completed' | 'failed';
  current_stage: number;
  total_cost: number;
  total_tokens: number;
  total_turns: number;
  stages_completed: number;
  created_at: string;
  completed_at?: string;
  stage_count: number;
  max_stage: number;
  agent_flow: string;
}

/** Filter options for the chain list */
export interface ChainListFilters {
  status?: string;
  source_type?: string;
  agent_id?: string;
  since?: number; // hours
}

/** Sort options */
export type ChainSortField = 'created_at' | 'total_cost' | 'total_tokens';

/** Which stage detail tab is active */
export type StageDetailTab = 'summary' | 'chat' | 'tools' | 'files';

/** Tool usage entry extracted from spans */
export interface ToolUsageEntry {
  toolName: string;
  count: number;
  totalDurationMs: number;
  errors: number;
}

/** File touch entry extracted from span attributes */
export interface FileTouchEntry {
  path: string;
  operations: string[];
  lastTouched: number;
}
