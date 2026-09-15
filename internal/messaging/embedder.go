package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/sunholo-data/ailang/internal/config"
)

// Embedder provides text embedding capabilities for semantic search
type Embedder interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(texts []string) ([][]float32, error)
	Dimension() int
	ModelName() string
}

// EmbedConfig configures the embedding provider
type EmbedConfig struct {
	Provider string            `yaml:"provider"` // "ollama", "openai", "gemini", or "none"
	Ollama   OllamaConfig      `yaml:"ollama"`
	OpenAI   OpenAIEmbedConfig `yaml:"openai"`
	Gemini   GeminiEmbedConfig `yaml:"gemini"`
}

// OpenAIEmbedConfig configures the OpenAI embedding provider
type OpenAIEmbedConfig struct {
	APIKey    string        `yaml:"api_key"`   // Defaults to OPENAI_API_KEY env
	Model     string        `yaml:"model"`     // e.g. "text-embedding-3-small"
	Dimension int           `yaml:"dimension"` // Output dimension (0 = model default)
	Timeout   time.Duration `yaml:"timeout"`
}

// GeminiEmbedConfig configures the Gemini embedding provider
type GeminiEmbedConfig struct {
	APIKey    string        `yaml:"api_key"`   // Defaults to GOOGLE_API_KEY env
	Model     string        `yaml:"model"`     // e.g. "text-embedding-004"
	Dimension int           `yaml:"dimension"` // Output dimension (0 = model default)
	Timeout   time.Duration `yaml:"timeout"`
}

// OllamaConfig configures the Ollama embedding provider
type OllamaConfig struct {
	Model     string        `yaml:"model"`
	Endpoint  string        `yaml:"endpoint"`
	Dimension int           `yaml:"dimension"`
	Timeout   time.Duration `yaml:"timeout"`
	BatchSize int           `yaml:"batch_size"`
}

// DefaultEmbedConfig returns the default embedding configuration
func DefaultEmbedConfig() EmbedConfig {
	return EmbedConfig{
		Provider: "ollama",
		Ollama: OllamaConfig{
			Model:     "nomic-embed-text", // Good balance of speed/quality
			Endpoint:  "http://localhost:11434",
			Dimension: 768, // nomic-embed-text dimension
			Timeout:   30 * time.Second,
			BatchSize: 10,
		},
	}
}

