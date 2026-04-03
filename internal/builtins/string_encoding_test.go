package builtins

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

// ============================================================================
// decodeQuotedPrintable tests (RFC 2045 §6.7)
// ============================================================================

func TestDecodeQP_BasicHex(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"space", "hello=20world", "hello world"},
		{"tab", "hello=09world", "hello\tworld"},
		{"equals sign", "3=3D4", "3=4"},
		{"crlf", "line1=0D=0Aline2", "line1\r\nline2"},
		{"lf only", "line1=0Aline2", "line1\nline2"},
		{"uppercase hex", "=C3=A9", "\xc3\xa9"},  // é in UTF-8
		{"lowercase hex", "=c3=a9", "\xc3\xa9"},  // also valid
		{"mixed case", "=C3=a9", "\xc3\xa9"},     // mixed case hex
		{"multiple escapes", "=41=42=43", "ABC"}, // ABC
		{"no escapes", "hello world", "hello world"},
		{"empty string", "", ""},
		{"all escaped", "=48=65=6C=6C=6F", "Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeQPImpl(nil, []eval.Value{
				&eval.StringValue{Value: tt.input},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			sv, ok := result.(*eval.StringValue)
			if !ok {
				t.Fatalf("expected StringValue, got %T", result)
			}
			if sv.Value != tt.want {
				t.Errorf("decodeQP(%q) = %q, want %q", tt.input, sv.Value, tt.want)
			}
		})
	}
}

func TestDecodeQP_SoftLineBreaks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"soft break CRLF", "line1=\r\nline2", "line1line2"},
		{"soft break LF", "line1=\nline2", "line1line2"},
		{"soft break CR only", "line1=\rline2", "line1line2"},
		{"multiple soft breaks", "a=\r\nb=\r\nc", "abc"},
		{"soft break at end", "hello=\r\n", "hello"},
		{"soft break then hex", "he=\r\nllo=20world", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeQPImpl(nil, []eval.Value{
				&eval.StringValue{Value: tt.input},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			sv := result.(*eval.StringValue)
			if sv.Value != tt.want {
				t.Errorf("decodeQP(%q) = %q, want %q", tt.input, sv.Value, tt.want)
			}
		})
	}
}

func TestDecodeQP_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"equals at end", "hello=", "hello="},
		{"equals then one char", "hello=2", "hello=2"},
		{"invalid hex GG", "=GG", "=GG"},
		{"invalid hex ZZ", "=ZZ", "=ZZ"},
		{"partial then valid", "=G=41", "=GA"},
		{"consecutive equals", "==20", "= "},
		{"just equals", "=", "="},
		{"just two equals", "==", "=="},
		{"real QP line", "If you believe that truth =3D beauty, then surely=20teleology...",
			"If you believe that truth = beauty, then surely teleology..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeQPImpl(nil, []eval.Value{
				&eval.StringValue{Value: tt.input},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			sv := result.(*eval.StringValue)
			if sv.Value != tt.want {
				t.Errorf("decodeQP(%q) = %q, want %q", tt.input, sv.Value, tt.want)
			}
		})
	}
}

func TestDecodeQP_Determinism(t *testing.T) {
	// Pure builtins must be deterministic — run multiple times
	input := "hello=20=C3=A9=0D=0Aworld=\r\nfoo=3Dbar"
	want := "hello \xc3\xa9\r\nworldfoo=bar"

	for i := 0; i < 20; i++ {
		result, err := decodeQPImpl(nil, []eval.Value{
			&eval.StringValue{Value: input},
		})
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v", i, err)
		}
		sv := result.(*eval.StringValue)
		if sv.Value != want {
			t.Errorf("run %d: got %q, want %q", i, sv.Value, want)
		}
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkDecodeQP_Small(b *testing.B) {
	input := &eval.StringValue{Value: "hello=20world=0D=0Afoo=3Dbar"}
	args := []eval.Value{input}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = decodeQPImpl(nil, args)
	}
}

// ============================================================================
// replaceMany tests
// ============================================================================

func makeTuplePair(old, new string) eval.Value {
	return &eval.TupleValue{Elements: []eval.Value{
		&eval.StringValue{Value: old},
		&eval.StringValue{Value: new},
	}}
}

