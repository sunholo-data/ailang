package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/pkg"
)

func pkgSearchCommand(args []string) error {
	flagSet := flag.NewFlagSet("search", flag.ExitOnError)
	tagFlag := flagSet.String("tag", "", "Filter by tag")
	helpFlag := flagSet.Bool("help", false, "Show help")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *helpFlag {
		fmt.Println("Usage: ailang search [query] [--tag TAG]")
		fmt.Println()
		fmt.Println("Search the AILANG package registry.")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --tag TAG    Filter by tag (e.g., gcp, auth, logging)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang search auth")
		fmt.Println("  ailang search --tag gcp")
		fmt.Println("  ailang search                    # list all packages")
		return nil
	}

	query := ""
	if flagSet.NArg() > 0 {
		query = flagSet.Arg(0)
	}

	client := pkg.NewRegistryClient()

	results, err := client.SearchPackages(query, *tagFlag)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		if query != "" {
			fmt.Printf("No packages found for %q\n", query)
		} else {
			fmt.Println("No packages in registry")
		}
		return nil
	}

	for _, p := range results {
		effects := "Pure"
		if len(p.Effects) > 0 {
			effects = strings.Join(p.Effects, ", ")
		}
		fmt.Printf("%s@%s — %s [%s]\n", cyan(p.Name), p.Latest, p.AISummary, effects)
	}

	fmt.Printf("\n%d package(s) found. Install with: %s\n", len(results), bold("ailang install vendor/name@version"))

	return nil
}
