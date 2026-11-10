package wire

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
)

// maxLineBytes is the longest NDJSON line a stream may carry (the SDK's
// 8 * format.MegaByte, decimal). A longer line fails the call with
// bufio.ErrTooLong ("bufio.Scanner: token too long").
const maxLineBytes = 8 * 1000 * 1000

// userAgent is the SDK's User-Agent, kept byte-identical (the SDK's
// version.Version is "0.0.0" outside an ollama release build).
var userAgent = fmt.Sprintf("ollama/0.0.0 (%s %s) Go/%s", runtime.GOARCH, runtime.GOOS, runtime.Version())

// StatusError is an HTTP error reply. Its fields deliberately carry no JSON
// tags but ErrorMessage's: the SDK decoded error bodies into it, so a body
// with a "status" key fills Status (encoding/json matches field names
// case-insensitively). Error() is the text callers classify.
type StatusError struct {
	StatusCode   int
	Status       string
	ErrorMessage string `json:"error"`
}

func (e StatusError) Error() string {
	switch {
	case e.Status != "" && e.ErrorMessage != "":
		return fmt.Sprintf("%s: %s", e.Status, e.ErrorMessage)
	case e.Status != "":
		return e.Status
	case e.ErrorMessage != "":
		return e.ErrorMessage
	default:
		return "something went wrong, please see the ollama server logs for details"
	}
}

// AuthorizationError is a 401 reply.
type AuthorizationError struct {
	StatusCode int
	Status     string
	SigninURL  string `json:"signin_url"`
}

func (e AuthorizationError) Error() string {
	if e.Status != "" {
		return e.Status
	}
	return "something went wrong, please see the ollama server logs for details"
}

// Client talks to one Ollama server.
type Client struct {
	base *url.URL
	http *http.Client
}

// NewClient returns a client for base using hc. It refuses ollama.com: the
// SDK signed requests to that host with the device key from `ollama signin`,
// which this client does not do, and an unsigned request there would fail
// with a 401 far from its cause. AILANG reaches Ollama Cloud through the
// local daemon, which signs on its behalf.
func NewClient(base *url.URL, hc *http.Client) (*Client, error) {
	if base.Hostname() == "ollama.com" {
		return nil, fmt.Errorf("ollama: direct requests to %s are not supported (they need request signing); point OLLAMA_HOST at a local ollama daemon, which proxies Ollama Cloud models", base.Host)
	}
	return &Client{base: base, http: hc}, nil
}

// ClientFromEnvironment returns a client for HostFromEnvironment() over
// http.DefaultClient, as the SDK's ClientFromEnvironment did.
func ClientFromEnvironment() (*Client, error) {
	return NewClient(HostFromEnvironment(), http.DefaultClient)
}

// HostFromEnvironment resolves OLLAMA_HOST exactly as the SDK's
// envconfig.Host did: surrounding whitespace and quotes are stripped, then
// ParseHost applies. Unset means http://127.0.0.1:11434.
func HostFromEnvironment() *url.URL {
	return ParseHost(strings.Trim(strings.TrimSpace(config.OllamaHost()), "\"'"))
}

// ParseHost turns an OLLAMA_HOST-style string into a base URL with the SDK's
// normalisation: an optional scheme (http by default; the bare "ollama.com"
// means https on 443), a host that may be an IP, a name or empty
// (127.0.0.1), a port that defaults per scheme (11434, or 80 / 443 when the
// scheme is explicit) and is replaced by that default when out of range, and
// an optional path. It does not strip quotes; HostFromEnvironment does.
func ParseHost(s string) *url.URL {
	defaultPort := "11434"

	s = strings.TrimSpace(s)
	scheme, hostport, ok := strings.Cut(s, "://")
	switch {
	case !ok:
		scheme, hostport = "http", s
		if s == "ollama.com" {
			scheme, hostport = "https", "ollama.com:443"
		}
	case scheme == "http":
		defaultPort = "80"
	case scheme == "https":
		defaultPort = "443"
	}

	hostport, path, _ := strings.Cut(hostport, "/")
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		host, port = "127.0.0.1", defaultPort
		if ip := net.ParseIP(strings.Trim(hostport, "[]")); ip != nil {
			host = ip.String()
		} else if hostport != "" {
			host = hostport
		}
	}

	if n, err := strconv.ParseInt(port, 10, 32); err != nil || n > 65535 || n < 0 {
		port = defaultPort
	}

	return &url.URL{Scheme: scheme, Host: net.JoinHostPort(host, port), Path: path}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body []byte, accept string) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base.JoinPath(path).String(), r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", userAgent)
	return req, nil
}

