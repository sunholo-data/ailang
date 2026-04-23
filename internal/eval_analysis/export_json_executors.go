package eval_analysis

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// Per-executor agent aggregates (claude, gemini, etc.) extracted from
// export_json.go to keep the main exporter under the 800-line soft limit.
//
// The caller feeds in agent results and gets back:
//   - executorsJS: ready for DashboardJSON.Executors
//   - perExecLang: executor -> lang -> ExecutorLangStats, consumed by the
//     language aggregate loop in ExportBenchmarkJSON to attach
//     agent_*_<executor> fields per language
//   - executorList: sorted executor names for aggregates.agentExecutors

// ExecutorLangStats exposes the counts the language aggregate loop needs
// to surface per-executor agent metrics alongside the default ones.
type ExecutorLangStats struct {
	Runs    int
	Success int
	Turns   int
	Tokens  int
	Cost    float64
}

type executorAgentTotals struct {
	runs        int
	success     int
	totalTurns  int
	totalTokens int
	totalCost   float64
}

type executorLangTotals struct {
	runs         int
	success      int
	turns        int
	tokens       int
	cost         float64
	successTurns int
	successCount int
	failureTurns int
	failureCount int
}

func buildExecutorAggregates(agentResults []*BenchmarkResult) (
	map[string]interface{},
	map[string]map[string]ExecutorLangStats,
	[]string,
) {
	perExec := make(map[string]*executorAgentTotals)
	perExecLang := make(map[string]map[string]*executorLangTotals)
	perExecModel := make(map[string]map[string]*executorAgentTotals)

	for _, r := range agentResults {
		executor := r.Executor
		if executor == "" {
			executor = "unknown"
		}

		if perExec[executor] == nil {
			perExec[executor] = &executorAgentTotals{}
		}
		es := perExec[executor]
		es.runs++
		if r.StdoutOk {
			es.success++
		}
		es.totalTurns += r.AgentTurns
		es.totalTokens += r.TotalTokens
		es.totalCost += r.CostUSD

		if perExecModel[executor] == nil {
			perExecModel[executor] = make(map[string]*executorAgentTotals)
		}
		if perExecModel[executor][r.Model] == nil {
			perExecModel[executor][r.Model] = &executorAgentTotals{}
		}
		ms := perExecModel[executor][r.Model]
		ms.runs++
		if r.StdoutOk {
			ms.success++
		}
		ms.totalTurns += r.AgentTurns
		ms.totalTokens += r.TotalTokens
		ms.totalCost += r.CostUSD

		if perExecLang[executor] == nil {
			perExecLang[executor] = make(map[string]*executorLangTotals)
		}
		if perExecLang[executor][r.Lang] == nil {
			perExecLang[executor][r.Lang] = &executorLangTotals{}
		}
		ls := perExecLang[executor][r.Lang]
		ls.runs++
		ls.turns += r.AgentTurns
		ls.tokens += r.TotalTokens
		ls.cost += r.CostUSD
		if r.StdoutOk {
			ls.success++
			ls.successTurns += r.AgentTurns
			ls.successCount++
		} else {
			ls.failureTurns += r.AgentTurns
			ls.failureCount++
		}
	}

	executorsJS := make(map[string]interface{})
	for executor, es := range perExec {
		if es.runs == 0 {
			continue
		}
		execData := map[string]interface{}{
			"runs":        es.runs,
			"successRate": float64(es.success) / float64(es.runs),
			"avgTurns":    float64(es.totalTurns) / float64(es.runs),
			"avgTokens":   float64(es.totalTokens) / float64(es.runs),
			"avgCost":     es.totalCost / float64(es.runs),
			"totalCost":   es.totalCost,
		}

		if langMap, ok := perExecLang[executor]; ok {
			langBreakdown := make(map[string]interface{})
			for lang, ls := range langMap {
				if ls.runs == 0 {
					continue
				}
				langEntry := map[string]interface{}{
					"runs":        ls.runs,
					"successRate": float64(ls.success) / float64(ls.runs),
					"avgTurns":    float64(ls.turns) / float64(ls.runs),
					"avgTokens":   float64(ls.tokens) / float64(ls.runs),
					"avgCost":     ls.cost / float64(ls.runs),
				}
				if ls.successCount > 0 {
					langEntry["avgTurnsSuccess"] = float64(ls.successTurns) / float64(ls.successCount)
				}
				if ls.failureCount > 0 {
					langEntry["avgTurnsFailure"] = float64(ls.failureTurns) / float64(ls.failureCount)
				}
				langBreakdown[lang] = langEntry
			}
			execData["languages"] = langBreakdown
		}

		if modelMap, ok := perExecModel[executor]; ok {
			modelBreakdown := make(map[string]interface{})
			for model, ms := range modelMap {
				if ms.runs == 0 {
					continue
				}
				modelBreakdown[model] = map[string]interface{}{
					"runs":        ms.runs,
					"successRate": float64(ms.success) / float64(ms.runs),
					"avgTurns":    float64(ms.totalTurns) / float64(ms.runs),
					"avgTokens":   float64(ms.totalTokens) / float64(ms.runs),
					"avgCost":     ms.totalCost / float64(ms.runs),
				}
			}
			execData["models"] = modelBreakdown
		}

		executorsJS[executor] = execData
	}

	simplified := make(map[string]map[string]ExecutorLangStats, len(perExecLang))
	for executor, langMap := range perExecLang {
		bucket := make(map[string]ExecutorLangStats, len(langMap))
		for lang, ls := range langMap {
			bucket[lang] = ExecutorLangStats{
				Runs:    ls.runs,
				Success: ls.success,
				Turns:   ls.turns,
				Tokens:  ls.tokens,
				Cost:    ls.cost,
			}
		}
		simplified[executor] = bucket
	}

	executorList := make([]string, 0, len(perExec))
	for executor := range perExec {
		executorList = append(executorList, executor)
	}
	sort.Strings(executorList)

	return executorsJS, simplified, executorList
}

