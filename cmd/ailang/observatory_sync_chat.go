package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/claudehistory"
	"github.com/sunholo-data/ailang/internal/observatory"
)

func observatorySyncChatCommand() {
	fs := flag.NewFlagSet("observatory sync-chat", flag.ExitOnError)
	sessionID := fs.String("session", "", "Sync a specific session ID only")
	showStatus := fs.Bool("status", false, "Show import status without syncing")
	force := fs.Bool("force", false, "Force re-import even if file unchanged")
	verbose := fs.Bool("verbose", false, "Show detailed progress")

	// Skip "ailang observatory sync-chat" args
	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Open observatory database
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening observatory database: %v\n", err)
		fmt.Fprintf(os.Stderr, "Database path: %s\n", dbPath)
		os.Exit(1)
	}
	defer backend.Close()

	// Create importer using the backend's DB connection
	importer := claudehistory.NewImporter(backend.DB())

	// Show status mode
	if *showStatus {
		showImportStatus(ctx, importer, *sessionID)
		return
	}

	// Sync mode
	if *sessionID != "" {
		// Sync specific session
		syncSingleSession(ctx, importer, *sessionID, *force, *verbose)
	} else {
		// Sync all sessions
		syncAllSessions(ctx, importer, *force, *verbose)
	}
}

func showImportStatus(ctx context.Context, importer *claudehistory.Importer, sessionID string) {
	if sessionID != "" {
		// Show status for specific session
		status, err := importer.GetImportStatus(ctx, sessionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting import status: %v\n", err)
			os.Exit(1)
		}
		if status == nil {
			fmt.Printf("Session %s has not been imported yet\n", sessionID)
			return
		}
		fmt.Printf("Session: %s\n", status.SessionID)
		fmt.Printf("  File: %s\n", status.FilePath)
		fmt.Printf("  File mtime: %s\n", status.FileMtime.Format(time.RFC3339))
		fmt.Printf("  Messages: %d\n", status.MessageCount)
		fmt.Printf("  Imported: %s\n", status.ImportedAt.Format(time.RFC3339))
		return
	}

	// Show all import statuses
	statuses, err := importer.GetAllImportStatus(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting import statuses: %v\n", err)
		os.Exit(1)
	}

	if len(statuses) == 0 {
		fmt.Println("No sessions have been imported yet")
		fmt.Println("\nRun 'ailang observatory sync-chat' to import chat history")
		return
	}

	fmt.Printf("Imported sessions: %d\n\n", len(statuses))
	fmt.Printf("%-40s  %8s  %s\n", "SESSION ID", "MESSAGES", "IMPORTED AT")
	fmt.Printf("%-40s  %8s  %s\n", "----------", "--------", "-----------")
	for _, s := range statuses {
		fmt.Printf("%-40s  %8d  %s\n",
			truncateString(s.SessionID, 40),
			s.MessageCount,
			s.ImportedAt.Format("2006-01-02 15:04:05"))
	}
}

func syncSingleSession(ctx context.Context, importer *claudehistory.Importer, sessionID string, force bool, verbose bool) {
	if verbose {
		fmt.Printf("Syncing session: %s\n", sessionID)
	}

	start := time.Now()
	msgCount, err := importer.SyncSession(ctx, sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error syncing session: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Synced session %s: %d messages (%.2fs)\n",
		sessionID, msgCount, time.Since(start).Seconds())
}

func syncAllSessions(ctx context.Context, importer *claudehistory.Importer, force bool, verbose bool) {
	fmt.Println("Scanning Claude Code conversation history...")

	start := time.Now()
	stats, err := importer.SyncAll(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during sync: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Println()
	fmt.Printf("Sync completed in %.2fs\n", time.Since(start).Seconds())
	fmt.Println()
	fmt.Printf("  Projects scanned:   %d\n", stats.ProjectsScanned)
	fmt.Printf("  Sessions scanned:   %d\n", stats.SessionsScanned)
	fmt.Printf("  Sessions imported:  %d\n", stats.SessionsImported)
	fmt.Printf("  Sessions skipped:   %d (unchanged)\n", stats.SessionsSkipped)
	fmt.Printf("  Messages imported:  %d\n", stats.MessagesImported)

	if len(stats.Errors) > 0 {
		fmt.Println()
		fmt.Printf("Errors: %d\n", len(stats.Errors))
		if verbose {
			for _, e := range stats.Errors {
				fmt.Printf("  - %s\n", e)
			}
		} else {
			fmt.Println("  (use --verbose to see details)")
		}
	}

	// Print summary
	fmt.Println()
	if stats.SessionsImported > 0 {
		fmt.Printf("✓ Chat history synced to observatory database\n")
		fmt.Printf("  View with: ailang observatory hierarchy --chat <session-id>\n")
	} else if stats.SessionsSkipped > 0 {
		fmt.Printf("✓ All sessions up to date (no changes detected)\n")
	} else {
		fmt.Printf("No Claude Code conversation history found\n")
		fmt.Printf("  Expected location: ~/.claude/projects/\n")
	}
}
