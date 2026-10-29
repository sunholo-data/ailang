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

func TestExplicitEmptyCallbackContainers(t *testing.T) {
	for _, tt := range []struct{ name, decl, call string }{
		{"tuple", `func apply(p: ((int) -> int ! {}, int)) -> int = match p { (f,x) => f(x) }`, `apply((CALLBACK,1))`},
		{"list", `func apply(fs: [(int) -> int ! {}]) -> int = match fs { [] => 0, [f,..._] => f(1) }`, `apply([CALLBACK])`},
		{"option", `func apply(o: Option[(int) -> int ! {}]) -> int = match o { None => 0, Some(f) => f(1) }`, `apply(Some(CALLBACK))`},
		{"record_alias", `type H = {run: (int) -> int ! {}}
func apply(h: H) -> int = h.run(1)`, `apply({run: CALLBACK})`},
		{"multiarg_record", `type H = {run: (int,int) -> int ! {}}
func apply(h: H) -> int = h.run(1,2)`, `apply({run: CALLBACK})`},
	} {
		for _, io := range []bool{false, true} {
			name := tt.name + "/pure"
			cb := "inc"
			if io {
				name = tt.name + "/IO"
				cb = "noisy"
			}
			if tt.name == "multiarg_record" {
				cb += "2"
			}
			t.Run(name, func(t *testing.T) {
				src := `module empty_container
import std/io (println)
import std/option (Option,Some,None)
pure func inc(x:int)->int = x+1
pure func inc2(x:int,y:int)->int = x+y
func noisy(x:int)->int ! {IO} = { println("marker"); x }
func noisy2(x:int,y:int)->int ! {IO} = { println("marker"); x+y }
` + tt.decl + "\nfunc caller()->int ! {IO} = " + strings.ReplaceAll(tt.call, "CALLBACK", cb) + "\n"
				err := checkModuleSource(t, "empty_container", src)
				if io {
					if err == nil || !strings.Contains(err.Error(), "IO") {
						t.Fatalf("closed empty container accepted IO: %v", err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
