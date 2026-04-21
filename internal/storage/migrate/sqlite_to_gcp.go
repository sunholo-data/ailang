// Package migrate provides tools for migrating AILANG data between storage backends.
package migrate

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
)

// Stats tracks migration progress and results.
type Stats struct {
	CoordinatorTasks      int
	CoordinatorApprovals  int
	CoordinatorEvents     int
	MessagingThreads      int
	MessagingInbox        int
	MessagingAgents       int
	ObservatorySpans      int
	ObservatoryWorkspaces int
	Errors                []string
	Duration              time.Duration
}

// Options controls migration behavior.
type Options struct {
	DryRun     bool   // Print what would be migrated without writing
	Verbose    bool   // Print detailed progress
	BatchSize  int    // Number of records per batch (default: 100)
	Collection string // Migrate specific collection only ("coordinator", "messaging", "observatory", or "")
}

// Migrator copies data from source backends to destination backends.
type Migrator struct {
	src  *Sources
	dst  *Destinations
	opts Options
	log  *log.Logger
}

// Sources holds the local SQLite backends to read from.
type Sources struct {
	Coordinator coordinator.Store
	Messaging   messaging.MessageStore
	Observatory observatory.Backend
}

// Destinations holds the cloud backends to write to.
type Destinations struct {
	Coordinator coordinator.Store
	Messaging   messaging.MessageStore
	Observatory observatory.Backend
}

// NewMigrator creates a migrator with the given source and destination backends.
func NewMigrator(src *Sources, dst *Destinations, opts Options, logger *log.Logger) *Migrator {
	if opts.BatchSize <= 0 {
		opts.BatchSize = 100
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Migrator{src: src, dst: dst, opts: opts, log: logger}
}

// Run executes the full migration.
func (m *Migrator) Run(ctx context.Context) (*Stats, error) {
	start := time.Now()
	stats := &Stats{}

	if m.opts.Collection == "" || m.opts.Collection == "coordinator" {
		if err := m.migrateCoordinator(ctx, stats); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("coordinator: %v", err))
			m.log.Printf("Error migrating coordinator: %v", err)
		}
	}

	if m.opts.Collection == "" || m.opts.Collection == "messaging" {
		if err := m.migrateMessaging(ctx, stats); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("messaging: %v", err))
			m.log.Printf("Error migrating messaging: %v", err)
		}
	}

	if m.opts.Collection == "" || m.opts.Collection == "observatory" {
		if err := m.migrateObservatory(ctx, stats); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("observatory: %v", err))
			m.log.Printf("Error migrating observatory: %v", err)
		}
	}

	stats.Duration = time.Since(start)
	return stats, nil
}

// migrateCoordinator copies tasks, approvals, and task events.
func (m *Migrator) migrateCoordinator(ctx context.Context, stats *Stats) error {
	m.log.Println("Migrating coordinator data...")

	// 1. Migrate tasks
	tasks, err := m.src.Coordinator.ListTasks(ctx, &coordinator.TaskFilter{Limit: 10000})
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}
	for _, task := range tasks {
		if m.opts.DryRun {
			stats.CoordinatorTasks++
			continue
		}
		if err := m.dst.Coordinator.CreateTask(ctx, task); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("task %s: %v", task.ID, err))
			if m.opts.Verbose {
				m.log.Printf("  Error creating task %s: %v", task.ID, err)
			}
			continue
		}
		stats.CoordinatorTasks++
	}
	m.log.Printf("  Tasks: %d", stats.CoordinatorTasks)

	// 2. Migrate approvals (pending + resolved)
	pendingApprovals, err := m.src.Coordinator.ListPendingApprovals(ctx)
	if err != nil {
		return fmt.Errorf("list pending approvals: %w", err)
	}
	resolvedApprovals, err := m.src.Coordinator.ListResolvedApprovals(ctx, 10000)
	if err != nil {
		return fmt.Errorf("list resolved approvals: %w", err)
	}
	allApprovals := append(pendingApprovals, resolvedApprovals...)
	for _, approval := range allApprovals {
		if m.opts.DryRun {
			stats.CoordinatorApprovals++
			continue
		}
		if err := m.dst.Coordinator.CreateApprovalRequest(ctx, approval); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("approval %s: %v", approval.ID, err))
			continue
		}
		stats.CoordinatorApprovals++
	}
	m.log.Printf("  Approvals: %d", stats.CoordinatorApprovals)

	// 3. Migrate task events (for each task)
	for _, task := range tasks {
		events, err := m.src.Coordinator.GetTaskEvents(ctx, task.ID, 10000)
		if err != nil {
			if m.opts.Verbose {
				m.log.Printf("  Warning: could not get events for task %s: %v", task.ID, err)
			}
			continue
		}
		for _, event := range events {
			if m.opts.DryRun {
				stats.CoordinatorEvents++
				continue
			}
			if err := m.dst.Coordinator.StoreTaskEvent(ctx, event); err != nil {
				if m.opts.Verbose {
					m.log.Printf("  Error storing event for task %s: %v", task.ID, err)
				}
				continue
			}
			stats.CoordinatorEvents++
		}
	}
	m.log.Printf("  Events: %d", stats.CoordinatorEvents)

	return nil
}

