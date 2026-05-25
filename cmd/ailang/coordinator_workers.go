package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// coordinatorWorkers dispatches `ailang coordinator workers <subcommand>`
// (M-COORD-MULTI-HOST-WORKERS, v0.24.0).
func coordinatorWorkers(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printWorkersHelp()
		return nil
	}
	sub := args[0]
	subargs := args[1:]
	switch sub {
	case "list":
		return workersList(subargs)
	case "ping":
		return workersPing(subargs)
	default:
		return fmt.Errorf("unknown coordinator workers subcommand: %s", sub)
	}
}

func printWorkersHelp() {
	fmt.Println("Usage: ailang coordinator workers <subcommand> [options]")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  list                  Show all worker hosts (bare-metal + Cloud Run history)")
	fmt.Println("  ping <host_id>        Round-trip probe a live worker via system:heartbeat tag")
	fmt.Println("")
	fmt.Println("List flags:")
	fmt.Println("  --type bare-metal|cloud-run  Filter by worker type")
	fmt.Println("  --since 7d                   Time window for Cloud Run history (default 7d)")
	fmt.Println("  --max-age 5m                 Bare-metal staleness cutoff (default 5m)")
	fmt.Println("  --json                       Output as JSON")
	fmt.Println("  --state-dir DIR              Override state directory (default ~/.ailang/state)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang coordinator workers list")
	fmt.Println("  ailang coordinator workers list --type bare-metal --json")
	fmt.Println("  ailang coordinator workers list --type cloud-run --since 24h")
	fmt.Println("  ailang coordinator workers ping studio.eval-rig")
	fmt.Println("")
	fmt.Println("Note: bare-metal hosts come from the worker_heartbeats backend (Firestore in")
	fmt.Println("cloud mode, in-memory otherwise — see SetHeartbeatStore wiring in daemon init).")
	fmt.Println("Cloud Run history comes from the existing coordinator task store.")
}

// workersList implements `ailang coordinator workers list`.
func workersList(args []string) error {
	jsonOutput := false
	stateDir := ""
	typeFilter := ""
	maxAge := 5 * time.Minute
	sinceDur := 7 * 24 * time.Hour

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--state-dir":
			if i+1 < len(args) {
				stateDir = args[i+1]
				i++
			}
		case "--type":
			if i+1 < len(args) {
				typeFilter = args[i+1]
				i++
			}
		case "--max-age":
			if i+1 < len(args) {
				d, err := time.ParseDuration(args[i+1])
				if err == nil {
					maxAge = d
				}
				i++
			}
		case "--since":
			if i+1 < len(args) {
				d, err := time.ParseDuration(args[i+1])
				if err == nil {
					sinceDur = d
				}
				i++
			}
		case "--help", "-h":
			printWorkersHelp()
			return nil
		}
	}

	rows := []workerRow{}

	// Bare-metal hosts: from the heartbeat store. The CLI doesn't have direct
	// access to whatever HeartbeatStore the running daemon uses. For v0.24.0
	// we expose the in-memory store; the live data lives across the wire and
	// will be picked up once the Firestore-backed implementation is wired
	// (see M-COORD-MULTI-HOST-WORKERS Future Work). When no live data is
	// available, the bare-metal column simply doesn't return rows.
	if typeFilter == "" || typeFilter == "bare-metal" {
		bm, err := loadBareMetalRows(maxAge)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: bare-metal worker query failed: %v\n", err)
		} else {
			rows = append(rows, bm...)
		}
	}

	// Cloud Run history: from the existing coordinator task store.
	if typeFilter == "" || typeFilter == "cloud-run" {
		cr, err := loadCloudRunRows(stateDir, sinceDur)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: cloud-run history query failed: %v\n", err)
		} else {
			rows = append(rows, cr...)
		}
	}

	if jsonOutput {
		return printWorkerRowsJSON(rows)
	}
	return printWorkerRowsTable(rows)
}

// workerRow is the unified view across bare-metal hosts + Cloud Run history.
type workerRow struct {
	HostID       string    `json:"host_id"`
	Type         string    `json:"type"`
	Tags         []string  `json:"tags,omitempty"`
	LastSeen     time.Time `json:"last_seen"`
	ActiveTasks  int       `json:"active_tasks"`
	TotalTasks7d int       `json:"total_tasks_7d,omitempty"`
	Version      string    `json:"version,omitempty"`
	UptimeSecs   int64     `json:"uptime_secs,omitempty"`
	Alive        bool      `json:"alive"`
	Note         string    `json:"note,omitempty"`
}

