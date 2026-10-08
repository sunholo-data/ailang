package apiserver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// M-MCP-FILE-HANDOFF F1b: @mcp_file("param", ...) declares that a tool param
// takes a file the host hands over, in OpenAI's openai/fileParams shape.
//
// OpenAI Apps SDK reference (https://developers.openai.com/plugins/reference,
// fetched 2026-10-07):
//
//	"_meta[\"openai/fileParams\"] — string[] — List of top-level input fields
//	 that represent files. Each field receives { download_url, file_id,
//	 mime_type?, file_name? }."
//	"Every file object schema must declare all four supported properties:
//	 download_url (required), file_id (required), mime_type, file_name … The
//	 Scan Tools step and plugin submission reject a file schema that omits any
//	 of the four properties, does not require download_url and file_id, marks
//	 either optional property as required, or requires a property other than
//	 download_url or file_id."
//	"ChatGPT always includes download_url and file_id; it may omit mime_type
//	 and file_name."
//
// The AILANG param is the closed record
//
//	{download_url: string, file_id: string, mime_type: string, file_name: string}
//
// written inline or as a same-module type alias. A record (not Json) because
// MCP arguments already bind JSON objects to records, and the function then
// reads file.download_url with no decoding. The two optional fields bind ""
// when the host omits them, so the record is always complete.
//
// @optional interplay: a file param is required unless it is also @optional;
// an omitted @optional file param binds the empty file record (all four
// fields ""), on MCP and REST alike — the zero of its declared record type
// (param_zero.go), never the empty record {} the type name "record" gives.
//
// Both MCP surfaces (/mcp/ and /mcp/connect/) carry the _meta and the file
// schema: the shape is the only one a host can fill, and hosts that do not
// know openai/fileParams ignore _meta.

// fileParamsMetaKey is the tool-descriptor _meta key ChatGPT reads.
const fileParamsMetaKey = "openai/fileParams"

// fileFields are the file object's properties: the first two are required.
var fileFields = []string{"download_url", "file_id", "mime_type", "file_name"}

const fileRecordShape = "{download_url: string, file_id: string, mime_type: string, file_name: string}"

// fileObjectSchema is OpenAI's documented file object schema ($defs/OpenAIFile
// in the reference's example), inlined at the param. A fresh map per call:
// callers may mutate the schema they are given.
func fileObjectSchema() map[string]any {
	props := make(map[string]any, len(fileFields))
	for _, f := range fileFields {
		props[f] = map[string]any{"type": "string"}
	}
	return map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             []string{"download_url", "file_id"},
		"additionalProperties": false,
	}
}

// emptyFileRecord is the base a supplied file object is normalised onto.
func emptyFileRecord() map[string]any {
	m := make(map[string]any, len(fileFields))
	for _, f := range fileFields {
		m[f] = ""
	}
	return m
}

