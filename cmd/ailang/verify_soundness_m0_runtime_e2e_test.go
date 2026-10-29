package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestVerifyM0_SignedRuntimeAndZero(t *testing.T) {
	m0Hermetic(t)
	bin := buildAilang(t)
	file := "cmd/ailang/testdata/verify_preflight/m0_runtime_signs.ail"
	pairs := [][2]int64{{7, 2}, {-7, 2}, {7, -2}, {-7, -2}, {0, -5}, {0, 5}, {-1, 7}, {1, -7}, {-6, 3}, {-6, -3}, {6, -3}, {9223372036854775807, -1}}
	want := make([][]int64, 2)
	for _, p := range pairs {
		want[0] = append(want[0], p[0]/p[1])
		want[1] = append(want[1], p[0]%p[1])
	}
	_, errout, code := runAilangBin(t, bin, "check", file)
	if code != 0 {
		t.Fatalf("check%d: %s", code, errout)
	}
	for _, flags := range [][]string{nil, {"--bytecode", "--strict-bytecode"}} {
		args := append([]string{"run"}, flags...)
		args = append(args, "--entry", "main", file)
		out, errout, code := runAilangBin(t, bin, args...)
		if code != 0 {
			t.Fatalf("runtime%d: %s %s", code, out, errout)
		}
		var got [][]int64
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("signed runtime got%v want%v", got, want)
		}
		for _, entry := range []string{"divZero", "modZero"} {
			args := append([]string{"run"}, flags...)
			args = append(args, "--entry", entry, file)
			out, errout, code := runAilangBin(t, bin, args...)
			if code != 1 || !strings.Contains(out+errout, "RT001") {
				t.Fatalf("zero%d: %s %s", code, out, errout)
			}
		}
	}
}
