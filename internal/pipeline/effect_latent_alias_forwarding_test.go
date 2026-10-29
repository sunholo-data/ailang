package pipeline

import (
	"fmt"
	"strings"
	"testing"
)

// Aliases must use the same data-arrow traversal for binder ownership as for
// caller charging, including callers forwarding rather than invoking the data.
func TestLatentAliasForwarding(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, imported := range []bool{false, true} {
		for _, count := range []int{1, 2} {
			for _, body := range []string{"0", "p.authority(1)"} {
				t.Run(fmt.Sprintf("imported_%v/fields_%d/body_%s", imported, count, body), func(t *testing.T) {
					fields := "authority: int -> int"
					if count == 2 {
						fields += ", appraisal: int -> int"
					}
					decl := "type Policies = {" + fields + "}\n"
					files := map[string]string{}
					prefix := decl
					if imported {
						files["policies.ail"] = "module policies\nexport " + decl
						prefix = "import policies (Policies)\n"
					}
					files["main.ail"] = "module main\n" + prefix + "export pure func accept(p: Policies) -> int = acceptReady(p)\npure func acceptReady(p: Policies) -> int = " + body
					if err := checkModules(t, files); err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestLatentAliasCallerCharge(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, imported := range []bool{false, true} {
		for _, effect := range []string{"IO", "FS"} {
			for _, field := range []string{"authority", "appraisal"} {
				for _, row := range []string{"", " ! {IO}", " ! {FS}"} {
					t.Run(fmt.Sprintf("imported_%v/%s/%s/row_%s", imported, effect, field, row), func(t *testing.T) {
						decl := "type Policies = {authority: int -> int, appraisal: int -> int}\n"
						files := map[string]string{}
						prefix := decl
						if imported {
							files["policies.ail"] = "module policies\nexport " + decl
							prefix = "import policies (Policies)\n"
						}
						files["main.ail"] = "module main\nimport std/io (println)\nimport std/fs (readFile)\n" + prefix + `
pure func accept(p: Policies) -> int = acceptReady(p)
pure func acceptReady(p: Policies) -> int = p.authority(1)
pure func quiet(x: int) -> int = x
func loud(x: int) -> int ! {` + effect + `} { let _ = ` + map[string]string{"IO": "println(\"EFFECT PERFORMED\")", "FS": "readFile(\"absent\")"}[effect] + `; x }
func caller() -> int` + row + ` = accept(` + map[string]string{"authority": "{authority: loud, appraisal: quiet}", "appraisal": "{authority: quiet, appraisal: loud}"}[field] + `)`
						err := checkModules(t, files)
						if row == " ! {"+effect+"}" {
							if err != nil {
								t.Fatal(err)
							}
							return
						}
						if err == nil {
							t.Fatal("accepted uncharged callback")
						}
						for _, want := range []string{"Effect checking failed for function 'caller'", "Missing effects: " + effect} {
							if !strings.Contains(err.Error(), want) {
								t.Fatalf("want %q: %v", want, err)
							}
						}
					})
				}
			}
		}
	}
}

func TestLatentGenericAliasForwarding(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, imported := range []bool{false, true} {
		t.Run(fmt.Sprintf("imported_%v", imported), func(t *testing.T) {
			decl := "type Policies[a] = {authority: a -> int, appraisal: a -> int}\n"
			files := map[string]string{}
			prefix := decl
			if imported {
				files["policies.ail"] = "module policies\nexport " + decl
				prefix = "import policies (Policies)\n"
			}
			files["main.ail"] = "module main\n" + prefix + `export pure func accept(p: Policies[int]) -> int = acceptReady(p)
pure func acceptReady(p: Policies[int]) -> int = p.authority(1)`
			if err := checkModules(t, files); err != nil {
				t.Fatal(err)
			}
		})
	}
}
