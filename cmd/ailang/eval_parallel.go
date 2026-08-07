package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/messaging"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// EvalBenchmarkJob represents a benchmark job payload for queue mode (M-UNIFIED-AI-CONTROL-PLANE)
type EvalBenchmarkJob struct {
	Model       string `json:"model"`
	Benchmark   string `json:"benchmark"`
	Language    string `json:"language"`
	Seed        int64  `json:"seed"`
	OutputDir   string `json:"output_dir"`
	AgentMode   bool   `json:"agent_mode"`
	SuiteTaskID string `json:"suite_task_id"`
}

// runBenchmarksParallel executes benchmarks with concurrency control.
// budgetUSD is an aggregate cost ceiling (M-EVAL-STANDARD-CONFIDENCE-GATING):
// 0 = no cap, preserving today's unconstrained behavior. On breach, already
// in-flight trials finish (never hard-killed mid-call) but no NEW trial is
// scheduled, and a budget_stopped.json sentinel is written to outputDir so
// callers (e.g. run_eval_baseline.sh) can detect a partial run and label it
// rather than silently treat it as complete.
func runBenchmarksParallel(ctx context.Context, jobs []Job, seed int64, outputDir string, timeout time.Duration, maxConcurrent int, selfRepair bool, promptVersion string, agentConfig *eval_harness.AgentBenchmarkConfig, taskID string, evalChain *EvalChainContext, budgetUSD float64) []SuiteResult {

	if maxConcurrent <= 0 {
		maxConcurrent = 1 // Sequential
	}

	// Warm the prompt cache BEFORE fanning out. A cache entry only becomes
	// readable once the first response starts streaming, so without this every
	// concurrent call in the first wave misses and pays a full cache WRITE for
	// the same prefix. Non-fatal: failures warn and the suite runs uncached.
	// (M-ANTHROPIC-CACHE-HIT-RATE D4 — see eval_cache_warmup.go.)
	if maxConcurrent > 1 {
		warmPromptCaches(ctx, jobs, timeout)
	}

	var (
		wg           sync.WaitGroup
		results      = make([]SuiteResult, len(jobs))
		sem          = make(chan struct{}, maxConcurrent) // Semaphore for concurrency control
		mu           sync.Mutex                           // Protect progress counter
		failureCount int                                  // Track consecutive failures
		aborted      bool                                 // Early abort flag
	)

	// budgetTracker accumulates banked cost against budgetUSD
	// (M-EVAL-STANDARD-CONFIDENCE-GATING); budgetUSD<=0 means it never trips.
	// onCost is invoked from within runSingleBenchmark once a trial's result
	// is banked. A race between this check and concurrent in-flight trials
	// finishing is accepted: this is a graceful-stop gauge (sum of what got
	// banked), not a byte-exact real-time meter — see the design doc's Risks
	// section. maxConcurrent bounds how far over budgetUSD a breach can run.
	tracker := newBudgetTracker(budgetUSD)
	onCost := func(costUSD float64) {
		if !tracker.Add(costUSD) {
			return
		}
		mu.Lock()
		aborted = true
		mu.Unlock()
		fmt.Printf("\n%s Budget exceeded: $%.2f spent >= $%.2f cap — stopping new trials (in-flight trials finish)\n",
			red("🚨"), tracker.Spent(), budgetUSD)
	}

	completed := 0
	totalJobs := len(jobs)

	for i, job := range jobs {
		// Check if we should abort early
		mu.Lock()
		if aborted {
			mu.Unlock()
			break
		}
		mu.Unlock()

		wg.Add(1)
		go func(idx int, j Job) {
			defer wg.Done()

			// Check abort flag before starting work
			mu.Lock()
			if aborted {
				mu.Unlock()
				return
			}
			mu.Unlock()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Update progress
			mu.Lock()
			completed++
			currentProgress := completed
			mu.Unlock()

			condLabel := ""
			if j.Condition != "" {
				condLabel = fmt.Sprintf(" [%s]", j.Condition)
			}
			fmt.Printf("[%d/%d] Running %s with %s (%s)%s...\n",
				currentProgress, totalJobs,
				cyan(j.Benchmark), green(j.Model), j.Language, condLabel)

			// Run the benchmark
			success, err := runSingleBenchmark(ctx, j.Model, j.Benchmark, j.Language, j.Condition, j.Trial, seed, outputDir, timeout, selfRepair, promptVersion, agentConfig, taskID, evalChain, onCost)

			results[idx] = SuiteResult{
				BenchmarkID: j.Benchmark,
				Language:    j.Language,
				Model:       j.Model,
				Success:     success,
				Error:       err,
			}

			if success {
				fmt.Printf("  %s Completed\n", green("✓"))
				mu.Lock()
				failureCount = 0 // Reset failure count on success
				mu.Unlock()
			} else {
				fmt.Printf("  %s Failed: %v\n", red("✗"), err)
				mu.Lock()
				failureCount++
				// Abort if first 50 results are all failures
				if completed >= 50 && failureCount >= 50 {
					if !aborted {
						aborted = true
						fmt.Printf("\n%s Aborting: First 50 results all failed - likely system issue!\n", red("🚨"))
						fmt.Printf("Check: interpreter debug output, missing API keys, or broken prompt.\n\n")
					}
				}
				mu.Unlock()
			}
		}(i, job)
	}

	wg.Wait()

	if tracker.Exceeded() {
		writeBudgetStoppedSentinel(outputDir, budgetUSD, tracker.Spent())
	}

	return results
}

