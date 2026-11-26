package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/types"
)

// inspectOptions configures a single `ailang inspect` invocation.
type inspectOptions struct {
	Format       string // "agent" (default) or "json"
	ExportsOnly  bool
	RelaxModules bool
}

// inspectReport is the top-level JSON schema for `ailang inspect --json`.
type inspectReport struct {
	Files []inspectFileReport `json:"files"`
}

// inspectFileReport describes one inspected `.ail` module file.
type inspectFileReport struct {
	File    string              `json:"file"`
	Module  string              `json:"module"`
	Types   []inspectTypeInfo   `json:"types"`
	Symbols []inspectSymbolInfo `json:"symbols"`
}

// inspectTypeInfo describes a declared type (record, ADT, or alias).
type inspectTypeInfo struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	Exported   bool     `json:"exported"`
	Line       int      `json:"line"`
	Definition string   `json:"definition"`
	Deriving   []string `json:"deriving"`

	typeParams []string
}

// inspectContractInfo describes one requires/ensures/invariant contract clause.
type inspectContractInfo struct {
	Kind string `json:"kind"`
	Expr string `json:"expr"`
}

// inspectSymbolInfo describes one top-level function declaration.
type inspectSymbolInfo struct {
	Name      string                `json:"name"`
	Kind      string                `json:"kind"`
	Exported  bool                  `json:"exported"`
	Pure      bool                  `json:"pure"`
	Line      int                   `json:"line"`
	Signature string                `json:"signature"`
	Effects   []string              `json:"effects"`
	Contracts []inspectContractInfo `json:"contracts"`
	Calls     []string              `json:"calls"`

	paramList  string
	returnType string
}

// runInspectCommand implements `ailang inspect <file-or-dir> [flags]` (and `ailang dev inspect`).
func runInspectCommand(args []string) error {
	return runInspectCommandTo(args, os.Stdout)
}

func runInspectCommandTo(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output structured JSON")
	formatFlag := fs.String("format", "agent", "Output format: agent (default) or json")
	exportsOnly := fs.Bool("exports-only", false, "Show only exported types and symbols")
	exportedOnly := fs.Bool("exported-only", false, "Alias for --exports-only")
	relaxModules := fs.Bool("relax-modules", false, "Relax MOD010 module path validation")
	helpFlag := fs.Bool("help", false, "Show help for inspect command")
	shortHelpFlag := fs.Bool("h", false, "Show help for inspect command")

	paths, err := parseInspectArgs(fs, args)
	if err != nil {
		return fmt.Errorf("%w\nRun 'ailang inspect --help' for usage", err)
	}
	if *helpFlag || *shortHelpFlag {
		printInspectHelp(out)
		return nil
	}
	if len(paths) == 0 {
		return fmt.Errorf("missing file or directory argument\nUsage: ailang inspect <file-or-dir> [--json | --format=agent | --exports-only | --relax-modules]")
	}

	format := strings.ToLower(strings.TrimSpace(*formatFlag))
	if *jsonFlag {
		format = "json"
	}
	if format != "agent" && format != "json" {
		return fmt.Errorf("unsupported --format %q (expected 'agent' or 'json')", *formatFlag)
	}

	opts := inspectOptions{
		Format:       format,
		ExportsOnly:  *exportsOnly || *exportedOnly,
		RelaxModules: *relaxModules || config.RelaxModules(),
	}

	files, err := collectInspectFiles(paths)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no .ail files found in %s", strings.Join(paths, ", "))
	}

	report := inspectReport{
		Files: make([]inspectFileReport, 0, len(files)),
	}
	for _, filePath := range files {
		fileRep, err := inspectFile(filePath, opts)
		if err != nil {
			return err
		}
		report.Files = append(report.Files, *fileRep)
	}

	if opts.Format == "json" {
		var buf strings.Builder
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return fmt.Errorf("marshal inspect report: %w", err)
		}
		_, err = io.WriteString(out, buf.String())
		return err
	}

	_, err = io.WriteString(out, renderInspectAgent(report))
	return err
}

func printInspectHelp(out io.Writer) {
	fmt.Fprintln(out, "Usage: ailang inspect <file-or-dir> [flags]")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Inspect an AILANG module or directory and emit a compact semantic X-ray:")
	fmt.Fprintln(out, "declared types, exported and local functions, resolved signatures, ! {Effects},")
	fmt.Fprintln(out, "requires/ensures contracts, source line numbers, and direct callees.")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Flags:")
	fmt.Fprintln(out, "  --format string     Output format: agent (default) or json")
	fmt.Fprintln(out, "  --json              Shorthand for --format=json")
	fmt.Fprintln(out, "  --exports-only      Show only exported types and functions (alias: --exported-only)")
	fmt.Fprintln(out, "  --relax-modules     Relax MOD010 module path validation")
	fmt.Fprintln(out, "  --help, -h          Show this help message")
}

