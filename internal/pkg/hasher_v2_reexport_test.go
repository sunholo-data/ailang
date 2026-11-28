package pkg

import (
	"slices"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

const reexportOriginBody = "export type Color = Red | Blue(int)\n" +
	"export pure func name(c: Color) -> string\n  requires { true }\n{ match c { Red => \"red\", Blue(n) => \"blue\" } }\n" +
	"export func shout(s: string) -> () ! {IO} = ()\n"

// linesOf is the signature lines of one module, with its prefix removed.
func linesOf(sigs []string, module string) []string {
	var out []string
	for _, s := range sigs {
		if rest, ok := strings.CutPrefix(s, module+":"); ok {
			out = append(out, rest)
		}
	}
	slices.Sort(out)
	return out
}

// A facade's signature lines for what it re-exports are the origin's lines
// under the facade's name: type, constructors, function type, effects and
// contracts (M-PKG-PARALLEL-AGENTS C1). So a module moved behind a facade
// measures as no removal, and dropping a re-export later measures as C.
func TestInterfaceHashV2_ReexportsRenderAsTheOrigin(t *testing.T) {
	facadeAll := "export import test/pkg/color (Color, Red, Blue, name, shout)\n"
	dir, m := writeV2Package(t, []v2Module{
		{"test/pkg/color", reexportOriginBody},
		{"test/pkg/facade", facadeAll},
	}, []string{"IO"})
	_, sigs := mustV2Hash(t, dir, m, DefaultPublishLimits())
	origin, facade := linesOf(sigs, "test/pkg/color"), linesOf(sigs, "test/pkg/facade")
	if len(origin) == 0 || !slices.Equal(origin, facade) {
		t.Fatalf("facade lines differ from the origin's:\norigin=%#v\nfacade=%#v", origin, facade)
	}
	if !slices.ContainsFunc(facade, func(s string) bool { return strings.HasPrefix(s, "contract:name:") }) {
		t.Fatalf("the re-exported function's contract is missing: %#v", facade)
	}

	// The facade alone, before and after it stops re-exporting `name`.
	dirOld, mOld := writeV2Package(t, []v2Module{{"test/pkg/color", reexportOriginBody}, {"test/pkg/facade", facadeAll}}, []string{"IO"})
	dirNew, mNew := writeV2Package(t, []v2Module{{"test/pkg/color", reexportOriginBody},
		{"test/pkg/facade", "export import test/pkg/color (Color, Red, Blue, shout)\n"}}, []string{"IO"})
	_, oldSigs := mustV2Hash(t, dirOld, mOld, DefaultPublishLimits())
	_, newSigs := mustV2Hash(t, dirNew, mNew, DefaultPublishLimits())
	got := messaging.ClassifyVersionChange(
		messaging.PackageVersionInfo{Version: "1.0.0", Signatures: oldSigs},
		messaging.PackageVersionInfo{Version: "1.0.1", Signatures: newSigs})
	if got != "C" {
		t.Fatalf("dropping a re-export classified %q, want C", got)
	}
}
