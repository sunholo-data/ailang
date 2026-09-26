package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// runPromptFreeze implements `ailang prompt freeze [<version>] [--migrate] [--check] [--repo DIR]`.
// Exit codes: 0 green; 1 violations found (--check); 2 usage/IO error.
func runPromptFreeze(args []string) {
	version := ""
	if len(args) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		version, args = args[0], args[1:]
	}
	fs := flag.NewFlagSet("prompt freeze", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	migrate := fs.Bool("migrate", false, "migrate all existing prompt versions")
	check := fs.Bool("check", false, "check prompt freeze invariants")
	repo := fs.String("repo", "", "repository root")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *repo == "" {
		var err error
		*repo, err = findFreezeRepoRoot()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	if fs.NArg() == 1 {
		if version != "" {
			fmt.Fprintln(os.Stderr, "only one prompt version may be specified")
			os.Exit(2)
		}
		version = fs.Arg(0)
	} else if fs.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "usage: ailang prompt freeze [<version>] [--migrate] [--check] [--repo DIR]")
		os.Exit(2)
	}
	selected := 0
	if version != "" {
		selected++
	}
	if *migrate {
		selected++
	}
	if *check {
		selected++
	}
	if selected != 1 {
		fmt.Fprintln(os.Stderr, "exactly one of <version>, --migrate, or --check is required")
		os.Exit(2)
	}
	today := time.Now().UTC().Format("2006-01-02")
	if *migrate {
		b, l, m, err := migrateRegistries(*repo, today)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Printf("migrated prompt registry: %d banked, %d legacy, %d mutable\n", b, l, m)
		return
	}
	if *check {
		violations, checked, err := checkRegistries(*repo)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		for _, v := range violations {
			fmt.Fprintln(os.Stderr, v)
		}
		fmt.Printf("checked %d registry entries\n", checked)
		if len(violations) > 0 {
			os.Exit(1)
		}
		return
	}
	if err := freezeVersion(*repo, version, today); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func findFreezeRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repository root not found")
		}
		dir = parent
	}
}
