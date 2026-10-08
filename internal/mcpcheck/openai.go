package mcpcheck

import (
	"fmt"
	"sort"
	"strings"
)

// ---- check 6 (openai): openai/fileParams schemas (O9) ----

// fileProps are the file object's four properties; the first two are required.
var fileProps = []string{"download_url", "file_id", "mime_type", "file_name"}

func checkFileParams(tools []tool) []Finding {
	var fs []Finding
	declared := 0
	for _, t := range tools {
		raw, ok := t.Meta["openai/fileParams"]
		if !ok {
			continue
		}
		declared++
		names, ok := stringList(raw)
		if !ok {
			fs = append(fs, Finding{"file-params", Fail, t.Name, `_meta["openai/fileParams"] is not a list of strings`, "O9"})
			continue
		}
		props, _ := t.InputSchema["properties"].(map[string]any)
		defs, _ := t.InputSchema["$defs"].(map[string]any)
		for _, name := range names {
			schema, ok := props[name].(map[string]any)
			if !ok {
				fs = append(fs, Finding{"file-params", Fail, t.Name, fmt.Sprintf("file param %q is not a top-level property", name), "O9"})
				continue
			}
			if issue := fileSchemaIssue(schema, defs); issue != "" {
				fs = append(fs, Finding{"file-params", Fail, t.Name, fmt.Sprintf("file param %q: %s", name, issue), "O9"})
			}
		}
	}
	if len(fs) == 0 {
		msg := "no tool declares openai/fileParams"
		if declared > 0 {
			msg = fmt.Sprintf("%d tool(s) declare openai/fileParams with the four-property file schema", declared)
		}
		fs = append(fs, Finding{"file-params", Pass, "", msg, "O9"})
	}
	return fs
}

// fileSchemaIssue applies OpenAI's Scan Tools rules to one file param schema
// ("" = it passes). An array of file objects is checked through its items;
// a local $ref resolves against the input schema's $defs.
func fileSchemaIssue(schema, defs map[string]any) string {
	if schema["type"] == "array" {
		items, ok := schema["items"].(map[string]any)
		if !ok {
			return "an array file param needs an items schema"
		}
		schema = items
	}
	if ref, ok := schema["$ref"].(string); ok {
		target, found := defs[strings.TrimPrefix(ref, "#/$defs/")].(map[string]any)
		if !strings.HasPrefix(ref, "#/$defs/") || !found {
			return fmt.Sprintf("$ref %q does not resolve to a $defs entry", ref)
		}
		schema = target
	}
	if schema["type"] != "object" {
		return "the schema is not type object"
	}
	props, _ := schema["properties"].(map[string]any)
	var missing []string
	for _, p := range fileProps {
		if _, ok := props[p]; !ok {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		return "the schema omits " + strings.Join(missing, ", ")
	}
	req, _ := stringList(schema["required"])
	var extra []string
	have := map[string]bool{}
	for _, r := range req {
		have[r] = true
		if r != "download_url" && r != "file_id" {
			extra = append(extra, r)
		}
	}
	if !have["download_url"] || !have["file_id"] {
		return "download_url and file_id must both be required"
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		return "only download_url and file_id may be required, not " + strings.Join(extra, ", ")
	}
	return ""
}

// ---- check 7 (openai): securitySchemes on gated tools (O5, O10) ----

// checkSecuritySchemes: every tool that answered 401 must declare an oauth2
// scheme, top level or in the _meta mirror, so ChatGPT knows to sign in.
func checkSecuritySchemes(tools []tool, gated map[string]bool) []Finding {
	var fs []Finding
	for _, t := range tools {
		if !gated[t.Name] {
			continue
		}
		if !hasScheme(t.SecuritySchemes, "oauth2") && !hasScheme(t.Meta["securitySchemes"], "oauth2") {
			fs = append(fs, Finding{"security-schemes", Fail, t.Name, `answers 401 but declares no securitySchemes [{"type":"oauth2"}]`, "O5,O10"})
		}
	}
	if len(fs) == 0 {
		fs = append(fs, Finding{"security-schemes", Pass, "", fmt.Sprintf("%d gated tool(s) declare an oauth2 security scheme", len(gated)), "O5,O10"})
	}
	return fs
}

func hasScheme(v any, typ string) bool {
	list, _ := v.([]any)
	for _, s := range list {
		if m, ok := s.(map[string]any); ok && m["type"] == typ {
			return true
		}
	}
	return false
}

func stringList(v any) ([]string, bool) {
	list, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(list))
	for _, x := range list {
		s, ok := x.(string)
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}