// LoadEmbedConfigFromEnv loads embedding config from config file and environment variables
// Priority: env vars > config file > defaults
func LoadEmbedConfigFromEnv() EmbedConfig {
	cfg := DefaultEmbedConfig()

	// Load from config file first
	yamlCfg, err := LoadEmbeddingsConfig()
	if err == nil && yamlCfg != nil {
		if yamlCfg.Provider != "" {
			cfg.Provider = yamlCfg.Provider
		}
		if yamlCfg.Ollama.Model != "" {
			cfg.Ollama.Model = yamlCfg.Ollama.Model
		}
		if yamlCfg.Ollama.Endpoint != "" {
			cfg.Ollama.Endpoint = yamlCfg.Ollama.Endpoint
		}
		if yamlCfg.Ollama.Dimension > 0 {
			cfg.Ollama.Dimension = yamlCfg.Ollama.Dimension
		}
		if yamlCfg.Ollama.Timeout != "" {
			if parsed, err := time.ParseDuration(yamlCfg.Ollama.Timeout); err == nil {
				cfg.Ollama.Timeout = parsed
			}
		}
		if yamlCfg.Ollama.BatchSize > 0 {
			cfg.Ollama.BatchSize = yamlCfg.Ollama.BatchSize
		}

		// OpenAI config from YAML
		if yamlCfg.OpenAI.APIKey != "" {
			cfg.OpenAI.APIKey = yamlCfg.OpenAI.APIKey
		}
		if yamlCfg.OpenAI.Model != "" {
			cfg.OpenAI.Model = yamlCfg.OpenAI.Model
		}
		if yamlCfg.OpenAI.Dimension > 0 {
			cfg.OpenAI.Dimension = yamlCfg.OpenAI.Dimension
		}
		if yamlCfg.OpenAI.Timeout != "" {
			if parsed, err := time.ParseDuration(yamlCfg.OpenAI.Timeout); err == nil {
				cfg.OpenAI.Timeout = parsed
			}
		}

		// Gemini config from YAML
		if yamlCfg.Gemini.APIKey != "" {
			cfg.Gemini.APIKey = yamlCfg.Gemini.APIKey
		}
		if yamlCfg.Gemini.Model != "" {
			cfg.Gemini.Model = yamlCfg.Gemini.Model
		}
		if yamlCfg.Gemini.Dimension > 0 {
			cfg.Gemini.Dimension = yamlCfg.Gemini.Dimension
		}
		if yamlCfg.Gemini.Timeout != "" {
			if parsed, err := time.ParseDuration(yamlCfg.Gemini.Timeout); err == nil {
				cfg.Gemini.Timeout = parsed
			}
		}
	}

	// Environment variables override config file
	if provider := os.Getenv("AILANG_EMBED_PROVIDER"); provider != "" {
		cfg.Provider = provider
	}
	if model := os.Getenv("AILANG_OLLAMA_MODEL"); model != "" {
		cfg.Ollama.Model = model
	}
	if endpoint := os.Getenv("AILANG_OLLAMA_ENDPOINT"); endpoint != "" {
		cfg.Ollama.Endpoint = endpoint
	}

	// OpenAI env var defaults. The MODEL is deliberately left empty here:
	// NewEmbedderFromConfig resolves it (env var, else the deprecated default
	// with a warning), because this function cannot return an error.
	if cfg.OpenAI.APIKey == "" {
		cfg.OpenAI.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if m := os.Getenv(EnvEmbedOpenAIModel); m != "" {
		cfg.OpenAI.Model = m
	}
	if cfg.OpenAI.Timeout == 0 {
		cfg.OpenAI.Timeout = 30 * time.Second
	}

	// Gemini env var defaults — same shape.
	if cfg.Gemini.APIKey == "" {
		cfg.Gemini.APIKey = os.Getenv("GOOGLE_API_KEY")
	}
	if m := os.Getenv(EnvEmbedGeminiModel); m != "" {
		cfg.Gemini.Model = m
	}
	if cfg.Gemini.Timeout == 0 {
		cfg.Gemini.Timeout = 30 * time.Second
	}

	return cfg
}

// Environment variables naming the OpenAI and Gemini embedding models
// (M-V1-SIMPLIFY-S4 M1). Precedence: env var > embeddings.<provider>.model in
// ~/.ailang/config.yaml > the deprecated default below. Ollama's model already
// has AILANG_OLLAMA_MODEL.
const (
	EnvEmbedOpenAIModel = "AILANG_EMBED_OPENAI_MODEL"
	EnvEmbedGeminiModel = "AILANG_EMBED_GEMINI_MODEL"
)

// The models served when nothing names one. Each fixes a VECTOR DIMENSION
// (1536 and 768): a brain indexed under one model and queried under another
// compares vectors of different geometry and returns confidently wrong
// neighbours — no error, just silently corrupted search. That is why the
// default is deprecated rather than merely documented: an operator must know
// which model their stored vectors came from.
const (
	deprecatedOpenAIEmbedModel = "text-embedding-3-small"
	deprecatedGeminiEmbedModel = "text-embedding-004"
)

// resolveEmbedModel serves the configured model, else the deprecated default
// through config.DeprecatedDefault — one stderr warning per process naming
// the env var, plus one line saying WHY the model must be pinned; under
// AILANG_STRICT_CONFIG=1 an error wrapping config.ErrDeprecatedDefault.
func resolveEmbedModel(configured, envName, deprecated string) (string, error) {
	if configured != "" {
		return configured, nil
	}
	model, err := config.DeprecatedDefault(envName, deprecated)
	if err != nil {
		return "", fmt.Errorf("embedding model: %w", err)
	}
	embedDimensionWarning.Do(func() {
		fmt.Fprintf(os.Stderr, "%s: the embedding model fixes the vector dimension; a brain indexed under a different model "+
			"will return silently wrong search results. Pin it so stored vectors and queries agree.\n", envName)
	})
	return model, nil
}

// embedDimensionWarning prints the dimension-mismatch consequence once per
// process, alongside config.DeprecatedDefault's own once-per-name line.
var embedDimensionWarning sync.Once

// NewEmbedderFromConfig creates the appropriate Embedder based on config.
// Returns (nil, nil) if provider is "none" — callers should check for nil.
func NewEmbedderFromConfig(cfg EmbedConfig) (Embedder, error) {
	switch cfg.Provider {
	case "ollama":
		return NewOllamaEmbedder(cfg.Ollama)
	case "openai":
		if cfg.OpenAI.APIKey == "" {
			return nil, fmt.Errorf("openai embedder requires OPENAI_API_KEY or openai.api_key config")
		}
		model, err := resolveEmbedModel(cfg.OpenAI.Model, EnvEmbedOpenAIModel, deprecatedOpenAIEmbedModel)
		if err != nil {
			return nil, err
		}
		cfg.OpenAI.Model = model
		return NewOpenAIEmbedder(cfg.OpenAI)
	case "gemini":
		if cfg.Gemini.APIKey == "" {
			return nil, fmt.Errorf("gemini embedder requires GOOGLE_API_KEY or gemini.api_key config")
		}
		model, err := resolveEmbedModel(cfg.Gemini.Model, EnvEmbedGeminiModel, deprecatedGeminiEmbedModel)
		if err != nil {
			return nil, err
		}
		cfg.Gemini.Model = model
		return NewGeminiEmbedder(cfg.Gemini)
	case "none", "":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown embedding provider %q: valid providers are ollama, openai, gemini, none", cfg.Provider)
	}
}

// OllamaEmbedder implements Embedder using local Ollama
type OllamaEmbedder struct {
	client    *api.Client
	model     string
	dimension int
	timeout   time.Duration
}

// NewOllamaEmbedder creates a new Ollama-based embedder
func NewOllamaEmbedder(cfg OllamaConfig) (*OllamaEmbedder, error) {
	// Set OLLAMA_HOST for the client
	if cfg.Endpoint != "" {
		os.Setenv("OLLAMA_HOST", cfg.Endpoint)
	}

	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	dimension := cfg.Dimension
	if dimension == 0 {
		// Auto-detect based on model
		dimension = detectDimension(cfg.Model)
	}

	return &OllamaEmbedder{
		client:    client,
		model:     cfg.Model,
		dimension: dimension,
		timeout:   timeout,
	}, nil
}

// detectDimension returns the embedding dimension for known models
func detectDimension(model string) int {
	dimensions := map[string]int{
		"nomic-embed-text":       768,
		"mxbai-embed-large":      1024,
		"gemma2:2b":              2048,
		"all-minilm":             384,
		"snowflake-arctic-embed": 1024,
	}
	if dim, ok := dimensions[model]; ok {
		return dim
	}
	return 768 // Default
}

// MaxChunkSize is the maximum characters per chunk for embedding
// embeddinggemma has 2K context (~8000 chars), we use 6000 to be safe
const MaxChunkSize = 6000

// Embed generates an embedding for a single text
// For long texts, it chunks and averages the embeddings
func (e *OllamaEmbedder) Embed(text string) ([]float32, error) {
	// If text fits in one chunk, embed directly
	if len(text) <= MaxChunkSize {
		return e.embedSingle(text)
	}

	// Chunk the text and embed each chunk
	chunks := chunkText(text, MaxChunkSize)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks generated from text")
	}

	// Embed each chunk
	var embeddings [][]float32
	for _, chunk := range chunks {
		emb, err := e.embedSingle(chunk)
		if err != nil {
			// Skip failed chunks but continue
			continue
		}
		embeddings = append(embeddings, emb)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("all chunks failed to embed")
	}

	// Average the embeddings
	return averageEmbeddings(embeddings), nil
}

