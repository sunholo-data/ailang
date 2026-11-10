package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Sources is everything the gate reads, relative to the root, sorted.
type Sources struct {
	Pages  []string // markdown pages: docs/docs/** (minus mirrors), README.md, docs/LIMITATIONS.md
	Prompt string   // the active teaching prompt's SOURCE file
	JS     []string // docs/src/**/*.{js,jsx,ts,tsx}: raw-loader imports and embedded literals
}

// Scope lists the documents the gate reads: docs/docs/** (minus generated prompt mirrors
// and registry package pages), README.md, docs/LIMITATIONS.md, the active teaching
// prompt's SOURCE file (prompts/versions.json .active -> .versions[active].file, not its
// docs mirror), and the site's own JavaScript under docs/src.
func Scope(root string) (Sources, error) {
	var s Sources
	docs := filepath.Join(root, "docs", "docs")
	err := filepath.WalkDir(docs, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		rel := relSlash(root, p)
		if !d.IsDir() && (strings.HasSuffix(rel, ".md") || strings.HasSuffix(rel, ".mdx")) && !excluded(rel) {
			s.Pages = append(s.Pages, rel)
		}
		return nil
	})
	if err != nil {
		return s, fmt.Errorf("walk %s: %w", docs, err)
	}
	for _, f := range []string{"README.md", "docs/LIMITATIONS.md"} {
		if _, e := os.Stat(filepath.Join(root, f)); e == nil {
			s.Pages = append(s.Pages, f)
		}
	}
	src := filepath.Join(root, "docs", "src")
	if _, e := os.Stat(src); e == nil {
		err = filepath.WalkDir(src, func(p string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return werr
			}
			switch filepath.Ext(p) {
			case ".js", ".jsx", ".ts", ".tsx":
				if !d.IsDir() {
					s.JS = append(s.JS, relSlash(root, p))
				}
			}
			return nil
		})
		if err != nil {
			return s, fmt.Errorf("walk %s: %w", src, err)
		}
	}
	sort.Strings(s.Pages)
	sort.Strings(s.JS)
	s.Prompt, err = activePrompt(root)
	return s, err
}

func relSlash(root, p string) string {
	rel, _ := filepath.Rel(root, p)
	return filepath.ToSlash(rel)
}

// excluded: pages that are generated copies of something checked elsewhere, or not ours.
func excluded(rel string) bool {
	if strings.HasPrefix(rel, "docs/docs/prompts/") {
		// Archived revisions and the synced mirror of the active prompt (which is checked
		// at its source). index.md is a hand-written landing page.
		return rel != "docs/docs/prompts/index.md"
	}
	if strings.HasPrefix(rel, "docs/docs/packages/") {
		// docs/docs/packages/<vendor>/<pkg>.md is synced from the registry (third-party READMEs).
		return strings.Count(strings.TrimPrefix(rel, "docs/docs/packages/"), "/") > 0
	}
	return false
}

func activePrompt(root string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(root, "prompts", "versions.json"))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var v struct {
		Active   string `json:"active"`
		Versions map[string]struct {
			File string `json:"file"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", fmt.Errorf("prompts/versions.json: %w", err)
	}
	e, ok := v.Versions[v.Active]
	if !ok || e.File == "" {
		return "", fmt.Errorf("prompts/versions.json: active %q has no file", v.Active)
	}
	return e.File, nil
}

var (
	rawImport = regexp.MustCompile(`import\s+(\w+)\s+from\s+['"]!!raw-loader!([^'"]+)['"]`)
	codeBlock = regexp.MustCompile(`<CodeBlock\b([^>]*)>\s*\{\s*(\w+)\s*\}\s*</CodeBlock>`)
	langAttr  = regexp.MustCompile(`\blanguage=["']([^"']*)["']`)
	jsLiteral = regexp.MustCompile("\\{`([^`]*)`\\}")
)

// Import is one raw-loader import of an .ail file from a docs source.
type Import struct {
	From string // the importing page or component
	Line int
	Path string // repo-relative .ail path
}

// ImportsOf finds raw-loader imports of .ail files in one docs source, and returns a
// structural block for every <CodeBlock> that renders one of them with a language other
// than ailang (it would highlight as that language). An import that does not resolve is
// structural too: the docs build would fail on it.
func ImportsOf(root, page, text string) ([]Import, []*Block) {
	var imps []Import
	var defects []*Block
	names := map[string]string{}
	for _, m := range rawImport.FindAllStringSubmatchIndex(text, -1) {
		name, target := text[m[2]:m[3]], text[m[4]:m[5]]
		if !strings.HasSuffix(target, ".ail") {
			continue
		}
		line := 1 + strings.Count(text[:m[0]], "\n")
		var rel string
		if strings.HasPrefix(target, "@site/") {
			rel = path.Clean(path.Join("docs", strings.TrimPrefix(target, "@site/")))
		} else {
			rel = path.Clean(path.Join(path.Dir(page), target))
		}
		if strings.HasPrefix(rel, "../") {
			defects = append(defects, structural(page, line, "DOC_MISSING_IMPORT", target, "raw-loader import escapes the repository: "+target))
			continue
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			defects = append(defects, structural(page, line, "DOC_MISSING_IMPORT", target, "raw-loader import does not resolve: "+rel))
			continue
		}
		names[name] = rel
		imps = append(imps, Import{From: page, Line: line, Path: rel})
	}
	for _, m := range codeBlock.FindAllStringSubmatchIndex(text, -1) {
		attrs, name := text[m[2]:m[3]], text[m[4]:m[5]]
		rel, ok := names[name]
		if !ok {
			continue
		}
		lang := ""
		if l := langAttr.FindStringSubmatch(attrs); l != nil {
			lang = l[1]
		}
		if lang != "ailang" && lang != "ail" {
			line := 1 + strings.Count(text[:m[0]], "\n")
			defects = append(defects, structural(page, line, "DOC_MISFENCED", rel+" "+lang,
				fmt.Sprintf(`<CodeBlock language=%q> renders %s as %s; use language="ailang"`, lang, rel, lang)))
		}
	}
	return imps, defects
}

// LiteralsOf returns AILANG code embedded in a JS/JSX source as a {`...`} template
// literal (the landing page's hero sample). Only literals the keyword hints recognise as
// AILANG are taken; CSS and HTML literals are not.
func LiteralsOf(page, text string) []*Block {
	var out []*Block
	for _, m := range jsLiteral.FindAllStringSubmatchIndex(text, -1) {
		body := text[m[2]:m[3]]
		if strings.Contains(body, "${") || !looksAILANG(body) {
			continue
		}
		out = append(out, &Block{Page: page, Source: page, Line: 1 + strings.Count(text[:m[0]], "\n"),
			Lang: "jsx-literal", Class: ClassProgram, Body: body, Key: KeyOf(body)})
	}
	return out
}

func structural(page string, line int, code, keyed, msg string) *Block {
	return &Block{Page: page, Source: page, Line: line, Class: ClassStructural, Defect: code, DefectMsg: msg, Key: KeyOf(code + " " + keyed)}
}
