package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// Class is what a block's info string says it is, and so what the gate does with it.
type Class string

const (
	ClassProgram    Class = "program"    // ```ailang            -> ailang check
	ClassParse      Class = "parse"      // ```ailang parse      -> ailang fmt must parse it
	ClassError      Class = "error"      // ```ailang error=CODE -> ailang check must FAIL with CODE first
	ClassSketch     Class = "sketch"     // ```ailang sketch     -> not run; listed in the unchecked ratchet
	ClassRepl       Class = "repl"       // ```ailang repl       -> not run; listed in the unchecked ratchet
	ClassFile       Class = "file"       // an .ail file the docs import (raw-loader) -> ailang check in place
	ClassCandidate  Class = "candidate"  // a non-ailang fence that may be AILANG -> parsed, then misfenced or dropped
	ClassMisfenced  Class = "misfenced"  // AILANG in a typescript/untagged/text fence -> structural failure
	ClassBadTag     Class = "badtag"     // an info-string word the gate does not know -> structural failure
	ClassStructural Class = "structural" // a fence the page cannot render as written -> structural failure
)

// PromptPage is the page key of the active teaching prompt. It names the ROLE, not the
// file (prompts/v0.16.12.md), so a new prompt revision that keeps a block keeps its
// baseline key, and one that edits a block has to fix or tag it.
const PromptPage = "prompt:active"

// Block is one code block (or imported .ail file) the gate checks or counts.
type Block struct {
	Page      string // repo-relative path, or PromptPage
	Source    string // repo-relative file the block was read from
	Line      int    // 1-based line of the opening fence in Source
	Ordinal   int    // 1-based index among the page's AILANG blocks
	Lang      string // the fence language as written ("" = none)
	Info      string // the rest of the info string
	Class     Class
	File      string // file=<path> (multi-file examples)
	WantCode  string // error=<CODE>
	BadTag    string // the unknown word, for ClassBadTag
	Defect    string // DOC_* code, for ClassStructural
	DefectMsg string // what is wrong, for ClassStructural / ClassMisfenced
	Body      string // dedented block text
	Key       string // sha1-12 of the normalised body
}

var (
	fenceOpen  = regexp.MustCompile("^([ \t]*)(`{3,}|~{3,})[ \t]*([^\\s`{]*)[ \t]*(.*)$")
	moduleLine = regexp.MustCompile(`(?m)^[ \t]*module[ \t]+([A-Za-z0-9_/]+)[ \t]*$`)
	mainDecl   = regexp.MustCompile(`(?m)^[ \t]*(export[ \t]+)?func[ \t]+main[ \t]*\(`)
	ailHints   = []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\s*(export\s+)?(pure\s+)?func\s+\w+\s*(\[[^\]]*\])?\s*\(`),
		regexp.MustCompile(`(?m)^\s*module\s+\w+/\w+`),
		regexp.MustCompile(`(?m)^\s*import\s+std/`),
		regexp.MustCompile(`(\)|->)\s*(->\s*[^!\n{]+)?!\s*\{\s*[A-Z][A-Za-z]*`), // effect row
		regexp.MustCompile(`\\[a-z_]\w*(\s*,\s*[a-z_]\w*)*\s*\.`),               // \x. lambda (Haskell writes \x ->)
		regexp.MustCompile(`(?m)^\s*--\s+\S`),                                   // -- line comment
		regexp.MustCompile(`(?m)^\s*(λ[^>\n]*|ailang)>\s`),                      // REPL transcript
		regexp.MustCompile(`#\[`),                                               // array literal
		regexp.MustCompile(`(?m)^\s*match\s+[A-Za-z_(\[]`),                      // `match e {` (TS has .match( / match =)
		regexp.MustCompile(`(?m)^\s*(Ok|Err|Some|None)\b[^\n]*=>`),              // ADT match arm
	}
	// Lines only another language writes. Matched at line start, not anywhere: an
	// AILANG comment saying "-- Lambda function" must not hide the block.
	notAilHint = regexp.MustCompile(`(?m)^\s*((export\s+)?(function|const|interface)\s|def\s|fn\s|package\s|console\.|\$\s)`)
	// ailKeyword gates the parse probe: a block must contain an AILANG-only keyword or
	// operator before "it parses as AILANG" counts as evidence (bare `42`, `(15, 5)` or
	// `owner/repo` parse too, and are output).
	ailKeyword = regexp.MustCompile(`\b(func|let|letrec|match|module|import|export|test|requires|ensures)\b|=>|\\[a-z_]\w*\s*\.`)
	// pseudoCodes are classes the gate invents for code-less diagnostics; `error=` must
	// name a code the binary itself prints.
	pseudoCode = regexp.MustCompile(`^(TC_UNDEFINED_VAR|TC_EFFECT|TC_TYPE|MODULE|OTHER|CHECK_NO_JSON|FMT_EXIT\d*|DOC_\w+)$`)
	realCode   = regexp.MustCompile(`^[A-Z]{2,5}(\d{3}|_[A-Z][A-Z0-9_]+)$`)
)