// harnessTotals accumulates agent results per harness (agent_cli value).
type harnessTotals struct {
	models     map[string]struct{} // model keys seen under this harness
	runs       int
	success    int
	totalCost  float64
	totalDurMs int64
	languages  map[string]*executorLangTotals
}

// buildHarnessAggregates groups agent results by their harness (agent_cli field
// from ModelConfig) and returns a map ready for DashboardJSON.Harnesses.
// Results with no matching ModelConfig entry are grouped under "unknown".
func buildHarnessAggregates(agentResults []*BenchmarkResult) map[string]interface{} {
	cfg := eval_harness.GlobalModelsConfig
	perHarness := make(map[string]*harnessTotals)

	for _, r := range agentResults {
		harness := "unknown"
		if cfg != nil {
			if cli, err := cfg.GetAgentCLI(r.Model); err == nil && cli != "" {
				harness = cli
			}
		}

		h := perHarness[harness]
		if h == nil {
			h = &harnessTotals{
				models:    make(map[string]struct{}),
				languages: make(map[string]*executorLangTotals),
			}
			perHarness[harness] = h
		}
		h.models[r.Model] = struct{}{}
		h.runs++
		if r.StdoutOk {
			h.success++
		}
		h.totalCost += r.CostUSD
		h.totalDurMs += r.DurationMs

		ls := h.languages[r.Lang]
		if ls == nil {
			ls = &executorLangTotals{}
			h.languages[r.Lang] = ls
		}
		ls.runs++
		ls.cost += r.CostUSD
		ls.tokens += r.TotalTokens
		ls.turns += r.AgentTurns
		if r.StdoutOk {
			ls.success++
		}
	}

	harnessDisplayNames := map[string]string{
		"claude":   "Claude Code CLI",
		"gemini":   "Gemini CLI",
		"opencode": "opencode CLI",
		"codex":    "Codex CLI",
	}

	result := make(map[string]interface{}, len(perHarness))
	for harness, h := range perHarness {
		if h.runs == 0 {
			continue
		}
		modelList := make([]string, 0, len(h.models))
		for m := range h.models {
			modelList = append(modelList, m)
		}
		sort.Strings(modelList)

		displayName := harnessDisplayNames[harness]
		if displayName == "" {
			displayName = harness
		}

		langBreakdown := make(map[string]interface{}, len(h.languages))
		for lang, ls := range h.languages {
			if ls.runs == 0 {
				continue
			}
			langBreakdown[lang] = map[string]interface{}{
				"runs":        ls.runs,
				"successRate": float64(ls.success) / float64(ls.runs),
				"avgTokens":   float64(ls.tokens) / float64(ls.runs),
				"avgCost":     ls.cost / float64(ls.runs),
			}
		}

		entry := map[string]interface{}{
			"name":            harness,
			"display_name":    displayName,
			"models":          modelList,
			"total_runs":      h.runs,
			"success_rate":    float64(h.success) / float64(h.runs),
			"avg_cost_usd":    h.totalCost / float64(h.runs),
			"avg_duration_ms": float64(h.totalDurMs) / float64(h.runs),
			"languages":       langBreakdown,
		}
		result[harness] = entry
	}
	return result
}