// parseInspectArgs parses flags in any position relative to positional file/dir paths.
func parseInspectArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var paths []string
	rem := args
	for len(rem) > 0 {
		if rem[0] == "--" {
			paths = append(paths, rem[1:]...)
			break
		}
		if strings.HasPrefix(rem[0], "-") && rem[0] != "-" {
			if err := fs.Parse(rem); err != nil {
				return nil, err
			}
			rem = fs.Args()
			continue
		}
		paths = append(paths, rem[0])
		rem = rem[1:]
	}
	return paths, nil
}

// collectInspectFiles resolves file/directory arguments into a deterministic sorted list of .ail files.
func collectInspectFiles(paths []string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("cannot access '%s': %w", p, err)
		}
		if !info.IsDir() {
			if !seen[p] {
				seen[p] = true
				files = append(files, p)
			}
			continue
		}
		var dirFiles []string
		err = filepath.Walk(p, func(path string, fi os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if fi.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".ail") {
				return nil
			}
			if pkg.IsNamedTestBodyFile(fi.Name()) {
				return nil
			}
			dirFiles = append(dirFiles, path)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("cannot walk directory '%s': %w", p, err)
		}
		sort.Strings(dirFiles)
		for _, f := range dirFiles {
			if !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}
	return files, nil
}

// inspectFile runs the compiler pipeline in ModeCheck (no evaluation, no Z3) and extracts
// types, functions, resolved signatures, effects, contracts, and direct callees.
func inspectFile(filename string, opts inspectOptions) (*inspectFileReport, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("cannot read file '%s': %w", filename, err)
	}

	cfg := pipeline.Config{
		Mode:         pipeline.ModeCheck,
		DryLink:      true,
		RelaxModules: opts.RelaxModules,
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	}

	res, err := pipeline.Run(cfg, src)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}
	if len(res.Errors) > 0 {
		return nil, fmt.Errorf("%s: %v", filename, res.Errors[0])
	}

	report := &inspectFileReport{
		File:    filename,
		Types:   []inspectTypeInfo{},
		Symbols: []inspectSymbolInfo{},
	}

	astFile := res.Artifacts.AST
	if astFile != nil && astFile.Module != nil {
		report.Module = astFile.Module.Path
	} else if res.Interface != nil {
		report.Module = res.Interface.Module
	}

	if astFile == nil {
		return report, nil
	}

	// Collect type declarations in source order.
	report.Types = extractInspectTypes(astFile, opts.ExportsOnly)

	// Index Core declarations and local function names for call-graph and type lookup.
	coreDecls := indexCoreDecls(res.Artifacts.Core)
	localFuncs := make(map[string]bool, len(astFile.Funcs))
	for _, fn := range astFile.Funcs {
		localFuncs[fn.Name] = true
	}

	// Collect function declarations in source order.
	for _, fn := range astFile.Funcs {
		if opts.ExportsOnly && !fn.IsExport {
			continue
		}
		sym := extractInspectSymbol(fn, &res, coreDecls, localFuncs)
		report.Symbols = append(report.Symbols, sym)
	}

	return report, nil
}

func extractInspectTypes(file *ast.File, exportsOnly bool) []inspectTypeInfo {
	var decls []*ast.TypeDecl
	seen := make(map[*ast.TypeDecl]bool)
	collect := func(nodes []ast.Node) {
		for _, n := range nodes {
			if td, ok := n.(*ast.TypeDecl); ok && !seen[td] {
				seen[td] = true
				decls = append(decls, td)
			}
		}
	}
	collect(file.Statements)
	collect(file.Decls)

	out := make([]inspectTypeInfo, 0, len(decls))
	for _, td := range decls {
		if exportsOnly && !td.Exported {
			continue
		}
		kind, def := formatTypeDef(td.Definition)
		deriving := make([]string, 0, len(td.Deriving))
		for _, dk := range td.Deriving {
			if s := dk.String(); s != "" {
				deriving = append(deriving, s)
			}
		}
		out = append(out, inspectTypeInfo{
			Name:       td.Name,
			Kind:       kind,
			Exported:   td.Exported,
			Line:       td.Pos.Line,
			Definition: def,
			Deriving:   deriving,
			typeParams: td.TypeParams,
		})
	}
	return out
}

