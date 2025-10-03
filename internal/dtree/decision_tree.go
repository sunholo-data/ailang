package dtree

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
)

// DecisionTree represents a compiled pattern matching decision tree
// This optimizes pattern matching by avoiding redundant tests
type DecisionTree interface {
	isDecisionTree()
	String() string
}

// LeafNode represents a match with a body to execute
type LeafNode struct {
	ArmIndex int // Index of the original match arm
	Body     core.CoreExpr
	Guard    core.CoreExpr // Optional guard
}

func (l *LeafNode) isDecisionTree() {}
func (l *LeafNode) String() string  { return fmt.Sprintf("Leaf(arm=%d)", l.ArmIndex) }

// FailNode represents no match (non-exhaustive)
type FailNode struct{}

func (f *FailNode) isDecisionTree() {}
func (f *FailNode) String() string  { return "Fail" }

// SwitchNode represents a choice based on a discriminator
type SwitchNode struct {
	Path    []int                        // Path to the value being tested (e.g., [0, 1] = first field of second field)
	Cases   map[interface{}]DecisionTree // Map from constructor/literal to subtree
	Default DecisionTree                 // Fallback for wildcard/variable patterns
}

func (s *SwitchNode) isDecisionTree() {}
func (s *SwitchNode) String() string {
	return fmt.Sprintf("Switch(path=%v, cases=%d, default=%v)", s.Path, len(s.Cases), s.Default != nil)
}

// DecisionTreeCompiler compiles match arms into a decision tree
type DecisionTreeCompiler struct {
	arms []core.MatchArm
}

// NewDecisionTreeCompiler creates a new compiler
func NewDecisionTreeCompiler(arms []core.MatchArm) *DecisionTreeCompiler {
	return &DecisionTreeCompiler{arms: arms}
}

// Compile builds a decision tree from match arms
func (c *DecisionTreeCompiler) Compile() DecisionTree {
	// Build initial matrix: each row is (pattern, armIndex, guard, body)
	var matrix []matchRow
	for i, arm := range c.arms {
		matrix = append(matrix, matchRow{
			patterns: []core.CorePattern{arm.Pattern},
			armIndex: i,
			guard:    arm.Guard,
			body:     arm.Body,
		})
	}

	// Compile from the matrix
	return c.compileMatrix(matrix, []int{}) // Start with empty path (root of scrutinee)
}

// matchRow represents one row in the pattern matrix
type matchRow struct {
	patterns []core.CorePattern // Patterns for each column (starts with just scrutinee)
	armIndex int                // Original arm index
	guard    core.CoreExpr      // Optional guard
	body     core.CoreExpr      // Body to execute
}

// compileMatrix builds a decision tree from a pattern matrix
func (c *DecisionTreeCompiler) compileMatrix(matrix []matchRow, path []int) DecisionTree {
	// Base cases
	if len(matrix) == 0 {
		// No rows left - this is a failure case (non-exhaustive match)
		return &FailNode{}
	}

	// If first row has only wildcards/variables in all columns, it's a leaf
	if c.isDefaultRow(matrix[0]) {
		return &LeafNode{
			ArmIndex: matrix[0].armIndex,
			Body:     matrix[0].body,
			Guard:    matrix[0].guard,
		}
	}

	// Find the best column to split on (for now, just use column 0)
	colIndex := 0
	if colIndex >= len(matrix[0].patterns) {
		// All columns exhausted - first row wins
		return &LeafNode{
			ArmIndex: matrix[0].armIndex,
			Body:     matrix[0].body,
			Guard:    matrix[0].guard,
		}
	}

	// Build switch node based on column 0
	return c.buildSwitch(matrix, path, colIndex)
}

// isDefaultRow checks if a row contains only wildcards/variables
func (c *DecisionTreeCompiler) isDefaultRow(row matchRow) bool {
	for _, pat := range row.patterns {
		switch pat.(type) {
		case *core.WildcardPattern, *core.VarPattern:
			continue
		default:
			return false
		}
	}
	return true
}

// buildSwitch creates a switch node for the given column
func (c *DecisionTreeCompiler) buildSwitch(matrix []matchRow, path []int, colIndex int) DecisionTree {
	// Group rows by their pattern in column colIndex
	cases := make(map[interface{}][]matchRow)
	var defaultRows []matchRow

	for _, row := range matrix {
		if colIndex >= len(row.patterns) {
			defaultRows = append(defaultRows, row)
			continue
		}

		pat := row.patterns[colIndex]
		switch p := pat.(type) {
		case *core.LitPattern:
			// Group by literal value
			cases[p.Value] = append(cases[p.Value], row)

		case *core.ConstructorPattern:
			// Group by constructor name
			cases[p.Name] = append(cases[p.Name], row)

		case *core.WildcardPattern, *core.VarPattern:
			// Wildcard/variable goes to default
			defaultRows = append(defaultRows, row)

		default:
			// For now, treat unknown patterns as default
			defaultRows = append(defaultRows, row)
		}
	}

	// If we only have defaults and no specific cases, collapse to the first default
	if len(cases) == 0 && len(defaultRows) > 0 {
		return &LeafNode{
			ArmIndex: defaultRows[0].armIndex,
			Body:     defaultRows[0].body,
			Guard:    defaultRows[0].guard,
		}
	}

	// Build subtrees for each case
	switchNode := &SwitchNode{
		Path:  append(path, colIndex),
		Cases: make(map[interface{}]DecisionTree),
	}

	for key, rows := range cases {
		// Remove the matched column from each row (pattern specialization)
		specialized := c.specializeRows(rows, colIndex)
		switchNode.Cases[key] = c.compileMatrix(specialized, append(path, colIndex))
	}

	// Build default subtree
	if len(defaultRows) > 0 {
		specialized := c.specializeRows(defaultRows, colIndex)
		switchNode.Default = c.compileMatrix(specialized, append(path, colIndex))
	} else {
		switchNode.Default = &FailNode{}
	}

	return switchNode
}

// specializeRows removes the matched column from rows
func (c *DecisionTreeCompiler) specializeRows(rows []matchRow, colIndex int) []matchRow {
	var result []matchRow
	for _, row := range rows {
		// Remove column colIndex from patterns
		newPatterns := make([]core.CorePattern, 0, len(row.patterns)-1)
		for i, pat := range row.patterns {
			if i == colIndex {
				// For constructor patterns, expand to their arguments
				if ctorPat, ok := pat.(*core.ConstructorPattern); ok {
					newPatterns = append(newPatterns, ctorPat.Args...)
				}
				// For literals/wildcards, just remove the column
				continue
			}
			newPatterns = append(newPatterns, pat)
		}

		result = append(result, matchRow{
			patterns: newPatterns,
			armIndex: row.armIndex,
			guard:    row.guard,
			body:     row.body,
		})
	}
	return result
}

// CanCompileToTree determines if a match can benefit from decision tree compilation
// For now, simple heuristic: worth it if there are multiple literal/constructor patterns
func CanCompileToTree(arms []core.MatchArm) bool {
	// Count how many arms have literal or constructor patterns
	count := 0
	for _, arm := range arms {
		switch arm.Pattern.(type) {
		case *core.LitPattern, *core.ConstructorPattern:
			count++
		}
	}

	// Worth compiling if we have multiple testable patterns
	// Decision trees excel when there are multiple specific cases to dispatch on
	return count >= 2
}
