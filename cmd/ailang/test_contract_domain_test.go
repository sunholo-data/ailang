package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// This file is the M3 CLI-level e2e for the design doc's AC1/AC2/AC3 contract
// soundness criteria (deferred from M1 by plan §3.4). It drives the built
// binary over the four checked-in fixtures in internal/testing/testdata/contracts/
// and asserts on the JSON verdict fields the design doc pins: status,
// generated_inputs, discarded_inputs, and the `unverified:` skip text — NOT
// wall time. Every run uses an explicit --seed 42 so the assertions are
// reproducible. runAilangBin runs with cwd = repo root, and each fixture
// declares its own module, so the relative paths here are stable.

// runContractFixture runs one checked-in fixture with --seed 42 and parses its
// JSON output. It returns the raw doc and the parsed property list.
func runContractFixture(t *testing.T, bin, relPath string, wantExit int) []map[string]interface{} {
	t.Helper()
	stdout, stderr, exit := runAilangBin(t, bin, "test", "--seed", "42", "--format", "json", "--no-color", relPath)
	if exit != wantExit {
		t.Fatalf("%s: exit = %d, want %d\nstderr:\n%s", relPath, exit, wantExit, stderr)
	}
	var doc struct {
		Properties []json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("%s: output is not JSON: %v\n%s", relPath, err, stdout)
	}
	props := make([]map[string]interface{}, 0, len(doc.Properties))
	for _, raw := range doc.Properties {
		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("%s: bad property object: %v", relPath, err)
		}
		props = append(props, m)
	}
	return props
}

func propStatus(p map[string]interface{}) string { return p["status"].(string) }
func propName(p map[string]interface{}) string   { return p["name"].(string) }
func propErr(p map[string]interface{}) string {
	if e, ok := p["error"].(string); ok {
		return e
	}
	return ""
}
func propNum(p map[string]interface{}, key string) int {
	f, _ := p[key].(float64)
	return int(f)
}

func propByName(t *testing.T, props []map[string]interface{}, substr string) map[string]interface{} {
	t.Helper()
	for _, p := range props {
		if strings.Contains(propName(p), substr) {
			return p
		}
	}
	t.Fatalf("no property whose name contains %q in %d properties", substr, len(props))
	return nil
}

// propByNameStatus returns the FIRST property whose name contains substr AND
// whose status equals want. Several contracts emit one requires property and
// one ensures property for the same function (e.g. big_property_1 vs
// big_property_2); the design-doc ACs pin the verdict on the ensures one, so
// selecting by name alone is ambiguous and we disambiguate by status.
func propByNameStatus(t *testing.T, props []map[string]interface{}, substr, want string) map[string]interface{} {
	t.Helper()
	for _, p := range props {
		if strings.Contains(propName(p), substr) && propStatus(p) == want {
			return p
		}
	}
	t.Fatalf("no %s property whose name contains %q in %d properties", want, substr, len(props))
	return nil
}

var counterexampleRe = regexp.MustCompile(`x=[1-9][0-9]{2,}`)

// TestContractDomain_AC1_ExcludedInputsNoLongerFail — precond2.ail: the big()
// ensures contract passes (100 in-contract inputs with discards); the requires
// property is an honest out_of_contract skip, never a fail. Suite exits 0.
func TestContractDomain_AC1_ExcludedInputsNoLongerFail(t *testing.T) {
	bin := buildAilang(t)
	props := runContractFixture(t, bin, "internal/testing/testdata/contracts/precond2.ail", 0)
	big := propByNameStatus(t, props, "big", "pass")
	if got := propStatus(big); got != "pass" {
		t.Errorf("big contract status = %q, want pass", got)
	}
	if got := propNum(big, "tests_run"); got != 100 {
		t.Errorf("big contract tests_run = %d, want 100", got)
	}
	if got := propNum(big, "discarded_inputs"); got <= 0 {
		t.Errorf("big contract discarded_inputs = %d, want > 0 (requires filter active)", got)
	}
}

