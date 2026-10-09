package testing

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// Match the compiler's heading, never arbitrary mentions of generated symbols.
// Property entries share the same purity contract as named-test entries.
var entryEffectHeading = regexp.MustCompile(`(?:^|\n|: )Effect checking failed for function '(__namedtest_(?:[0-9]+|prop_[0-9]+|entry)|\$tmp[0-9]+)'\n`)

func (e *Executor) mapEntryEffectError(msg string, ent namedEntry) string {
	if !entryEffectHeading.MatchString(msg) {
		return msg
	}
	kind := "named test"
	if ent.label == "property" {
		kind = "property"
	}
	var detail []string
	var caps string
	for _, line := range strings.Split(msg, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Missing effects:") {
			detail = append(detail, line)
			caps = strings.ReplaceAll(strings.TrimSpace(strings.TrimPrefix(trimmed, "Missing effects:")), ", ", ",")
		} else if strings.HasPrefix(trimmed, "Effect ") && strings.Contains(trimmed, " mismatch:") {
			detail = append(detail, line)
		}
	}
	route := "ailang run --caps " + caps
	if caps == "" {
		route = "ailang run --caps <required-effects>"
	}
	return fmt.Sprintf("%s %q (%s:%d:1): bodies are checked pure (m-named-test-blocks.md, v0.29.0), but this body requires effects not allowed in a pure signature:\n%s\n\nEffectful verification belongs in an exported entry run with `%s`, or in a pure test over byte-mirror fixtures.", kind, ent.title, e.modulePath, ent.testLine, strings.Join(detail, "\n"), route)
}

func (e *Executor) mapEntryCompileError(msg string, ent namedEntry, lines []int) string {
	return e.mapEntryEffectError(e.mapPositions(msg, ent, lines), ent)
}

// Recover the original construct for the legacy per-body engine API.
func (e *Executor) entryForBody(body []ast.Expr) namedEntry {
	ent := namedEntry{label: "test body"}
	if len(body) == 0 || e.sourceFile == nil {
		return ent
	}
	for _, d := range e.sourceFile.Decls {
		if td, ok := d.(*ast.TestDecl); ok && len(td.Body) > 0 && td.Body[0] == body[0] {
			ent.title = td.Name
			ent.testLine = td.Pos.Line
			break
		}
	}
	return ent
}