// do is a buffered call: the whole reply is read, a status >= 400 becomes a
// StatusError (or AuthorizationError on 401) decoded from the body, and a
// non-empty body otherwise decodes into out.
func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body []byte
	if in != nil {
		var err error
		if body, err = json.Marshal(in); err != nil {
			return err
		}
	}
	req, err := c.newRequest(ctx, method, path, body, "application/json")
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		if resp.StatusCode == http.StatusUnauthorized {
			authErr := AuthorizationError{StatusCode: resp.StatusCode}
			_ = json.Unmarshal(data, &authErr)
			return authErr
		}
		apiErr := StatusError{StatusCode: resp.StatusCode}
		if json.Unmarshal(data, &apiErr) != nil {
			apiErr.ErrorMessage = string(data)
		}
		return apiErr
	}
	if len(data) > 0 && out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

// ErrEmptyStream is a 2xx streaming reply with no lines at all. /api/generate
// and /api/chat always end with a done chunk, so an empty reply is a truncated
// one, not an empty answer.
var ErrEmptyStream = errors.New("ollama: empty streaming reply: no response lines before the body ended")

// stream POSTs in and calls fn with each NDJSON line of the reply. Every line
// is first checked for an {"error": ...} object; on a status >= 400 the first
// line becomes the error.
//
// A reply with no lines at all is an error at every status. This is the one
// deliberate departure from the SDK, which returned nil: its caller then saw a
// successful call with empty text and zero tokens, so an HTTP 500 with an empty
// body was banked as "the model said nothing".
func (c *Client) stream(ctx context.Context, path string, in any, fn func([]byte) error) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := c.newRequest(ctx, http.MethodPost, path, body, "application/x-ndjson")
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, maxLineBytes), maxLineBytes)
	lines := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		lines++
		var errLine struct {
			Error     string `json:"error,omitempty"`
			SigninURL string `json:"signin_url,omitempty"`
		}
		if err := json.Unmarshal(line, &errLine); err != nil {
			if resp.StatusCode >= http.StatusBadRequest {
				return StatusError{StatusCode: resp.StatusCode, Status: resp.Status, ErrorMessage: string(line)}
			}
			return errors.New(string(line))
		}
		switch {
		case resp.StatusCode == http.StatusUnauthorized:
			return AuthorizationError{StatusCode: resp.StatusCode, Status: resp.Status, SigninURL: errLine.SigninURL}
		case resp.StatusCode >= http.StatusBadRequest:
			return StatusError{StatusCode: resp.StatusCode, Status: resp.Status, ErrorMessage: errLine.Error}
		case errLine.Error != "":
			return errors.New(errLine.Error)
		}
		if err := fn(line); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if lines == 0 {
		switch {
		case resp.StatusCode == http.StatusUnauthorized:
			return AuthorizationError{StatusCode: resp.StatusCode, Status: resp.Status}
		case resp.StatusCode >= http.StatusBadRequest:
			return StatusError{StatusCode: resp.StatusCode, Status: resp.Status, ErrorMessage: "empty response body"}
		default:
			return ErrEmptyStream
		}
	}
	return nil
}

// Generate streams POST /api/generate, calling fn per chunk.
func (c *Client) Generate(ctx context.Context, req *GenerateRequest, fn func(GenerateResponse) error) error {
	return c.stream(ctx, "/api/generate", req, func(line []byte) error {
		var r GenerateResponse
		if err := json.Unmarshal(line, &r); err != nil {
			return err
		}
		return fn(r)
	})
}

// Chat streams POST /api/chat, calling fn per chunk.
func (c *Client) Chat(ctx context.Context, req *ChatRequest, fn func(ChatResponse) error) error {
	return c.stream(ctx, "/api/chat", req, func(line []byte) error {
		var r ChatResponse
		if err := json.Unmarshal(line, &r); err != nil {
			return err
		}
		return fn(r)
	})
}

// Embed calls POST /api/embed.
func (c *Client) Embed(ctx context.Context, req *EmbedRequest) (*EmbedResponse, error) {
	var r EmbedResponse
	if err := c.do(ctx, http.MethodPost, "/api/embed", req, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// List calls GET /api/tags.
func (c *Client) List(ctx context.Context) (*ListResponse, error) {
	var r ListResponse
	if err := c.do(ctx, http.MethodGet, "/api/tags", nil, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
