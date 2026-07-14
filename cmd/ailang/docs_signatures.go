package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// exportSignatures maps an exported function name -> its AST-rendered signature
// for a single stdlib file. Signatures are rendered from ast.FuncDecl fields
// (NOT from the exportSigRe regex, which truncates effect rows at `{` — the V16
// defect: `now() -> int ! {Clock}` became `now() -> int ! `). This is the same
// parser-backed extraction pattern iteration 29 used for imports; cmd/ailang
// already links internal/parser (see debug.go / test.go).
//
// The parser is used READ-ONLY via its public New(lexer)/Parse() API plus the
// ast package's existing Type.String()/FormatEffects renderers — no grammar or
// AST changes.
type exportSignatures map[string]string

// parseExportSignatures parses a single stdlib .ail file and renders the
// signature of every exported function from its AST. It also returns the export
// names in FILE ORDER (declaration order), so callers can emit deterministic,
// diffable output.
//
// A parse failure is returned as an error naming the file — the caller MUST fail
// loudly (non-zero exit), never drop or partial-render a row. Stdlib always
// parsing is CI's contract.
func parseExportSignatures(filePath string) (exportSignatures, []string, error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read %s: %w", filePath, err)
	}

	l := lexer.New(string(source), filePath)
	p := parser.New(l)
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		return nil, nil, fmt.Errorf("cannot parse stdlib file %s: %v", filePath, errs[0])
	}
	if prog == nil || prog.File == nil {
		return nil, nil, fmt.Errorf("cannot parse stdlib file %s: no AST produced", filePath)
	}

	sigs := make(exportSignatures)
	var order []string
	for _, fn := range prog.File.Funcs {
		if fn == nil || !fn.IsExport {
			continue
		}
		sigs[fn.Name] = renderFuncSignature(fn)
		order = append(order, fn.Name)
	}
	return sigs, order, nil
}

// renderFuncSignature renders `name[T1,T2](p1type, p2type) -> rettype ! {Eff}`
// from an ast.FuncDecl using only declaration-level facts (fully annotated for
// stdlib exports by policy) and the ast package's own String()/FormatEffects
// renderers — the SAME renderers the compiler and error messages use.
func renderFuncSignature(fn *ast.FuncDecl) string {
	var sb strings.Builder
	sb.WriteString(fn.Name)

	if len(fn.TypeParams) > 0 {
		sb.WriteString("[")
		sb.WriteString(strings.Join(fn.TypeParams, ", "))
		sb.WriteString("]")
	}

	params := make([]string, 0, len(fn.Params))
	for _, param := range fn.Params {
		params = append(params, typeString(param.Type))
	}
	sb.WriteString("(")
	sb.WriteString(strings.Join(params, ", "))
	sb.WriteString(")")

	sb.WriteString(" -> ")
	sb.WriteString(typeString(fn.ReturnType))

	if eff := ast.FormatEffects(fn.Effects); eff != "" {
		sb.WriteString(" ")
		sb.WriteString(eff)
	}

	return sb.String()
}

// typeString renders an ast.Type, defensively handling a nil type node (an
// un-annotated declaration would produce this; stdlib exports are fully
// annotated, but we never panic in a docs command).
func typeString(t ast.Type) string {
	if t == nil {
		return "?"
	}
	return t.String()
}
