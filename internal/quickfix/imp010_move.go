package quickfix

import (
	"fmt"
	"regexp"
	"strings"
)

var imp010ExportedByRe = regexp.MustCompile(`symbol '([^']+)' not exported by '([^']+)'[^—]*—\s*['` + "`" + `][^'` + "`" + `]+['` + "`" + `]\s+is exported by\s+([^;\n]+)`)

// computeIMP010MoveEdits handles IMP010 diagnostics that identify a single
// canonical exporting module (e.g. `symbol 'join' not exported by 'std/list' —
// 'join' is exported by std/string; import it from there, not 'std/list'`).
// It removes `<sym>` from the offending `import <fromMod> (...)` list (or
// removes/replaces the entire import line when `<sym>` was its sole symbol)
// and merges `<sym>` into an existing `import <targetMod> (...)` or inserts a
// new `import <targetMod> (<sym>)` declaration.
func computeIMP010MoveEdits(file string, lines []string, msg string) []TextEdit {
	if strings.Contains(msg, "(qualified access ") {
		return nil
	}
	m := imp010ExportedByRe.FindStringSubmatch(msg)
	if m == nil {
		return nil
	}
	sym, fromMod, rawMods := m[1], m[2], strings.TrimSpace(m[3])
	candidates := strings.Split(rawMods, ",")
	if len(candidates) != 1 {
		return nil // ambiguous: multiple modules export sym
	}
	targetMod := strings.Trim(strings.TrimSpace(candidates[0]), "'\"`")
	if targetMod == "" || targetMod == fromMod {
		return nil
	}

	fromIdx, openIdx, closeIdx, kept, movedItem, ok := findSelectiveImportRemoval(lines, fromMod, sym)
	if !ok {
		return nil
	}

	targetLineIdx, targetCloseIdx, targetAlreadyHas := findSelectiveTargetImport(lines, targetMod, sym)

	// Single-symbol offending import with no existing target import: replace in place.
	if len(kept) == 0 && targetLineIdx < 0 {
		origLine := lines[fromIdx]
		indent := leadingWhitespace(origLine)
		suffix := origLine[closeIdx+1:]
		newLine := fmt.Sprintf("%simport %s (%s)%s", indent, targetMod, movedItem, suffix)
		return []TextEdit{{
			File:      file,
			StartLine: fromIdx + 1,
			StartCol:  1,
			EndLine:   fromIdx + 1,
			EndCol:    len(origLine) + 1,
			NewText:   newLine,
			Title:     fmt.Sprintf("Move %s from %s to %s", sym, fromMod, targetMod),
		}}
	}

	var edits []TextEdit
	if len(kept) == 0 {
		// Delete the entire single-symbol offending import line.
		if fromIdx+1 < len(lines) {
			edits = append(edits, TextEdit{
				File:      file,
				StartLine: fromIdx + 1,
				StartCol:  1,
				EndLine:   fromIdx + 2,
				EndCol:    1,
				NewText:   "",
				Title:     fmt.Sprintf("Remove mis-imported %s from %s", sym, fromMod),
			})
		} else {
			edits = append(edits, TextEdit{
				File:      file,
				StartLine: fromIdx + 1,
				StartCol:  1,
				EndLine:   fromIdx + 1,
				EndCol:    len(lines[fromIdx]) + 1,
				NewText:   "",
				Title:     fmt.Sprintf("Remove mis-imported %s from %s", sym, fromMod),
			})
		}
	} else {
		origLine := lines[fromIdx]
		newLine := origLine[:openIdx+1] + strings.Join(kept, ", ") + origLine[closeIdx:]
		edits = append(edits, TextEdit{
			File:      file,
			StartLine: fromIdx + 1,
			StartCol:  1,
			EndLine:   fromIdx + 1,
			EndCol:    len(origLine) + 1,
			NewText:   newLine,
			Title:     fmt.Sprintf("Remove %s from %s import", sym, fromMod),
		})
	}

	if targetLineIdx >= 0 {
		if !targetAlreadyHas {
			edits = append(edits, TextEdit{
				File:      file,
				StartLine: targetLineIdx + 1,
				StartCol:  targetCloseIdx + 1,
				EndLine:   targetLineIdx + 1,
				EndCol:    targetCloseIdx + 1,
				NewText:   ", " + movedItem,
				Title:     fmt.Sprintf("Add %s to import %s", sym, targetMod),
			})
		}
		return edits
	}

	insertAfter := 0
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "module ") || strings.HasPrefix(t, "import ") {
			insertAfter = i + 1
		}
	}
	edits = append(edits, TextEdit{
		File:      file,
		StartLine: insertAfter + 1,
		StartCol:  1,
		EndLine:   insertAfter + 1,
		EndCol:    1,
		NewText:   fmt.Sprintf("import %s (%s)\n", targetMod, movedItem),
		Title:     fmt.Sprintf("Add import %s (%s)", targetMod, movedItem),
	})
	return edits
}

func matchesImportModuleHeader(line, mod string) bool {
	trimmed := strings.TrimSpace(line)
	prefix := "import " + mod
	if !strings.HasPrefix(trimmed, prefix) {
		return false
	}
	rest := trimmed[len(prefix):]
	return rest == "" || rest[0] == ' ' || rest[0] == '\t' || rest[0] == '('
}

func importItemSymbolName(item string) string {
	fields := strings.Fields(item)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func findSelectiveImportRemoval(lines []string, fromMod, sym string) (lineIdx, openIdx, closeIdx int, kept []string, movedItem string, ok bool) {
	for i, line := range lines {
		if !matchesImportModuleHeader(line, fromMod) {
			continue
		}
		open := strings.Index(line, "(")
		closeParen := strings.Index(line, ")")
		if open < 0 || closeParen <= open {
			continue
		}
		parts := strings.Split(line[open+1:closeParen], ",")
		var rem []string
		found := ""
		for _, p := range parts {
			t := strings.TrimSpace(p)
			if t == "" {
				continue
			}
			if found == "" && importItemSymbolName(t) == sym {
				found = t
				continue
			}
			rem = append(rem, t)
		}
		if found != "" {
			return i, open, closeParen, rem, found, true
		}
	}
	return -1, -1, -1, nil, "", false
}

func findSelectiveTargetImport(lines []string, targetMod, sym string) (lineIdx, closeIdx int, alreadyHas bool) {
	for i, line := range lines {
		if !matchesImportModuleHeader(line, targetMod) {
			continue
		}
		open := strings.Index(line, "(")
		closeParen := strings.Index(line, ")")
		if open < 0 || closeParen <= open {
			continue
		}
		for _, p := range strings.Split(line[open+1:closeParen], ",") {
			if importItemSymbolName(strings.TrimSpace(p)) == sym {
				return i, closeParen, true
			}
		}
		return i, closeParen, false
	}
	return -1, -1, false
}
