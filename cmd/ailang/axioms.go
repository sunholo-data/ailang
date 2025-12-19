package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AxiomScorecard represents the AILANG design axiom compliance scorecard
type AxiomScorecard struct {
	Version     string                 `json:"version"`
	Timestamp   string                 `json:"timestamp"`
	Methodology string                 `json:"methodology"`
	Summary     AxiomSummary           `json:"summary"`
	Axioms      map[string]AxiomDetail `json:"axioms"`
	History     []AxiomHistoryEntry    `json:"history"`
}

type AxiomSummary struct {
	TotalScore     int     `json:"totalScore"`
	MaxScore       int     `json:"maxScore"`
	Percentage     float64 `json:"percentage"`
	StrongCount    int     `json:"strongCount"`
	PartialCount   int     `json:"partialCount"`
	ViolationCount int     `json:"violationCount"`
}

type AxiomDetail struct {
	Name          string   `json:"name"`
	Score         int      `json:"score"`
	MaxScore      int      `json:"maxScore"`
	Status        string   `json:"status"`
	Justification string   `json:"justification"`
	Evidence      []string `json:"evidence"`
	Gaps          []string `json:"gaps"`
}

type AxiomHistoryEntry struct {
	Version    string  `json:"version"`
	Date       string  `json:"date"`
	Score      int     `json:"score"`
	MaxScore   int     `json:"maxScore"`
	Percentage float64 `json:"percentage"`
	Notes      string  `json:"notes"`
}

func axiomsCommand() {
	fs := flag.NewFlagSet("axioms", flag.ExitOnError)
	jsonFlag := fs.Bool("json", false, "Output raw JSON")
	compactFlag := fs.Bool("compact", false, "Compact output (no details)")
	helpFlag := fs.Bool("help", false, "Show help")

	_ = fs.Parse(flag.Args()[1:])

	if *helpFlag {
		printAxiomsHelp()
		return
	}

	// Find scorecard file
	scorecardPath := findScorecardPath()
	if scorecardPath == "" {
		fmt.Fprintf(os.Stderr, "%s: axiom scorecard not found\n", red("Error"))
		fmt.Fprintf(os.Stderr, "Expected at: docs/static/benchmarks/axiom_scorecard.json\n")
		os.Exit(1)
	}

	// Load scorecard
	data, err := os.ReadFile(scorecardPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to read scorecard: %v\n", red("Error"), err)
		os.Exit(1)
	}

	var scorecard AxiomScorecard
	if err := json.Unmarshal(data, &scorecard); err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to parse scorecard: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if *jsonFlag {
		// Output raw JSON
		if *compactFlag {
			out, _ := json.Marshal(scorecard)
			fmt.Println(string(out))
		} else {
			out, _ := json.MarshalIndent(scorecard, "", "  ")
			fmt.Println(string(out))
		}
		return
	}

	// Human-readable output
	printAxiomsScorecard(&scorecard, *compactFlag)
}

