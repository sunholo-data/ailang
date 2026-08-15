package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestReadFindingsClassifiesModuleOnlyAndReaching(t *testing.T) {
	input := strings.NewReader(`
{"config":{"protocol_version":"v1.0.0"}}
{"finding":{"osv":"GO-2026-5750","trace":[{"module":"github.com/ollama/ollama","function":""}]}}
{"progress":{"message":"Scanning your code..."}}
{"finding":{"osv":"GO-2026-6218","trace":[{"module":"example.com/dependency","function":""},{"module":"example.com/ailang","function":"main.run"}]}}
`)

	reaching, moduleOnly, err := readFindings(input)
	if err != nil {
		t.Fatalf("readFindings() error = %v", err)
	}
	assertStringsEqual(t, "moduleOnly", moduleOnly, []string{"GO-2026-5750"})
	assertStringsEqual(t, "reaching", reaching, []string{"GO-2026-6218"})

	// Mutation killed: reverting readFindings to skip non-function traces
	// (the pre-#703 `Function == ""` continue) empties moduleOnly and reds this test.
}

func TestPrintModuleOnlyAnnotations(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		entries map[string]allowEntry
		want    string
		notWant string
	}{
		{"expired", map[string]allowEntry{"GO-PAST": {ID: "GO-PAST", Expires: "2020-01-01"}}, "[allowlisted, EXPIRED 2020-01-01]", ""},
		{"future", map[string]allowEntry{"GO-FUTURE": {ID: "GO-FUTURE", Expires: "2030-01-01"}}, "[allowlisted]", "EXPIRED"},
		{"not allowlisted", map[string]allowEntry{}, "[NOT allowlisted]", "EXPIRED"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			id := "GO-UNLISTED"
			for key := range tc.entries {
				id = key
			}
			printModuleOnly(&out, []string{id}, tc.entries, now)
			if !strings.Contains(out.String(), tc.want) {
				t.Errorf("output %q does not contain %q", out.String(), tc.want)
			}
			if tc.notWant != "" && strings.Contains(out.String(), tc.notWant) {
				t.Errorf("output %q unexpectedly contains %q", out.String(), tc.notWant)
			}
		})
	}
}

func TestReachingExpiryBehaviorUnchanged(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	_, expired, err := classifyReaching([]string{"GO-PAST"}, map[string]allowEntry{
		"GO-PAST": {ID: "GO-PAST", Expires: "2020-01-01"},
	}, now)
	if err != nil {
		t.Fatalf("classifyReaching() error = %v", err)
	}
	assertStringsEqual(t, "expired", expired, []string{"GO-PAST (expired 2020-01-01)"})
}

// TestDecideExitCodes pins the exit-code contract end to end, through the same
// decision path the binary runs.
//
// The #703 invariant — module-level findings are reported but never gate — can
// only be pinned here. An earlier version of this test called
// classifyReaching(nil, ...) and asserted the result was empty; that passes for
// ANY implementation, because the loop body never runs on a nil slice. Measured:
// mutating classifyReaching to append every id to `blocking` left that test
// green. These arms red on that mutant.
func TestDecideExitCodes(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	const modOnly = `{"finding":{"osv":"GO-MODULE","trace":[{"module":"example.com/m"}]}}`
	const reachingListed = `{"finding":{"osv":"GO-REACH","trace":[{"module":"example.com/m","function":"Boom"}]}}`
	const reachingUnlisted = `{"finding":{"osv":"GO-UNLISTED","trace":[{"module":"example.com/m","function":"Boom"}]}}`

	expiredModule := allowEntry{ID: "GO-MODULE", Reason: "unreachable", Expires: "2020-01-01"}
	freshReaching := allowEntry{ID: "GO-REACH", Reason: "pending fix", Expires: "2030-01-01"}
	badModule := allowEntry{ID: "GO-MODULE", Reason: "typo", Expires: "not-a-date"}

	tests := []struct {
		name     string
		findings string
		allow    []allowEntry
		wantCode int
		wantOut  string // substring required in stdout+stderr
	}{
		{
			// The load-bearing arm: an EXPIRED module-only entry is surfaced
			// and the process still exits 0.
			name:     "expired module-only does not gate",
			findings: modOnly,
			allow:    []allowEntry{expiredModule},
			wantCode: 0,
			wantOut:  "GO-MODULE [allowlisted, EXPIRED 2020-01-01]",
		},
		{
			// Same, alongside a healthy reaching finding — the module-only
			// class must not drag an otherwise-passing run to non-zero.
			name:     "expired module-only alongside allowlisted reaching",
			findings: modOnly + "\n" + reachingListed,
			allow:    []allowEntry{expiredModule, freshReaching},
			wantCode: 0,
			wantOut:  "GO-MODULE [allowlisted, EXPIRED 2020-01-01]",
		},
		{
			// A genuinely blocking REACHING finding still exits 1, so arm 1's
			// zero is a decision and not a filter that never gates anything.
			name:     "unallowlisted reaching finding gates",
			findings: reachingUnlisted,
			allow:    nil,
			wantCode: 1,
			wantOut:  "GO-UNLISTED",
		},
		{
			// Malformed expiry on a module-only entry is an input error (2),
			// not a silent pass.
			name:     "malformed module-only expiry exits 2",
			findings: modOnly,
			allow:    []allowEntry{badModule},
			wantCode: 2,
			wantOut:  `bad expires date "not-a-date" for GO-MODULE`,
		},
		{
			// The reaching sibling of the arm above. This branch predates the
			// module-only fix and was unpinned at 38641e216 too — measured:
			// neutering `if parseErr != nil` in classifyReaching redded nothing.
			// It is the direct counterpart of the #717 deliverable, so it is
			// pinned here rather than left as debt.
			name:     "malformed reaching expiry exits 2",
			findings: reachingListed,
			allow:    []allowEntry{{ID: "GO-REACH", Reason: "typo", Expires: "not-a-date"}},
			wantCode: 2,
			wantOut:  `bad expires date "not-a-date" for GO-REACH`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := decide(&stdout, &stderr, &allowlist{Allow: tc.allow},
				strings.NewReader(tc.findings), now)
			if got != tc.wantCode {
				t.Errorf("decide() = %d, want %d\nstdout: %s\nstderr: %s",
					got, tc.wantCode, stdout.String(), stderr.String())
			}
			combined := stdout.String() + stderr.String()
			if !strings.Contains(combined, tc.wantOut) {
				t.Errorf("output %q does not contain %q", combined, tc.wantOut)
			}
		})
	}
}

func TestValidateModuleOnlyRejectsMalformedExpiry(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	err := validateModuleOnly([]string{"GO-BAD"}, map[string]allowEntry{
		"GO-BAD": {ID: "GO-BAD", Expires: "not-a-date"},
	}, now)
	if err == nil || !strings.Contains(err.Error(), `bad expires date "not-a-date" for GO-BAD:`) {
		t.Fatalf("validateModuleOnly() error = %v, want bad expires date for GO-BAD", err)
	}
}

func assertStringsEqual(t *testing.T, name string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}
