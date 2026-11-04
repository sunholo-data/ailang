package builtins

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var schemaAnchorPattern = regexp.MustCompile(`^[A-Za-z_][-A-Za-z0-9._]*$`)
var schemaVocabularies = map[string]bool{
	"https://json-schema.org/draft/2020-12/vocab/core":              true,
	"https://json-schema.org/draft/2020-12/vocab/applicator":        true,
	"https://json-schema.org/draft/2020-12/vocab/unevaluated":       true,
	"https://json-schema.org/draft/2020-12/vocab/validation":        true,
	"https://json-schema.org/draft/2020-12/vocab/meta-data":         true,
	"https://json-schema.org/draft/2020-12/vocab/format-annotation": true,
}

func auditSchemaID(s, path string, add func(string, string, string)) {
	u, e := url.Parse(s)
	if e != nil || u.IsAbs() || u.Host != "" || u.Fragment != "" {
		add(path, "$id", "only relative $id values without nonempty fragments are supported")
	}
}
func (p *schemaProgram) resolveReferences(add func(string, string, string)) {
	paths := make([]string, 0, len(p.nodes))
	for path := range p.nodes {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		n := p.nodes[path]
		obj, ok := n.value.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range []string{"$ref", "$dynamicRef"} {
			raw, ok := obj[key].(string)
			if !ok {
				continue
			}
			target, reason := p.resolveReference(raw)
			if reason != "" {
				add(schemaPointer(path, key), key, reason)
				continue
			}
			if key == "$ref" {
				n.ref = target
			} else {
				n.dynamicRef = raw
				n.children["$dynamicRef"] = target
			}
		}
	}
}
func (p *schemaProgram) resolveReference(raw string) (*schemaNode, string) {
	if !strings.HasPrefix(raw, "#") {
		return nil, "only local fragment references (#, #/pointer or #anchor) are supported"
	}
	fragment, e := url.PathUnescape(strings.TrimPrefix(raw, "#"))
	if e != nil {
		return nil, "invalid reference fragment encoding"
	}
	if fragment == "" {
		return p.root, ""
	}
	if !strings.HasPrefix(fragment, "/") {
		if n := p.anchors[fragment]; n != nil {
			return n, ""
		}
		return nil, "dangling local anchor " + raw
	}
	// Validate escapes before comparing canonical pointers: malformed ~ escapes are
	// not literals, while a percent-encoded slash is a pointer delimiter.
	parts := strings.Split(fragment[1:], "/")
	canonical := ""
	for _, part := range parts {
		for i := 0; i < len(part); i++ {
			if part[i] == '~' {
				if i+1 >= len(part) || (part[i+1] != '0' && part[i+1] != '1') {
					return nil, "invalid RFC 6901 escape in " + raw
				}
				i++
			}
		}
		decoded := strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		canonical = schemaPointer(canonical, decoded)
	}
	if n := p.nodes[canonical]; n != nil {
		return n, ""
	}
	return nil, "dangling reference or target is not a schema: " + raw
}
