package parser

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

// M-AGENT-STUCK-FIXES M1: a malformed `pure func ...` must yield a clean parse error,
// never a PAR999 parser panic. parsePureLambda used to assert parseLambda().(*ast.Lambda)
// with no nil guard; any malformed pure-lambda made parseLambda return nil and the
// assertion panicked. A benchmark agent looped 87 steps on the resulting useless
// "interface conversion: ast.Expr is nil" message because it had nothing to act on.
func TestPureLambdaMalformed_NoPanic(t *testing.T) {
	cases := []string{
		"module t\nexport func f() -> int = pure func",
		"module t\nexport func f() -> int = pure func(",
		"module t\nexport func f() -> int = pure func()",
		"module t\nexport func f() -> int = pure func(x:int)",
		"module t\nexport func f() -> int = pure func(x:int) ->",
	}
	for _, src := range cases {
		t.Run(strings.TrimPrefix(strings.SplitN(src, "= ", 2)[1], ""), func(t *testing.T) {
			p := New(lexer.New(src, "test.ail"))
			// Must not panic. parsePureLambda is a registered prefix parser, so a nil
			// return is the normal error signal and the Pratt loop tolerates it.
			_ = p.Parse()
			errs := p.Errors()
			if len(errs) == 0 {
				t.Fatalf("expected a parse error for %q, got none", src)
			}
			for _, e := range errs {
				if strings.Contains(e.Error(), "PAR999") || strings.Contains(e.Error(), "parser panic") {
					t.Fatalf("got a PAR999 panic instead of a clean error for %q: %v", src, e)
				}
			}
		})
	}
}