func formatTypeDef(def ast.TypeDef) (string, string) {
	switch d := def.(type) {
	case *ast.RecordType:
		return "record", d.String()
	case *ast.AlgebraicType:
		ctors := make([]string, 0, len(d.Constructors))
		for _, c := range d.Constructors {
			if len(c.Fields) == 0 {
				ctors = append(ctors, c.Name)
				continue
			}
			fields := make([]string, len(c.Fields))
			for i, f := range c.Fields {
				if f.Name != "" {
					fields[i] = fmt.Sprintf("%s: %s", f.Name, f.Type.String())
				} else {
					fields[i] = f.Type.String()
				}
			}
			ctors = append(ctors, fmt.Sprintf("%s(%s)", c.Name, strings.Join(fields, ", ")))
		}
		return "adt", strings.Join(ctors, " | ")
	case *ast.TypeAlias:
		if d.Target != nil {
			return "alias", d.Target.String()
		}
		return "alias", ""
	default:
		return "type", ""
	}
}

func indexCoreDecls(prog *core.Program) map[string]core.CoreExpr {
	decls := make(map[string]core.CoreExpr)
	if prog == nil {
		return decls
	}
	for _, d := range prog.Decls {
		switch x := d.(type) {
		case *core.Let:
			decls[x.Name] = x.Value
		case *core.LetRec:
			for _, b := range x.Bindings {
				decls[b.Name] = b.Value
			}
		}
	}
	return decls
}

func extractInspectSymbol(
	fn *ast.FuncDecl,
	res *pipeline.Result,
	coreDecls map[string]core.CoreExpr,
	localFuncs map[string]bool,
) inspectSymbolInfo {
	resolvedType := resolveFuncType(fn, res, coreDecls)
	paramTypes, retType, effects := decomposeFuncType(fn, resolvedType)

	// Build signature string: "(T1, T2) -> Ret" [+ " ! {Effects}"]
	sig := fmt.Sprintf("(%s) -> %s", strings.Join(paramTypes, ", "), retType)
	if len(effects) > 0 {
		sig += fmt.Sprintf(" ! {%s}", strings.Join(effects, ", "))
	}

	// Build named parameter list for agent format: "p1: T1, p2: T2"
	namedParams := make([]string, 0, len(fn.Params))
	for i, p := range fn.Params {
		pType := ""
		if i < len(paramTypes) {
			pType = paramTypes[i]
		} else if p.Type != nil {
			pType = p.Type.String()
		}
		if p.Name != "" && pType != "" {
			namedParams = append(namedParams, fmt.Sprintf("%s: %s", p.Name, pType))
		} else if pType != "" {
			namedParams = append(namedParams, pType)
		} else {
			namedParams = append(namedParams, p.Name)
		}
	}

	contracts := extractContracts(fn, res.Artifacts.Core)
	calls := collectDirectCalls(coreDecls[fn.Name], localFuncs)

	return inspectSymbolInfo{
		Name:       fn.Name,
		Kind:       "func",
		Exported:   fn.IsExport,
		Pure:       fn.IsPure,
		Line:       fn.Pos.Line,
		Signature:  sig,
		Effects:    effects,
		Contracts:  contracts,
		Calls:      calls,
		paramList:  strings.Join(namedParams, ", "),
		returnType: retType,
	}
}

func resolveFuncType(fn *ast.FuncDecl, res *pipeline.Result, coreDecls map[string]core.CoreExpr) types.Type {
	if res.Interface != nil {
		if item, ok := res.Interface.Exports[fn.Name]; ok && item != nil && item.Type != nil && item.Type.Type != nil {
			return item.Type.Type
		}
	}
	if coreVal, ok := coreDecls[fn.Name]; ok && res.Artifacts.CoreTI != nil {
		if t, found := res.Artifacts.CoreTI.GetForExpr(coreVal); found && t != nil {
			return t
		}
		if da, isDict := coreVal.(*core.DictAbs); isDict {
			if t, found := res.Artifacts.CoreTI.GetForExpr(da.Body); found && t != nil {
				return t
			}
		}
	}
	return nil
}

