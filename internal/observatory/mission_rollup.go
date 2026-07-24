package observatory

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Mission cost rollup (M-MISSION-COST-CHAINS, M3).
//
// `ailang chains stats --by-mission` groups chains by their source_ref prefix
// (mission:<name>/iter-<N>) and reports, per mission: the split metered/estimated/
// quota/unknown totals (reusing the M1 classifier), per-bucket quota-stage counts
// (the bucket is encoded by M2 in the free-text agent_id, e.g. "…(quota:opus)"),
// and the top-N most expensive stages. CLI only — no dashboards.

// MissionStageCost is one classified stage within a mission rollup, kept for the
// top-N most-expensive listing.
type MissionStageCost struct {
	AgentID string     `json:"agent_id"`
	Status  CostStatus `json:"status"`
	CostUSD float64    `json:"cost_usd"`
	Tokens  int64      `json:"tokens"`
	Model   string     `json:"model,omitempty"`
}

// MissionRollup is the per-mission cost summary.
type MissionRollup struct {
	Mission string     `json:"mission"` // e.g. "mission:v1"
	Rollup  CostRollup `json:"rollup"`
	// QuotaByBucket counts quota-lane stages per subscription bucket
	// (fable|opus|sonnet|…), parsed from the free-text agent_id.
	QuotaByBucket map[string]int `json:"quota_by_bucket"`
	// TopStages are the most expensive classified stages (reported+estimated), desc.
	TopStages []MissionStageCost `json:"top_stages"`
}

// GetMissionRollups groups chains by mission (source_ref prefix up to the first
// '/', i.e. "mission:<name>") within the window and returns a per-mission split
// rollup, per-bucket quota counts, and the top-N most expensive stages per mission.
// sourcePrefix restricts which chains are considered (e.g. "mission:").
func (s *Store) GetMissionRollups(ctx context.Context, createdAfter *time.Time, sourcePrefix string, topN int) ([]MissionRollup, error) {
	if topN <= 0 {
		topN = 5
	}

	query := `
		SELECT c.source_ref, cs.agent_id, cs.cost, cs.tokens_in, cs.tokens_out,
		       COALESCE(json_extract(cs.eval_assessment, '$.model'), '') AS ea_model,
		       COALESCE((
		           SELECT sp.model FROM spans sp
		           WHERE sp.stage_id = cs.id AND sp.model IS NOT NULL AND sp.model != ''
		           ORDER BY sp.start_time ASC LIMIT 1
		       ), '') AS span_model
		FROM chain_stages cs
		JOIN execution_chains c ON cs.chain_id = c.id
		WHERE c.source_ref LIKE ?
	`
	args := []interface{}{sourcePrefix + "%"}
	if createdAfter != nil {
		query += " AND c.created_at > ?"
		args = append(args, *createdAfter)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query mission rollups: %w", err)
	}
	defer rows.Close()

	// Accumulate per-mission.
	type acc struct {
		rollup    CostRollup
		byBucket  map[string]int
		allStages []MissionStageCost
	}
	missions := map[string]*acc{}
	var order []string // preserve first-seen order for stable output

	for rows.Next() {
		var sourceRef, agentID, eaModel, spanModel string
		var cost float64
		var tokensIn, tokensOut int
		var sourceRefNull, agentIDNull sql.NullString
		if err := rows.Scan(&sourceRefNull, &agentIDNull, &cost, &tokensIn, &tokensOut, &eaModel, &spanModel); err != nil {
			return nil, fmt.Errorf("failed to scan mission rollup row: %w", err)
		}
		sourceRef = sourceRefNull.String
		agentID = agentIDNull.String

		missionName := missionKey(sourceRef)
		a, ok := missions[missionName]
		if !ok {
			a = &acc{byBucket: map[string]int{}}
			missions[missionName] = a
			order = append(order, missionName)
		}

		stage := &ChainStage{Cost: cost, TokensIn: tokensIn, TokensOut: tokensOut, AgentID: agentID}
		if eaModel != "" {
			stage.EvalAssessment = &EvalAssessment{Model: eaModel}
		} else if spanModel != "" {
			stage.Spans = []*Span{{Model: spanModel}}
		}
		sc := ClassifyStageCost(stage)
		a.rollup.AddStage(stage)

		if sc.Status == CostStatusQuota {
			if bucket := parseQuotaBucket(agentID); bucket != "" {
				a.byBucket[bucket]++
			} else {
				a.byBucket["unlabeled"]++
			}
		}
		a.allStages = append(a.allStages, MissionStageCost{
			AgentID: agentID,
			Status:  sc.Status,
			CostUSD: sc.CostUSD,
			Tokens:  int64(tokensIn) + int64(tokensOut),
			Model:   sc.Model,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	results := make([]MissionRollup, 0, len(order))
	for _, name := range order {
		a := missions[name]
		// Top-N by cost desc (only reported/estimated carry dollars).
		sort.Slice(a.allStages, func(i, j int) bool {
			return a.allStages[i].CostUSD > a.allStages[j].CostUSD
		})
		top := a.allStages
		if len(top) > topN {
			top = top[:topN]
		}
		results = append(results, MissionRollup{
			Mission:       name,
			Rollup:        a.rollup,
			QuotaByBucket: a.byBucket,
			TopStages:     top,
		})
	}
	return results, nil
}

// missionKey reduces a source_ref like "mission:v1/iter-42" to "mission:v1".
// Anything without a '/' is returned unchanged.
func missionKey(sourceRef string) string {
	if idx := strings.Index(sourceRef, "/"); idx >= 0 {
		return sourceRef[:idx]
	}
	return sourceRef
}

// parseQuotaBucket extracts a subscription bucket from a free-text agent_id.
// M2 encodes it as "…(quota:opus)". Returns "" if no bucket marker is present.
func parseQuotaBucket(agentID string) string {
	const marker = "quota:"
	idx := strings.Index(agentID, marker)
	if idx < 0 {
		return ""
	}
	rest := agentID[idx+len(marker):]
	// Bucket runs until a delimiter: ')', space, or end.
	end := strings.IndexAny(rest, ") ")
	if end < 0 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:end])
}
