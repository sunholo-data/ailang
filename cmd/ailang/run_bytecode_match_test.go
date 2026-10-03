package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestCLI_RunBytecode_MatchLoweringParity is the m-vm-match-lowering parity
// gate (#1420, #1505, #1517, #1473, #1503). Each row names one zero-argument
// entry in tests/golden/bytecode/nested_patterns.ail; the entry exercises a
// pattern shape on both a matching and a near-miss input. The row must print
// the hand-computed value on the evaluator AND under --strict-bytecode, where
// an evaluator-only function or a VM crash fails the run instead of being
// bridged back to the evaluator.
//
// The families mirror the superseded design docs' symptom tables:
//
//	A  nested cons / nested sub-patterns   (were evaluator-only: unbound variable)
//	B  literals inside list/tuple/record    (were silent wrong results)
//	C  constructors in if-chain positions   (were VM crashes: _record_get on ADT)
//	G  guards over the arm's own bindings   (were evaluator-only: unbound variable)
//	V  variable-pattern arms                (unbound default, arm-order violation)
//	N  constructor matches, refutable args  (switch path vs if-chain routing)
//	X  guard ANF lets, non-variable scrutinee, match inside a lambda
func TestCLI_RunBytecode_MatchLoweringParity(t *testing.T) {
	src := filepath.Join("tests", "golden", "bytecode", "nested_patterns.ail")
	rows := []struct{ entry, want string }{
		// A
		{"p_cons2", "2|0|0"},
		{"p_cons2_wild", "5|0"},
		{"p_cons3", "7|0|6"},
		{"p_cons_closed_tail", "pair|other|other"},
		{"p_cons_in_list", "1|0|0"},
		{"p_ctor_nested", "7|1|0"},
		{"p_cons_in_tuple", "0|14"},
		// B
		{"p_lit_list", "0.0|1.0"},
		{"p_lit_tuple", "0|5"},
		{"p_lit_record", "other|alice"},
		{"p_lit_prefix", "no|ok1"},
		{"p_lit_cons_head", "A1|x|e"},
		{"p_record_nested", "baby a@x|b@x"},
		// C
		{"p_ctor_cons_head", "hi|mx|empty"},
		{"p_opt_cons_head", "ok5:2|none|empty"},
		{"p_opt_list_elem", "one4|other|other"},
		{"p_ctor_in_tuple", "7|3"},
		{"p_lit_ctor_cons_head", "greeting|text bye|other"},
		{"p_list_in_ctor", "3|0|-1"},
		{"p_deep_mix", "6|1|0"},
		// G
		{"p_guard_list", "desc|other|other"},
		{"p_guard_cons", "big|small|empty"},
		{"p_guard_var", "A|B|Z"},
		{"p_guard_record", "big|small"},
		{"p_guard_tuple", "lt|ge"},
		{"p_guard_nested_cons", "desc|asc|short"},
		{"p_guard_ctor_dup", "big|small2|none"},
		{"p_guard_ctor_default", "big|other|other"},
		{"p_guard_chain", "one|two|rest|same|huge|rest"},
		{"p_guard_value_pos", "50|-10|0"},
		// V
		{"p_var_default", "p|committed|idle"},
		{"p_var_before_ctor", "o|c|o"},
		{"p_var_guarded", "a|c|z"},
		{"p_all_var", "42"},
		{"p_str_var", "one|hello!"},
		// N
		{"p_ctor_lit", "true|false|false"},
		{"p_ctor_lit_dup", "zero|n3|none"},
		{"p_ctor_deep", "dbl4|neg2|zl9|add|other|other"},
		// X
		{"p_guard_let_switch", "yes|no|no"},
		{"p_guard_param", "empty+|empty|over|under"},
		{"p_scrut_call", "100|9|2"},
		{"p_lambda_match", "30|20|0"},
	}

	for _, row := range rows {
		row := row
		t.Run(row.entry, func(t *testing.T) {
			t.Parallel()
			base := []string{"--quiet", "--relax-modules", "--entry", row.entry}

			evalOut, evalErr, evalExit := runCLI(t, append(append([]string{"run"}, base...), src)...)
			if evalExit != 0 || strings.TrimSpace(evalOut) != row.want {
				t.Fatalf("evaluator: exit %d, got %q, want %q\nstderr=%s", evalExit, strings.TrimSpace(evalOut), row.want, evalErr)
			}

			vmArgs := append(append([]string{"run", "--bytecode", "--strict-bytecode"}, base...), src)
			vmOut, vmErr, vmExit := runCLI(t, vmArgs...)
			if vmExit != 0 || strings.TrimSpace(vmOut) != row.want {
				t.Fatalf("strict VM: exit %d, got %q, want %q (evaluator agrees with want)\nstderr=%s", vmExit, strings.TrimSpace(vmOut), row.want, vmErr)
			}
		})
	}
}