// TestContractDomain_AC2_GenuineViolationStillFails — precond_negative.ail: an
// in-domain violation is still a fail with an x > 100 counterexample (the
// negative control that proves filtering did not create a vacuous pass).
func TestContractDomain_AC2_GenuineViolationStillFails(t *testing.T) {
	bin := buildAilang(t)
	props := runContractFixture(t, bin, "internal/testing/testdata/contracts/precond_negative.ail", 1)
	broken := propByNameStatus(t, props, "broken", "fail")
	if got := propStatus(broken); got != "fail" {
		t.Errorf("broken contract status = %q, want fail", got)
	}
	if errText := propErr(broken); !strings.Contains(errText, "ensures violated") {
		t.Errorf("broken contract error missing %q: %q", "ensures violated", errText)
	} else if !counterexampleRe.MatchString(errText) {
		t.Errorf("broken contract error has no x>100 counterexample: %q", errText)
	}
}

// TestContractDomain_AC3_UnreachableDomainIsUnverified — requires_unreachable.ail:
// exhausting 1000 attempts before 100 acceptances is a bounded out_of_contract
// skip carrying the `unverified:` text with exact counts — never pass or fail.
func TestContractDomain_AC3_UnreachableDomainIsUnverified(t *testing.T) {
	bin := buildAilang(t)
	props := runContractFixture(t, bin, "internal/testing/testdata/contracts/requires_unreachable.ail", 0)
	// Both impossible_property_{1,2} are out_of_contract skips; select the
	// cap-exhausted ensures one (generated_inputs == 1000), not the requires
	// one that exits after 1 generated input.
	var impossible map[string]interface{}
	for _, p := range props {
		if strings.Contains(propName(p), "impossible") && propNum(p, "generated_inputs") == 1000 {
			impossible = p
		}
	}
	if impossible == nil {
		t.Fatalf("no cap-exhausted impossible property (generated_inputs=1000) in %d properties", len(props))
	}
	if got := propStatus(impossible); got != "skip" {
		t.Errorf("impossible contract status = %q, want skip", got)
	}
	if got := propNum(impossible, "tests_run"); got != 0 {
		t.Errorf("impossible contract tests_run = %d, want 0", got)
	}
	if got := propNum(impossible, "generated_inputs"); got != 1000 {
		t.Errorf("impossible contract generated_inputs = %d, want 1000", got)
	}
	if got := propNum(impossible, "discarded_inputs"); got != 1000 {
		t.Errorf("impossible contract discarded_inputs = %d, want 1000", got)
	}
	if errText := propErr(impossible); !strings.HasPrefix(errText, "unverified:") {
		t.Errorf("impossible contract error does not start with %q: %q", "unverified:", errText)
	}
}

// TestContractDomain_OrderedRequiresBothInDomainPass — ordered.ail: the
// comma-separated requires contract (g) accepts enough in-domain inputs to pass
// 100 cases with discards, and the no-requires contract (h) passes with zero
// discards. Guards source-order enumeration of the requires conjunction.
func TestContractDomain_OrderedRequiresBothInDomainPass(t *testing.T) {
	bin := buildAilang(t)
	props := runContractFixture(t, bin, "internal/testing/testdata/contracts/ordered.ail", 0)
	g := propByName(t, props, "g_property_3")
	if got := propStatus(g); got != "pass" {
		t.Errorf("g contract status = %q, want pass", got)
	}
	if got := propNum(g, "tests_run"); got != 100 {
		t.Errorf("g contract tests_run = %d, want 100", got)
	}
	if got := propNum(g, "discarded_inputs"); got <= 0 {
		t.Errorf("g contract discarded_inputs = %d, want > 0", got)
	}
	h := propByName(t, props, "h_property_1")
	if got := propStatus(h); got != "pass" {
		t.Errorf("h contract status = %q, want pass", got)
	}
	if got := propNum(h, "discarded_inputs"); got != 0 {
		t.Errorf("h contract discarded_inputs = %d, want 0 (no requires)", got)
	}
}
