package policytool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #1872: read takes an optional 1-based line window and reports total_lines,
// so an agent pages a large file without writing a line-reader program.

func intp(n int) *int { return &n }

func writeSandbox(t *testing.T, f *fixture, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.sandbox, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPolicyTool_ReadRange(t *testing.T) {
	f := newFixture(t, "")
	writeSandbox(t, f, "five.txt", "one\ntwo\nthree\nfour\nfive")
	for _, tc := range []struct {
		name                   string
		offset, limit          *int
		content                string
		total, wantOff, served int
	}{
		{"middle", intp(2), intp(2), "two\nthree\n", 5, 2, 2},
		{"offset only runs to the end", intp(4), nil, "four\nfive", 5, 4, 2},
		{"limit only starts at line 1", nil, intp(1), "one\n", 5, 1, 1},
		{"limit past the end", intp(5), intp(100), "five", 5, 5, 1},
		{"offset past the end is empty, not an error", intp(9), intp(3), "", 5, 9, 0},
	} {
		r := f.host.Dispatch(Request{Op: "read", Path: "five.txt", Offset: tc.offset, Limit: tc.limit})
		if !r.OK || r.Content != tc.content || r.TotalLines == nil || *r.TotalLines != tc.total ||
			r.Offset == nil || *r.Offset != tc.wantOff || r.Limit == nil || *r.Limit != tc.served {
			t.Errorf("%s: %+v (total %v offset %v limit %v)", tc.name, r, deref(r.TotalLines), deref(r.Offset), deref(r.Limit))
		}
	}
	// Consecutive windows reassemble the file byte for byte.
	var b strings.Builder
	for off := 1; off <= 5; off += 2 {
		b.WriteString(f.host.Dispatch(Request{Op: "read", Path: "five.txt", Offset: intp(off), Limit: intp(2)}).Content)
	}
	if b.String() != "one\ntwo\nthree\nfour\nfive" {
		t.Errorf("windows do not reassemble the file: %q", b.String())
	}
	// No window: same response as before — whole content, no range fields.
	if r := f.host.Dispatch(Request{Op: "read", Path: "five.txt"}); !r.OK || r.Content != "one\ntwo\nthree\nfour\nfive" || r.TotalLines != nil || r.Offset != nil || r.Limit != nil {
		t.Errorf("whole read changed: %+v", r)
	}
	writeSandbox(t, f, "empty.txt", "")
	if r := f.host.Dispatch(Request{Op: "read", Path: "empty.txt", Offset: intp(1)}); !r.OK || r.Content != "" || *r.TotalLines != 0 {
		t.Errorf("empty file: %+v", r)
	}
	writeSandbox(t, f, "nl.txt", "a\nb\n")
	if r := f.host.Dispatch(Request{Op: "read", Path: "nl.txt", Offset: intp(1)}); *r.TotalLines != 2 || r.Content != "a\nb\n" {
		t.Errorf("a trailing newline is not an extra line: %+v total %d", r, *r.TotalLines)
	}
}

func TestPolicyTool_ReadRangeBounds(t *testing.T) {
	f := newFixture(t, "")
	writeSandbox(t, f, "x.txt", "x\n")
	for name, req := range map[string]Request{
		"offset 0":       {Op: "read", Path: "x.txt", Offset: intp(0)},
		"negative limit": {Op: "read", Path: "x.txt", Limit: intp(-1)},
		"limit 0":        {Op: "read", Path: "x.txt", Offset: intp(1), Limit: intp(0)},
		"directory":      {Op: "read", Path: "sub", Offset: intp(1)},
	} {
		if r := f.host.Dispatch(req); r.OK || r.Refused == "" {
			t.Errorf("%s must be refused: %+v", name, r)
		}
	}
	writeSandbox(t, f, "long.txt", strings.Repeat("l\n", maxReadLines+10))
	r := f.host.Dispatch(Request{Op: "read", Path: "long.txt", Offset: intp(1), Limit: intp(maxReadLines * 5)})
	if !r.OK || *r.Limit != maxReadLines || *r.TotalLines != maxReadLines+10 {
		t.Errorf("limit must be capped at %d and reported: limit %v total %v", maxReadLines, deref(r.Limit), deref(r.TotalLines))
	}
}

// A ranged read is refused outside the sandbox exactly as a whole read is.
func TestPolicyTool_ReadRangeDenyOutside(t *testing.T) {
	f := newFixture(t, "")
	for _, p := range []string{"../marker.txt", "link.txt", "linkdir/marker.txt", f.marker, f.policyPath, "sub/../../marker.txt"} {
		if r := f.host.Dispatch(Request{Op: "read", Path: p, Offset: intp(1), Limit: intp(1)}); r.OK || strings.Contains(r.Content, "SENTINEL") || strings.Contains(r.Content, "allowed_caps") {
			t.Errorf("ranged read %s: %+v", p, r)
		}
	}
	f.outsideIntact(t)
}

// A file over the transfer cap cannot be read whole but can be paged; the
// window is cut at a whole line under the cap, and one line over it refuses.
func TestPolicyTool_ReadRangeTransferCap(t *testing.T) {
	f := newFixture(t, "max_fs_transfer_bytes = 16\n")
	writeSandbox(t, f, "big.txt", strings.Repeat("abcde\n", 10)) // 60 bytes
	if r := f.host.Dispatch(Request{Op: "read", Path: "big.txt"}); r.OK {
		t.Fatalf("whole read over the cap must be refused: %+v", r)
	}
	r := f.host.Dispatch(Request{Op: "read", Path: "big.txt", Offset: intp(9), Limit: intp(5)})
	if !r.OK || r.Content != "abcde\nabcde\n" || *r.Limit != 2 || *r.TotalLines != 10 {
		t.Fatalf("paging past the cap: %+v", r)
	}
	r = f.host.Dispatch(Request{Op: "read", Path: "big.txt", Offset: intp(1), Limit: intp(10)})
	if !r.OK || r.Content != "abcde\nabcde\n" || *r.Limit != 2 {
		t.Fatalf("window must stop at a whole line under the cap: %+v", r)
	}
	writeSandbox(t, f, "wide.txt", "short\n"+strings.Repeat("w", 40)+"\nend\n")
	if r := f.host.Dispatch(Request{Op: "read", Path: "wide.txt", Offset: intp(2), Limit: intp(1)}); r.OK || !strings.Contains(r.Refused, "line 2") {
		t.Fatalf("a single line over the cap: %+v", r)
	}
	if r := f.host.Dispatch(Request{Op: "read", Path: "wide.txt", Offset: intp(3)}); !r.OK || r.Content != "end\n" || *r.TotalLines != 3 {
		t.Fatalf("a wide line outside the window is only counted: %+v", r)
	}
}

func TestDecodeRequest_ReadRange(t *testing.T) {
	req, err := DecodeRequest([]byte(`{"op":"read","path":"x","offset":3,"limit":20}`))
	if err != nil || req.Offset == nil || *req.Offset != 3 || req.Limit == nil || *req.Limit != 20 {
		t.Fatalf("decoded %+v %v", req, err)
	}
	for name, raw := range map[string]string{
		"string offset":    `{"op":"read","path":"x","offset":"3"}`,
		"fractional limit": `{"op":"read","path":"x","limit":2.5}`,
		"aliased offset":   `{"op":"read","path":"x","Offset":3}`,
		"duplicate limit":  `{"op":"read","path":"x","limit":1,"limit":2}`,
		"object offset":    `{"op":"read","path":"x","offset":{}}`,
	} {
		if _, err := DecodeRequest([]byte(raw)); err == nil {
			t.Errorf("%s: %s decoded without error", name, raw)
		}
	}
}

func deref(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
