package gemini

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	gcpauth "github.com/sunholo-data/ailang/internal/auth/gcp"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/strutil"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var geminiAITracer = telemetry.Tracer("ai.gemini")

const (
	// AI Studio base URL (uses API key)
	aiStudioBaseURL = "https://generativelanguage.googleapis.com/v1beta"
)

// vertexAIBaseURL is the Vertex AI endpoint for a location. The host is what
// Google's data-residency guarantee rests on, not the locations/{location}
// path segment: the global host aiplatform.googleapis.com accepts any location
// in the path, but Google documents global endpoints as giving no regional
// isolation or residency guarantee. So the host follows the location:
//
//	"global"              -> https://aiplatform.googleapis.com/v1
//	"eu", "us" (multi)    -> https://aiplatform.{eu,us}.rep.googleapis.com/v1
//	a region, e.g. "europe-west4" -> https://europe-west4-aiplatform.googleapis.com/v1
//
// Until v0.56.x every location went to the global host (docparse found its
// GOOGLE_CLOUD_LOCATION=eu traffic on aiplatform.googleapis.com, 2026-11-03).
// A location that is not a plain lowercase name is refused rather than spliced
// into a hostname.
func vertexAIBaseURL(location string) (string, error) {
	switch location {
	case "global":
		return "https://aiplatform.googleapis.com/v1", nil
	case "eu", "us":
		return "https://aiplatform." + location + ".rep.googleapis.com/v1", nil
	}
	if !vertexLocationPattern.MatchString(location) {
		return "", ai.NewProviderError("gemini", 0,
			fmt.Sprintf("invalid Vertex AI location %q (want global, eu, us or a region such as europe-west4)", location), nil)
	}
	return "https://" + location + "-aiplatform.googleapis.com/v1", nil
}

// vertexLocationPattern: a Google Cloud region name (europe-west4, us-central1).
var vertexLocationPattern = regexp.MustCompile(`^[a-z]+(-[a-z]+)*[0-9]+$`)