func decomposeFuncType(fn *ast.FuncDecl, resolved types.Type) (paramTypes []string, retType string, effects []string) {
	effects = []string{}
	if tf, ok := resolved.(*types.TFunc2); ok {
		paramTypes = make([]string, len(tf.Params))
		for i, pt := range tf.Params {
			paramTypes[i] = pt.String()
		}
		if tf.Return != nil {
			retType = tf.Return.String()
		}
		effects = extractRowEffects(tf.EffectRow, fn.Effects)
		return paramTypes, retType, effects
	}

	// Fallback to surface AST annotations when CoreTI/Interface does not yield a TFunc2.
	paramTypes = make([]string, len(fn.Params))
	for i, p := range fn.Params {
		if p.Type != nil {
			paramTypes[i] = p.Type.String()
		} else {
			paramTypes[i] = "_"
		}
	}
	if fn.ReturnType != nil {
		retType = fn.ReturnType.String()
	} else if resolved != nil {
		retType = resolved.String()
	} else {
		retType = "_"
	}
	for _, eff := range fn.Effects {
		effects = append(effects, eff.String())
	}
	sort.Strings(effects)
	return paramTypes, retType, effects
}

func extractRowEffects(row *types.Row, astEffects []ast.EffectAnnotation) []string {
	effects := []string{}
	if row != nil {
		keys := make([]string, 0, len(row.Labels))
		for k := range row.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			effects = append(effects, formatRowEffectLabel(row, k))
		}
	}
	// Preserve explicit user-written effect row variables (e.g. `! {e}` on higher-order functions)
	// without leaking compiler-synthesized fresh row variables like `ε12`.
	for _, ae := range astEffects {
		if ae.IsRowVar && ae.Name != "" {
			effects = append(effects, ae.Name)
		}
	}
	return effects
}

func formatRowEffectLabel(r *types.Row, k string) string {
	head := k
	if r.Params != nil {
		if pmap, ok := r.Params[k]; ok && len(pmap) > 0 {
			pkeys := make([]string, 0, len(pmap))
			for pk := range pmap {
				pkeys = append(pkeys, pk)
			}
			sort.Strings(pkeys)
			parts := make([]string, len(pkeys))
			for i, pk := range pkeys {
				parts[i] = fmt.Sprintf("%s=%s", pk, pmap[pk])
			}
			head = fmt.Sprintf("%s[%s]", k, strings.Join(parts, ", "))
		}
	}
	parts := []string{head}
	if r.MinBudgets != nil {
		if min, ok := r.MinBudgets[k]; ok && min != nil {
			parts = append(parts, fmt.Sprintf("@min=%d", *min))
		}
	}
	if r.Budgets != nil {
		if budget, ok := r.Budgets[k]; ok && budget != nil {
			parts = append(parts, fmt.Sprintf("@limit=%d", *budget))
		}
	}
	return strings.Join(parts, " ")
}

func extractContracts(fn *ast.FuncDecl, prog *core.Program) []inspectContractInfo {
	contracts := []inspectContractInfo{}
	for _, prop := range fn.Properties {
		if prop == nil || prop.Kind == ast.PropertyKind {
			continue
		}
		exprStr := ""
		if prop.Expr != nil {
			exprStr = prop.Expr.String()
		}
		contracts = append(contracts, inspectContractInfo{
			Kind: prop.Kind.String(),
			Expr: exprStr,
		})
	}
	if len(contracts) > 0 || prog == nil || prog.Meta == nil {
		return contracts
	}
	if meta, ok := prog.Meta[fn.Name]; ok && meta != nil {
		for _, c := range meta.Contracts {
			if c == nil {
				continue
			}
			exprStr := c.Message
			if exprStr == "" && c.Expr != nil {
				exprStr = c.Expr.String()
			}
			contracts = append(contracts, inspectContractInfo{
				Kind: c.Kind.String(),
				Expr: exprStr,
			})
		}
	}
	return contracts
}

