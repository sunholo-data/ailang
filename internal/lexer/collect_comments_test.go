package lexer

import "testing"

// collect_comments_test.go verifies CollectComments: byte-exact comment spans,
// literal-region mapping, and the interpolation-hole classification that the
// formatter's fail-closed carve-out depends on.

func TestCollectComments_Spans(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		want    []Comment
		regions int // number of literal regions expected (nil-safe count)
	}{
		{
			name: "single dash comment",
			src:  "let x = 1 -- note\n",
			want: []Comment{{Kind: LineCommentDash, Text: "-- note", Start: 10, End: 17}},
		},
		{
			name: "slash comment at EOF no newline",
			src:  "// header",
			want: []Comment{{Kind: LineCommentSlash, Text: "// header", Start: 0, End: 9}},
		},
		{
			name: "dash inside string is not a comment",
			src:  `let s = "a -- b"` + "\n",
			want: nil,
		},
		{
			name:    "string region recorded",
			src:     `"abc"`,
			want:    nil,
			regions: 1,
		},
		{
			name: "two comments preserve order",
			src:  "-- one\n-- two\n",
			want: []Comment{
				{Kind: LineCommentDash, Text: "-- one", Start: 0, End: 6},
				{Kind: LineCommentDash, Text: "-- two", Start: 7, End: 13},
			},
		},
		{
			name: "unicode before comment (rune vs byte)",
			// "héπ" is 2+2+... multibyte; the comment offsets are BYTE offsets.
			src:  `x = "héπ" -- c` + "\n",
			want: []Comment{{Kind: LineCommentDash, Text: "-- c", Start: 12, End: 16}},
		},
		{
			name: "triple-quote body dash is literal",
			src:  `sql"""a -- b"""` + "\n",
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			comments, regions := CollectComments([]byte(tc.src))
			if len(comments) != len(tc.want) {
				t.Fatalf("got %d comments, want %d: %+v", len(comments), len(tc.want), comments)
			}
			for i, w := range tc.want {
				g := comments[i]
				if g.Kind != w.Kind || g.Text != w.Text || g.Start != w.Start || g.End != w.End {
					t.Errorf("comment[%d] = %+v, want %+v", i, g, w)
				}
				// Text must equal the exact normalized-source slice [Start,End).
				norm := string(Normalize([]byte(tc.src)))
				if got := norm[g.Start:g.End]; got != g.Text {
					t.Errorf("comment[%d] Text %q != source slice %q", i, g.Text, got)
				}
			}
			if tc.regions != 0 && len(regions) != tc.regions {
				t.Errorf("got %d regions, want %d: %+v", len(regions), tc.regions, regions)
			}
		})
	}
}

// TestCollectComments_InterpolationClassification proves that a comment inside a
// `${...}` interpolation hole is reported as a REAL comment whose span lies
// OUTSIDE every literal region — the fail-closed carve-out signal — while a
// comment introducer in ordinary literal bytes is not reported at all.
func TestCollectComments_InterpolationClassification(t *testing.T) {
	// A comment inside the interpolation hole is code-level → reported.
	src := `let s = "pre${ x -- interior` + "\n" + ` }post"` + "\n"
	comments, regions := CollectComments([]byte(src))
	if len(comments) != 1 {
		t.Fatalf("want exactly 1 interior comment, got %d: %+v", len(comments), comments)
	}
	c := comments[0]
	if c.Kind != LineCommentDash {
		t.Errorf("kind = %v, want dash", c.Kind)
	}
	// The comment must NOT fall inside any recorded literal region.
	for _, r := range regions {
		if c.Start >= r.Start && c.Start < r.End {
			t.Errorf("interior comment at %d falls inside literal region %+v — carve-out would MISS it", c.Start, r)
		}
	}
}

// TestCollectComments_NestedInterpolation covers the design V19 case: nested
// `${ f("${base}/x") }` must not terminate the region at the first inner `"`.
func TestCollectComments_NestedInterpolation(t *testing.T) {
	src := `"${f("${base}/x")}" -- after` + "\n"
	comments, _ := CollectComments([]byte(src))
	if len(comments) != 1 {
		t.Fatalf("want exactly the trailing comment, got %d: %+v", len(comments), comments)
	}
	if comments[0].Text != "-- after" {
		t.Errorf("text = %q, want %q", comments[0].Text, "-- after")
	}
}

// TestCollectComments_NoInteriorComment verifies the common case: an
// interpolation with no comment yields zero comments (no false positive).
func TestCollectComments_NoInteriorComment(t *testing.T) {
	src := `let s = "hello ${name}!"` + "\n"
	comments, _ := CollectComments([]byte(src))
	if len(comments) != 0 {
		t.Errorf("want 0 comments, got %+v", comments)
	}
}
