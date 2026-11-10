package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Class is what a block's info string says it is, and so what the gate does with it.
type Class string

const (
	ClassProgram   Class = "program"   // ```ailang            -> ailang check (module line synthesised if absent)
	ClassParse     Class = "parse"     // ```ailang parse      -> ailang fmt must parse it
	ClassError     Class = "error"     // ```ailang error=CODE -> ailang check must FAIL with CODE
	ClassSketch    Class = "sketch"    // ```ailang sketch     -> not run, counted
	ClassRepl      Class = "repl"      // ```ailang repl       -> not run, counted
	ClassMisfenced Class = "misfenced" // AILANG in a typescript/untagged fence -> failure
	ClassBadTag    Class = "badtag"    // an info-string word the gate does not know -> failure
)

// PromptPage is the page key of the active teaching prompt. It names the ROLE, not the
// file (prompts/v0.16.12.md), so a new prompt revision that keeps a block keeps its
// baseline key, and one that edits a block has to fix or tag it.
const PromptPage = "prompt:active"

// Block is one fenced code block the gate checks or counts.
type Block struct {
	Page     string // repo-relative path, or PromptPage
	Source   string // repo-relative file the block was read from
	Line     int    // 1-based line of the opening fence in Source
	Ordinal  int    // 1-based index among the page's AILANG blocks
	Lang     string // the fence language as written ("" = none)
	Info     string // the rest of the info string
	Class    Class
	File     string // file=<path> (multi-file examples)
	WantCode string // error=<CODE>
	BadTag   string // the unknown word, for ClassBadTag
	Body     string // dedented block text
	Key      string // sha1-12 of the normalised body
}

var (
	fenceOpen  = regexp.MustCompile("^([ \t]*)(`{3,}|~{3,})[ \t]*([^\\s`{]*)[ \t]*(.*)$")
	moduleLine = regexp.MustCompile(`(?m)^[ \t]*module[ \t]+([A-Za-z0-9_/]+)[ \t]*$`)
	ailHints   = []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\s*(export\s+)?(pure\s+)?func\s+\w+\s*(\[[^\]]*\])?\s*\(`),
		regexp.MustCompile(`(?m)^\s*module\s+\w+/\w+`),
		regexp.MustCompile(`(?m)^\s*import\s+std/`),
		regexp.MustCompile(`(\)|->)\s*(->\s*[^!\n{]+)?!\s*\{\s*[A-Z][A-Za-z]*`), // effect row
		regexp.MustCompile(`\\[a-z_]\w*(\s*,\s*[a-z_]\w*)*\s*\.`),               // \x. lambda (Haskell writes \x ->)
		regexp.MustCompile(`(?m)^\s*--\s+\S`),                                   // -- line comment
		regexp.MustCompile(`(?m)^\s*(λ[^>\n]*|ailang)>\s`),                      // REPL transcript
		regexp.MustCompile(`#\[`),                                               // array literal
	}
	// Lines only another language writes. Matched at line start, not anywhere: an
	// AILANG comment saying "-- Lambda function" must not hide the block.
	notAilHint = regexp.MustCompile(`(?m)^\s*((export\s+)?(function|const|interface)\s|def\s|fn\s|package\s|console\.|\$\s)`)
)

// candidateLangs are fence languages that AILANG has historically been mis-fenced as
// (`typescript` was the only way to get highlighting before the site had an ailang
// grammar). Other languages (bash, text, json, go, ...) are never inspected.
var candidateLangs = map[string]bool{"": true, "typescript": true, "ts": true}

// docusaurusMeta are metastring words Docusaurus itself interprets; the gate ignores them.
func docusaurusMeta(w string) bool {
	return strings.HasPrefix(w, "title=") || strings.HasPrefix(w, "showLineNumbers") ||
		strings.HasPrefix(w, "{") || w == "live" || w == "noInline"
}

