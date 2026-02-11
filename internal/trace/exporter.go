package trace

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// TrainingExample is a single training data record for AI fine-tuning.
type TrainingExample struct {
	Source   string       `json:"source"`   // AILANG source code
	Trace    string       `json:"trace"`    // JSONL trace events (newline-separated)
	Score    float64      `json:"score"`    // Quality score 0.0-1.0
	Metadata TrainingMeta `json:"metadata"` // Summary metadata
}

// TrainingMeta holds summary metadata for a training example.
type TrainingMeta struct {
	Module    string   `json:"module,omitempty"`  // Module name
	Caps      []string `json:"caps,omitempty"`    // Capabilities used
	Functions int      `json:"functions"`         // Distinct function count
	Effects   []string `json:"effects,omitempty"` // Distinct effect names
	MaxDepth  int      `json:"max_depth"`         // Maximum call depth
	Events    int      `json:"events"`            // Total event count
}

// ExportConfig controls training data export behavior.
type ExportConfig struct {
	MinScore  float64 // Minimum quality score (0.0-1.0), default 0.0
	SourceDir string  // Directory to resolve source files from
}

// ExportTrainingData processes JSONL trace files and produces training examples.
// Reads each .jsonl file in the directory, scores it, and writes qualifying
// examples as JSONL to the writer.
func ExportTrainingData(w io.Writer, traceFiles []string, cfg ExportConfig) (exported, skipped int, err error) {
	enc := json.NewEncoder(w)
	for _, path := range traceFiles {
		events, readErr := readTraceFile(path)
		if readErr != nil {
			skipped++
			continue
		}
		if len(events) == 0 {
			skipped++
			continue
		}

		score := ScoreTrace(events)
		if score.Total < cfg.MinScore {
			skipped++
			continue
		}

		example, buildErr := buildExample(events, score, path, cfg.SourceDir)
		if buildErr != nil {
			skipped++
			continue
		}

		if err := enc.Encode(example); err != nil {
			return exported, skipped, err
		}
		exported++
	}

	return exported, skipped, nil
}

// ScoreTraceFile reads and scores a single JSONL trace file.
func ScoreTraceFile(path string) (TraceScore, error) {
	events, err := readTraceFile(path)
	if err != nil {
		return TraceScore{}, err
	}
	return ScoreTrace(events), nil
}

// readTraceFile reads a JSONL trace file.
func readTraceFile(path string) ([]TraceEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadJSONL(f)
}

// buildExample constructs a training example from events and score.
func buildExample(events []TraceEvent, score TraceScore, tracePath, sourceDir string) (TrainingExample, error) {
	moduleName, caps := TraceMetadata(events)

	// Reconstruct trace JSONL
	var traceBuf strings.Builder
	if err := WriteJSONL(&traceBuf, events); err != nil {
		return TrainingExample{}, err
	}

	// Try to read source code
	source := resolveAndReadSource(moduleName, tracePath, sourceDir)

	// Collect distinct effect names
	var effectNames []string
	for name := range score.EffectBreakdown {
		effectNames = append(effectNames, name)
	}

	return TrainingExample{
		Source: source,
		Trace:  traceBuf.String(),
		Score:  score.Total,
		Metadata: TrainingMeta{
			Module:    moduleName,
			Caps:      caps,
			Functions: score.Stats.DistinctFunctions,
			Effects:   effectNames,
			MaxDepth:  score.Stats.MaxDepth,
			Events:    score.Stats.TotalEvents,
		},
	}, nil
}

// resolveAndReadSource tries to find and read the AILANG source file.
func resolveAndReadSource(moduleName, tracePath, sourceDir string) string {
	if moduleName == "" {
		return ""
	}

	candidates := []string{}

	// Try source dir + module name
	if sourceDir != "" {
		candidates = append(candidates, filepath.Join(sourceDir, moduleName+".ail"))
	}

	// Try relative to trace file
	traceDir := filepath.Dir(tracePath)
	candidates = append(candidates, filepath.Join(traceDir, moduleName+".ail"))

	// Try CWD
	candidates = append(candidates, moduleName+".ail")

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data)
		}
	}

	return ""
}
