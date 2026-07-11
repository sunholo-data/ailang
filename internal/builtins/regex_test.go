package builtins

import (
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// mkStr is a tiny helper for building StringValue args.
func mkStr(s string) eval.Value { return &eval.StringValue{Value: s} }

func TestRegexCompile_OK(t *testing.T) {
	got, err := regexCompileImpl(nil, []eval.Value{mkStr(`^\d+$`)})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	tv, ok := got.(*eval.TaggedValue)
	if !ok || tv.CtorName != "Ok" {
		t.Fatalf("expected Ok(...), got %v", got)
	}
	if s, ok := tv.Fields[0].(*eval.StringValue); !ok || s.Value != `^\d+$` {
		t.Fatalf("expected Ok to carry the pattern, got %v", tv.Fields[0])
	}
}

func TestRegexCompile_ErrNeverPanics(t *testing.T) {
	// Unbalanced paren, lookaround, and backreference: all unsupported/invalid in
	// RE2. Each must return Err(msg), NOT panic (CLAUDE.md CP2).
	for _, pat := range []string{`(`, `(?=x)`, `(a)\1`} {
		got, err := regexCompileImpl(nil, []eval.Value{mkStr(pat)})
		if err != nil {
			t.Fatalf("pattern %q: unexpected Go error (should be Err value): %v", pat, err)
		}
		tv, ok := got.(*eval.TaggedValue)
		if !ok || tv.CtorName != "Err" {
			t.Fatalf("pattern %q: expected Err(...), got %v", pat, got)
		}
		if s, ok := tv.Fields[0].(*eval.StringValue); !ok || s.Value == "" {
			t.Fatalf("pattern %q: Err message must be non-empty", pat)
		}
	}
}

func TestRegexIsMatch(t *testing.T) {
	cases := []struct {
		pat, s string
		want   bool
	}{
		{`^\d+$`, "12345", true},
		{`^\d+$`, "12a45", false},
		{`foo`, "a foo b", true},
	}
	for _, c := range cases {
		got, err := regexIsMatchImpl(nil, []eval.Value{mkStr(c.pat), mkStr(c.s)})
		if err != nil {
			t.Fatalf("%q/%q: %v", c.pat, c.s, err)
		}
		b, ok := got.(*eval.BoolValue)
		if !ok || b.Value != c.want {
			t.Fatalf("isMatch(%q,%q) = %v, want %v", c.pat, c.s, got, c.want)
		}
	}
}

func TestRegexFindFirst_GroupsAndSpans(t *testing.T) {
	got, err := regexFindFirstImpl(nil, []eval.Value{mkStr(`^\[(\w+)\]\s+(.*)$`), mkStr("[ERROR] disk full")})
	if err != nil {
		t.Fatal(err)
	}
	tv := got.(*eval.TaggedValue)
	if tv.CtorName != "Some" {
		t.Fatalf("expected Some, got %v", tv.CtorName)
	}
	rec := tv.Fields[0].(*eval.RecordValue)
	if txt := rec.Fields["text"].(*eval.StringValue).Value; txt != "[ERROR] disk full" {
		t.Fatalf("whole-match text = %q", txt)
	}
	groups := rec.Fields["groups"].(*eval.ListValue).Elements
	if len(groups) != 3 { // group 0 = whole, 1 = level, 2 = body
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	g1 := groups[1].(*eval.RecordValue)
	g2 := groups[2].(*eval.RecordValue)
	if g1.Fields["text"].(*eval.StringValue).Value != "ERROR" {
		t.Fatalf("group1 text = %q", g1.Fields["text"].(*eval.StringValue).Value)
	}
	if g2.Fields["text"].(*eval.StringValue).Value != "disk full" {
		t.Fatalf("group2 text = %q", g2.Fields["text"].(*eval.StringValue).Value)
	}
}

func TestRegexFindFirst_NoMatch(t *testing.T) {
	got, err := regexFindFirstImpl(nil, []eval.Value{mkStr(`\d+`), mkStr("no digits here")})
	if err != nil {
		t.Fatal(err)
	}
	if tv := got.(*eval.TaggedValue); tv.CtorName != "None" {
		t.Fatalf("expected None, got %v", tv.CtorName)
	}
}

func TestRegexFindFirst_NonParticipatingGroup(t *testing.T) {
	// Optional group (a)? does not participate when matching "b" → start = -1.
	got, err := regexFindFirstImpl(nil, []eval.Value{mkStr(`(a)?b`), mkStr("b")})
	if err != nil {
		t.Fatal(err)
	}
	rec := got.(*eval.TaggedValue).Fields[0].(*eval.RecordValue)
	g1 := rec.Fields["groups"].(*eval.ListValue).Elements[1].(*eval.RecordValue)
	if g1.Fields["start"].(*eval.IntValue).Value != -1 {
		t.Fatalf("non-participating group should have start=-1, got %d", g1.Fields["start"].(*eval.IntValue).Value)
	}
	if g1.Fields["text"].(*eval.StringValue).Value != "" {
		t.Fatalf("non-participating group text should be empty")
	}
}

func TestRegexFindAll(t *testing.T) {
	got, err := regexFindAllImpl(nil, []eval.Value{mkStr(`\d+`), mkStr("a1 b22 c333")})
	if err != nil {
		t.Fatal(err)
	}
	elems := got.(*eval.ListValue).Elements
	if len(elems) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(elems))
	}
	want := []string{"1", "22", "333"}
	for i, e := range elems {
		if txt := e.(*eval.RecordValue).Fields["text"].(*eval.StringValue).Value; txt != want[i] {
			t.Fatalf("match %d = %q, want %q", i, txt, want[i])
		}
	}
}

