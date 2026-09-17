package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/types"
)

// std/web builtins (M-DANEEL-AILANG-EXECUTOR M2): web search and page fetch
// through a fixed backend. Effect {Net} only — the API key is read in Go
// (OLLAMA_API_KEY), never by the program.

func init() {
	registerWebSearch()
	registerWebFetch()
}

func webSearchResultType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("title", T.String()),
		types.Field("url", T.String()),
		types.Field("content", T.String()),
	)
}

func webFetchResultType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("title", T.String()),
		types.Field("content", T.String()),
		types.Field("links", T.List(T.String())),
	)
}

func registerWebSearch() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/web",
		Name:    "_web_search",
		NumArgs: 2,
		IsPure:  false,
		Effect:  "Net",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.String(), T.Int()).
				Returns(T.App("Result", T.List(webSearchResultType(T)), T.Con("NetError"))).
				Effects("Net")
		},
		Impl: effects.WebSearch,
		Metadata: &BuiltinMetadata{
			Description: "Search the web through the configured backend (Ollama web search)",
			LongDesc: `Runs a web search and returns up to max results as {title, url, content}
records. The request goes to a FIXED backend endpoint on ollama.com, so the
policy's net_allow must list that host; the API key comes from OLLAMA_API_KEY
in the runtime, never from the program. max must be 1..10.`,
			Params: []ParamDoc{
				{Name: "query", Description: "Search query (non-empty)"},
				{Name: "max", Description: "Maximum results, 1..10"},
			},
			Returns: "Result[list[{title, url, content}], NetError]",
			Examples: []Example{{
				Code:        `webSearch("AILANG programming language", 3)`,
				Description: "Three results for a query",
			}},
			Since:     "v0.39.2",
			Stability: StabilityStable,
			Tags:      []string{"web", "search", "network"},
			Category:  "network",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _web_search: %v", err))
	}
}

func registerWebFetch() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/web",
		Name:    "_web_fetch",
		NumArgs: 1,
		IsPure:  false,
		Effect:  "Net",
		Type: func() types.Type {
			T := types.NewBuilder()
			return T.Func(T.String()).
				Returns(T.App("Result", webFetchResultType(T), T.Con("NetError"))).
				Effects("Net")
		},
		Impl: effects.WebFetch,
		Metadata: &BuiltinMetadata{
			Description: "Fetch a web page as text through the configured backend",
			LongDesc: `Fetches the page at url THROUGH the backend's fetch API (not directly), and
returns {title, content, links}. The policy's net_allow governs the backend
host, not the page's host; the API key comes from OLLAMA_API_KEY in the runtime.`,
			Params: []ParamDoc{
				{Name: "url", Description: "Page URL (non-empty)"},
			},
			Returns: "Result[{title, content, links}, NetError]",
			Examples: []Example{{
				Code:        `webFetch("https://ailang.sunholo.com/")`,
				Description: "The page's title, text content and links",
			}},
			Since:     "v0.39.2",
			Stability: StabilityStable,
			Tags:      []string{"web", "fetch", "network"},
			Category:  "network",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _web_fetch: %v", err))
	}
}