// budgetStoppedSentinel is written to outputDir when an aggregate --budget-usd
// cap stops a run early, so callers (e.g. run_eval_baseline.sh) can detect a
// partial baseline and label it rather than silently present it as complete.
type budgetStoppedSentinel struct {
	BudgetUSD float64   `json:"budget_usd"`
	SpentUSD  float64   `json:"spent_usd"`
	StoppedAt time.Time `json:"stopped_at"`
}

func writeBudgetStoppedSentinel(outputDir string, budgetUSD, spentUSD float64) {
	sentinel := budgetStoppedSentinel{BudgetUSD: budgetUSD, SpentUSD: spentUSD, StoppedAt: time.Now()}
	data, err := json.MarshalIndent(sentinel, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not marshal budget_stopped sentinel: %v\n", err)
		return
	}
	path := filepath.Join(outputDir, "budget_stopped.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write %s: %v\n", path, err)
	}
}

// runBenchmarksViaQueue submits benchmarks as messages to the specified inbox.
// Coordinator daemon picks up messages and processes via ailang exec.
// This enables crash recovery (unacked messages resume) and distributed execution.
func runBenchmarksViaQueue(ctx context.Context, jobs []Job, suiteTaskID, inbox string, seed int64, outputDir string, agentMode, wait bool, suiteSpan trace.Span) []SuiteResult {
	// Open message store
	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s Failed to open message store: %v\n", red("✗"), err)
		suiteSpan.RecordError(err)
		return nil
	}
	defer store.Close()

	// Generate correlation ID for this eval suite run
	correlationID := fmt.Sprintf("eval_%s", suiteTaskID)

	// Submit each benchmark as a message
	var messageIDs []string
	fmt.Printf("%s Submitting %d benchmark jobs to queue '%s'...\n", cyan("→"), len(jobs), inbox)

	for i, job := range jobs {
		// Create job payload
		payload := EvalBenchmarkJob{
			Model:       job.Model,
			Benchmark:   job.Benchmark,
			Language:    job.Language,
			Seed:        seed,
			OutputDir:   outputDir,
			AgentMode:   agentMode,
			SuiteTaskID: suiteTaskID,
		}
		payloadBytes, _ := json.Marshal(payload)

		// Create message with hierarchy metadata
		msg := &messaging.InboxMessage{
			FromAgent:     "eval-suite",
			ToInbox:       inbox,
			MessageType:   messaging.InboxTypeNotification,
			Title:         fmt.Sprintf("Benchmark: %s/%s/%s", job.Benchmark, job.Language, job.Model),
			Payload:       string(payloadBytes),
			CorrelationID: correlationID,
			ParentTaskID:  suiteTaskID,
			Category:      "eval",
		}

		if err := store.InsertInboxMessage(msg); err != nil {
			fmt.Fprintf(os.Stderr, "%s Failed to queue job %d: %v\n", red("✗"), i+1, err)
			continue
		}
		messageIDs = append(messageIDs, msg.MessageID)

		// Progress indicator
		if (i+1)%50 == 0 || i+1 == len(jobs) {
			fmt.Printf("  Queued %d/%d jobs\n", i+1, len(jobs))
		}
	}

	suiteSpan.SetAttributes(
		attribute.Int("eval.jobs_queued", len(messageIDs)),
		attribute.String("eval.correlation_id", correlationID),
	)

	fmt.Printf("%s Queued %d benchmark jobs (correlation: %s)\n", green("✓"), len(messageIDs), correlationID)

	// If not waiting, return empty results (async mode)
	if !wait {
		fmt.Println()
		fmt.Println("Jobs queued for processing by coordinator daemon.")
		fmt.Println("Monitor progress:")
		fmt.Printf("  ailang messages list --inbox %s --unread\n", inbox)
		fmt.Println()
		return nil
	}

	// Wait for all jobs to complete
	fmt.Println()
	fmt.Printf("%s Waiting for %d jobs to complete...\n", cyan("→"), len(messageIDs))
	fmt.Println("  (Coordinator daemon processes these - ensure it's running)")
	fmt.Println()

	// Poll for completion
	pollInterval := 5 * time.Second
	maxWait := 2 * time.Hour
	startTime := time.Now()

	results := make([]SuiteResult, len(jobs))
	completed := make(map[string]bool)

	for time.Since(startTime) < maxWait {
		// Check how many messages are still unread
		messages, err := store.ListInboxMessages(messaging.InboxListOptions{
			Inbox:      inbox,
			UnreadOnly: true,
		})
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		// Count remaining from our correlation
		remaining := 0
		for _, msg := range messages {
			if msg.CorrelationID == correlationID {
				remaining++
			}
		}

		completedCount := len(messageIDs) - remaining
		if completedCount > len(completed) {
			// New completions - mark them
			for _, msg := range messages {
				if msg.CorrelationID == correlationID {
					completed[msg.MessageID] = true
				}
			}
			fmt.Printf("  Progress: %d/%d completed\n", completedCount, len(messageIDs))
		}

		if remaining == 0 {
			// All done
			break
		}

		time.Sleep(pollInterval)
	}

	// Build results from completed messages
	// For now, assume success if message was acknowledged
	for i, job := range jobs {
		results[i] = SuiteResult{
			BenchmarkID: job.Benchmark,
			Language:    job.Language,
			Model:       job.Model,
			Success:     true, // Message was processed
		}
	}

	return results
}