// metaWords splits an info string on spaces, keeping quoted values and {ranges} whole.
func metaWords(s string) []string {
	var out []string
	var cur strings.Builder
	var quote byte
	depth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '{':
			depth++
		case c == '}':
			if depth > 0 {
				depth--
			}
		case (c == ' ' || c == '\t') && depth == 0:
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteByte(c)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// classify applies the info-string grammar to an `ailang` fence.
func classify(b *Block) {
	b.Class = ClassProgram
	classWords := 0
	for _, w := range metaWords(b.Info) {
		switch {
		case docusaurusMeta(w):
		case w == "parse":
			b.Class, classWords = ClassParse, classWords+1
		case w == "sketch":
			b.Class, classWords = ClassSketch, classWords+1
		case w == "repl":
			b.Class, classWords = ClassRepl, classWords+1
		case strings.HasPrefix(w, "error=") && len(w) > len("error="):
			b.Class, b.WantCode, classWords = ClassError, strings.TrimPrefix(w, "error="), classWords+1
		case strings.HasPrefix(w, "file=") && strings.HasSuffix(w, ".ail") && !strings.Contains(w, ".."):
			b.File = strings.TrimPrefix(w, "file=")
		default:
			b.Class, b.BadTag = ClassBadTag, w
			return
		}
	}
	if classWords > 1 {
		b.Class, b.BadTag = ClassBadTag, b.Info
	}
	if b.File != "" && (b.Class == ClassSketch || b.Class == ClassRepl || b.Class == ClassParse) {
		b.Class, b.BadTag = ClassBadTag, "file= with "+string(b.Class)
	}
}

// looksAILANG is the mis-fence detector. It is deliberately conservative: a false
// positive fails the gate on someone's TypeScript, so a hint must be AILANG-only
// syntax (an effect row, `module a/b`, `import std/`, a `func` declaration).
func looksAILANG(src string) bool {
	if notAilHint.MatchString(src) {
		return false
	}
	for _, h := range ailHints {
		if h.MatchString(src) {
			return true
		}
	}
	return false
}

// Normalize is what the baseline key hashes: trailing whitespace and surrounding
// blank lines removed, so a re-indent or a trailing space does not churn the key.
func Normalize(body string) string {
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t\r")
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// KeyOf is the stable block identity: sha1 of the normalised body, 12 hex chars.
func KeyOf(body string) string {
	sum := sha1.Sum([]byte(Normalize(body)))
	return hex.EncodeToString(sum[:])[:12]
}

// ExtractBlocks returns the AILANG blocks (tagged, and mis-fenced) of one markdown text.
func ExtractBlocks(page, source, text string) []*Block {
	lines := strings.Split(text, "\n")
	var out []*Block
	ordinal := 0
	for i := 0; i < len(lines); i++ {
		m := fenceOpen.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		indent, fence, lang, info := m[1], m[2], strings.ToLower(m[3]), strings.TrimSpace(m[4])
		closeRe := regexp.MustCompile(`^[ \t]*` + regexp.QuoteMeta(fence[:1]) + `{` + fmt.Sprint(len(fence)) + `,}[ \t]*$`)
		var body []string
		j := i + 1
		for ; j < len(lines) && !closeRe.MatchString(lines[j]); j++ {
			body = append(body, strings.TrimPrefix(lines[j], indent))
		}
		text := strings.Join(body, "\n")
		b := &Block{Page: page, Source: source, Line: i + 1, Lang: lang, Info: info, Body: text, Key: KeyOf(text)}
		switch {
		case lang == "ailang" || lang == "ail":
			classify(b)
		case candidateLangs[lang] && !strings.Contains(info, "not-ailang") && looksAILANG(text):
			b.Class = ClassMisfenced
		default:
			i = j
			continue
		}
		ordinal++
		b.Ordinal = ordinal
		out = append(out, b)
		i = j
	}
	return out
}

// moduleOf returns the block's own `module` path, "" if it has none.
func moduleOf(body string) string {
	if m := moduleLine.FindStringSubmatch(body); m != nil {
		return m[1]
	}
	return ""
}

// Scope lists the documents the gate reads, relative to root, sorted:
// docs/docs/** (minus generated prompt mirrors and registry package pages),
// README.md, docs/LIMITATIONS.md, and the active teaching prompt's SOURCE file
// (prompts/versions.json .active -> .versions[active].file), not its docs mirror.
func Scope(root string) (pages []string, prompt string, err error) {
	docs := filepath.Join(root, "docs", "docs")
	err = filepath.WalkDir(docs, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(rel, ".md") && !strings.HasSuffix(rel, ".mdx") {
			return nil
		}
		if excluded(rel) {
			return nil
		}
		pages = append(pages, rel)
		return nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("walk %s: %w", docs, err)
	}
	for _, f := range []string{"README.md", "docs/LIMITATIONS.md"} {
		if _, e := os.Stat(filepath.Join(root, f)); e == nil {
			pages = append(pages, f)
		}
	}
	sort.Strings(pages)
	prompt, err = activePrompt(root)
	return pages, prompt, err
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
