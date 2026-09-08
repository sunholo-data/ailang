package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/mission/comms"
)

// `ailang mission` — the mission↔human GitHub channel, moved off the shell.
//
// M3 of M-MISSION-COMMS-P1. Only `report` ships in this sprint; `decisions` and
// `telemetry` wait on HD-1 and HD-2 being ratified, and the shell cutover is a
// separate sprint with its own window because three live loops run pinned driver
// copies that re-sync from origin/dev at every fire.

// missionPoster is the seam the dry-run test asserts against: a --dry-run run must
// never construct a real client, so the test injects a poster that fails if called.
type missionPoster interface {
	AddComment(repo string, number int, body string) error
}

// newMissionPoster is a variable so tests can replace it. Production builds the
// real GitHubClient, whose gh shell-outs are bounded as of M1.
var newMissionPoster = func() (missionPoster, error) {
	cfg, err := messaging.LoadGitHubConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load GitHub config: %w", err)
	}
	return messaging.NewGitHubClient(cfg), nil
}

func runMissionCommand(args []string) {
	if len(args) == 0 {
		missionUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "report":
		os.Exit(runMissionReport(args[1:]))
	case "-h", "--help", "help":
		missionUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "ailang mission: unknown subcommand %q\n\n", args[0])
		missionUsage()
		os.Exit(1)
	}
}

func missionUsage() {
	fmt.Fprint(os.Stderr, `ailang mission — mission↔human comms

USAGE:
  ailang mission report --mission <name> --body-file <path> [--dry-run]

SUBCOMMANDS:
  report    Post one iteration report to the mission's bookkeeping issue.
            The body is capped at 400 characters and truncation is marked:
            the report is a POINTER to the mission log, not a second copy of it.

FLAGS (report):
  --mission <name>     v1 | world | docs | motoko  (required)
  --body-file <path>   file holding the report body ("-" for stdin, required)
  --dry-run            print what would be posted and exit 0, making NO network
                       call. This is how a cutover is rehearsed safely.

NOTES:
  The bookkeeping issue is read from $AILANG_STATE_DIR/mission-<name>-gh-issue
  (the threads rotate weekly, so it is never hardcoded). MISSION_GH_ISSUE
  overrides it, which is how you point a rehearsal at a scratch issue.
`)
}

func runMissionReport(args []string) int {
	fs := flag.NewFlagSet("mission report", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	mission := fs.String("mission", "", "mission name (v1|world|docs|motoko)")
	bodyFile := fs.String("body-file", "", `file holding the report body ("-" for stdin)`)
	dryRun := fs.Bool("dry-run", false, "print what would be posted; make no network call")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *mission == "" {
		fmt.Fprintln(os.Stderr, "ailang mission report: --mission is required")
		return 2
	}
	if *bodyFile == "" {
		fmt.Fprintln(os.Stderr, "ailang mission report: --body-file is required")
		return 2
	}

	// Resolve identity BEFORE reading the body: a bad mission name should fail
	// immediately and by name, not after side effects.
	m, err := comms.ResolveMission(*mission)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ailang mission report: %v\n", err)
		return 1
	}

	var raw []byte
	if *bodyFile == "-" {
		raw, err = readAllStdin()
	} else {
		raw, err = os.ReadFile(*bodyFile) //nolint:gosec // operator-supplied path
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ailang mission report: cannot read body: %v\n", err)
		return 1
	}

	body := strings.TrimRight(string(raw), "\n")
	if strings.TrimSpace(body) == "" {
		fmt.Fprintln(os.Stderr, "ailang mission report: body is empty — refusing to post an empty comment")
		return 1
	}

	if *dryRun {
		// No client is constructed at all. Building one here would make the
		// rehearsal depend on gh auth, and the whole point is that it doesn't.
		fmt.Printf("DRY RUN — would post to %s#%d (%d chars):\n%s\n", m.Repo, m.Issue, len(body), body)
		return 0
	}

	poster, err := newMissionPoster()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ailang mission report: %v\n", err)
		return 1
	}
	if err := poster.AddComment(m.Repo, m.Issue, body); err != nil {
		// Loud, and carrying M1's typed error when the cause was a deadline, so a
		// caller can tell "GitHub is slow" from "gh is broken".
		fmt.Fprintf(os.Stderr, "ailang mission report: failed to post to %s#%d: %v\n", m.Repo, m.Issue, err)
		return 1
	}
	fmt.Printf("✓ posted to %s#%d (%d chars)\n", m.Repo, m.Issue, len(body))
	return 0
}

func readAllStdin() ([]byte, error) {
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return []byte(sb.String()), nil
		}
	}
	return []byte(sb.String()), nil
}
