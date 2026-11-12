package main

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// exportMarks is what `ailang docs` shows about an export's promise: its tier
// and, for a deprecated export, its replacement (M-V1-STDLIB-SURFACE-POLISH
// §6.1 item 3).
//
// It is read from the parsed declaration (@experimental, @deprecated) and the
// module tier from std/STABILITY.toml, through the SAME iface.ModuleTier /
// iface.FuncTier the compiler uses to write the golden — so the two readers of
// the annotation cannot disagree on the rule. TestDocsTiersMatchGoldens checks
// they agree on the data too, for every std module.
type exportMarks struct {
	Tier       string
	ReplacedBy string
	Note       string
}

// tag renders the marks for a docs line: "[experimental]",
// "[deprecated → headOr]", both, or "" for a plain export of a module whose
// tier the module header already states.
func (m exportMarks) tag(moduleTier string) string {
	var parts []string
	if m.Tier != "" && m.Tier != moduleTier {
		parts = append(parts, "["+m.Tier+"]")
	}
	if m.ReplacedBy != "" {
		parts = append(parts, "[deprecated → "+m.ReplacedBy+"]")
	}
	return strings.Join(parts, " ")
}

// stdModuleName maps a stdlib file name ("list.ail", "ai/streaming.ail") to its
// module path ("std/list").
func stdModuleName(fileName string) string {
	return "std/" + strings.TrimSuffix(fileName, ".ail")
}

// parseExportMarks returns the marks of every exported function of one stdlib
// file, plus the module's own tier.
func parseExportMarks(stdlib fs.FS, fileName string) (map[string]exportMarks, string, error) {
	filePath := "std/" + fileName
	source, err := fs.ReadFile(stdlib, fileName)
	if err != nil {
		return nil, "", fmt.Errorf("cannot read %s: %w", filePath, err)
	}
	p := parser.New(lexer.New(string(source), filePath))
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		return nil, "", fmt.Errorf("cannot parse stdlib file %s: %v", filePath, errs[0])
	}
	if prog == nil || prog.File == nil {
		return nil, "", fmt.Errorf("cannot parse stdlib file %s: no AST produced", filePath)
	}
	modTier, err := iface.ModuleTier(stdModuleName(fileName))
	if err != nil {
		return nil, "", err
	}
	marks := map[string]exportMarks{}
	for _, fn := range prog.File.Funcs {
		if fn == nil || !fn.IsExport {
			continue
		}
		m := exportMarks{Tier: iface.FuncTier(modTier, fn)}
		if repl, note, ok := fn.Deprecation(); ok {
			m.ReplacedBy, m.Note = repl, note
		}
		marks[fn.Name] = m
	}
	return marks, modTier, nil
}
