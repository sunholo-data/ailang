package builtins

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// HTML5 parsing builtins for AILANG.
// Wraps golang.org/x/net/html (WHATWG HTML5 spec-compliant parser) and emits
// the same XmlNode ADT that std/xml uses, so all tree-walking helpers
// (findAll, findFirst, getText, getAttr, …) work identically on HTML trees.
//
// Part of M-STDLIB-HTML (v0.19.1). See design_docs/implemented/v0_19_1/m-stdlib-html.md.

const (
	// Reuse the same caps as std/xml; html.go and xml.go are peer parsers and
	// callers should not have to think about which has a different budget.
	htmlMaxDepth     = xmlMaxDepth
	htmlMaxInputSize = xmlMaxInputSize

	htmlParseErrFmt = "HTML parse error: %v"
)

func init() {
	registerHtmlParse()
	registerHtmlParseFragment()
}

// ============================================================================
// _html_parse: string -> Result[XmlNode, string]
// ============================================================================

func registerHtmlParse() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/html",
		Name:    "_html_parse",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeHtmlParseType,
		Impl:    htmlParseImpl,
		Metadata: &BuiltinMetadata{
			Description: "Parse an HTML5 document into an XmlNode tree",
			LongDesc:    "Lenient WHATWG HTML5 parse via golang.org/x/net/html. Handles unclosed tags, boolean attributes, overlapping nesting, mixed-case tags, and other real-world HTML5 deviations. Returns the <html> element at the top level (DOCTYPE and the synthetic document root are stripped). Rejects input >50MB and tree depth >256.",
			Params: []ParamDoc{
				{Name: "html", Description: "HTML string to parse"},
			},
			Returns: "Result[XmlNode, string] - Ok(XmlNode tree rooted at <html>) or Err(error message)",
			Examples: []Example{
				{Code: `_html_parse("<p>hello<p>world")`, Description: `Returns Ok with two sibling <p> elements (lenient auto-close)`},
				{Code: `_html_parse("<input disabled>")`, Description: `Boolean attribute "disabled" becomes {name: "disabled", value: ""}`},
			},
			SeeAlso:   []string{"_html_parseFragment", "_xml_parse", "_xml_findAll", "_xml_getText"},
			Since:     "v0.19.1",
			Stability: StabilityStable,
			Tags:      []string{"html", "parsing", "tree", "adt"},
			Category:  "html",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _html_parse: %v", err))
	}
}

func makeHtmlParseType() types.Type {
	T := types.NewBuilder()
	xmlNodeType := T.Con("XmlNode")
	return T.Func(T.String()).Returns(
		T.App("Result", xmlNodeType, T.String()),
	).Build()
}

func htmlParseImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_html_parse: expected String, got %T", args[0])
	}

	input := strVal.Value
	if len(input) > htmlMaxInputSize {
		return xmlMakeErr(fmt.Sprintf("HTML input too large: %d bytes (max %d)", len(input), htmlMaxInputSize)), nil
	}

	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return xmlMakeErr(fmt.Sprintf(htmlParseErrFmt, err)), nil
	}

	// html.Parse always returns a DocumentNode. We want the first ElementNode
	// child (the <html> element). Skip Doctype/Comment siblings at the top
	// level — those are document-level metadata, not part of the content tree.
	var rootElement *html.Node
	for c := doc.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			rootElement = c
			break
		}
	}
	if rootElement == nil {
		return xmlMakeErr("HTML parse error: no root element found"), nil
	}

	node, err := htmlNodeToXmlNode(rootElement, 0)
	if err != nil {
		return xmlMakeErr(fmt.Sprintf(htmlParseErrFmt, err)), nil
	}
	return xmlMakeOk(node), nil
}

// ============================================================================
// _html_parseFragment: string -> Result[List[XmlNode], string]
// ============================================================================