func TestReplaceMany_Basic(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		replacements []eval.Value
		want         string
	}{
		{
			"single pair",
			"a&amp;b",
			[]eval.Value{makeTuplePair("&amp;", "&")},
			"a&b",
		},
		{
			"multiple pairs",
			"a&amp;b&lt;c&gt;d",
			[]eval.Value{
				makeTuplePair("&amp;", "&"),
				makeTuplePair("&lt;", "<"),
				makeTuplePair("&gt;", ">"),
			},
			"a&b<c>d",
		},
		{
			"empty list",
			"hello world",
			[]eval.Value{},
			"hello world",
		},
		{
			"no matches",
			"hello world",
			[]eval.Value{makeTuplePair("xyz", "abc")},
			"hello world",
		},
		{
			"empty string input",
			"",
			[]eval.Value{makeTuplePair("a", "b")},
			"",
		},
		{
			"unicode replacements",
			"cafe\u0301",
			[]eval.Value{makeTuplePair("e\u0301", "é")},
			"café",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list := &eval.ListValue{Elements: tt.replacements}
			result, err := strReplaceManyImpl(nil, []eval.Value{
				&eval.StringValue{Value: tt.input},
				list,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			sv := result.(*eval.StringValue)
			if sv.Value != tt.want {
				t.Errorf("replaceMany(%q) = %q, want %q", tt.input, sv.Value, tt.want)
			}
		})
	}
}

func TestReplaceMany_Determinism(t *testing.T) {
	replacements := &eval.ListValue{Elements: []eval.Value{
		makeTuplePair("&amp;", "&"),
		makeTuplePair("&lt;", "<"),
		makeTuplePair("&gt;", ">"),
	}}
	input := "a&amp;b&lt;c&gt;d"
	want := "a&b<c>d"

	for i := 0; i < 20; i++ {
		result, err := strReplaceManyImpl(nil, []eval.Value{
			&eval.StringValue{Value: input},
			replacements,
		})
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v", i, err)
		}
		sv := result.(*eval.StringValue)
		if sv.Value != want {
			t.Errorf("run %d: got %q, want %q", i, sv.Value, want)
		}
	}
}

func BenchmarkReplaceMany_23Patterns_50KB(b *testing.B) {
	// Simulate HTML entity decoding on a 50KB string
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("Some text with &amp; entities &lt;tag&gt; and &quot;quotes&quot; here. ")
	}
	input := sb.String()

	replacements := &eval.ListValue{Elements: []eval.Value{
		makeTuplePair("&amp;", "&"),
		makeTuplePair("&lt;", "<"),
		makeTuplePair("&gt;", ">"),
		makeTuplePair("&quot;", "\""),
		makeTuplePair("&#39;", "'"),
		makeTuplePair("&nbsp;", " "),
		makeTuplePair("&mdash;", "—"),
		makeTuplePair("&ndash;", "–"),
		makeTuplePair("&lsquo;", "'"),
		makeTuplePair("&rsquo;", "'"),
		makeTuplePair("&ldquo;", "\u201C"),
		makeTuplePair("&rdquo;", "\u201D"),
		makeTuplePair("&hellip;", "…"),
		makeTuplePair("&copy;", "©"),
		makeTuplePair("&reg;", "®"),
		makeTuplePair("&trade;", "™"),
		makeTuplePair("&bull;", "•"),
		makeTuplePair("&middot;", "·"),
		makeTuplePair("&laquo;", "«"),
		makeTuplePair("&raquo;", "»"),
		makeTuplePair("&deg;", "°"),
		makeTuplePair("&plusmn;", "±"),
		makeTuplePair("&times;", "×"),
	}}

	args := []eval.Value{&eval.StringValue{Value: input}, replacements}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = strReplaceManyImpl(nil, args)
	}
}

// ============================================================================
// foldSlices tests
// ============================================================================

