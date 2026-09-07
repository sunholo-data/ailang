package main

import (
	"bytes"
	"strings"
	"testing"
)

// runHeadroom invokes run() with the log on stdin ("-") and the given budget,
// returning the exit code and captured stdout.
func runHeadroom(t *testing.T, log string, budget string) (int, string) {
	t.Helper()
	var buf bytes.Buffer
	rc := run([]string{"headroom", "-", budget}, strings.NewReader(log), &buf)
	return rc, buf.String()
}

// genOK builds n synthetic `ok\t<pkg>\t<sec>s` lines. sec is the raw seconds
// value (e.g. "100"); genOK appends the "s".
func genOK(n int, sec string) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString("ok\tgithub.com/x/pkg")
		b.WriteString(strings.Repeat("0", 3-len(itoa(i))))
		b.WriteString(itoa(i))
		b.WriteString("\t")
		b.WriteString(sec)
		b.WriteString("s\n")
	}
	return b.String()
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

// TestParseGoTestOutput pins the parser against every record shape the go test
// step can emit: ok, FAIL, the (cached) and [no tests to run] suffixes, and the
// non-record shapes (a `FAIL <pkg> [build failed]` line, a bare `FAIL`) that
// must parse to NO record.
func TestParseGoTestOutput(t *testing.T) {
	log := "" +
		"ok\tgithub.com/sunholo-data/ailang/cmd/ailang\t35.5s\n" +
		"ok\tgithub.com/sunholo-data/ailang/internal/coordinator\t300.5s\n" +
		"FAIL\tgithub.com/sunholo-data/ailang/internal/eval_harness\t148.7s\n" +
		"ok\tgithub.com/sunholo-data/ailang/internal/types\t12.3s (cached)\n" +
		"ok\tgithub.com/sunholo-data/ailang/internal/effects\t0.1s [no tests to run]\n" +
		"FAIL\tgithub.com/sunholo-data/ailang/internal/broken [build failed]\n" +
		"FAIL\n" +
		"panic: test timed out after 6m56s\n"

	recs := parse([]byte(log))
	if len(recs) != 5 {
		t.Fatalf("parse: got %d records, want 5 (the [build failed] and bare FAIL lines must not parse)", len(recs))
	}

	byPkg := map[string]record{}
	for _, r := range recs {
		byPkg[r.pkg] = r
	}

	if r := byPkg["github.com/sunholo-data/ailang/cmd/ailang"]; r.seconds != 35.5 || r.failed {
		t.Errorf("cmd/ailang: got %+v, want seconds=35.5 failed=false", r)
	}
	if r := byPkg["github.com/sunholo-data/ailang/internal/coordinator"]; r.seconds != 300.5 || r.failed {
		t.Errorf("coordinator: got %+v, want seconds=300.5 failed=false", r)
	}
	if r := byPkg["github.com/sunholo-data/ailang/internal/eval_harness"]; r.seconds != 148.7 || !r.failed {
		t.Errorf("eval_harness: got %+v, want seconds=148.7 failed=true", r)
	}
	if r := byPkg["github.com/sunholo-data/ailang/internal/types"]; r.seconds != 12.3 || r.failed {
		t.Errorf("types (cached): got %+v, want seconds=12.3 failed=false", r)
	}
	if r := byPkg["github.com/sunholo-data/ailang/internal/effects"]; r.seconds != 0.1 || r.failed {
		t.Errorf("effects [no tests to run]: got %+v, want seconds=0.1 failed=false", r)
	}
}

// TestPercentOfBudget pins the rounding: 100s/416s = 24%, 380s/416s = 91%,
// 12.3s/416s = 3%.
func TestPercentOfBudget(t *testing.T) {
	cases := []struct {
		seconds, budget float64
		want            int
	}{
		{100, 416, 24},
		{380, 416, 91},
		{12.3, 416, 3},
		{416, 416, 100},
		{0, 416, 0},
	}
	for _, c := range cases {
		if got := percentOfBudget(c.seconds, c.budget); got != c.want {
			t.Errorf("percentOfBudget(%v, %v) = %d, want %d", c.seconds, c.budget, got, c.want)
		}
	}
}

