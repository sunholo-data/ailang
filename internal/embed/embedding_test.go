package embed

import (
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddingEvaluatorBudget(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "std"))
	fixturePath := filepath.Join(t.TempDir(), "embedding.json")
	if err := os.WriteFile(fixturePath, []byte(`{"version":1,"fixtures":[
 {"kind":"embed","request":{"model":"m","text":"hi"},"error":{"message":"fixture offline"}},
 {"kind":"embed","request":{"model":"m","text":"hi","image":false,"audio":false},"error":{"message":"fixture offline"}}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	fixtures, err := effects.LoadStubFixtures(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, fn := range []string{"text", "textResult", "media", "mediaResult"} {
		for _, mode := range []string{"success", "provider", "no-cap", "budget", "operator"} {
			t.Run(fn+"/"+mode, func(t *testing.T) {
				engine := New(root)
				defer engine.Close()
				base := effects.NewEffContext(nil)
				if mode != "no-cap" {
					base.Grant(effects.NewCapability("AI"))
				}
				engine.runtime.GetEvaluator().SetEffContext(base)
				var ctx *effects.EffContext
				v, err := engine.CallPrepared(func(raw interface{}) {
					ctx = raw.(*effects.EffContext)
					if mode != "no-cap" {
						ctx.Grant(effects.NewCapability("AI"))
					}
					ctx.AI = effects.NewAIContext(effects.NewStubAIHandler())
					if mode == "provider" {
						ctx.AI = effects.NewAIContext(effects.NewFixtureAIHandler(fixtures))
					}
					n := 1
					if mode == "budget" {
						n = 0
					}
					ctx.SetBudget(effects.NewBudgetContext(map[string]*int{"AI": &n}))
					ceiling := 1
					if mode == "operator" {
						ceiling = 0
					}
					ctx.SetOperatorBudget(effects.NewOperatorBudget(map[string]int{"AI": ceiling}))
				}, "internal/embed/testdata/embedding", fn)
				if mode == "provider" {
					if ctx.Budget.Used("AI") != 1 || ctx.OperatorBudgetUsage()["AI"] != 1 {
						t.Fatal("failure charge", ctx.Budget.Used("AI"), ctx.OperatorBudgetUsage())
					}
					if strings.HasSuffix(fn, "Result") {
						if err != nil {
							t.Fatal(err)
						}
						tag := v.(*eval.TaggedValue)
						if tag.CtorName != "Err" || tag.Fields[0].(*eval.StringValue).Value != "fixture offline" {
							t.Fatal(v)
						}
					} else if err == nil || !strings.Contains(err.Error(), "fixture offline") {
						t.Fatal(v, err)
					}
					return
				}
				if mode != "success" {
					if err == nil {
						t.Fatalf("denial became catchable value: %v", v)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if ctx.Budget.Used("AI") != 1 || ctx.OperatorBudgetUsage()["AI"] != 1 {
					t.Fatal("incorrect charge", ctx.Budget.Used("AI"), ctx.OperatorBudgetUsage())
				}
				if tag, ok := v.(*eval.TaggedValue); ok {
					if tag.CtorName != "Ok" {
						t.Fatal(v)
					}
					v = tag.Fields[0]
				}
				if len(v.(*eval.ListValue).Elements) != 8 {
					t.Fatal(v)
				}
			})
		}
	}
}