func TestRegexReplaceAll_WithGroupRef(t *testing.T) {
	got, err := regexReplaceAllImpl(nil, []eval.Value{mkStr(`(\w+)@(\w+)`), mkStr("user@host"), mkStr("$2.$1")})
	if err != nil {
		t.Fatal(err)
	}
	if s := got.(*eval.StringValue).Value; s != "host.user" {
		t.Fatalf("replaceAll = %q, want %q", s, "host.user")
	}
}

func TestRegexSplit(t *testing.T) {
	got, err := regexSplitImpl(nil, []eval.Value{mkStr(`\s+`), mkStr("one   two\tthree")})
	if err != nil {
		t.Fatal(err)
	}
	elems := got.(*eval.ListValue).Elements
	want := []string{"one", "two", "three"}
	if len(elems) != len(want) {
		t.Fatalf("expected %d parts, got %d", len(want), len(elems))
	}
	for i, e := range elems {
		if s := e.(*eval.StringValue).Value; s != want[i] {
			t.Fatalf("part %d = %q, want %q", i, s, want[i])
		}
	}
}

// TestRegexLinearTime is the headline guarantee: the classic catastrophic-
// backtracking pattern completes in milliseconds because RE2 has no backtracking.
// A PCRE/Python/JS engine would take exponential time on this input.
func TestRegexLinearTime(t *testing.T) {
	pat := `(a+)+$`
	subject := strings.Repeat("a", 40) + "!"
	start := time.Now()
	got, err := regexIsMatchImpl(nil, []eval.Value{mkStr(pat), mkStr(subject)})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if got.(*eval.BoolValue).Value {
		t.Fatalf("pattern should NOT match (trailing '!')")
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("linear-time guarantee violated: took %v (a backtracking engine leaked in?)", elapsed)
	}
}

// TestRegexRuneIndices proves exposed spans are RUNE indices (consistent with
// std/string), NOT byte offsets, on a multibyte-UTF8 subject.
func TestRegexRuneIndices(t *testing.T) {
	// "héllo" — é is 2 bytes. Match "llo": byte offset 3 (h=1, é=2 bytes) but
	// rune index 2 (h=0, é=1, l=2...).
	got, err := regexFindFirstImpl(nil, []eval.Value{mkStr(`llo`), mkStr("héllo")})
	if err != nil {
		t.Fatal(err)
	}
	rec := got.(*eval.TaggedValue).Fields[0].(*eval.RecordValue)
	if st := rec.Fields["start"].(*eval.IntValue).Value; st != 2 {
		t.Fatalf("start should be rune index 2, got %d (byte offset leaked?)", st)
	}
	if en := rec.Fields["end"].(*eval.IntValue).Value; en != 5 {
		t.Fatalf("end should be rune index 5, got %d", en)
	}
	if txt := rec.Fields["text"].(*eval.StringValue).Value; txt != "llo" {
		t.Fatalf("text = %q, want %q", txt, "llo")
	}
}

// TestRegexCacheMemoization: compiling the same pattern twice is consistent
// (pure memoization → same result, deterministic).
func TestRegexCacheMemoization(t *testing.T) {
	pat := `^\d{3}-\d{4}$`
	re1, err1 := getCompiled(pat)
	re2, err2 := getCompiled(pat)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected compile error: %v / %v", err1, err2)
	}
	if re1 != re2 {
		t.Fatalf("expected memoized *regexp.Regexp to be identical across calls")
	}
}