// TestThresholds proves both WARN tiers report AND the job stays green (exit 0):
// a slow package is a data point, not a failure.
func TestThresholds(t *testing.T) {
	// 58 ok at 100s (24%), one at 380s (91% -> 90% tier), one at 333s (80% -> 75% tier).
	log := genOK(58, "100") +
		"ok\tgithub.com/x/pkg58\t380s\n" +
		"ok\tgithub.com/x/pkg59\t333s\n"

	rc, out := runHeadroom(t, log, "416")
	if rc != 0 {
		t.Fatalf("WARN-ONLY violated: exit %d, want 0 (a slow package must not red the job)\n%s", rc, out)
	}
	if !strings.Contains(out, "::warning::headroom: github.com/x/pkg58 at 91% of budget (warn threshold 90%)") {
		t.Errorf("missing 90%%-tier warning; got:\n%s", out)
	}
	if !strings.Contains(out, "::warning::headroom: github.com/x/pkg59 at 80% of budget (warn threshold 75%)") {
		t.Errorf("missing 75%%-tier warning; got:\n%s", out)
	}
}

// TestTopN pins the report to the top-5 slowest packages, sorted by seconds
// descending (ties by package name ascending).
func TestTopN(t *testing.T) {
	// 60 packages; the five slowest are pkg55..pkg59 at 55..59s.
	var b strings.Builder
	for i := 0; i < 60; i++ {
		b.WriteString("ok\tgithub.com/x/pkg")
		b.WriteString(itoa(i))
		b.WriteString("\t")
		b.WriteString(itoa(i))
		b.WriteString("s\n")
	}

	rc, out := runHeadroom(t, b.String(), "416")
	if rc != 0 {
		t.Fatalf("exit %d, want 0\n%s", rc, out)
	}
	if !strings.Contains(out, "HEADROOM: top 5 slowest packages (budget 416s)") {
		t.Fatalf("missing report header; got:\n%s", out)
	}

	lines := strings.Split(out, "\n")
	var rows []string
	for _, l := range lines {
		if strings.Contains(l, " of budget)") {
			rows = append(rows, l)
		}
	}
	if len(rows) != topN {
		t.Fatalf("report has %d rows, want %d; got:\n%s", len(rows), topN, out)
	}
	wantOrder := []string{"pkg59", "pkg58", "pkg57", "pkg56", "pkg55"}
	for i, w := range wantOrder {
		if !strings.Contains(rows[i], w) {
			t.Errorf("row %d = %q, want it to contain %q (top-5 descending)", i, rows[i], w)
		}
	}
}

// TestEmptyParseOnNonEmptyInput proves the anti-vacuity guard: a non-empty log
// that parses to zero records is a broken INSTRUMENT (exit 1), not a slow
// package.
func TestEmptyParseOnNonEmptyInput(t *testing.T) {
	log := "garbage\nnot a go test line\n"
	rc, out := runHeadroom(t, log, "416")
	if rc != 1 {
		t.Fatalf("exit %d, want 1 (anti-vacuity guard)\n%s", rc, out)
	}
	if !strings.Contains(out, "::error:: headroom: parsed 0 packages from non-empty log — parser may be stale") {
		t.Errorf("missing anti-vacuity error; got:\n%s", out)
	}
}

// TestPackageCountFloor proves the floor: a parser finding 3 of 130 packages is
// as broken as one finding 0 (exit 1).
func TestPackageCountFloor(t *testing.T) {
	log := genOK(3, "100")
	rc, out := runHeadroom(t, log, "416")
	if rc != 1 {
		t.Fatalf("exit %d, want 1 (package-count floor)\n%s", rc, out)
	}
	if !strings.Contains(out, "::error:: headroom: parsed 3 packages (< 50) — parser may be stale") {
		t.Errorf("missing floor error; got:\n%s", out)
	}
}
