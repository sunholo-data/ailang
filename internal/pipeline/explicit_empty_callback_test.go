package pipeline

import (
	"strings"
	"testing"
)

func TestExplicitEmptyCallbackContract(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		reject     bool
	}{
		{"reject_IO_even_in_IO_caller", `func apply(f: (int) -> int ! {}, x: int) -> int = f(x)
func caller() -> int ! {IO} = apply(noisy, 0)`, true},
		{"accept_pure", `func apply(f: (int) -> int ! {}, x: int) -> int = f(x)
pure func caller() -> int = apply(inc, 0)`, false},
		{"omitted_still_charges", `func apply(f: (int) -> int, x: int) -> int = f(x)
pure func caller() -> int = apply(noisy, 0)`, true},
		{"omitted_declared_caller", `func apply(f: (int) -> int, x: int) -> int = f(x)
func caller() -> int ! {IO} = apply(noisy, 0)`, false},
		{"explicit_rowpoly_unchanged", `func apply[e](f: (int) -> int ! {e}, x: int) -> int ! {e} = f(x)
func caller() -> int ! {IO} = apply(noisy, 0)`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			src := "module empty_contract\nimport std/io (println)\nfunc noisy(x: int) -> int ! {IO} = { println(\"must not run\"); x }\npure func inc(x: int) -> int = x+1\n" + tt.body + "\n"
			err := checkModuleSource(t, "empty_contract", src)
			if tt.reject {
				if err == nil || !strings.Contains(err.Error(), "IO") {
					t.Fatalf("want IO rejection: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
