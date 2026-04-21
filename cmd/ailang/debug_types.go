package main

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/types"
)

// TypeDebugDumper formats type debugging information for CLI output.
// It consumes events from VerboseDebugSink and TypeReport data
// to produce human-readable (and AI-parseable) debug output.
type TypeDebugDumper struct {
	sink       *types.VerboseDebugSink
	tc         *types.CoreTypeChecker
	filterNode uint64 // 0 means no filter
}

// NewTypeDebugDumper creates a dumper that will format debug output.
func NewTypeDebugDumper(sink *types.VerboseDebugSink, tc *types.CoreTypeChecker, filterNode uint64) *TypeDebugDumper {
	return &TypeDebugDumper{
		sink:       sink,
		tc:         tc,
		filterNode: filterNode,
	}
}

// Dump prints the formatted debug output to stdout.
func (d *TypeDebugDumper) Dump() {
	fmt.Println(cyan("=== Type Inference Debug ==="))
	fmt.Println()

	d.dumpSubstitutionMap()
	d.dumpConstraints()
	d.dumpCoreTI()
}

// dumpSubstitutionMap shows the substitution map with chain resolution.
func (d *TypeDebugDumper) dumpSubstitutionMap() {
	fmt.Println(bold("[Substitution Map]"))

	// Collect substitution events
	subs := make(map[string]types.Type)
	for _, event := range d.sink.Events() {
		if event.Kind == types.EventSubstitute {
			if tv, ok := event.TypeVar.(*types.TVar2); ok {
				subs[tv.Name] = event.Result
			}
		}
	}

	if len(subs) == 0 {
		fmt.Println("  (empty)")
		fmt.Println()
		return
	}

	// Sort for deterministic output
	var names []string
	for name := range subs {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		resolved := subs[name]
		// Check if this is a chain (resolved type is also a type var)
		if tv, ok := resolved.(*types.TVar2); ok {
			// It's a chain - show intermediate step
			if final, hasFinal := subs[tv.Name]; hasFinal {
				fmt.Printf("  %s → %s → %s %s\n",
					green(name), tv.Name, final.String(), dim("(CHAIN)"))
			} else {
				fmt.Printf("  %s → %s\n", green(name), resolved.String())
			}
		} else {
			fmt.Printf("  %s → %s %s\n", green(name), resolved.String(), dim("(direct)"))
		}
	}
	fmt.Println()
}

// dumpConstraints shows constraint add/resolve events.
func (d *TypeDebugDumper) dumpConstraints() {
	fmt.Println(bold("[Constraints]"))

	var adds []types.DebugEvent
	var resolves []types.DebugEvent

	for _, event := range d.sink.Events() {
		switch event.Kind {
		case types.EventConstraintAdd:
			if d.filterNode == 0 || event.NodeID == d.filterNode {
				adds = append(adds, event)
			}
		case types.EventConstraintResolve:
			if d.filterNode == 0 || event.NodeID == d.filterNode {
				resolves = append(resolves, event)
			}
		}
	}

	if len(adds) == 0 && len(resolves) == 0 {
		fmt.Println("  (no constraints)")
		fmt.Println()
		return
	}

	if len(adds) > 0 {
		fmt.Println("  " + dim("Added:"))
		for _, event := range adds {
			typeStr := "?"
			if event.Result != nil {
				typeStr = event.Result.String()
			}
			fmt.Printf("    %s %s at node %d\n",
				yellow(event.ClassName), typeStr, event.NodeID)
		}
	}

	if len(resolves) > 0 {
		fmt.Println("  " + dim("Resolved:"))
		for _, event := range resolves {
			typeStr := "?"
			if event.Result != nil {
				typeStr = event.Result.String()
			}
			fmt.Printf("    %s %s → %s at node %d\n",
				green(event.ClassName), typeStr, event.Method, event.NodeID)
		}
	}

	fmt.Println()
}

// dumpCoreTI shows CoreTypeInfo entries with raw vs resolved types.
func (d *TypeDebugDumper) dumpCoreTI() {
	fmt.Println(bold("[CoreTI Entries]"))

	if d.tc == nil {
		fmt.Println("  (type checker not available)")
		fmt.Println()
		return
	}

	// Get all node IDs from CoreTI directly
	var nodeIDs []uint64
	for nodeID := range d.tc.CoreTI {
		if d.filterNode == 0 || nodeID == d.filterNode {
			nodeIDs = append(nodeIDs, nodeID)
		}
	}

	if len(nodeIDs) == 0 {
		if d.filterNode != 0 {
			fmt.Printf("  (no entries for node %d)\n", d.filterNode)
		} else {
			fmt.Println("  (no entries)")
		}
		fmt.Println()
		return
	}

	// Sort for deterministic output
	sort.Slice(nodeIDs, func(i, j int) bool { return nodeIDs[i] < nodeIDs[j] })

	for _, nodeID := range nodeIDs {
		report := d.tc.TypeReport(nodeID)
		if !report.Found {
			continue
		}

		rawStr := "?"
		if report.Raw != nil {
			rawStr = report.Raw.String()
		}

		resolvedStr := "?"
		if report.Resolved != nil {
			resolvedStr = report.Resolved.String()
		}

		// Check if raw and resolved are different (interesting case)
		if rawStr != resolvedStr {
			fmt.Printf("  NodeID %d: %s\n", nodeID, green(resolvedStr))
			fmt.Printf("    Raw: %s\n", rawStr)
			fmt.Printf("    Resolved: %s\n", resolvedStr)
		} else {
			fmt.Printf("  NodeID %d: %s\n", nodeID, green(resolvedStr))
		}

		// Show constraints if any
		if len(report.Constraints) > 0 {
			for _, c := range report.Constraints {
				status := "(pending)"
				if c.Resolved && c.Method != "" {
					status = fmt.Sprintf("→ %s", c.Method)
				} else if c.Resolved {
					status = "(resolved)"
				}
				fmt.Printf("    Constraint: %s %s\n", c.ClassName, dim(status))
			}
		}

		// M-DX11: Show provenance if available
		if len(report.Origins) > 0 {
			fmt.Printf("    Origins:\n")
			for _, origin := range report.Origins {
				line := fmt.Sprintf("      - %s", origin.Kind.String())
				if origin.Note != "" {
					line += ": " + origin.Note
				}
				if origin.Span.Line != 0 || origin.Span.File != "" {
					line += " at " + origin.Span.String()
				}
				fmt.Println(line)
			}
		}
	}
	fmt.Println()
}

// Helper color functions (reuse from main.go)
func dim(s string) string {
	return "\033[2m" + s + "\033[0m"
}

// FormatTypeDebugOutput is a convenience function that creates a dumper
// and prints the output.
func FormatTypeDebugOutput(sink *types.VerboseDebugSink, tc *types.CoreTypeChecker, filterNode uint64) {
	dumper := NewTypeDebugDumper(sink, tc, filterNode)
	dumper.Dump()
}