// hintLangs are fences AILANG was historically mis-fenced as (`typescript` was the only
// way to get highlighting before the site had an ailang grammar): a keyword hint alone
// convicts. probeLangs are additionally PARSED (see ClassCandidate): text and JavaScript
// fences are mostly output and objects, so they need both a keyword and a clean parse.
var (
	hintLangs  = map[string]bool{"": true, "typescript": true, "ts": true}
	probeLangs = map[string]bool{"": true, "typescript": true, "ts": true, "text": true, "javascript": true, "js": true}
)

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
		case strings.HasPrefix(w, "error="):
			code := strings.TrimPrefix(w, "error=")
			if !realCode.MatchString(code) || pseudoCode.MatchString(code) {
				b.Class, b.BadTag = ClassBadTag, w+" (error= names a code the binary prints, e.g. PAR015 or EXT002)"
				return
			}
			b.Class, b.WantCode, classWords = ClassError, code, classWords+1
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

// looksAILANG is the keyword-hint half of the mis-fence detector. It is deliberately
// conservative: a false positive fails the gate on someone's TypeScript, so a hint must be
// AILANG-only syntax. The parse probe (ClassCandidate) is the other half.
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

// ExtractBlocks returns the AILANG blocks of one markdown text: `ailang` fences,
// mis-fence suspects, and structural fence defects in ANY fence (an opening fence nested
// inside a block of the same kind, or a fence that never closes) — those swallow the
// blocks after them, which then are never checked.
func ExtractBlocks(page, source, text string) []*Block {
	// CRLF (a Windows checkout with autocrlf) would leave `\r` on every fence line and
	// no closing fence would match; the key is unaffected (Normalize trims `\r`).
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var out []*Block
	ordinal := 0
	for i := 0; i < len(lines); i++ {
		m := fenceOpen.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		indent, fence, lang, info := m[1], m[2], strings.ToLower(m[3]), strings.TrimSpace(m[4])
		ch, n := regexp.QuoteMeta(fence[:1]), fmt.Sprint(len(fence))
		closeRe := regexp.MustCompile(`^[ \t]*` + ch + `{` + n + `,}[ \t]*$`)
		nestedRe := regexp.MustCompile(`^[ \t]*` + ch + `{` + n + `,}[ \t]*[^\s` + "`" + `~]`)
		var body []string
		nested := 0
		j := i + 1
		for ; j < len(lines) && !closeRe.MatchString(lines[j]); j++ {
			if nested == 0 && nestedRe.MatchString(lines[j]) {
				nested = j + 1
			}
			body = append(body, strings.TrimPrefix(lines[j], indent))
		}
		text := strings.Join(body, "\n")
		b := &Block{Page: page, Source: source, Line: i + 1, Lang: lang, Info: info, Body: text, Key: KeyOf(text)}
		switch {
		case j >= len(lines):
			b.Class, b.Defect = ClassStructural, "DOC_UNTERMINATED"
			b.DefectMsg = fmt.Sprintf("fence %s opened here never closes: everything to the end of the file renders as one code block", fence)
		case nested > 0:
			b.Class, b.Defect = ClassStructural, "DOC_FENCE_PAIRING"
			b.DefectMsg = fmt.Sprintf("line %d opens a fence (%s) inside this block, so it is rendered as text and its block is never checked; "+
				"a stray or missing fence above usually causes this (block runs %d-%d)", nested, strings.TrimSpace(lines[nested-1]), i+1, j+1)
		case lang == "ailang" || lang == "ail":
			classify(b)
		case strings.Contains(info, "not-ailang"):
			i = j
			continue
		case hintLangs[lang] && looksAILANG(text):
			b.Class = ClassMisfenced
		case probeLangs[lang] && ailKeyword.MatchString(text):
			b.Class = ClassCandidate // CheckAll parses it; a clean parse convicts
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

// definesMain reports whether a block declares `main` — a program a reader will copy
// into a file and run, so it must carry its own `module` line (MOD014 otherwise).
func definesMain(body string) bool { return mainDecl.MatchString(body) }
