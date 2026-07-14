package main

import (
	"fmt"
	"os"
	"strings"
)

// allFunctionsCommand implements `ailang docs --all-functions [filter]`: one
// deterministic, grep-able line per stdlib export (plus the prelude pseudo-module),
// signatures rendered from the AST. This is the one-shot, whole-stdlib view for
// agents/humans/grep pipelines that do NOT get the 91KB canonical prompt.
//
// Output line form:
//
//	std/clock.now: () -> int ! {Clock} -- Returns epoch time...
//	prelude.println: (string) -> () ! {IO} -- Print with newline (no import needed)
//
// Modules are sorted (discoverModules sorts); exports are in file (declaration)
// order. A stdlib file that does not parse fails the command loudly (non-zero
// exit, file named) — never a silently dropped or partial row.
func allFunctionsCommand(stdlibPath, filter string) {
	lines := buildAllFunctionsLines(stdlibPath)
	lowerFilter := strings.ToLower(filter)

	for _, line := range lines {
		if filter != "" && !strings.Contains(strings.ToLower(line), lowerFilter) {
			continue
		}
		fmt.Println(line)
	}
}

// buildAllFunctionsLines renders every stdlib export line (prelude first, then
// modules in sorted order, exports in file order). It exits non-zero on a
// stdlib parse failure. Returned lines are the FULL rendered line (module +
// name + signature + description) so the caller can substring-filter over all
// of it.
func buildAllFunctionsLines(stdlibPath string) []string {
	var lines []string

	// Prelude pseudo-module first (single source of truth: M4's renderer).
	for _, e := range preludeDocEntries() {
		lines = append(lines, formatFunctionLine("prelude", e.name, e.signature, e.doc))
	}

	// discoverModules is already sorted by module name (docs.go).
	modules := discoverModules(stdlibPath)
	for _, mod := range modules {
		astSigs, order, err := parseExportSignatures(mod.FilePath)
		if err != nil {
			// Loud failure: name the file, non-zero exit. Never drop a row.
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

		// Doc-comment prose is only available via the regex scan (the AST
		// carries no comments); map export name -> doc line.
		docByName := make(map[string]string, len(mod.Exports))
		for _, exp := range mod.Exports {
			docByName[exp.Name] = exp.DocLine
		}

		// Exports in file (declaration) order.
		for _, name := range order {
			lines = append(lines, formatFunctionLine(mod.Name, name, astSigs[name], docByName[name]))
		}
	}

	return lines
}

// formatFunctionLine renders one grep-able line. The signature already carries
// the leading function name (renderFuncSignature emits `name(...) -> ...`), so
// we prefix `module.` and strip the redundant leading name from the signature
// to produce `module.name: sig`. Description (if any) is appended as ` -- doc`.
func formatFunctionLine(module, name, signature, doc string) string {
	// signature is `name[...]( ... ) -> ...`; drop the leading name so the line
	// reads `module.name: (params) -> ret` without doubling the name.
	sig := strings.TrimPrefix(signature, name)
	var sb strings.Builder
	sb.WriteString(module)
	sb.WriteString(".")
	sb.WriteString(name)
	sb.WriteString(": ")
	sb.WriteString(strings.TrimSpace(sig))
	if doc != "" {
		sb.WriteString(" -- ")
		sb.WriteString(doc)
	}
	return sb.String()
}