// embedSingle embeds a single chunk of text
func (e *OllamaEmbedder) embedSingle(text string) ([]float32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	resp, err := e.client.Embed(ctx, &api.EmbedRequest{
		Model: e.model,
		Input: text,
	})
	if err != nil {
		return nil, fmt.Errorf("ollama embed failed: %w", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return resp.Embeddings[0], nil
}

// chunkText splits text into chunks of maxSize characters
// Uses markdown-aware boundaries: headers, code blocks, paragraphs, sentences
func chunkText(text string, maxSize int) []string {
	if len(text) <= maxSize {
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= maxSize {
			chunks = append(chunks, remaining)
			break
		}

		// Try to find a good split point in priority order
		chunk := remaining[:maxSize]
		splitPoint := maxSize

		// Priority 1: Markdown header (## or #)
		if idx := findLastIndex(chunk, "\n## "); idx > maxSize/3 {
			splitPoint = idx + 1 // Keep newline with previous chunk
		} else if idx := findLastIndex(chunk, "\n# "); idx > maxSize/3 {
			splitPoint = idx + 1
		} else if idx := findLastIndex(chunk, "\n### "); idx > maxSize/3 {
			splitPoint = idx + 1
		} else if idx := findLastIndex(chunk, "\n```"); idx > maxSize/3 {
			// Priority 2: Code block boundary
			splitPoint = idx + 1
		} else if idx := findLastIndex(chunk, "\n\n"); idx > maxSize/3 {
			// Priority 3: Paragraph boundary
			splitPoint = idx + 2
		} else if idx := findLastIndex(chunk, "\n- "); idx > maxSize/3 {
			// Priority 4: List item
			splitPoint = idx + 1
		} else if idx := findLastIndex(chunk, ". "); idx > maxSize/2 {
			// Priority 5: Sentence boundary
			splitPoint = idx + 2
		} else if idx := findLastIndex(chunk, "\n"); idx > maxSize/2 {
			// Priority 6: Any line break
			splitPoint = idx + 1
		} else if idx := findLastIndex(chunk, " "); idx > maxSize/2 {
			// Priority 7: Word boundary
			splitPoint = idx + 1
		}

		chunks = append(chunks, remaining[:splitPoint])
		remaining = remaining[splitPoint:]
	}

	return chunks
}

// findLastIndex finds the last occurrence of substr in s
func findLastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// averageEmbeddings computes the mean of multiple embeddings
func averageEmbeddings(embeddings [][]float32) []float32 {
	if len(embeddings) == 0 {
		return nil
	}

	dim := len(embeddings[0])
	result := make([]float32, dim)

	for _, emb := range embeddings {
		for i, v := range emb {
			result[i] += v
		}
	}

	n := float32(len(embeddings))
	for i := range result {
		result[i] /= n
	}

	return result
}

// EmbedBatch generates embeddings for multiple texts
func (e *OllamaEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))

	for i, text := range texts {
		embedding, err := e.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		results[i] = embedding
	}

	return results, nil
}

// Dimension returns the embedding dimension
func (e *OllamaEmbedder) Dimension() int {
	return e.dimension
}

// ModelName returns the model identifier
func (e *OllamaEmbedder) ModelName() string {
	return "ollama:" + e.model
}

// EmbeddingToJSON converts an embedding to JSON string for storage
func EmbeddingToJSON(embedding []float32) string {
	data, _ := json.Marshal(embedding)
	return string(data)
}

// EmbeddingFromJSON parses an embedding from JSON string
func EmbeddingFromJSON(data string) ([]float32, error) {
	var embedding []float32
	if err := json.Unmarshal([]byte(data), &embedding); err != nil {
		return nil, err
	}
	return embedding, nil
}
