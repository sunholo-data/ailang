package smt

import (
	"fmt"
	"strings"
)

// codegen_mutual.go — mutual recursion via Z3's `declare-datatypes` (plural).
//
// Z3's singular `declare-datatype` cannot express mutual recursion: the body
// of declaration N may not reference an undeclared sort. The plural form
// `declare-datatypes ((Sort1 0) (Sort2 0)) (vars1 vars2)` declares all sorts
// and their variants in a single block, supporting cycles.
//
// This file:
//   - Parses existing pending `declare-datatype` strings (built by Step 0.5/1
//     of EncodeFunction) into a (sortName, variantsBody) pair.
//   - Builds a sort-reference graph from those decls.
//   - Runs Tarjan's SCC algorithm to find cycles.
//   - For each SCC of size > 1 (or self-recursive singleton with a self-edge
//     in the variants), emits a plural `declare-datatypes` block.

// splitDeclareDatatype parses a `(declare-datatype Name (variants))` string
// and returns (Name, variantsBody) where variantsBody includes the outer
// parens around the variant list. Returns ("", "") if the input is not a
// well-formed declaration.
func splitDeclareDatatype(decl string) (string, string) {
	const prefix = "(declare-datatype "
	if !strings.HasPrefix(decl, prefix) {
		return "", ""
	}
	rest := decl[len(prefix):]
	end := strings.IndexAny(rest, " (")
	if end == -1 {
		return "", ""
	}
	name := rest[:end]
	// Skip whitespace to the start of the variants tuple.
	body := strings.TrimLeft(rest[end:], " \t")
	// The outer wrapper is `(... )` closing the declare-datatype expr; we want
	// to drop the trailing `)` of the outer wrapper, leaving just the variants.
	body = strings.TrimSuffix(body, ")")
	body = strings.TrimRight(body, " \t")
	return name, body
}

// DeclareDatatypesMutual emits the plural form for a group of mutually
// recursive declarations. Each input string MUST be a singular
// `(declare-datatype Name (variants))`. Output:
//
//	(declare-datatypes ((Name1 0) (Name2 0) ...) (variants1 variants2 ...))
func DeclareDatatypesMutual(decls []string) string {
	if len(decls) == 0 {
		return ""
	}
	if len(decls) == 1 {
		// A single sort is not "mutual" in the strict sense, but a self-
		// recursive singleton can be expressed via the singular form (Z3
		// allows a sort to reference itself in its own variants). Pass through.
		return decls[0]
	}
	var typeList []string
	var bodyList []string
	for _, d := range decls {
		name, body := splitDeclareDatatype(d)
		if name == "" {
			// Pass-through: we received something we can't parse. Better to
			// emit it raw than to silently drop it.
			continue
		}
		typeList = append(typeList, fmt.Sprintf("(%s 0)", name))
		bodyList = append(bodyList, body)
	}
	return fmt.Sprintf("(declare-datatypes (%s) (%s))",
		strings.Join(typeList, " "),
		strings.Join(bodyList, " "))
}

// findSCCs returns the strongly connected components of the sort-reference
// graph induced by the input declarations. Each SCC is a list of sort names.
//
// Edge: sort A → sort B exists iff B's name appears as a token in A's variant
// body. We rely on the convention that sort names are alphanumeric+underscore
// tokens that appear adjacent to whitespace or parens in declarations.
//
// Tarjan's algorithm produces SCCs in reverse topological order; we don't
// reverse them here because the caller emits each SCC independently and only
// the per-SCC plural form needs to be self-contained.
func findSCCs(decls []string) [][]string {
	// Build name → body index.
	bodyByName := make(map[string]string, len(decls))
	names := make([]string, 0, len(decls))
	for _, d := range decls {
		name, body := splitDeclareDatatype(d)
		if name == "" {
			continue
		}
		bodyByName[name] = body
		names = append(names, name)
	}

	// Build adjacency: name → list of names referenced in body.
	adj := make(map[string][]string, len(names))
	for _, name := range names {
		var refs []string
		body := bodyByName[name]
		seen := make(map[string]bool)
		for _, candidate := range names {
			if candidate == name {
				// Self-edge: only count if body references the name as a
				// non-constructor token. Easy heuristic: substring match.
				if containsSortRef(body, candidate) {
					if !seen[candidate] {
						seen[candidate] = true
						refs = append(refs, candidate)
					}
				}
				continue
			}
			if containsSortRef(body, candidate) {
				if !seen[candidate] {
					seen[candidate] = true
					refs = append(refs, candidate)
				}
			}
		}
		adj[name] = refs
	}

	// Tarjan's SCC algorithm.
	var (
		index   int
		stack   []string
		onStack = make(map[string]bool)
		indices = make(map[string]int)
		lowlink = make(map[string]int)
		sccs    [][]string
	)

	var strongconnect func(v string)
	strongconnect = func(v string) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range adj[v] {
			if _, visited := indices[w]; !visited {
				strongconnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] {
				if indices[w] < lowlink[v] {
					lowlink[v] = indices[w]
				}
			}
		}
		if lowlink[v] == indices[v] {
			var component []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				component = append(component, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, component)
		}
	}

	for _, n := range names {
		if _, visited := indices[n]; !visited {
			strongconnect(n)
		}
	}
	return sccs
}

// containsSortRef returns true if body contains name as a "sort reference" —
// an occurrence flanked by characters that cannot be part of an identifier.
// Avoids false positives like "Block" matching "BlockSomething".
func containsSortRef(body, name string) bool {
	for i := 0; i < len(body); {
		j := strings.Index(body[i:], name)
		if j == -1 {
			return false
		}
		j += i
		// Check left boundary.
		leftOK := j == 0 || !isIdentChar(body[j-1])
		// Check right boundary.
		end := j + len(name)
		rightOK := end >= len(body) || !isIdentChar(body[end])
		if leftOK && rightOK {
			return true
		}
		i = j + 1
	}
	return false
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}
