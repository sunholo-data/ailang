package smt

import (
	"regexp"
	"strings"
)

var pluralSortPattern = regexp.MustCompile(`\(([A-Za-z_][A-Za-z_0-9]*)\s+[0-9]+\)`)

func pluralDatatypeSortNames(decl string) []string {
	if !strings.HasPrefix(decl, "(declare-datatypes (") {
		return nil
	}
	headerEnd := strings.Index(decl, "))")
	if headerEnd < 0 {
		return nil
	}
	matches := pluralSortPattern.FindAllStringSubmatch(decl[:headerEnd+2], -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		names = append(names, match[1])
	}
	return names
}

func constantDeclarationHeader(decl string) (name, sortStr string, ok bool) {
	fieldsStart := strings.IndexByte(decl, ' ')
	if fieldsStart < 0 {
		return "", "", false
	}
	rest := strings.TrimSpace(decl[fieldsStart+1:])
	nameEnd := strings.IndexAny(rest, " \t\r\n")
	if nameEnd < 1 {
		return "", "", false
	}
	name = rest[:nameEnd]
	rest = strings.TrimSpace(rest[nameEnd:])
	if rest == "" {
		return "", "", false
	}
	if rest[0] != '(' {
		sortEnd := strings.IndexAny(rest, " \t\r\n)")
		if sortEnd < 0 {
			return "", "", false
		}
		return name, rest[:sortEnd], true
	}
	depth := 0
	for i, r := range rest {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return name, rest[:i+1], true
			}
		}
	}
	return "", "", false
}

func firstUnresolvedSort(sortStr string, declared map[string]bool) string {
	return validateSortApplication(sortStr, declared)
}