func findScorecardPath() string {
	// Try relative to working directory
	paths := []string{
		"docs/static/benchmarks/axiom_scorecard.json",
		"../docs/static/benchmarks/axiom_scorecard.json",
		"../../docs/static/benchmarks/axiom_scorecard.json",
	}

	// Try relative to executable
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		paths = append(paths,
			filepath.Join(execDir, "..", "docs", "static", "benchmarks", "axiom_scorecard.json"),
			filepath.Join(execDir, "..", "..", "docs", "static", "benchmarks", "axiom_scorecard.json"),
		)
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

func printAxiomsScorecard(sc *AxiomScorecard, compact bool) {
	fmt.Println()
	fmt.Printf("  %s AILANG Design Axiom Scorecard\n", bold("📊"))
	fmt.Printf("  %s\n", strings.Repeat("─", 50))
	fmt.Println()

	// Summary
	grade := getGrade(sc.Summary.Percentage)
	gradeColor := getGradeColor(grade)
	fmt.Printf("  Version:     %s\n", cyan(sc.Version))
	fmt.Printf("  Score:       %s / %d (%s)\n",
		bold(fmt.Sprintf("%d", sc.Summary.TotalScore)),
		sc.Summary.MaxScore,
		gradeColor(fmt.Sprintf("%.0f%% %s", sc.Summary.Percentage, grade)))
	fmt.Printf("  Status:      %s strong, %s partial, %s violations\n",
		green(fmt.Sprintf("%d", sc.Summary.StrongCount)),
		yellow(fmt.Sprintf("%d", sc.Summary.PartialCount)),
		red(fmt.Sprintf("%d", sc.Summary.ViolationCount)))
	fmt.Println()

	// Progress bar
	filled := int(sc.Summary.Percentage / 100 * 20)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", 20-filled)
	fmt.Printf("  Progress:    [%s] %.0f%%\n", gradeColor(bar), sc.Summary.Percentage)
	fmt.Println()

	if !compact {
		// Axiom details
		fmt.Printf("  %s\n", strings.Repeat("─", 50))
		fmt.Printf("  %-4s %-35s %s\n", "ID", "Axiom", "Score")
		fmt.Printf("  %s\n", strings.Repeat("─", 50))

		// Order axioms by their ID
		axiomOrder := []string{
			"A1_determinism", "A2_replayability", "A3_effect_legibility",
			"A4_explicit_authority", "A5_bounded_verification", "A6_safe_concurrency",
			"A7_machines_first", "A8_minimal_syntax", "A9_cost_visibility",
			"A10_composability", "A11_structured_failure", "A12_system_boundary",
		}

		for _, key := range axiomOrder {
			axiom, ok := sc.Axioms[key]
			if !ok {
				continue
			}

			// Format ID
			id := strings.Split(key, "_")[0]

			// Status icon and color
			var statusIcon string
			var scoreStr string
			switch axiom.Status {
			case "strong":
				statusIcon = green("✅")
				scoreStr = green(fmt.Sprintf("%d/%d", axiom.Score, axiom.MaxScore))
			case "partial":
				statusIcon = yellow("🔶")
				scoreStr = yellow(fmt.Sprintf("%d/%d", axiom.Score, axiom.MaxScore))
			default:
				statusIcon = red("❌")
				scoreStr = red(fmt.Sprintf("%d/%d", axiom.Score, axiom.MaxScore))
			}

			// Truncate name if too long
			name := axiom.Name
			if len(name) > 32 {
				name = name[:29] + "..."
			}

			fmt.Printf("  %-4s %-35s %s %s\n", id, name, scoreStr, statusIcon)
		}

		fmt.Printf("  %s\n", strings.Repeat("─", 50))
		fmt.Println()

		// Show gaps for partial axioms
		fmt.Printf("  %s Improvement Areas:\n", yellow("📋"))
		for _, key := range axiomOrder {
			axiom, ok := sc.Axioms[key]
			if !ok || axiom.Status != "partial" || len(axiom.Gaps) == 0 {
				continue
			}

			id := strings.Split(key, "_")[0]
			fmt.Printf("  %s %s:\n", yellow(id), axiom.Name)
			for _, gap := range axiom.Gaps {
				fmt.Printf("     • %s\n", gap)
			}
		}
		fmt.Println()
	}

	// Reference
	fmt.Printf("  %s Reference: docs/docs/references/axioms.mdx\n", cyan("📖"))
	fmt.Println()
}

func getGrade(percentage float64) string {
	switch {
	case percentage >= 90:
		return "A"
	case percentage >= 80:
		return "B"
	case percentage >= 70:
		return "C"
	case percentage >= 60:
		return "D"
	default:
		return "F"
	}
}

func getGradeColor(grade string) func(a ...interface{}) string {
	switch grade {
	case "A":
		return green
	case "B":
		return cyan
	case "C":
		return yellow
	case "D", "F":
		return red
	default:
		return fmt.Sprint
	}
}

func printAxiomsHelp() {
	fmt.Println()
	fmt.Printf("  %s AILANG Design Axiom Scorecard\n", bold("📊"))
	fmt.Println()
	fmt.Println("  Usage: ailang axioms [options]")
	fmt.Println()
	fmt.Println("  Options:")
	fmt.Println("    --json      Output raw JSON")
	fmt.Println("    --compact   Compact output (no details)")
	fmt.Println("    --help      Show this help")
	fmt.Println()
	fmt.Println("  The axiom scorecard measures AILANG's compliance with its 12 design axioms.")
	fmt.Println("  Axioms are defined in: docs/docs/references/axioms.mdx")
	fmt.Println()
	fmt.Println("  Scoring:")
	fmt.Println("    +2  Strong compliance (fully implemented)")
	fmt.Println("    +1  Partial compliance (partially implemented)")
	fmt.Println("     0  Neutral (not applicable)")
	fmt.Println("    -1  Violation (design choice contradicts axiom)")
	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println("    ailang axioms              # Show scorecard")
	fmt.Println("    ailang axioms --compact    # Summary only")
	fmt.Println("    ailang axioms --json       # JSON output")
	fmt.Println()
}