// collectDirectCalls walks a function's Core AST and returns sorted unique direct callees.
func collectDirectCalls(expr core.CoreExpr, localFuncs map[string]bool) []string {
	if expr == nil {
		return []string{}
	}
	seen := make(map[string]bool)
	aliases := make(map[string]string)

	var walk func(e core.CoreExpr)
	walk = func(e core.CoreExpr) {
		if e == nil {
			return
		}
		switch x := e.(type) {
		case *core.Lambda:
			walk(x.Body)
		case *core.DictAbs:
			walk(x.Body)
		case *core.Let:
			if target, ok := resolveCalleeExpr(x.Value, localFuncs, aliases); ok {
				aliases[x.Name] = target
			}
			walk(x.Value)
			walk(x.Body)
		case *core.LetRec:
			for _, b := range x.Bindings {
				walk(b.Value)
			}
			walk(x.Body)
		case *core.App:
			if target, ok := resolveCalleeExpr(x.Func, localFuncs, aliases); ok {
				seen[target] = true
			}
			walk(x.Func)
			for _, a := range x.Args {
				walk(a)
			}
		case *core.If:
			walk(x.Cond)
			walk(x.Then)
			walk(x.Else)
		case *core.Match:
			walk(x.Scrutinee)
			for _, arm := range x.Arms {
				walk(arm.Guard)
				walk(arm.Body)
			}
		case *core.BinOp:
			walk(x.Left)
			walk(x.Right)
		case *core.UnOp:
			walk(x.Operand)
		case *core.Intrinsic:
			for _, a := range x.Args {
				walk(a)
			}
		case *core.Record:
			for _, v := range x.Fields {
				walk(v)
			}
		case *core.RecordAccess:
			walk(x.Record)
		case *core.RecordUpdate:
			walk(x.Base)
			for _, v := range x.Updates {
				walk(v)
			}
		case *core.List:
			for _, elem := range x.Elements {
				walk(elem)
			}
		case *core.Array:
			for _, elem := range x.Elements {
				walk(elem)
			}
		case *core.Tuple:
			for _, elem := range x.Elements {
				walk(elem)
			}
		case *core.DictApp:
			walk(x.Dict)
			for _, a := range x.Args {
				walk(a)
			}
		case *core.Forall:
			walk(x.Lo)
			walk(x.Hi)
			walk(x.Body)
		}
	}

	walk(expr)

	calls := make([]string, 0, len(seen))
	for name := range seen {
		calls = append(calls, name)
	}
	sort.Strings(calls)
	return calls
}

func resolveCalleeExpr(e core.CoreExpr, localFuncs map[string]bool, aliases map[string]string) (string, bool) {
	switch x := e.(type) {
	case *core.VarGlobal:
		if strings.HasPrefix(x.Ref.Module, "$") {
			return "", false
		}
		if x.Ref.Module == "" {
			return x.Ref.Name, x.Ref.Name != ""
		}
		return x.Ref.Module + "." + x.Ref.Name, true
	case *core.Var:
		if target, ok := aliases[x.Name]; ok {
			return target, true
		}
		if localFuncs[x.Name] {
			return x.Name, true
		}
	}
	return "", false
}

// renderInspectAgent formats an inspectReport in dense 1-line-per-declaration agent format.
func renderInspectAgent(report inspectReport) string {
	var b strings.Builder
	for i, f := range report.Files {
		if i > 0 {
			b.WriteByte('\n')
		}
		if f.Module != "" {
			fmt.Fprintf(&b, "module %s (%s)\n", f.Module, f.File)
		} else {
			fmt.Fprintf(&b, "file %s\n", f.File)
		}

		for _, t := range f.Types {
			name := t.Name
			if len(t.typeParams) > 0 {
				name = fmt.Sprintf("%s[%s]", t.Name, strings.Join(t.typeParams, ", "))
			}
			line := fmt.Sprintf("type %s = %s", name, t.Definition)
			if len(t.Deriving) > 0 {
				line += fmt.Sprintf(" deriving (%s)", strings.Join(t.Deriving, ", "))
			}
			var meta []string
			if t.Exported {
				meta = append(meta, "export")
			}
			if t.Line > 0 {
				meta = append(meta, fmt.Sprintf("line %d", t.Line))
			}
			if len(meta) > 0 {
				line += fmt.Sprintf(" [%s]", strings.Join(meta, ", "))
			}
			b.WriteString(line)
			b.WriteByte('\n')
		}

		for _, s := range f.Symbols {
			var parts []string
			if s.Exported {
				parts = append(parts, "export")
			}
			parts = append(parts, fmt.Sprintf("func %s(%s) -> %s ! {%s}",
				s.Name, s.paramList, s.returnType, strings.Join(s.Effects, ", ")))
			for _, c := range s.Contracts {
				parts = append(parts, fmt.Sprintf("%s { %s }", c.Kind, c.Expr))
			}
			var meta []string
			if s.Line > 0 {
				meta = append(meta, fmt.Sprintf("line %d", s.Line))
			}
			if len(s.Calls) > 0 {
				meta = append(meta, "calls: "+strings.Join(s.Calls, ", "))
			}
			if len(meta) > 0 {
				parts = append(parts, fmt.Sprintf("[%s]", strings.Join(meta, ", ")))
			}
			b.WriteString(strings.Join(parts, " "))
			b.WriteByte('\n')
		}
	}
	return b.String()
}
