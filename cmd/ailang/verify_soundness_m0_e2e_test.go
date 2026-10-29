package main

import (
	"encoding/json"
	"github.com/sunholo-data/ailang/internal/smt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type m0Row struct {
	Function string `json:"function"`
	Status   string `json:"status"`
	Model    []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"model"`
	SMT string `json:"smtlib"`
}

func m0Hermetic(t *testing.T) {
	t.Helper()
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if strings.HasPrefix(key, "OTEL_") || strings.HasPrefix(key, "OTLP_") || strings.HasPrefix(key, "AILANG_TRACE") || strings.HasPrefix(key, "AILANG_OTLP_") || key == "GOOGLE_CLOUD_PROJECT" || key == "OTLP_GOOGLE_CLOUD_PROJECT" {
			t.Setenv(key, "")
		}
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "std"))
	t.Setenv("AILANG_NO_CACHE", "1")
}
func m0Verify(t *testing.T, name string, wants map[string]string) map[string]m0Row {
	t.Helper()
	if !smt.Z3Available() {
		t.Skip("Z3 unavailable (Windows CI)")
	}
	m0Hermetic(t)
	out, errout, exit := runAilangBin(t, buildAilang(t), "verify", "--json", "--verbose", "--timeout", "3s", "cmd/ailang/testdata/verify_preflight/"+name+".ail")
	var payload struct {
		Results []m0Row `json:"results"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("JSON: %v\n%s\n%s", err, out, errout)
	}
	got := map[string]m0Row{}
	for _, row := range payload.Results {
		got[row.Function] = row
	}
	for fn, want := range wants {
		if got[fn].Status != want {
			t.Errorf("%s: got %s want %s\n%s", fn, got[fn].Status, want, out)
		}
	}
	// These suites deliberately contain a refuted contract; secondary ERROR rows
	// must never make their aggregate nonzero exit a substitute for row evidence.
	if exit != 1 {
		t.Fatalf("expected counterexample exit1, got%d\n%s\n%s", exit, out, errout)
	}
	return got
}
func TestVerifyM0_ArithmeticAndIdentity(t *testing.T) {
	rows := m0Verify(t, "m0_int_divmod", map[string]string{"parity": "counterexample", "halfDown": "counterexample", "halfNonNeg": "verified", "parityBounded": "verified", "halfTowardZero": "verified", "remSign": "verified"})
	for _, name := range []string{"parity", "halfDown"} {
		negative := false
		for _, b := range rows[name].Model {
			if b.Name == "$p_x" {
				v := strings.TrimSuffix(strings.TrimPrefix(b.Value, "(- "), ")")
				n, err := strconv.Atoi(v)
				if err != nil {
					t.Fatal(err)
				}
				negative = strings.HasPrefix(b.Value, "(- ") && n > 0
			}
		}
		if !negative {
			t.Fatalf("%s: expected negative witness: %+v", name, rows[name])
		}
	}
	rows = m0Verify(t, "m0_callee_identity", map[string]string{"gap": "counterexample", "selfGap": "verified", "sumBoth": "verified", "atLeast": "verified", "atLeastUnguarded": "counterexample", "total": "counterexample", "lowBound": "counterexample"})
	for _, row := range rows {
		for _, b := range row.Model {
			if strings.HasPrefix(b.Name, "$spec_") {
				t.Fatalf("UF displayed as value: %+v", b)
			}
		}
	}
}
func TestVerifyM0_LexicalHelpersAndSelectedPaths(t *testing.T) {
	m0Verify(t, "m0_lexical_facts", map[string]string{"shadowFalse": "counterexample", "shadowTrue": "verified", "capturedTrue": "verified", "capturedFalse": "counterexample"})
	m0Verify(t, "m0_helper_environment", map[string]string{"diamond": "verified", "diamondFalse": "counterexample", "boundShadow": "verified", "nestedBoundFalse": "counterexample", "nestedBoundTrue": "verified", "nullary": "error", "nullaryCaller": "verified", "nullaryFalse": "counterexample", "total": "counterexample"})
	m0Verify(t, "m0_selected_paths", map[string]string{"andSkipped": "verified", "orFalseGoal": "counterexample", "predicateSkipped": "verified", "predicateFalse": "counterexample", "requiresSkipped": "counterexample", "matchUnmatched": "counterexample", "matchLater": "counterexample", "unreachableOpaque": "verified", "zeroOnly": "counterexample", "zeroSkipped": "verified"})
}
func TestVerifyM0_PartialCalleeNoGlobalAxiom(t *testing.T) {
	m0Verify(t, "m0_partial_vacuity", map[string]string{"wrong": "counterexample", "partial": "counterexample"})
	bin := buildAilang(t)
	file := "cmd/ailang/testdata/verify_preflight/m0_partial_vacuity.ail"
	for _, flags := range [][]string{nil, {"--bytecode", "--strict-bytecode"}} {
		args := append([]string{"run"}, flags...)
		args = append(args, "--entry", "main", file)
		out, errout, code := runAilangBin(t, bin, args...)
		if code != 0 || strings.TrimSpace(out) != "0" {
			t.Fatalf("runtime%d: %s %s", code, out, errout)
		}
	}
	out, errout, code := runAilangBin(t, bin, "run", "--verify-contracts", "--entry", "main", file)
	if code != 1 || !strings.Contains(out+errout, "ensures") {
		t.Fatalf("runtime contract%d: %s %s", code, out, errout)
	}
}