func loadBareMetalRows(maxAge time.Duration) ([]workerRow, error) {
	// M-COORD-MULTI-HOST-WORKERS (v0.24.0): read from the on-host JSON file
	// the daemon writes (~/.ailang/state/worker_heartbeats.json by default).
	// This gives same-host cross-process visibility. Cross-host visibility
	// will land via FirestoreHeartbeatStore in v0.25, using the same interface.
	store := coordinator.NewFileHeartbeatStore(coordinator.DefaultHeartbeatPath(""))
	hbs, err := store.List(context.Background(), maxAge)
	if err != nil {
		return nil, err
	}
	rows := make([]workerRow, 0, len(hbs))
	now := time.Now()
	for _, hb := range hbs {
		rows = append(rows, workerRow{
			HostID:      hb.HostID,
			Type:        defaultStr(hb.Type, "bare-metal"),
			Tags:        hb.Tags,
			LastSeen:    hb.LastSeen,
			ActiveTasks: hb.ActiveTasks,
			Version:     hb.Version,
			UptimeSecs:  hb.UptimeSecs,
			Alive:       now.Sub(hb.LastSeen) <= maxAge,
		})
	}
	return rows, nil
}

func loadCloudRunRows(stateDir string, since time.Duration) ([]workerRow, error) {
	cfg := coordinator.DefaultConfig()
	if stateDir != "" {
		cfg.StateDir = stateDir
	}
	dbPath := filepath.Join(cfg.StateDir, "coordinator.db")
	store, err := coordinator.NewSQLiteStore(dbPath)
	if err != nil {
		return nil, err
	}
	defer store.Close()

	cutoff := time.Now().Add(-since)
	filter := &coordinator.TaskFilter{
		Since: &cutoff,
		Limit: 500,
	}
	tasks, err := store.ListTasks(context.Background(), filter)
	if err != nil {
		return nil, err
	}

	// Aggregate by AgentID — that's the closest existing "worker" notion for
	// coordinator tasks. Each unique agent becomes one row marked cloud-run.
	type agg struct {
		host   string
		agent  string
		count  int
		latest time.Time
	}
	groups := map[string]*agg{}
	for _, task := range tasks {
		host := task.AgentID
		if host == "" {
			host = "cloud-run.unknown"
		} else if !strings.HasPrefix(host, "cloud-run") {
			host = "cloud-run." + host
		}
		g, ok := groups[host]
		if !ok {
			g = &agg{host: host, agent: task.AgentID}
			groups[host] = g
		}
		g.count++
		if task.CreatedAt.After(g.latest) {
			g.latest = task.CreatedAt
		}
	}

	rows := make([]workerRow, 0, len(groups))
	for _, g := range groups {
		rows = append(rows, workerRow{
			HostID:       g.host,
			Type:         "cloud-run",
			LastSeen:     g.latest,
			TotalTasks7d: g.count,
			Alive:        true,
			Note:         "ephemeral — per-task; row aggregates recent history",
		})
	}
	return rows, nil
}

func printWorkerRowsTable(rows []workerRow) error {
	if len(rows) == 0 {
		fmt.Println("No workers found.")
		fmt.Println("")
		fmt.Println("Bare-metal hosts appear once a coordinator with a HeartbeatStore is running.")
		fmt.Println("Cloud Run rows aggregate the coordinator task store; ensure tasks exist in")
		fmt.Println("the configured state directory.")
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "HOST\tTYPE\tTAGS\tLAST SEEN\tTASKS (7d)\tALIVE")
	for _, r := range rows {
		tags := strings.Join(r.Tags, ",")
		if tags == "" {
			tags = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%v\n",
			r.HostID,
			r.Type,
			tags,
			formatAge(r.LastSeen),
			r.TotalTasks7d,
			r.Alive,
		)
	}
	return tw.Flush()
}

func printWorkerRowsJSON(rows []workerRow) error {
	if rows == nil {
		rows = []workerRow{}
	}
	out, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

// workersPing is the round-trip probe — placeholder for now since it requires
// the full Pub/Sub round-trip with a system:heartbeat tag. Returns a clear
// "not yet implemented" message so the surface area is present.
func workersPing(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ailang coordinator workers ping <host_id>")
	}
	host := args[0]
	// Resolve via the on-host heartbeat file so users get a useful error if
	// the host doesn't exist (or hasn't heartbeated recently).
	store := coordinator.NewFileHeartbeatStore(coordinator.DefaultHeartbeatPath(""))
	hbs, err := store.List(context.Background(), 5*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to query heartbeat store: %w", err)
	}
	found := false
	for _, hb := range hbs {
		if hb.HostID == host {
			found = true
			break
		}
	}
	if !found {
		fmt.Fprintf(os.Stderr, "no live worker with host_id=%s\n", host)
		fmt.Fprintln(os.Stderr, "Tip: run `ailang coordinator workers list` to see available hosts.")
		os.Exit(1)
	}
	fmt.Printf("%s: probe via system:heartbeat tag is not yet wired (M-COORD-MULTI-HOST-WORKERS Future Work).\n", host)
	fmt.Println("The host IS visible in the heartbeat store and was last seen recently.")
	return nil
}

func defaultStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