func TestFoldSlices_CountSegments(t *testing.T) {
	ctx := newTestEffCtx()
	countFn := goFn(func(args []eval.Value) (eval.Value, error) {
		acc := args[0].(*eval.IntValue).Value
		return &eval.IntValue{Value: acc + 1}, nil
	})

	result, err := strFoldSlicesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: "a,b,c,d"},
		&eval.StringValue{Value: ","},
		&eval.IntValue{Value: 0},
		countFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	iv := result.(*eval.IntValue)
	if iv.Value != 4 {
		t.Errorf("expected 4 segments, got %d", iv.Value)
	}
}

func TestFoldSlices_ConcatSegments(t *testing.T) {
	ctx := newTestEffCtx()
	concatFn := goFn(func(args []eval.Value) (eval.Value, error) {
		acc := args[0].(*eval.StringValue).Value
		seg := args[1].(*eval.StringValue).Value
		if acc == "" {
			return &eval.StringValue{Value: seg}, nil
		}
		return &eval.StringValue{Value: acc + "|" + seg}, nil
	})

	result, err := strFoldSlicesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: "hello world foo"},
		&eval.StringValue{Value: " "},
		&eval.StringValue{Value: ""},
		concatFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv := result.(*eval.StringValue)
	if sv.Value != "hello|world|foo" {
		t.Errorf("got %q, want %q", sv.Value, "hello|world|foo")
	}
}

func TestFoldSlices_NoDelimiter(t *testing.T) {
	ctx := newTestEffCtx()
	concatFn := goFn(func(args []eval.Value) (eval.Value, error) {
		acc := args[0].(*eval.StringValue).Value
		seg := args[1].(*eval.StringValue).Value
		return &eval.StringValue{Value: acc + seg}, nil
	})

	result, err := strFoldSlicesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: "hello"},
		&eval.StringValue{Value: "X"}, // no X in input
		&eval.StringValue{Value: ""},
		concatFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv := result.(*eval.StringValue)
	if sv.Value != "hello" {
		t.Errorf("got %q, want %q", sv.Value, "hello")
	}
}

func TestFoldSlices_EmptyString(t *testing.T) {
	ctx := newTestEffCtx()
	countFn := goFn(func(args []eval.Value) (eval.Value, error) {
		acc := args[0].(*eval.IntValue).Value
		return &eval.IntValue{Value: acc + 1}, nil
	})

	result, err := strFoldSlicesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: ""},
		&eval.StringValue{Value: ","},
		&eval.IntValue{Value: 0},
		countFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	iv := result.(*eval.IntValue)
	if iv.Value != 1 {
		t.Errorf("expected 1 segment (empty string), got %d", iv.Value)
	}
}

