package builtins

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// M-STREAMING-ZIP-XML: Combined ZIP entry + XML fold builtin
//
// Pipes a ZIP entry's decompressed stream directly into an XML decoder,
// folding over matching elements via callback. The decompressed XML is
// never materialized as a contiguous string in memory.
//
// Memory: O(largest matched element + accumulator) instead of O(entire entry)

func init() {
	registerZipXmlScanFold()
}

// ============================================================================
// _zip_xml_scanFold: Streaming ZIP+XML fold
// (zipPath, entryName, tag, init, handler) -> Result[a, string] ! {FS}
// ============================================================================

func registerZipXmlScanFold() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/zip",
		Name:    "_zip_xml_scanFold",
		NumArgs: 5,
		IsPure:  false,
		Effect:  "FS",
		Type:    makeZipXmlScanFoldType,
		Impl:    zipXmlScanFoldImpl,
		Metadata: &BuiltinMetadata{
			Description: "Stream a ZIP entry through XML parser, folding over matching elements",
			LongDesc:    "Opens a ZIP entry as a decompressed stream, pipes it directly into an XML decoder, and folds over elements matching the given tag. The decompressed XML is never held entirely in memory — only the current element and accumulator exist at any time. Memory usage is O(largest element + accumulator) instead of O(entire decompressed entry).",
			Params: []ParamDoc{
				{Name: "zipPath", Description: "Path to the ZIP archive"},
				{Name: "entryName", Description: "Name of the XML entry within the archive"},
				{Name: "tagName", Description: "XML tag name to match"},
				{Name: "init", Description: "Initial accumulator value"},
				{Name: "handler", Description: "Fold function: (acc, XmlNode) -> acc"},
			},
			Returns: "Result[a, string] - Ok(final accumulator) or Err(message)",
			Examples: []Example{
				{Code: `_zip_xml_scanFold("doc.xlsx", "xl/sharedStrings.xml", "si", [], \acc, node. acc ++ [getText(node)])`, Description: "Extract shared strings from XLSX without OOM"},
			},
			SeeAlso:   []string{"_xml_parseFold", "_zip_readEntry", "_xml_parseElements"},
			Since:     "v0.11.0",
			Stability: StabilityStable,
			Tags:      []string{"zip", "xml", "streaming", "fold", "performance"},
			Category:  "zip",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _zip_xml_scanFold: %v", err))
	}
}

// Type: forall a. (string, string, string, a, (a, XmlNode) -> a) -> Result[a, string] ! {FS}
func makeZipXmlScanFoldType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	xmlNodeType := T.Con("XmlNode")
	fn := T.Func(a, xmlNodeType).Returns(a).Build()
	return T.Func(T.String(), T.String(), T.String(), a, fn).Returns(
		T.App("Result", a, T.String()),
	).Effects("FS")
}

func zipXmlScanFoldImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	pathVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_zip_xml_scanFold: expected String for zipPath, got %T", args[0])
	}
	entryVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_zip_xml_scanFold: expected String for entryName, got %T", args[1])
	}
	tagVal, ok := args[2].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_zip_xml_scanFold: expected String for tagName, got %T", args[2])
	}
	acc := args[3]
	handler := args[4]

	if ctx == nil || ctx.FnCallerN == nil {
		return nil, fmt.Errorf("_zip_xml_scanFold: FnCallerN not set (evaluator not wired)")
	}

	entryName := entryVal.Value
	if strings.Contains(entryName, "..") {
		return zipMakeErr(fmt.Sprintf("path traversal rejected: %s", entryName)), nil
	}

	zipPath := pathVal.Value
	if ctx.Env.Sandbox != "" {
		zipPath = filepath.Join(ctx.Env.Sandbox, zipPath)
	}

	// 1. Open ZIP archive
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return zipMakeErr(fmt.Sprintf("cannot open ZIP: %v", err)), nil
	}
	defer archive.Close()

	// 2. Find entry
	var entry *zip.File
	for _, f := range archive.File {
		if f.Name == entryName {
			entry = f
			break
		}
	}
	if entry == nil {
		return zipMakeErr(fmt.Sprintf("entry not found: %s", entryName)), nil
	}

	// 3. Open decompressed stream (NOT io.ReadAll!)
	rc, err := entry.Open()
	if err != nil {
		return zipMakeErr(fmt.Sprintf("cannot read entry: %v", err)), nil
	}
	defer rc.Close()

	// 4. Pipe stream directly into XML decoder
	decoder := xml.NewDecoder(rc)
	tagName := tagVal.Value

	// 5. Fold over matching elements using shared scanForElementsFold
	var foldErr error
	acc, foldErr = scanForElementsFold(decoder, tagName, acc, func(node eval.Value, currentAcc eval.Value) (eval.Value, error) {
		return ctx.FnCallerN(handler, []eval.Value{currentAcc, node})
	})
	if foldErr != nil {
		return zipMakeErr(fmt.Sprintf("fold handler error: %v", foldErr)), nil
	}

	return zipMakeOk(acc), nil
}
