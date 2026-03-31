package builtins

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// ============================================================================
// Tests for _zip_xml_scanFold (combined streaming ZIP+XML fold)
// ============================================================================

// zipXmlTestCtx creates an EffContext with FS capability and FnCallerN wired.
func zipXmlTestCtx(t *testing.T) *xmlFoldCtx {
	t.Helper()
	ctx := makeTestCtx(t) // FS capability granted
	ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
		goFn, ok := fn.(*eval.BuiltinFunction)
		if !ok {
			return nil, fmt.Errorf("test FnCaller: expected BuiltinFunction, got %T", fn)
		}
		return goFn.Fn([]eval.Value{arg})
	}
	ctx.FnCallerN = func(fn eval.Value, args []eval.Value) (eval.Value, error) {
		goFn, ok := fn.(*eval.BuiltinFunction)
		if !ok {
			return nil, fmt.Errorf("test FnCallerN: expected BuiltinFunction, got %T", fn)
		}
		return goFn.Fn(args)
	}
	return ctx
}

// Type alias to avoid confusing the compiler — just use *effects.EffContext
type xmlFoldCtx = effects.EffContext

func TestZipXmlScanFold_Basic(t *testing.T) {
	dir := t.TempDir()
	xmlContent := `<root><item>one</item><item>two</item><item>three</item></root>`
	zipPath := createTestZip(t, dir, map[string][]byte{
		"data.xml": []byte(xmlContent),
	})

	ctx := zipXmlTestCtx(t)
	handler := &eval.BuiltinFunction{
		Name: "collect_text",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.ListValue)
			node := args[1]
			var buf strings.Builder
			collectText(node, &buf)
			newElems := make([]eval.Value, len(acc.Elements)+1)
			copy(newElems, acc.Elements)
			newElems[len(acc.Elements)] = &eval.StringValue{Value: buf.String()}
			return &eval.ListValue{Elements: newElems}, nil
		},
	}

	result, err := zipXmlScanFoldImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "data.xml"},
		&eval.StringValue{Value: "item"},
		&eval.ListValue{Elements: nil},
		handler,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := assertOk(t, result)
	list, ok := inner.(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", inner)
	}
	if len(list.Elements) != 3 {
		t.Fatalf("expected 3 items, got %d", len(list.Elements))
	}
	for i, expected := range []string{"one", "two", "three"} {
		sv := list.Elements[i].(*eval.StringValue)
		if sv.Value != expected {
			t.Errorf("element %d: expected %q, got %q", i, expected, sv.Value)
		}
	}
}

func TestZipXmlScanFold_SharedStringsPattern(t *testing.T) {
	// Simulate XLSX shared strings: <sst><si><t>text</t></si>...</sst>
	dir := t.TempDir()
	var sb strings.Builder
	sb.WriteString(`<sst count="100">`)
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&sb, `<si><t>string_%d</t></si>`, i)
	}
	sb.WriteString(`</sst>`)

	zipPath := createTestZip(t, dir, map[string][]byte{
		"xl/sharedStrings.xml": []byte(sb.String()),
	})

	ctx := zipXmlTestCtx(t)
	handler := &eval.BuiltinFunction{
		Name: "collect_text",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.ListValue)
			node := args[1]
			var buf strings.Builder
			collectText(node, &buf)
			newElems := make([]eval.Value, len(acc.Elements)+1)
			copy(newElems, acc.Elements)
			newElems[len(acc.Elements)] = &eval.StringValue{Value: buf.String()}
			return &eval.ListValue{Elements: newElems}, nil
		},
	}

	result, err := zipXmlScanFoldImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "xl/sharedStrings.xml"},
		&eval.StringValue{Value: "si"},
		&eval.ListValue{Elements: nil},
		handler,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := assertOk(t, result)
	list := inner.(*eval.ListValue)
	if len(list.Elements) != 100 {
		t.Fatalf("expected 100 shared strings, got %d", len(list.Elements))
	}

	// Check first and last
	first := list.Elements[0].(*eval.StringValue)
	if first.Value != "string_0" {
		t.Errorf("first: expected 'string_0', got %q", first.Value)
	}
	last := list.Elements[99].(*eval.StringValue)
	if last.Value != "string_99" {
		t.Errorf("last: expected 'string_99', got %q", last.Value)
	}
}

func TestZipXmlScanFold_CountElements(t *testing.T) {
	dir := t.TempDir()
	var sb strings.Builder
	sb.WriteString("<root>")
	for i := 0; i < 500; i++ {
		fmt.Fprintf(&sb, `<row id="%d"><c>%d</c></row>`, i, i)
	}
	sb.WriteString("</root>")

	zipPath := createTestZip(t, dir, map[string][]byte{
		"sheet1.xml": []byte(sb.String()),
	})

	ctx := zipXmlTestCtx(t)
	counter := &eval.BuiltinFunction{
		Name: "counter",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.IntValue)
			return &eval.IntValue{Value: acc.Value + 1}, nil
		},
	}

	result, err := zipXmlScanFoldImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "sheet1.xml"},
		&eval.StringValue{Value: "row"},
		&eval.IntValue{Value: 0},
		counter,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := assertOk(t, result)
	iv := inner.(*eval.IntValue)
	if iv.Value != 500 {
		t.Errorf("expected count 500, got %d", iv.Value)
	}
}