func registerHtmlParseFragment() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/html",
		Name:    "_html_parseFragment",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeHtmlParseFragmentType,
		Impl:    htmlParseFragmentImpl,
		Metadata: &BuiltinMetadata{
			Description: "Parse an HTML5 fragment into a list of XmlNode roots",
			LongDesc:    "Lenient HTML5 fragment parse — no <html>/<head>/<body> wrapper is synthesized. Useful for parsing snippets like email bodies or CMS-authored content. Returns a list of top-level nodes (Elements, Text, Comments) in document order.",
			Params: []ParamDoc{
				{Name: "html", Description: "HTML fragment string to parse"},
			},
			Returns: "Result[List[XmlNode], string] - Ok(list of fragment roots) or Err(message)",
			Examples: []Example{
				{Code: `_html_parseFragment("<p>a</p><p>b</p>")`, Description: `Returns Ok with two top-level <p> elements`},
			},
			SeeAlso:   []string{"_html_parse"},
			Since:     "v0.19.1",
			Stability: StabilityStable,
			Tags:      []string{"html", "parsing", "fragment"},
			Category:  "html",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _html_parseFragment: %v", err))
	}
}

func makeHtmlParseFragmentType() types.Type {
	T := types.NewBuilder()
	xmlNodeType := T.Con("XmlNode")
	return T.Func(T.String()).Returns(
		T.App("Result", T.List(xmlNodeType), T.String()),
	).Build()
}

func htmlParseFragmentImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_html_parseFragment: expected String, got %T", args[0])
	}

	input := strVal.Value
	if len(input) > htmlMaxInputSize {
		return xmlMakeErr(fmt.Sprintf("HTML input too large: %d bytes (max %d)", len(input), htmlMaxInputSize)), nil
	}

	bodyCtx := &html.Node{Type: html.ElementNode, DataAtom: atom.Body, Data: "body"}
	nodes, err := html.ParseFragment(strings.NewReader(input), bodyCtx)
	if err != nil {
		return xmlMakeErr(fmt.Sprintf(htmlParseErrFmt, err)), nil
	}

	out := make([]eval.Value, 0, len(nodes))
	for _, n := range nodes {
		converted, err := htmlNodeToXmlNode(n, 0)
		if err != nil {
			return xmlMakeErr(fmt.Sprintf(htmlParseErrFmt, err)), nil
		}
		if converted == nil {
			continue
		}
		out = append(out, converted)
	}
	return xmlMakeOk(&eval.ListValue{Elements: out}), nil
}

// ============================================================================
// Converter: *html.Node -> XmlNode TaggedValue
// ============================================================================

// htmlNodeToXmlNode converts a *html.Node subtree into the same XmlNode shape
// std/xml produces. Returns nil for DoctypeNode and DocumentNode (caller filters).
func htmlNodeToXmlNode(n *html.Node, depth int) (eval.Value, error) {
	if depth > htmlMaxDepth {
		return nil, fmt.Errorf("maximum depth exceeded (%d)", htmlMaxDepth)
	}

	switch n.Type {
	case html.ElementNode:
		attrs := make([]eval.Value, 0, len(n.Attr))
		for _, a := range n.Attr {
			// HTML5 doesn't really namespace; flatten to local name.
			attrs = append(attrs, makeXmlAttr(a.Key, a.Val))
		}

		children := make([]eval.Value, 0)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child, err := htmlNodeToXmlNode(c, depth+1)
			if err != nil {
				return nil, err
			}
			if child != nil {
				children = append(children, child)
			}
		}

		return makeXmlElement(n.Data, attrs, children), nil

	case html.TextNode:
		return makeXmlText(n.Data), nil

	case html.CommentNode:
		return makeXmlComment(n.Data), nil

	case html.DoctypeNode, html.DocumentNode:
		// Skip — callers see the <html> element directly.
		return nil, nil

	default:
		// ErrorNode and any future node kinds: drop silently.
		return nil, nil
	}
}
