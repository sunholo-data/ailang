package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIEmbedder implements Embedder using the OpenAI Embeddings API.
type OpenAIEmbedder struct {
	apiKey    string
	model     string
	dimension int
	timeout   time.Duration
	client    *http.Client
}

// NewOpenAIEmbedder creates a new OpenAI-based embedder.
func NewOpenAIEmbedder(cfg OpenAIEmbedConfig) (*OpenAIEmbedder, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	model := cfg.Model
	if model == "" {
		model = "text-embedding-3-small"
	}

	dimension := cfg.Dimension
	if dimension == 0 {
		dimension = detectOpenAIDimension(model)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &OpenAIEmbedder{
		apiKey:    cfg.APIKey,
		model:     model,
		dimension: dimension,
		timeout:   timeout,
		client:    &http.Client{Timeout: timeout},
	}, nil
}

func detectOpenAIDimension(model string) int {
	switch model {
	case "text-embedding-3-small":
		return 1536
	case "text-embedding-3-large":
		return 3072
	case "text-embedding-ada-002":
		return 1536
	default:
		return 1536
	}
}

type openaiEmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openaiEmbedResponse struct {
	Data  []openaiEmbedData `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

type openaiEmbedData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

// Embed generates an embedding for a single text.
// For long texts, it chunks and averages the embeddings.
func (e *OpenAIEmbedder) Embed(text string) ([]float32, error) {
	if len(text) <= MaxChunkSize {
		return e.embedSingle(text)
	}

	chunks := chunkText(text, MaxChunkSize)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks generated from text")
	}

	var embeddings [][]float32
	for _, chunk := range chunks {
		emb, err := e.embedSingle(chunk)
		if err != nil {
			continue
		}
		embeddings = append(embeddings, emb)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("all chunks failed to embed")
	}

	return averageEmbeddings(embeddings), nil
}

func (e *OpenAIEmbedder) embedSingle(text string) ([]float32, error) {
	reqBody := openaiEmbedRequest{
		Model: e.model,
		Input: text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai embed request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai embed failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var embedResp openaiEmbedResponse
	if err := json.Unmarshal(respBody, &embedResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if embedResp.Error != nil {
		return nil, fmt.Errorf("openai embed error: %s", embedResp.Error.Message)
	}

	if len(embedResp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return embedResp.Data[0].Embedding, nil
}

// EmbedBatch generates embeddings for multiple texts.
func (e *OpenAIEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := e.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		results[i] = emb
	}
	return results, nil
}

// Dimension returns the embedding dimension.
func (e *OpenAIEmbedder) Dimension() int {
	return e.dimension
}

// ModelName returns the model identifier.
func (e *OpenAIEmbedder) ModelName() string {
	return "openai:" + e.model
}
