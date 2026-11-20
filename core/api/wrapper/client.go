package wrapper

// client.go — no-op HTTP transport for the offline Aparok build.

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	v2model "github.com/anyproto/anytype-heart/core/api/v2/model"
)

// DefaultBaseURL is preserved for API compatibility.
const DefaultBaseURL = "http://127.0.0.1:31009"

// clientMaxAttempts is preserved for API compatibility.
const clientMaxAttempts = 3

// Client is a no-op wrapper client in offline builds.
type Client struct {
	BaseURL string
	APIKey  string
	// HTTP is intentionally not an *HTTP client in offline builds.
	HTTP   struct{}
	Backoff func(attempt int) time.Duration
}

// NewClient builds a no-op client.
func NewClient(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		APIKey:  apiKey,
		Backoff: func(attempt int) time.Duration { return time.Duration(attempt) * time.Second },
	}
}

// RawRequest is one call to the server as the full tool table makes it.
type RawRequest struct {
	Method         string
	Path           string
	Query          url.Values
	Headers        http.Header
	Body           []byte
	IdempotencyKey string
}

// DoRaw always fails in offline builds.
func (c *Client) DoRaw(ctx context.Context, r RawRequest) (int, []byte, error) {
	return 0, nil, errors.New("API wrapper calls are disabled in offline builds")
}

// ToolError is a server-reported C6 error surfaced by a tool.
type ToolError struct {
	Status  int
	Code    string
	Message string
	Text    string
	Issues  []v2model.Issue
}

func (e *ToolError) Error() string { return e.Text }

// decode always fails in offline builds.
func (c *Client) decode(ctx context.Context, r apiRequest, out any) error {
	return errors.New("API wrapper calls are disabled in offline builds")
}

// raw always fails in offline builds.
func (c *Client) raw(ctx context.Context, r apiRequest) ([]byte, error) {
	return nil, errors.New("API wrapper calls are disabled in offline builds")
}

// apiRequest is an simplified, internal request descriptor.
type apiRequest struct {
	method         string
	path           string
	query          url.Values
	body           any
	rawBody        []byte
	headers        http.Header
	idempotencyKey string
	ifMatch        string
}

// RenderErrorText renders a C6 envelope as the one-string error.
func RenderErrorText(message string, issues []v2model.Issue) string {
	return renderErrorText(message, issues)
}

// renderErrorText renders a C6 envelope as the one-string error.
func renderErrorText(message string, issues []v2model.Issue) string {
	var b strings.Builder
	b.WriteString(message)
	for _, issue := range issues {
		b.WriteString("\n  ")
		if issue.Path != "" {
			b.WriteString(issue.Path)
			b.WriteString(": ")
		}
		b.WriteString(issue.Message)
		if issue.Hint != "" {
			b.WriteString(" (")
			b.WriteString(issue.Hint)
			b.WriteString(")")
		}
	}
	return b.String()
}

// isAmbiguous reports whether an error is the server's 400 ambiguous_input.
func isAmbiguous(err error) bool {
	var te *ToolError
	return errors.As(err, &te) && te.Code == v2model.CodeAmbiguousInput
}

// seg escapes one path segment.
func seg(s string) string { return url.PathEscape(s) }
