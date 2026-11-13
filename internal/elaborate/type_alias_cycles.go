package elaborate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// checkAliasCycles rejects a type alias that refers to itself, directly or
// through other aliases (TC_TYPE_ALIAS_CYCLE_001): `type L[a] = [L[a]]`,
// `type T2[a] = (T2[a], T2[a])`, `type A = B` with `type B = A`. An alias
// stands for its body, so a cycle has no finite meaning; such declarations
// were accepted and failed later with confusing errors, or made alias
// expansion run without bound. Recursion needs a nominal type, so a cycle
// that passes through an ADT declaration is fine (`type Tree[a] = Leaf |
// Node(Tree[a], a, Tree[a])`). So, deliberately, is one that passes through a
// record declaration (`type Node = {label: string, kids: [Node]}`): dev
// checks and runs those correctly, and real programs use them.
//
// Only this module's aliases can form a cycle: an imported alias's body
// names its own module's types, and a module cannot import its importer
// (dependency cycles are rejected before compilation).
func (e *Elaborator) checkAliasCycles(file *ast.File) error {
	aliases := map[string]*ast.TypeDecl{}
	var order []*ast.TypeDecl
	for _, list := range [][]ast.Node{file.Decls, file.Statements} {
		for _, d := range list {
			td, ok := d.(*ast.TypeDecl)
			if !ok {
				continue
			}
			if _, isAlias := td.Definition.(*ast.TypeAlias); isAlias && aliases[td.Name] == nil {
				aliases[td.Name] = td
				order = append(order, td)
			}
		}
	}
	if len(aliases) == 0 {
		return nil
	}
	edges := map[string][]string{}
	for name, td := range aliases {
		seen := map[string]bool{}
		walkType(td.Definition.(*ast.TypeAlias).Target, func(t ast.Type) {
			var ref string
			switch x := t.(type) {
			case *ast.SimpleType:
				ref = x.Name
			case *ast.TypeApp:
				ref = x.Constructor
			}
			if aliases[ref] != nil && !seen[ref] {
				seen[ref] = true
				edges[name] = append(edges[name], ref)
			}
		})
		sort.Strings(edges[name])
	}
	// Report the first alias in source order that lies on a cycle.
	for _, td := range order {
		if path := aliasCycleFrom(td.Name, edges); path != nil {
			return fmt.Errorf("at %s: %w", e.sourcePos(td.Pos), aliasCycleError(td, path))
		}
	}
	return nil
}

// aliasCycleFrom returns the aliases of a path from start back to start
// (start first), or nil when start is on no cycle.
func aliasCycleFrom(start string, edges map[string][]string) []string {
	var path []string
	onPath := map[string]bool{}
	var dfs func(n string) bool
	dfs = func(n string) bool {
		for _, m := range edges[n] {
			if m == start {
				path = append(path, n)
				return true
			}
			if onPath[m] {
				continue
			}
			onPath[m] = true
			if dfs(m) {
				path = append(path, n)
				return true
			}
		}
		return false
	}
	onPath[start] = true
	if !dfs(start) {
		return nil
	}
	// path was built from the end of the cycle back to start; reverse it.
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// aliasCycleError builds TC_TYPE_ALIAS_CYCLE_001 with the ADT the alias can
// become: the same body wrapped in a constructor of the alias's name.
func aliasCycleError(td *ast.TypeDecl, path []string) error {
	via := ""
	if len(path) > 1 {
		via = " through " + strings.Join(path[1:], ", ")
	}
	head := td.Name
	if len(td.TypeParams) > 0 {
		head += "[" + strings.Join(td.TypeParams, ", ") + "]"
	}
	body := renderType(td.Definition.(*ast.TypeAlias).Target)
	return fmt.Errorf("%s: type alias '%s' refers to itself%s; a recursive type must be an ADT: write `type %s = %s(%s)` (a constructor) instead",
		types.TC_TYPE_ALIAS_CYCLE_001, td.Name, via, head, td.Name, body)
}

// renderType prints a type in source syntax (`[x]` for a list).
func renderType(t ast.Type) string {
	switch x := t.(type) {
	case *ast.TypeApp:
		args := make([]string, len(x.Args))
		for i, a := range x.Args {
			args[i] = renderType(a)
		}
		if x.Constructor == "list" && len(args) == 1 {
			return "[" + args[0] + "]"
		}
		return x.Constructor + "[" + strings.Join(args, ", ") + "]"
	case *ast.TupleType:
		elems := make([]string, len(x.Elements))
		for i, el := range x.Elements {
			elems[i] = renderType(el)
		}
		return "(" + strings.Join(elems, ", ") + ")"
	case *ast.ListType:
		return "[" + renderType(x.Element) + "]"
	default:
		return t.String()
	}
}
