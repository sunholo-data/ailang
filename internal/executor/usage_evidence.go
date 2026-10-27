package executor

// UsageObservation is one source event, captured by the existing harness parser.
// Pointer fields preserve absent vs explicitly reported zero. Output may include
// reasoning; OutputScheme declares this rather than subtracting unknown zeros.
type UsageObservation struct {
	ImportID      string  `json:"importId"`
	EventDigest   string  `json:"eventDigest"`
	RequestID     *string `json:"requestId"`
	Provider      *string `json:"provider"`
	Model         *string `json:"model"`
	Input         *int    `json:"input"`
	Output        *int    `json:"output"`
	CacheRead     *int    `json:"cacheRead"`
	CacheWrite    *int    `json:"cacheWrite"`
	Reasoning     *int    `json:"reasoning"`
	ReportedTotal *int    `json:"reportedTotal"`
	Contract      string  `json:"contract"`
	OutputScheme  string  `json:"outputScheme"`
	CostAmount    *string `json:"costAmount"`
	CostBasis     string  `json:"costBasis"`
	Failed        bool    `json:"failed"`
	RawUsage      []byte  `json:"rawUsage"`
}

// UsageEvidence never derives completeness from aggregate counters. A settled
// stream can still omit failed requests or compaction usage. Consumers retain an
// unchecked summary whenever CoverageComplete is false, so partial subtotals
// cannot masquerade as total maintenance cost.
type UsageEvidence struct {
	Source             string             `json:"source"`
	Version            string             `json:"version"`
	StreamDigest       string             `json:"streamDigest"`
	Observations       []UsageObservation `json:"observations"`
	CoverageComplete   bool               `json:"coverageComplete"`
	Settled            bool               `json:"settled"`
	RetryEvents        int                `json:"retryEvents"`
	CompactionEvents   int                `json:"compactionEvents"`
	UnavailableReasons []string           `json:"unavailableReasons"`
}
