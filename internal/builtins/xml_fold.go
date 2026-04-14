package builtins

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// M-STREAMING-ZIP-XML: XML fold builtin + streaming element scanner
//
// parseFold scans the XML token stream for matching elements and calls a
// handler for each one, threading an accumulator. Memory: O(largest element
// + accumulator) instead of O(all matched elements).
//
// scanForElementsFold is the shared helper used by both _xml_parseFold
// (pure, over a string) and _zip_xml_scanFold (effectful, over a ZIP stream).

func init() {
	registerXmlParseFold()
	registerXmlParseFoldStep()
}

// ============================================================================
// _xml_parseFold: Fold over matching XML elements without collecting into list
// ============================================================================

func registerXmlParseFold() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_parseFold",
		NumArgs: 4,
		IsPure:  true,
		Type:    makeXmlParseFoldType,
		Impl:    xmlParseFoldImpl,
		Metadata: &BuiltinMetadata{
			Description: "Fold over XML elements matching a tag without building a result list",
			LongDesc:    "Scans the XML token stream for elements matching the given tag. For each match, calls the handler function with the accumulator and the matched XmlNode, threading the accumulator through. Memory usage is O(largest matched element + accumulator) instead of O(all matched elements).",
			Params: []ParamDoc{
				{Name: "xml", Description: "XML string to parse"},
				{Name: "tagName", Description: "Tag name to match"},
				{Name: "init", Description: "Initial accumulator value"},
				{Name: "handler", Description: "Fold function: (acc, XmlNode) -> acc"},
			},
			Returns: "Result[a, string] - Ok(final accumulator) or Err(message)",
			Examples: []Example{
				{Code: `_xml_parseFold(xml, "si", [], \acc, node. append(acc, getText(node)))`, Description: "Fold shared strings into an array without holding all XmlNodes in memory"},
			},
			SeeAlso:   []string{"_xml_parseElements", "_xml_parse"},
			Since:     "v0.11.0",
			Stability: StabilityStable,
			Tags:      []string{"xml", "parsing", "streaming", "fold", "performance"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_parseFold: %v", err))
	}
}

// Type: forall a. (string, string, a, (a, XmlNode) -> a) -> Result[a, string]
func makeXmlParseFoldType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	xmlNodeType := T.Con("XmlNode")
	fn := T.Func(a, xmlNodeType).Returns(a).Build()
	return T.Func(T.String(), T.String(), a, fn).Returns(
		T.App("Result", a, T.String()),
	).Build()
}

func xmlParseFoldImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_parseFold: expected String for xml, got %T", args[0])
	}
	tagVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_parseFold: expected String for tagName, got %T", args[1])
	}
	acc := args[2]
	handler := args[3]

	input := strVal.Value
	if len(input) > xmlMaxInputSize {
		return xmlMakeErr(fmt.Sprintf("XML input too large: %d bytes (max %d)", len(input), xmlMaxInputSize)), nil
	}

	if ctx == nil || ctx.FnCallerN == nil {
		return nil, fmt.Errorf("_xml_parseFold: FnCallerN not set (evaluator not wired)")
	}

	tagName := tagVal.Value
	decoder := xml.NewDecoder(strings.NewReader(input))

	var foldErr error
	acc, foldErr = scanForElementsFold(decoder, tagName, acc, func(node eval.Value, currentAcc eval.Value) (eval.Value, error) {
		return ctx.FnCallerN(handler, []eval.Value{currentAcc, node})
	})
	if foldErr != nil {
		return xmlMakeErr(fmt.Sprintf("fold handler error: %v", foldErr)), nil
	}

	return xmlMakeOk(acc), nil
}

// ============================================================================
// _xml_parseFoldStep: Bounded fold — handler returns FoldStep[a] = Continue(a) | Stop(a)
// ============================================================================

func registerXmlParseFoldStep() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/xml",
		Name:    "_xml_parseFoldStep",
		NumArgs: 4,
		IsPure:  true,
		Type:    makeXmlParseFoldStepType,
		Impl:    xmlParseFoldStepImpl,
		Metadata: &BuiltinMetadata{
			Description: "Fold over XML elements with early termination via FoldStep[a]",
			LongDesc:    "Like _xml_parseFold, but the handler returns FoldStep[a] = Continue(a) | Stop(a). Returning Stop(acc') ends the scan immediately without consuming the rest of the input — critical for extracting a bounded prefix of matches from a large XML stream.",
			Params: []ParamDoc{
				{Name: "xml", Description: "XML string to parse"},
				{Name: "tagName", Description: "Tag name to match"},
				{Name: "init", Description: "Initial accumulator value"},
				{Name: "handler", Description: "Fold function: (acc, XmlNode) -> FoldStep[acc]"},
			},
			Returns:   "Result[a, string] - Ok(final accumulator) or Err(message)",
			SeeAlso:   []string{"_xml_parseFold", "_zip_xml_scanFoldStep"},
			Since:     "v0.11.3",
			Stability: StabilityStable,
			Tags:      []string{"xml", "fold", "streaming", "bounded"},
			Category:  "xml",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _xml_parseFoldStep: %v", err))
	}
}

// Type: forall a. (string, string, a, (a, XmlNode) -> FoldStep[a]) -> Result[a, string]
func makeXmlParseFoldStepType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	xmlNodeType := T.Con("XmlNode")
	stepType := T.App("FoldStep", a)
	fn := T.Func(a, xmlNodeType).Returns(stepType).Build()
	return T.Func(T.String(), T.String(), a, fn).Returns(
		T.App("Result", a, T.String()),
	).Build()
}

func xmlParseFoldStepImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_parseFoldStep: expected String for xml, got %T", args[0])
	}
	tagVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_xml_parseFoldStep: expected String for tagName, got %T", args[1])
	}
	acc := args[2]
	handler := args[3]

	input := strVal.Value
	if len(input) > xmlMaxInputSize {
		return xmlMakeErr(fmt.Sprintf("XML input too large: %d bytes (max %d)", len(input), xmlMaxInputSize)), nil
	}

	if ctx == nil || ctx.FnCallerN == nil {
		return nil, fmt.Errorf("_xml_parseFoldStep: FnCallerN not set (evaluator not wired)")
	}

	decoder := xml.NewDecoder(strings.NewReader(input))
	finalAcc, _, foldErr := scanForElementsFoldStepInner(decoder, tagVal.Value, acc, func(node eval.Value, currentAcc eval.Value) (eval.Value, error) {
		return ctx.FnCallerN(handler, []eval.Value{currentAcc, node})
	})
	if foldErr != nil {
		return xmlMakeErr(fmt.Sprintf("fold handler error: %v", foldErr)), nil
	}
	return xmlMakeOk(finalAcc), nil
}

// scanForElementsFoldStepInner is like scanForElementsFoldInner, but the handler
// returns a FoldStep tagged value. If Stop(acc') is returned, scanning halts
// immediately and the bool flag propagates up so outer recursion also halts.
func scanForElementsFoldStepInner(decoder *xml.Decoder, tagName string, acc eval.Value, handler func(node eval.Value, acc eval.Value) (eval.Value, error)) (eval.Value, bool, error) {
	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return acc, false, nil
			}
			return acc, false, nil
		}

		switch t := tok.(type) {
		case xml.StartElement:
			resolvedName := resolveTagName(t.Name, nil)
			if resolvedName == tagName {
				localPM := extractPrefixMap(t, nil)
				attrs := buildAttrs(t, localPM)
				childNodes, parseErr := parseXmlChildren(decoder, 1, localPM)
				if parseErr != nil {
					return acc, false, parseErr
				}
				finalTag := resolveTagName(t.Name, localPM)
				node := makeXmlElement(finalTag, attrs, childNodes)

				stepVal, handlerErr := handler(node, acc)
				if handlerErr != nil {
					return acc, false, handlerErr
				}
				tagged, ok := stepVal.(*eval.TaggedValue)
				if !ok || len(tagged.Fields) != 1 {
					return acc, false, fmt.Errorf("handler must return FoldStep[a] (Continue(acc)|Stop(acc)), got %T", stepVal)
				}
				acc = tagged.Fields[0]
				switch tagged.CtorName {
				case "Stop":
					return acc, true, nil
				case "Continue":
					// fall through to next token
				default:
					return acc, false, fmt.Errorf("handler returned unknown FoldStep constructor %q (expected Continue or Stop)", tagged.CtorName)
				}
			} else {
				var descErr error
				var stop bool
				acc, stop, descErr = scanForElementsFoldStepInner(decoder, tagName, acc, handler)
				if descErr != nil {
					return acc, false, descErr
				}
				if stop {
					return acc, true, nil
				}
			}
		case xml.EndElement:
			return acc, false, nil
		}
	}
}

// ============================================================================
// scanForElementsFold: Shared streaming XML fold helper
// ============================================================================

// scanForElementsFold walks the XML token stream, calling handler for each
// element matching tagName with a threaded accumulator. This is the fold
// variant of scanForElements — O(largest element + accumulator) memory.
//
// Used by both _xml_parseFold (string input) and _zip_xml_scanFold (stream input).
func scanForElementsFold(decoder *xml.Decoder, tagName string, acc eval.Value, handler func(node eval.Value, acc eval.Value) (eval.Value, error)) (eval.Value, error) {
	return scanForElementsFoldInner(decoder, tagName, acc, handler)
}

func scanForElementsFoldInner(decoder *xml.Decoder, tagName string, acc eval.Value, handler func(node eval.Value, acc eval.Value) (eval.Value, error)) (eval.Value, error) {
	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return acc, nil
			}
			return acc, nil // non-EOF error — stop scanning
		}

		switch t := tok.(type) {
		case xml.StartElement:
			resolvedName := resolveTagName(t.Name, nil)
			if resolvedName == tagName {
				// Build subtree for this matched element
				localPM := extractPrefixMap(t, nil)
				attrs := buildAttrs(t, localPM)
				childNodes, parseErr := parseXmlChildren(decoder, 1, localPM)
				if parseErr != nil {
					return acc, parseErr
				}
				finalTag := resolveTagName(t.Name, localPM)
				node := makeXmlElement(finalTag, attrs, childNodes)

				// Call handler with current accumulator
				newAcc, handlerErr := handler(node, acc)
				if handlerErr != nil {
					return acc, handlerErr
				}
				acc = newAcc
			} else {
				// Descend into non-matching element
				var descErr error
				acc, descErr = scanForElementsFoldInner(decoder, tagName, acc, handler)
				if descErr != nil {
					return acc, descErr
				}
			}
		case xml.EndElement:
			return acc, nil // End of current element scope
		}
		// CharData, Comment, etc. — skip without allocation
	}
}
