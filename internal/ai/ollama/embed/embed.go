package embed

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/ollama/ollama/api"
	"time"
)

// MediaInput builds Ollama's multimodal embed input object, omitting empty parts
func MediaInput(text string, image, audio []byte) map[string]string {
	input := map[string]string{}
	if text != "" {
		input["text"] = text
	}
	if len(image) > 0 {
		input["image"] = base64.StdEncoding.EncodeToString(image)
	}
	if len(audio) > 0 {
		input["audio"] = base64.StdEncoding.EncodeToString(audio)
	}
	return input
}

// EmbedRequest calls Ollama's embed API with one input and returns its vector
func EmbedRequest(name, model string, input any, timeout time.Duration) ([]float64, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create Ollama client: %w", name, err)
	}

	goCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := client.Embed(goCtx, &api.EmbedRequest{
		Model: model,
		Input: input,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: Ollama embed failed: %w", name, err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("%s: no embeddings returned", name)
	}

	embedding := resp.Embeddings[0]
	vector := make([]float64, len(embedding))
	for i, v := range embedding {
		vector[i] = float64(v)
	}
	return vector, nil
}
