package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
)

// humanDuration supports human-friendly duration parsing including "d" for days.
// Examples: "7d", "30d", "24h", "1h30m", "168h"
type humanDuration time.Duration

func (d *humanDuration) String() string {
	return time.Duration(*d).String()
}

func (d *humanDuration) Set(s string) error {
	// Check for day suffix (e.g., "7d", "30d")
	dayRegex := regexp.MustCompile(`^(\d+)d$`)
	if matches := dayRegex.FindStringSubmatch(s); len(matches) == 2 {
		days, err := strconv.Atoi(matches[1])
		if err != nil {
			return err
		}
		*d = humanDuration(time.Duration(days) * 24 * time.Hour)
		return nil
	}

	// Fall back to standard Go duration parsing
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = humanDuration(parsed)
	return nil
}

// messagesCommand handles the 'messages' (alias: 'msg') subcommand.
// This uses the unified collaboration.db for both CLI and dashboard access.
func messagesCommand() {
	if len(os.Args) < 3 {
		runMessagesList([]string{})
		return
	}

	subCmd := os.Args[2]
	args := os.Args[3:]

	switch subCmd {
	case "list", "ls":
		runMessagesList(args)
	case "search":
		runMessagesSearch(args)
	case "dedupe":
		runMessagesDedupe(args)
	case "ack":
		runMessagesAck(args)
	case "unack":
		runMessagesUnack(args)
	case "send":
		runMessagesSend(args)
	case "read":
		runMessagesRead(args)
	case "watch":
		runMessagesWatch(args)
	case "cleanup":
		runMessagesCleanup(args)
	case "import-github":
		runMessagesImportGitHub(args)
	case "reply":
		runMessagesReply(args)
	case "--help", "-h", "help":
		printMessagesHelp()
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown subcommand '%s'\n", red("Error"), subCmd)
		printMessagesHelp()
		os.Exit(1)
	}
}

// openStore opens the unified collaboration database
func openStore() (*messaging.Store, error) {
	dbPath := messaging.GetDefaultDatabasePath()
	return messaging.OpenStore(dbPath)
}