// extractMCPFileAnnotations records @mcp_file names on each export and checks
// them against the signature. Every problem is a load error: a tool that
// advertises a file param it cannot bind would fail on every call.
func extractMCPFileAnnotations(modInfo *ModuleInfo, file *ast.File) error {
	aliases := recordAliases(file)
	for _, fn := range file.Funcs {
		var names []string
		for _, a := range fn.Annotations {
			if a.Name == "mcp_file" {
				names = append(names, stringLitArgs(a)...)
			}
		}
		if len(names) == 0 {
			continue
		}
		idx := -1
		for i := range modInfo.Exports {
			if modInfo.Exports[i].Name == fn.Name {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("@mcp_file on %s: only exported functions are MCP tools", fn.Name)
		}
		e := &modInfo.Exports[idx]
		seen := map[string]bool{}
		for _, name := range names {
			if seen[name] {
				return fmt.Errorf("@mcp_file(%q) on %s: named twice", name, fn.Name)
			}
			seen[name] = true
			if err := checkFileParam(fn, e, name, aliases); err != nil {
				return fmt.Errorf("%s.%s: %w", modInfo.Path, fn.Name, err)
			}
		}
		e.MCPFile = names
	}
	return nil
}

func checkFileParam(fn *ast.FuncDecl, e *ExportInfo, name string, aliases map[string]*ast.RecordType) error {
	pi := -1
	for i, p := range fn.Params {
		if p.Name == name {
			pi = i
			break
		}
	}
	params := make([]string, len(fn.Params))
	for i, p := range fn.Params {
		params[i] = p.Name
	}
	switch {
	case name == headersParam:
		return fmt.Errorf("@mcp_file(%q): _headers is bound from the request, not supplied by the client", name)
	case pi < 0:
		return fmt.Errorf("@mcp_file(%q): no such parameter (params: %s)", name, strings.Join(params, ", "))
	case isSecretParam(*e, name):
		return fmt.Errorf("@mcp_file(%q): a param cannot be both @mcp_file and @mcp_secret", name)
	}
	rec, ok := fn.Params[pi].Type.(*ast.RecordType)
	if !ok {
		if st, isSimple := fn.Params[pi].Type.(*ast.SimpleType); isSimple {
			rec = aliases[st.Name]
		}
	}
	if issue := fileRecordIssue(rec); issue != "" {
		return fmt.Errorf("@mcp_file(%q): %s; declare the param as %s", name, issue, fileRecordShape)
	}
	// A same-module alias reads as its name; bind it as the record it is.
	if pi < len(e.ParamTypes) {
		e.ParamTypes[pi] = "record"
	}
	return nil
}

// fileRecordIssue is why rec is not the closed four-string file record ("" =
// it is).
func fileRecordIssue(rec *ast.RecordType) string {
	if rec == nil {
		return "the param is not a record"
	}
	if rec.Row != nil {
		return "the record is open (| r); it must be closed"
	}
	got := map[string]bool{}
	for _, f := range rec.Fields {
		st, ok := f.Type.(*ast.SimpleType)
		if !ok || st.Name != "string" {
			return fmt.Sprintf("field %s must be string", f.Name)
		}
		got[f.Name] = true
	}
	var missing, extra []string
	for _, f := range fileFields {
		if !got[f] {
			missing = append(missing, f)
		}
		delete(got, f)
	}
	for f := range got {
		extra = append(extra, f)
	}
	sort.Strings(extra)
	switch {
	case len(missing) > 0:
		return "the record lacks " + strings.Join(missing, ", ")
	case len(extra) > 0:
		return "the record has fields OpenAI does not send: " + strings.Join(extra, ", ")
	}
	return ""
}

// recordAliases maps each same-module `type X = {…}` to its record.
func recordAliases(file *ast.File) map[string]*ast.RecordType {
	out := map[string]*ast.RecordType{}
	for _, d := range file.Decls {
		td, ok := d.(*ast.TypeDecl)
		if !ok || len(td.TypeParams) > 0 {
			continue
		}
		switch def := td.Definition.(type) {
		case *ast.RecordType:
			out[td.Name] = def
		case *ast.TypeAlias:
			if rec, ok := def.Target.(*ast.RecordType); ok {
				out[td.Name] = rec
			}
		}
	}
	return out
}

func isFileParam(e ExportInfo, name string) bool {
	for _, f := range e.MCPFile {
		if f == name {
			return true
		}
	}
	return false
}

// applyFileSchema swaps each file param's property for the file object schema.
func applyFileSchema(schema map[string]any, e ExportInfo) {
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return
	}
	for _, name := range e.MCPFile {
		if _, present := props[name]; present {
			props[name] = fileObjectSchema()
		}
	}
}

// fileParamsMeta is the tool's _meta entry, or nil when it has no file param.
func fileParamsMeta(e ExportInfo) []string {
	if len(e.MCPFile) == 0 {
		return nil
	}
	return append([]string(nil), e.MCPFile...)
}

// bindFileArg normalises a client-supplied file object to the four-string
// record: the required fields must be strings, the optional ones default to
// "" and must be strings when present, and anything else the host adds is
// dropped (the AILANG record is closed).
func bindFileArg(name string, v any) (map[string]any, error) {
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("parameter %q is a file object {download_url, file_id, mime_type?, file_name?}, got %s", name, jsonKind(v))
	}
	out := emptyFileRecord()
	for i, f := range fileFields {
		raw, present := obj[f]
		if !present || raw == nil {
			if i < 2 {
				return nil, fmt.Errorf("parameter %q: the file object has no %s", name, f)
			}
			continue
		}
		s, isStr := raw.(string)
		if !isStr {
			return nil, fmt.Errorf("parameter %q: %s must be a string, got %s", name, f, jsonKind(raw))
		}
		out[f] = s
	}
	return out, nil
}

func jsonKind(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case string:
		return "a string"
	case float64:
		return "a number"
	case bool:
		return "a boolean"
	case []any:
		return "an array"
	case map[string]any:
		return "an object"
	default:
		return fmt.Sprintf("%T", v)
	}
}

// bindFileArgs replaces each supplied file param in args with its normalised
// record. args is in param order (named or positional binding).
func bindFileArgs(e ExportInfo, args []any) error {
	for i, name := range e.ParamNames {
		if i >= len(args) || !isFileParam(e, name) {
			continue
		}
		rec, err := bindFileArg(name, args[i])
		if err != nil {
			return err
		}
		args[i] = rec
	}
	return nil
}
