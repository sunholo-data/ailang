package openrouter

import (
	"net/http"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

func TestFrozenAttribution_PrecedenceAndLaterEnvironment(t *testing.T) {
	t.Setenv("OPENROUTER_HTTP_REFERER", "https://selected.example")
	t.Setenv("OPENROUTER_X_TITLE", "selected title")
	t.Setenv("OPENROUTER_CATEGORIES", "selected-category")
	original := &ai.Attribution{Title: "explicit title"}
	frozen := FrozenAttribution(original)
	original.Title = "mutated"
	t.Setenv("OPENROUTER_HTTP_REFERER", "https://late.example")
	t.Setenv("OPENROUTER_X_TITLE", "late title")
	t.Setenv("OPENROUTER_CATEGORIES", "late-category")
	req, _ := http.NewRequest(http.MethodPost, "https://example.com", nil)
	setAttributionHeaders(req, frozen)
	for name, want := range map[string]string{"HTTP-Referer": "https://selected.example", "X-OpenRouter-Title": "explicit title", "X-OpenRouter-Categories": "selected-category"} {
		if got := req.Header.Get(name); got != want {
			t.Errorf("%s=%q want=%q", name, got, want)
		}
	}
}