func TestFoldSlices_DelimiterAtEdges(t *testing.T) {
	ctx := newTestEffCtx()
	countFn := goFn(func(args []eval.Value) (eval.Value, error) {
		acc := args[0].(*eval.IntValue).Value
		return &eval.IntValue{Value: acc + 1}, nil
	})

	// ",a,b," splits into ["", "a", "b", ""]
	result, err := strFoldSlicesImpl(ctx, []eval.Value{
		&eval.StringValue{Value: ",a,b,"},
		&eval.StringValue{Value: ","},
		&eval.IntValue{Value: 0},
		countFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	iv := result.(*eval.IntValue)
	if iv.Value != 4 {
		t.Errorf("expected 4 segments, got %d", iv.Value)
	}
}

// ============================================================================
// mapSlicesJoin tests
// ============================================================================

func TestMapSlicesJoin_ToUpper(t *testing.T) {
	ctx := newTestEffCtx()
	upperFn := goFn(func(args []eval.Value) (eval.Value, error) {
		s := args[0].(*eval.StringValue).Value
		return &eval.StringValue{Value: strings.ToUpper(s)}, nil
	})

	result, err := strMapSlicesJoinImpl(ctx, []eval.Value{
		&eval.StringValue{Value: "hello,world,foo"},
		&eval.StringValue{Value: ","},
		upperFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv := result.(*eval.StringValue)
	if sv.Value != "HELLOWORLDFOO" {
		t.Errorf("got %q, want %q", sv.Value, "HELLOWORLDFOO")
	}
}

func TestMapSlicesJoin_Identity(t *testing.T) {
	ctx := newTestEffCtx()
	identityFn := goFn(func(args []eval.Value) (eval.Value, error) {
		return args[0], nil
	})

	// Identity transform with delimiter removed = join without delimiter
	result, err := strMapSlicesJoinImpl(ctx, []eval.Value{
		&eval.StringValue{Value: "a=b=c"},
		&eval.StringValue{Value: "="},
		identityFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv := result.(*eval.StringValue)
	if sv.Value != "abc" {
		t.Errorf("got %q, want %q", sv.Value, "abc")
	}
}

func TestMapSlicesJoin_EmptyString(t *testing.T) {
	ctx := newTestEffCtx()
	identityFn := goFn(func(args []eval.Value) (eval.Value, error) {
		return args[0], nil
	})

	result, err := strMapSlicesJoinImpl(ctx, []eval.Value{
		&eval.StringValue{Value: ""},
		&eval.StringValue{Value: ","},
		identityFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv := result.(*eval.StringValue)
	if sv.Value != "" {
		t.Errorf("got %q, want %q", sv.Value, "")
	}
}

func TestMapSlicesJoin_Expansion(t *testing.T) {
	ctx := newTestEffCtx()
	// Each segment gets wrapped in brackets
	wrapFn := goFn(func(args []eval.Value) (eval.Value, error) {
		s := args[0].(*eval.StringValue).Value
		return &eval.StringValue{Value: "[" + s + "]"}, nil
	})

	result, err := strMapSlicesJoinImpl(ctx, []eval.Value{
		&eval.StringValue{Value: "a,b,c"},
		&eval.StringValue{Value: ","},
		wrapFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv := result.(*eval.StringValue)
	if sv.Value != "[a][b][c]" {
		t.Errorf("got %q, want %q", sv.Value, "[a][b][c]")
	}
}

func TestMapSlicesJoin_Determinism(t *testing.T) {
	ctx := newTestEffCtx()
	upperFn := goFn(func(args []eval.Value) (eval.Value, error) {
		s := args[0].(*eval.StringValue).Value
		return &eval.StringValue{Value: strings.ToUpper(s)}, nil
	})

	want := "HELLOWORLDFOO"
	for i := 0; i < 20; i++ {
		result, err := strMapSlicesJoinImpl(ctx, []eval.Value{
			&eval.StringValue{Value: "hello,world,foo"},
			&eval.StringValue{Value: ","},
			upperFn,
		})
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v", i, err)
		}
		sv := result.(*eval.StringValue)
		if sv.Value != want {
			t.Errorf("run %d: got %q, want %q", i, sv.Value, want)
		}
	}
}

func BenchmarkMapSlicesJoin_3000Segs(b *testing.B) {
	ctx := newTestEffCtx()
	identityFn := goFn(func(args []eval.Value) (eval.Value, error) {
		return args[0], nil
	})

	// Build a string with 3000 segments separated by "="
	var sb strings.Builder
	for i := 0; i < 3000; i++ {
		if i > 0 {
			sb.WriteString("=")
		}
		sb.WriteString("segment")
	}
	input := sb.String()
	args := []eval.Value{
		&eval.StringValue{Value: input},
		&eval.StringValue{Value: "="},
		identityFn,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = strMapSlicesJoinImpl(ctx, args)
	}
}

// ============================================================================
// ASCII fast-path tests
// ============================================================================

func TestIsASCII(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"hello world", true},
		{"", true},
		{"abc123!@#", true},
		{"\t\n\r", true},
		{"caf\xc3\xa9", false}, // café
		{"\xff", false},
		{"hello\x80world", false},
	}
	for _, tt := range tests {
		got := isASCII(tt.input)
		if got != tt.want {
			t.Errorf("isASCII(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestStrSlice_ASCIIFastPath(t *testing.T) {
	// Verify ASCII strings produce same results as before
	tests := []struct {
		name       string
		input      string
		start, end int
		want       string
	}{
		{"basic", "hello", 1, 4, "ell"},
		{"full", "hello", 0, 5, "hello"},
		{"empty", "hello", 2, 2, ""},
		{"clamp end", "hello", 0, 100, "hello"},
		{"clamp start", "hello", -5, 3, "hel"},
		{"reversed", "hello", 4, 2, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := strSliceImpl(nil, []eval.Value{
				&eval.StringValue{Value: tt.input},
				&eval.IntValue{Value: tt.start},
				&eval.IntValue{Value: tt.end},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			sv := result.(*eval.StringValue)
			if sv.Value != tt.want {
				t.Errorf("got %q, want %q", sv.Value, tt.want)
			}
		})
	}
}

func TestStrSlice_UnicodeStillWorks(t *testing.T) {
	// café — 4 runes, 5 bytes (é is 2 bytes)
	result, err := strSliceImpl(nil, []eval.Value{
		&eval.StringValue{Value: "caf\xc3\xa9"},
		&eval.IntValue{Value: 0},
		&eval.IntValue{Value: 4},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sv := result.(*eval.StringValue)
	if sv.Value != "caf\xc3\xa9" {
		t.Errorf("got %q, want %q", sv.Value, "café")
	}
}

func TestStrFind_ASCIIFastPath(t *testing.T) {
	result, err := strFindImpl(nil, []eval.Value{
		&eval.StringValue{Value: "hello world"},
		&eval.StringValue{Value: "world"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	iv := result.(*eval.IntValue)
	if iv.Value != 6 {
		t.Errorf("got %d, want 6", iv.Value)
	}
}

func BenchmarkStrSlice_ASCII_50KB(b *testing.B) {
	input := strings.Repeat("a", 50000)
	args := []eval.Value{
		&eval.StringValue{Value: input},
		&eval.IntValue{Value: 100},
		&eval.IntValue{Value: 49900},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = strSliceImpl(nil, args)
	}
}

func BenchmarkStrSlice_Unicode_50KB(b *testing.B) {
	// Mix of ASCII and multi-byte — forces rune path
	input := strings.Repeat("a\xc3\xa9", 25000)
	args := []eval.Value{
		&eval.StringValue{Value: input},
		&eval.IntValue{Value: 100},
		&eval.IntValue{Value: 49900},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = strSliceImpl(nil, args)
	}
}

func BenchmarkDecodeQP_37KB(b *testing.B) {
	// Simulate a 37KB QP-encoded email body
	// Mix of plain text and =XX escapes (roughly 1 escape per 12 chars)
	var sb strings.Builder
	for i := 0; i < 3000; i++ {
		sb.WriteString("Some text=20")
	}
	input := &eval.StringValue{Value: sb.String()}
	args := []eval.Value{input}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = decodeQPImpl(nil, args)
	}
}

// ============================================================================
// M5: startsWithIgnoreCase Tests
// ============================================================================

func TestStartsWithIgnoreCase(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		prefix string
		want   bool
	}{
		{"exact match", "Hello", "Hello", true},
		{"case differs", "Hello World", "hello", true},
		{"upper prefix", "content-type", "CONTENT-TYPE", true},
		{"partial prefix", "Content-Type: text/html", "content-type", true},
		{"no match", "Hello", "World", false},
		{"prefix longer", "Hi", "Hello", false},
		{"empty prefix", "Hello", "", true},
		{"empty string", "", "Hello", false},
		{"both empty", "", "", true},
		{"single char", "A", "a", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{
				&eval.StringValue{Value: tt.s},
				&eval.StringValue{Value: tt.prefix},
			}
			result, err := strStartsWithICImpl(nil, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			bv, ok := result.(*eval.BoolValue)
			if !ok {
				t.Fatalf("expected BoolValue, got %T", result)
			}
			if bv.Value != tt.want {
				t.Errorf("startsWithIC(%q, %q) = %v, want %v", tt.s, tt.prefix, bv.Value, tt.want)
			}
		})
	}
}

func TestStartsWithIgnoreCase_Determinism(t *testing.T) {
	args := []eval.Value{
		&eval.StringValue{Value: "Content-Type: text/html"},
		&eval.StringValue{Value: "content-type"},
	}
	for i := 0; i < 20; i++ {
		result, err := strStartsWithICImpl(nil, args)
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v", i, err)
		}
		bv := result.(*eval.BoolValue)
		if !bv.Value {
			t.Fatalf("run %d: expected true", i)
		}
	}
}