func TestZipXmlScanFold_EntryNotFound(t *testing.T) {
	dir := t.TempDir()
	zipPath := createTestZip(t, dir, map[string][]byte{
		"exists.xml": []byte("<root/>"),
	})

	ctx := zipXmlTestCtx(t)
	handler := &eval.BuiltinFunction{
		Name: "noop",
		Fn:   func(args []eval.Value) (eval.Value, error) { return args[0], nil },
	}

	result, err := zipXmlScanFoldImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "missing.xml"},
		&eval.StringValue{Value: "item"},
		&eval.IntValue{Value: 0},
		handler,
	})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}

	tv := result.(*eval.TaggedValue)
	if tv.CtorName != "Err" {
		t.Fatalf("expected Err for missing entry, got %s", tv.CtorName)
	}
	errMsg := tv.Fields[0].(*eval.StringValue).Value
	if !strings.Contains(errMsg, "entry not found") {
		t.Errorf("expected 'entry not found' in error, got %q", errMsg)
	}
}

func TestZipXmlScanFold_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	zipPath := createTestZip(t, dir, map[string][]byte{
		"data.xml": []byte("<root/>"),
	})

	ctx := zipXmlTestCtx(t)
	handler := &eval.BuiltinFunction{
		Name: "noop",
		Fn:   func(args []eval.Value) (eval.Value, error) { return args[0], nil },
	}

	result, err := zipXmlScanFoldImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "../../etc/passwd"},
		&eval.StringValue{Value: "item"},
		&eval.IntValue{Value: 0},
		handler,
	})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}

	tv := result.(*eval.TaggedValue)
	if tv.CtorName != "Err" {
		t.Fatalf("expected Err for path traversal, got %s", tv.CtorName)
	}
	errMsg := tv.Fields[0].(*eval.StringValue).Value
	if !strings.Contains(errMsg, "path traversal") {
		t.Errorf("expected 'path traversal' in error, got %q", errMsg)
	}
}

func TestZipXmlScanFold_HandlerError(t *testing.T) {
	dir := t.TempDir()
	xmlContent := `<root><item>a</item><item>b</item></root>`
	zipPath := createTestZip(t, dir, map[string][]byte{
		"data.xml": []byte(xmlContent),
	})

	ctx := zipXmlTestCtx(t)
	errorHandler := &eval.BuiltinFunction{
		Name: "error_handler",
		Fn: func(args []eval.Value) (eval.Value, error) {
			return nil, fmt.Errorf("handler exploded")
		},
	}

	result, err := zipXmlScanFoldImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "data.xml"},
		&eval.StringValue{Value: "item"},
		&eval.IntValue{Value: 0},
		errorHandler,
	})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}

	tv := result.(*eval.TaggedValue)
	if tv.CtorName != "Err" {
		t.Fatalf("expected Err for handler error, got %s", tv.CtorName)
	}
	errMsg := tv.Fields[0].(*eval.StringValue).Value
	if !strings.Contains(errMsg, "handler exploded") {
		t.Errorf("expected 'handler exploded' in error, got %q", errMsg)
	}
}

func TestZipXmlScanFold_NoMatches(t *testing.T) {
	dir := t.TempDir()
	xmlContent := `<root><item>one</item></root>`
	zipPath := createTestZip(t, dir, map[string][]byte{
		"data.xml": []byte(xmlContent),
	})

	ctx := zipXmlTestCtx(t)
	counter := &eval.BuiltinFunction{
		Name: "counter",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.IntValue)
			return &eval.IntValue{Value: acc.Value + 1}, nil
		},
	}

	result, err := zipXmlScanFoldImpl(ctx, []eval.Value{
		&eval.StringValue{Value: zipPath},
		&eval.StringValue{Value: "data.xml"},
		&eval.StringValue{Value: "nonexistent"},
		&eval.IntValue{Value: 0},
		counter,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := assertOk(t, result)
	iv := inner.(*eval.IntValue)
	if iv.Value != 0 {
		t.Errorf("expected 0 (no matches), got %d", iv.Value)
	}
}

func TestZipXmlScanFold_Deterministic(t *testing.T) {
	// Run 20 times to verify deterministic output (A1 compliance)
	dir := t.TempDir()
	var sb strings.Builder
	sb.WriteString("<root>")
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&sb, `<si><t>s%d</t></si>`, i)
	}
	sb.WriteString("</root>")

	zipPath := createTestZip(t, dir, map[string][]byte{
		"strings.xml": []byte(sb.String()),
	})

	handler := &eval.BuiltinFunction{
		Name: "collect",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.ListValue)
			node := args[1]
			var buf strings.Builder
			collectText(node, &buf)
			newElems := make([]eval.Value, len(acc.Elements)+1)
			copy(newElems, acc.Elements)
			newElems[len(acc.Elements)] = &eval.StringValue{Value: buf.String()}
			return &eval.ListValue{Elements: newElems}, nil
		},
	}

	var firstRun []string
	for run := 0; run < 20; run++ {
		ctx := zipXmlTestCtx(t)
		result, err := zipXmlScanFoldImpl(ctx, []eval.Value{
			&eval.StringValue{Value: zipPath},
			&eval.StringValue{Value: "strings.xml"},
			&eval.StringValue{Value: "si"},
			&eval.ListValue{Elements: nil},
			handler,
		})
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v", run, err)
		}

		inner := assertOk(t, result)
		list := inner.(*eval.ListValue)
		strs := make([]string, len(list.Elements))
		for i, el := range list.Elements {
			strs[i] = el.(*eval.StringValue).Value
		}

		if run == 0 {
			firstRun = strs
		} else {
			if len(strs) != len(firstRun) {
				t.Fatalf("run %d: length mismatch: %d vs %d", run, len(strs), len(firstRun))
			}
			for i := range strs {
				if strs[i] != firstRun[i] {
					t.Errorf("run %d element %d: %q vs %q", run, i, strs[i], firstRun[i])
				}
			}
		}
	}
}
