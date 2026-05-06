// gen-error-codes parses internal/errors/codes.go and emits dist/error_codes.json
// (schema-v1). Run via: make error-codes
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
)

// ErrorRecord is one row in the output JSON.
type ErrorRecord struct {
	Code     string `json:"code"`
	Category string `json:"category"`
	Summary  string `json:"summary"`
	FixHint  string `json:"fix_hint"`
}

// ErrorCodesOutput is the top-level schema-v1 envelope.
type ErrorCodesOutput struct {
	SchemaVersion string        `json:"schema_version"`
	Records       []ErrorRecord `json:"records"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gen-error-codes <codes.go> <output.json>")
		os.Exit(1)
	}
	codesPath := os.Args[1]
	outPath := os.Args[2]

	records, err := parseErrorCodes(codesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-error-codes: %v\n", err)
		os.Exit(1)
	}

	out := ErrorCodesOutput{
		SchemaVersion: "v1",
		Records:       records,
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-error-codes: marshal: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(strings.TrimSuffix(outPath, "/"+outPath[strings.LastIndex(outPath, "/")+1:]), 0o755); err != nil {
		// ignore mkdir errors (path may already exist)
		_ = err
	}
	if err := os.WriteFile(outPath, append(data, '\n'), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "gen-error-codes: write: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("gen-error-codes: wrote %d records to %s\n", len(records), outPath)
}

// parseErrorCodes parses the given codes.go file and returns all ErrorRecords.
// It uses go/parser to extract string constant names + the leading comment that
// describes each constant, then queries the ErrorRegistry (via the same file's
// comments) for category and summary.
func parseErrorCodes(codesPath string) ([]ErrorRecord, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, codesPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", codesPath, err)
	}

	// Build comment map so each GenDecl's doc comments are accessible
	cmap := ast.NewCommentMap(fset, f, f.Comments)

	// Walk top-level const declarations
	type rawCode struct {
		code    string
		comment string // leading comment on the const
	}
	var raws []rawCode

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		for _, spec := range genDecl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if !isErrorCodeIdent(name.Name) {
					continue
				}
				// Value must be a string literal equal to the name
				if i >= len(vs.Values) {
					continue
				}
				bl, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || bl.Kind != token.STRING {
					continue
				}
				val := strings.Trim(bl.Value, `"`)
				if val != name.Name {
					continue
				}

				// Extract comment: prefer doc comment on the ValueSpec,
				// fall back to the GenDecl doc, then cmap comments.
				comment := ""
				if vs.Comment != nil {
					comment = strings.TrimSpace(vs.Comment.Text())
				} else if vs.Doc != nil {
					comment = strings.TrimSpace(vs.Doc.Text())
				} else if genDecl.Doc != nil && len(genDecl.Specs) == 1 {
					comment = strings.TrimSpace(genDecl.Doc.Text())
				} else if docs := cmap[genDecl]; len(docs) > 0 {
					comment = strings.TrimSpace(docs[0].Text())
				}
				// Strip leading "// " prefixes left by Text()
				comment = strings.TrimPrefix(comment, "// ")
				raws = append(raws, rawCode{code: name.Name, comment: comment})
			}
		}
	}

	// Also parse the ErrorRegistry map to get category + description
	registry := extractRegistry(f)

	// Build final records
	records := make([]ErrorRecord, 0, len(raws))
	for _, r := range raws {
		rec := ErrorRecord{Code: r.code}
		if info, ok := registry[r.code]; ok {
			rec.Category = info.category
			rec.Summary = info.description
		}
		// Populate summary and fix_hint from comment when registry doesn't cover them
		if rec.Summary == "" {
			rec.Summary = r.comment
		}
		if rec.FixHint == "" && r.comment != "" && r.comment != rec.Summary {
			rec.FixHint = r.comment
		}
		// Ensure non-empty category/summary even when registry entry is missing
		if rec.Category == "" {
			rec.Category = categoryFromCode(r.code)
		}
		// Guarantee fix_hint is always present — fall back to summary text
		if rec.FixHint == "" {
			rec.FixHint = rec.Summary
		}
		records = append(records, rec)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Code < records[j].Code
	})
	return records, nil
}

// registryInfo holds the category + description extracted from ErrorRegistry.
type registryInfo struct {
	category    string
	description string
}

// extractRegistry walks the ErrorRegistry var declaration and extracts
// the Category and Description fields for each error code.
func extractRegistry(f *ast.File) map[string]registryInfo {
	result := make(map[string]registryInfo)

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}
		for _, spec := range genDecl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			// Look for: var ErrorRegistry = map[string]ErrorInfo{...}
			for _, name := range vs.Names {
				if name.Name != "ErrorRegistry" {
					continue
				}
				for _, val := range vs.Values {
					cl, ok := val.(*ast.CompositeLit)
					if !ok {
						continue
					}
					for _, elt := range cl.Elts {
						kv, ok := elt.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						// Key is the error code constant (e.g., PAR001)
						keyIdent, ok := kv.Key.(*ast.Ident)
						if !ok {
							continue
						}
						// Value is ErrorInfo{code, phase, category, description}
						inner, ok := kv.Value.(*ast.CompositeLit)
						if !ok || len(inner.Elts) < 4 {
							continue
						}
						// Positional: {Code, Phase, Category, Description}
						catLit, ok := inner.Elts[2].(*ast.BasicLit)
						if !ok {
							continue
						}
						descLit, ok := inner.Elts[3].(*ast.BasicLit)
						if !ok {
							continue
						}
						result[keyIdent.Name] = registryInfo{
							category:    strings.Trim(catLit.Value, `"`),
							description: strings.Trim(descLit.Value, `"`),
						}
					}
				}
			}
		}
	}
	return result
}

// isErrorCodeIdent returns true for identifiers that look like AILANG error codes
// (2-4 uppercase letters followed by 2-4 digits, e.g. PAR001, MOD013, RT007).
func isErrorCodeIdent(s string) bool {
	if len(s) < 4 || len(s) > 8 {
		return false
	}
	i := 0
	for i < len(s) && s[i] >= 'A' && s[i] <= 'Z' {
		i++
	}
	if i < 2 || i > 4 {
		return false
	}
	j := i
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	return j == len(s) && (j-i) >= 2
}

// categoryFromCode derives a fallback category from the error code prefix.
func categoryFromCode(code string) string {
	i := 0
	for i < len(code) && code[i] >= 'A' && code[i] <= 'Z' {
		i++
	}
	switch code[:i] {
	case "PAR":
		return "parser"
	case "MOD":
		return "module"
	case "LDR":
		return "loader"
	case "IMP":
		return "import"
	case "DSG":
		return "desugar"
	case "TC":
		return "typecheck"
	case "ELB":
		return "elaborate"
	case "LNK":
		return "link"
	case "EVA":
		return "eval"
	case "RT":
		return "runtime"
	default:
		return "unknown"
	}
}
