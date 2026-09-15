package config

// The message-store embedder (internal/messaging). Precedence for each: the
// variable, else the embeddings section of ~/.ailang/config.yaml, else a
// deprecated default served through DeprecatedDefault — the model FIXES the
// vector dimension, so a silent default corrupts search.
const (
	EnvEmbedProvider    = "AILANG_EMBED_PROVIDER"
	EnvEmbedOllamaModel = "AILANG_OLLAMA_MODEL"
	EnvEmbedOllamaURL   = "AILANG_OLLAMA_ENDPOINT"
	EnvEmbedOpenAIModel = "AILANG_EMBED_OPENAI_MODEL"
	EnvEmbedGeminiModel = "AILANG_EMBED_GEMINI_MODEL"
)

var embedVars = []Var{
	{EnvEmbedProvider, "", AreaEmbed, "Embedding provider (ollama, openai, gemini, none) over the config file's embeddings.provider."},
	{EnvEmbedOllamaModel, "", AreaEmbed, "Ollama embedding model over the config file's embeddings.ollama.model."},
	{EnvEmbedOllamaURL, "", AreaEmbed, "Ollama endpoint the embedder uses over the config file's embeddings.ollama.endpoint; set, it is exported as OLLAMA_HOST for the ollama client library."},
	{EnvEmbedOpenAIModel, "", AreaEmbed, "OpenAI embedding model over the config file; unset serves text-embedding-3-small as a deprecated default (1536 dimensions)."},
	{EnvEmbedGeminiModel, "", AreaEmbed, "Gemini embedding model over the config file; unset serves text-embedding-004 as a deprecated default (768 dimensions)."},
}

// EmbedProvider returns AILANG_EMBED_PROVIDER, "" when unset.
func EmbedProvider() string { return get(EnvEmbedProvider) }

// EmbedOllamaModel returns AILANG_OLLAMA_MODEL, "" when unset.
func EmbedOllamaModel() string { return get(EnvEmbedOllamaModel) }

// EmbedOllamaEndpoint returns AILANG_OLLAMA_ENDPOINT, "" when unset.
func EmbedOllamaEndpoint() string { return get(EnvEmbedOllamaURL) }

// EmbedOpenAIModel returns AILANG_EMBED_OPENAI_MODEL, "" when unset.
func EmbedOpenAIModel() string { return get(EnvEmbedOpenAIModel) }

// EmbedGeminiModel returns AILANG_EMBED_GEMINI_MODEL, "" when unset.
func EmbedGeminiModel() string { return get(EnvEmbedGeminiModel) }