// Client implements ai.Provider for Google's Gemini API.
// It supports both AI Studio and Vertex AI endpoints.
type Client struct {
	apiKey           string   // API key for AI Studio
	projectID        string   // GCP project for Vertex AI
	location         string   // Vertex AI location; see NewVertexAIClient for how it resolves
	authType         AuthType // Authentication type
	httpClient       *http.Client
	baseURL          string // Override base URL (for testing)
	preparedProject  func(context.Context) (string, error)
	preparedLocation bool
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithLocation sets the GCP location for Vertex AI.
func WithLocation(location string) ClientOption {
	return func(c *Client) {
		c.location = location
	}
}

// WithBaseURL sets a custom base URL (useful for testing).
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// NewClient creates a new Gemini client using API key (AI Studio). The AI
// Studio endpoint has no location; the field stays empty unless WithLocation
// sets it, so no regional default is ever implied here.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:     apiKey,
		authType:   AuthAPIKey,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// EnvVertexLocation names the Vertex AI location a request is routed to. It
// decides the regional endpoint, data residency and — since Vertex prices
// some models per region — the bill, so it is not a cosmetic default.
const EnvVertexLocation = config.EnvGoogleCloudLocation

// deprecatedVertexLocation is the value served when nothing names one
// (M-V1-SIMPLIFY-S4 M1): warned once via config.DeprecatedDefault, refused
// under AILANG_STRICT_CONFIG=1.
const deprecatedVertexLocation = "global"

// NewVertexAIClient creates a new Gemini client using ADC (Vertex AI).
// If projectID is empty, it will be fetched from gcloud config. The location
// is WithLocation, else GOOGLE_CLOUD_LOCATION, else the deprecated "global".
func NewVertexAIClient(projectID string, opts ...ClientOption) (*Client, error) {
	c := &Client{
		projectID:  projectID,
		authType:   AuthADC,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.location == "" && !c.preparedLocation {
		c.location = strings.TrimSpace(config.GoogleCloudLocation())
	}
	if c.location == "" {
		loc, err := config.DeprecatedDefault(EnvVertexLocation, deprecatedVertexLocation)
		if err != nil {
			return nil, ai.NewProviderError("gemini", 0, "no Vertex AI location", err)
		}
		c.location = loc
	}

	// Get project ID from gcloud if not provided
	if c.projectID == "" {
		project, err := c.resolveProject()
		if err != nil {
			return nil, ai.NewProviderError("gemini", 0, "failed to get GCP project", err)
		}
		c.projectID = project
	}

	return c, nil
}

// Generate implements ai.Provider.
func (c *Client) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	if req.Routing != nil && (req.Routing.HasRouting() || req.Routing.PriceCapSet()) {
		return nil, ai.NewProviderError("gemini", 0,
			"this provider does not support AIRoutingPolicy; use openrouter instead",
			ai.ErrRoutingNotSupported)
	}
	// Start OTEL span
	ctx, span := telemetry.StartSpan(ctx, geminiAITracer, "gemini.generate",
		trace.WithAttributes(
			attribute.String("ai.provider", "gemini"),
			attribute.String("ai.model", req.Model),
			attribute.String("ai.auth_type", string(c.authType)),
			attribute.String("ai.prompt_preview", strutil.Truncate(req.UserPrompt, 100)),
		),
	)
	defer span.End()

	resp, err := c.generateContent(ctx, req)
	if err != nil {
		span.SetAttributes(
			attribute.String("error.message", strutil.Truncate(err.Error(), 200)),
			attribute.String("error.category", telemetry.CategorizeError(err)),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Record success metrics on span
	span.SetAttributes(
		attribute.Int("ai.tokens_in", resp.InputTokens),
		attribute.Int("ai.tokens_out", resp.OutputTokens),
		attribute.Int("ai.tokens_total", resp.TotalTokens),
		attribute.String("ai.response_preview", strutil.Truncate(resp.Text, 100)),
	)

	return resp, nil
}

// Name implements ai.Provider.
func (c *Client) Name() string {
	return "gemini"
}

// NewHandler creates an ai.Handler wrapping this client.
func (c *Client) NewHandler(model string, opts ...ai.HandlerOption) *ai.Handler {
	return ai.NewHandler(c, model, opts...)
}

// getAccessToken retrieves an ADC access token for Vertex AI. Delegates to
// the shared internal/auth/gcp helper (metadata-first, gcloud fallback) so
// the executor side has a single source of truth. The wrap with ai.NewProviderError
// preserves the existing error shape callers of this package expect.
func getAccessToken() (string, error) {
	token, err := gcpauth.AccessToken(context.Background())
	if err != nil {
		return "", ai.NewProviderError("gemini", 0, err.Error(), nil)
	}
	return token, nil
}

// getGCPProject gets the current GCP project ID: config.CloudProject (env,
// config file, metadata server — one precedence for the whole binary), then
// the gcloud CLI as a local-dev fallback. The CLI step stays here, not in the
// resolver: a Vertex call from a laptop is the one place "whatever gcloud is
// pointed at" is what the developer means, and the store selectors must never
// inherit it.
func getGCPProject() (string, error) {
	if project, err := config.CloudProject(context.Background()); err == nil {
		return project, nil
	}

	return discoverGcloudProject()
}

func discoverGcloudProject() (string, error) {
	cmd := exec.Command("gcloud", "config", "get-value", "project")
	output, err := cmd.Output()
	if err != nil {
		return "", ai.NewProviderError("gemini", 0,
			"no GCP project: set GOOGLE_CLOUD_PROJECT env var, or run 'gcloud config set project PROJECT'", err)
	}

	project := strings.TrimSpace(string(output))
	if project == "" {
		return "", ai.NewProviderError("gemini", 0, "no GCP project set", nil)
	}

	return project, nil
}