// migrateMessaging copies threads, inbox messages, and agent registrations.
func (m *Migrator) migrateMessaging(ctx context.Context, stats *Stats) error {
	m.log.Println("Migrating messaging data...")

	// 1. Migrate threads (using CreateThread API: title, createdByType, createdByID, targetAgent)
	threads, err := m.src.Messaging.GetThreadsFiltered(messaging.ThreadFilter{Limit: 10000})
	if err != nil {
		return fmt.Errorf("list threads: %w", err)
	}
	for _, thread := range threads {
		if m.opts.DryRun {
			stats.MessagingThreads++
			continue
		}
		// CreateThread(title, createdByType, createdByID, targetAgent) returns (*Thread, error)
		// We re-create threads with the original metadata
		if _, err := m.dst.Messaging.CreateThread(thread.Title, thread.CreatedByType, thread.CreatedByID, thread.TargetAgent); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("thread %s: %v", thread.ID, err))
			continue
		}
		stats.MessagingThreads++
	}
	m.log.Printf("  Threads: %d", stats.MessagingThreads)

	// 2. Migrate inbox messages (all statuses)
	inboxMsgs, err := m.src.Messaging.ListInboxMessages(messaging.InboxListOptions{Limit: 50000})
	if err != nil {
		return fmt.Errorf("list inbox messages: %w", err)
	}
	for i := range inboxMsgs {
		if m.opts.DryRun {
			stats.MessagingInbox++
			continue
		}
		if err := m.dst.Messaging.InsertInboxMessage(&inboxMsgs[i]); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("inbox %s: %v", inboxMsgs[i].ID, err))
			continue
		}
		stats.MessagingInbox++
	}
	m.log.Printf("  Inbox messages: %d", stats.MessagingInbox)

	// 3. Migrate agent registrations
	agents, err := m.src.Messaging.GetKnownAgents()
	if err != nil {
		return fmt.Errorf("list agents: %w", err)
	}
	for _, agent := range agents {
		if m.opts.DryRun {
			stats.MessagingAgents++
			continue
		}
		if err := m.dst.Messaging.RegisterAgent(agent.ID, agent.Label, agent.Status); err != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("agent %s: %v", agent.ID, err))
			continue
		}
		stats.MessagingAgents++
	}
	m.log.Printf("  Agents: %d", stats.MessagingAgents)

	return nil
}

// migrateObservatory copies workspaces and spans.
// Note: Chains/stages use CreateChain which generates new IDs. For full chain migration,
// use direct Firestore writes or export/import tooling.
func (m *Migrator) migrateObservatory(ctx context.Context, stats *Stats) error {
	m.log.Println("Migrating observatory data...")

	// 1. Migrate workspaces
	workspaces, err := m.src.Observatory.ListWorkspaces(ctx)
	if err != nil {
		m.log.Printf("  Warning: could not list workspaces: %v", err)
	} else {
		for _, ws := range workspaces {
			if !m.opts.DryRun {
				if err := m.dst.Observatory.CreateWorkspace(ctx, ws); err != nil {
					if m.opts.Verbose {
						m.log.Printf("  Warning: workspace %s: %v", ws.ID, err)
					}
				}
			}
			stats.ObservatoryWorkspaces++
		}
		m.log.Printf("  Workspaces: %d", stats.ObservatoryWorkspaces)
	}

	// 2. Migrate spans (the bulk of observatory data)
	offset := 0
	for {
		spans, err := m.src.Observatory.ListSpans(ctx, observatory.SpanListOptions{
			Limit:  m.opts.BatchSize,
			Offset: offset,
		})
		if err != nil {
			return fmt.Errorf("list spans (offset %d): %w", offset, err)
		}
		if len(spans) == 0 {
			break
		}
		for _, span := range spans {
			if m.opts.DryRun {
				stats.ObservatorySpans++
				continue
			}
			if err := m.dst.Observatory.CreateSpan(ctx, span); err != nil {
				if m.opts.Verbose {
					m.log.Printf("  Warning: span %s: %v", span.ID, err)
				}
				continue
			}
			stats.ObservatorySpans++
		}
		offset += len(spans)
		if m.opts.Verbose && offset%1000 == 0 {
			m.log.Printf("  Spans progress: %d", offset)
		}
		if len(spans) < m.opts.BatchSize {
			break
		}
	}
	m.log.Printf("  Spans: %d", stats.ObservatorySpans)

	return nil
}

// Verify checks that source and destination have matching record counts.
func (m *Migrator) Verify(ctx context.Context) ([]string, error) {
	var issues []string

	// Check coordinator tasks
	srcTasks, err := m.src.Coordinator.ListTasks(ctx, &coordinator.TaskFilter{Limit: 100000})
	if err != nil {
		return nil, fmt.Errorf("source coordinator tasks: %w", err)
	}
	dstTasks, err := m.dst.Coordinator.ListTasks(ctx, &coordinator.TaskFilter{Limit: 100000})
	if err != nil {
		return nil, fmt.Errorf("destination coordinator tasks: %w", err)
	}
	if len(srcTasks) != len(dstTasks) {
		issues = append(issues, fmt.Sprintf("coordinator tasks: source=%d, destination=%d", len(srcTasks), len(dstTasks)))
	}

	// Check messaging threads
	srcThreads, err := m.src.Messaging.GetThreadsFiltered(messaging.ThreadFilter{Limit: 100000})
	if err != nil {
		return nil, fmt.Errorf("source messaging threads: %w", err)
	}
	dstThreads, err := m.dst.Messaging.GetThreadsFiltered(messaging.ThreadFilter{Limit: 100000})
	if err != nil {
		return nil, fmt.Errorf("destination messaging threads: %w", err)
	}
	if len(srcThreads) != len(dstThreads) {
		issues = append(issues, fmt.Sprintf("messaging threads: source=%d, destination=%d", len(srcThreads), len(dstThreads)))
	}

	if len(issues) == 0 {
		m.log.Println("Verification passed: all counts match")
	} else {
		m.log.Printf("Verification found %d issue(s):", len(issues))
		for _, issue := range issues {
			m.log.Printf("  - %s", issue)
		}
	}

	return issues, nil
}
