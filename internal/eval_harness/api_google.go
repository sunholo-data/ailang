package eval_harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
)

// Google Gemini API structures (Vertex AI)
type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// callGemini makes a request to the Google Gemini API via Vertex AI
func (a *AIAgent) callGemini(ctx context.Context, prompt string) (*GenerateResult, error) {
	// Get access token from gcloud ADC
	accessToken, err := getGoogleAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get Google access token: %w", err)
	}

	// Get GCP project ID
	projectID, err := getGCPProject()
	if err != nil {
		return nil, fmt.Errorf("failed to get GCP project: %w", err)
	}

	// Vertex AI endpoint
	// Format: https://{REGION}-aiplatform.googleapis.com/v1/projects/{PROJECT}/locations/{REGION}/publishers/google/models/{MODEL}:generateContent
	region := "us-central1" // Default region
	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
		region, projectID, region, a.model)

	systemPrompt := "You are a programming assistant. Generate ONLY code without explanations or markdown formatting."
	fullPrompt := fmt.Sprintf("%s\n\n%s", systemPrompt, prompt)

	req := geminiRequest{
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{Text: fullPrompt},
				},
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp geminiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	code := apiResp.Candidates[0].Content.Parts[0].Text
	totalTokens := apiResp.UsageMetadata.TotalTokenCount

	return &GenerateResult{
		Code:   extractCodeFromMarkdown(code),
		Tokens: totalTokens,
		Model:  a.model,
	}, nil
}

// getGoogleAccessToken gets an access token from gcloud ADC
func getGoogleAccessToken() (string, error) {
	cmd := exec.Command("gcloud", "auth", "application-default", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gcloud command failed (run 'gcloud auth application-default login'): %w", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("empty token from gcloud")
	}

	return token, nil
}

// getGCPProject gets the current GCP project ID
func getGCPProject() (string, error) {
	cmd := exec.Command("gcloud", "config", "get-value", "project")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get GCP project (run 'gcloud config set project PROJECT_ID'): %w", err)
	}

	project := strings.TrimSpace(string(output))
	if project == "" {
		return "", fmt.Errorf("no GCP project set")
	}

	return project, nil
}
