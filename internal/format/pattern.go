package format

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// pattern.go prints match-case patterns to canonical source. A pattern is a
// leaf grammar (no operator precedence), so each printer returns a plain string.
//
// Note: an *ast.Identifier and an *ast.Literal both implement ast.Pattern (they
// carry patternNode markers). A bare identifier in pattern position is a binder;
// a literal in pattern position is a literal match. Both are handled here so the
// pattern visitor is exhaustive over every concrete Pattern implementation.

// patternString renders any ast.Pattern to canonical source, or errors on a
// nil-required child or unknown concrete pattern.
func (p *printer) patternString(pat ast.Pattern) (string, error) {
	if pat == nil {
		return "", fmt.Errorf("nil pattern node")
	}
	switch n := pat.(type) {
	case *ast.WildcardPattern:
		return "_", nil
	case *ast.Identifier:
		return n.Name, nil
	case *ast.Literal:
		return literalString(n)
	case *ast.ConsPattern:
		head, err := p.patternString(n.Head)
		if err != nil {
			return "", err
		}
		tail, err := p.patternString(n.Tail)
		if err != nil {
			return "", err
		}
		// Canonical cons-pattern spelling matches the parser's list-cons pattern:
		// [head, ...tail].
		return "[" + head + ", ..." + tail + "]", nil
	case *ast.ListPattern:
		return p.listPatternString(n)
	case *ast.TuplePattern:
		parts, err := p.patternList(n.Elements)
		if err != nil {
			return "", err
		}
		return "(" + strings.Join(parts, ", ") + ")", nil
	case *ast.RecordPattern:
		return p.recordPatternString(n)
	case *ast.ConstructorPattern:
		return p.constructorPatternString(n)
	default:
		return "", fmt.Errorf("unsupported pattern node: %T", pat)
	}
}

func (p *printer) patternList(pats []ast.Pattern) ([]string, error) {
	parts := make([]string, len(pats))
	for i, pat := range pats {
		s, err := p.patternString(pat)
		if err != nil {
			return nil, err
		}
		parts[i] = s
	}
	return parts, nil
}

func (p *printer) listPatternString(n *ast.ListPattern) (string, error) {
	parts, err := p.patternList(n.Elements)
	if err != nil {
		return "", err
	}
	if n.Rest != nil {
		rest, err := p.patternString(n.Rest)
		if err != nil {
			return "", err
		}
		parts = append(parts, "..."+rest)
	}
	return "[" + strings.Join(parts, ", ") + "]", nil
}

func (p *printer) recordPatternString(n *ast.RecordPattern) (string, error) {
	fields := make([]string, 0, len(n.Fields)+1)
	for _, f := range n.Fields {
		sub, err := p.patternString(f.Pattern)
		if err != nil {
			return "", err
		}
		fields = append(fields, f.Name+": "+sub)
	}
	if n.Rest {
		fields = append(fields, "...")
	}
	if len(fields) == 0 {
		return "{}", nil
	}
	return "{ " + strings.Join(fields, ", ") + " }", nil
}

func (p *printer) constructorPatternString(n *ast.ConstructorPattern) (string, error) {
	if len(n.Patterns) == 0 {
		return n.Name, nil
	}
	parts, err := p.patternList(n.Patterns)
	if err != nil {
		return "", err
	}
	return n.Name + "(" + strings.Join(parts, ", ") + ")", nil
}
