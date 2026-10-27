package eval_analysis

// BenchmarkEvidence is the additive, scoped read model for the performance page.
// It consumes already validated/deduplicated results, never launches evaluations.
type BenchmarkEvidence struct {
	SchemaVersion   int            `json:"schemaVersion"`
	RequiredCoreIDs []string       `json:"requiredCoreIds"`
	Rows            []*EvidenceRow `json:"rows"`
}

type EvidenceRow struct {
	ID      string                               `json:"id"`
	Model   string                               `json:"model"`
	Mode    string                               `json:"mode"`
	Harness string                               `json:"harness"`
	Scopes  map[string]map[string]*EvidenceStats `json:"scopes"`
}

type EvidenceStats struct {
	Runs                int                        `json:"runs"`
	Passes              int                        `json:"passes"`
	FirstAttemptPasses  int                        `json:"firstAttemptPasses"`
	FirstAttemptUnknown int                        `json:"firstAttemptUnknown"`
	Repairs             int                        `json:"repairs"`
	Benchmarks          []string                   `json:"benchmarks"`
	CoreCompleted       []string                   `json:"coreCompleted"`
	FirstRun            string                     `json:"firstRun,omitempty"`
	LastRun             string                     `json:"lastRun,omitempty"`
	LatestVersion       string                     `json:"latestVersion,omitempty"`
	Versions            map[string]int             `json:"versions"`
	UnknownVersionRuns  int                        `json:"unknownVersionRuns"`
	PromptVersions      map[string]int             `json:"promptVersions"`
	OutputTokens        int                        `json:"outputTokens"`
	TotalTokens         int                        `json:"totalTokens"`
	OutputTokenSamples  int                        `json:"outputTokenSamples"`
	TotalTokenSamples   int                        `json:"totalTokenSamples"`
	Outcomes            map[string]int             `json:"outcomes"`
	Costs               map[string]*EvidenceCost   `json:"costs"`
	Timing              map[string]*EvidenceTiming `json:"timing"`
}

// Complete distinguishes a genuine recorded zero from unknown legacy cost.
// Categories never pool metered, subscription-equivalent and free-local dollars.
type EvidenceCost struct {
	Runs          int      `json:"runs"`
	Passes        int      `json:"passes"`
	TotalUSD      float64  `json:"totalUsd"`
	Complete      bool     `json:"complete"`
	PerSuccessUSD *float64 `json:"perSuccessUsd"`
}

type EvidenceTiming struct {
	Samples  int     `json:"samples"`
	MedianMs float64 `json:"medianMs"`
}
