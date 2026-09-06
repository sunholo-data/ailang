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
	//
	// RAW spellings, deliberately unchanged: existing callers read this as a count and a
	// past rollup must still reproduce. The canonical view lives beside it.
	QuotaByBucket map[string]int `json:"quota_by_bucket"`
	// QuotaTokensByBucket sums TOKENS per CANONICAL bucket (codex|anthropic|openrouter|
	// ollama). Counts cannot be rationed against — an iteration is not a unit of quota —
	// and the raw spellings cannot be summed, since one bucket is written four ways.
	QuotaTokensByBucket map[string]int64 `json:"quota_tokens_by_bucket"`
	// QuotaStagesByCanonicalBucket counts stages under the same canonical key, so a token
	// total can be read against the number of stages that produced it.
	QuotaStagesByCanonicalBucket map[string]int `json:"quota_stages_by_canonical_bucket"`
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
		canonTok  map[string]int64
		canonN    map[string]int
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
			a = &acc{byBucket: map[string]int{}, canonTok: map[string]int64{}, canonN: map[string]int{}}
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
			bucket := parseQuotaBucket(agentID)
			if bucket != "" {
				a.byBucket[bucket]++
			} else {
				a.byBucket["unlabeled"]++
			}
			// Tokens under the CANONICAL key. An unlabelled stage is counted as
			// "unlabeled" rather than dropped or attributed to a neighbour: it still
			// consumed a bucket, and a ration that cannot see it is measuring low. 43
			// such stages exist in v1 alone.
			ck := canonicalBucket(bucket)
			if ck == "" {
				ck = "unlabeled"
			}
			a.canonTok[ck] += int64(stage.TokensIn) + int64(stage.TokensOut)
			a.canonN[ck]++
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
			Mission:                      name,
			Rollup:                       a.rollup,
			QuotaByBucket:                a.byBucket,
			QuotaTokensByBucket:          a.canonTok,
			QuotaStagesByCanonicalBucket: a.canonN,
			TopStages:                    top,
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

// canonicalBucket folds the spellings of one subscription bucket into a single key.
//
// The bucket is parsed out of a FREE-TEXT agent_id, so nothing has ever constrained how it
// is written, and four spellings of codex accumulated in the v1 mission alone:
//
//	codex 70 · codex-chatgpt 6 · Codex-OAuth 1 · codex-oauth 4
//
// That is harmless while the value is only displayed, and fatal the moment anything RATIONS
// against it: a limit computed over `codex` would see 70 stages and miss 11. Canonicalising
// happens at READ time and the stored agent_id is left untouched, so the raw record stays
// auditable and a past rollup can still be re-derived from it.
//
// Unrecognised values are returned trimmed-and-lowercased rather than folded into a
// neighbour. An unknown bucket must stay visible as itself; quietly attaching it to the
// nearest real one is how a ration ends up measuring the wrong thing and saying nothing.
func canonicalBucket(raw string) string {
	b := strings.ToLower(strings.TrimSpace(raw))
	if b == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(b, "codex"), b == "chatgpt", strings.HasPrefix(b, "chatgpt-"),
		strings.HasPrefix(b, "gpt-"), strings.HasPrefix(b, "gpt5"), strings.HasPrefix(b, "gpt6"):
		return "codex"
	case b == "opus", b == "sonnet", b == "haiku", b == "fable",
		strings.HasPrefix(b, "claude"), strings.HasPrefix(b, "weekly-"):
		return "anthropic"
	case strings.HasPrefix(b, "openrouter"), strings.HasPrefix(b, "or-"):
		return "openrouter"
	case strings.HasPrefix(b, "ollama"), strings.HasPrefix(b, "pi-"), strings.HasPrefix(b, "pi:"):
		return "ollama"
	}
	return b
}
