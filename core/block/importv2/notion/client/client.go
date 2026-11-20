// Package client is a no-op Notion API transport for the offline Aparok build.
package client

import (
	"context"
	"errors"
	"io"
	"net/http"
)

const (
	BaseURL = "https://api.notion.com/v1"
)

// PageRateCeiling returns 0 in offline builds.
func PageRateCeiling() float64 { return 0 }

// Transport mirrors the original transport interface.
type Transport interface {
	Do(req *http.Request) (*http.Response, error)
}

// FileFetcher mirrors the original file fetcher interface.
type FileFetcher interface {
	Fetch(ctx context.Context, url string, dst io.Writer) error
}

// Client is the offline no-op Notion client.
type Client struct {
	status StatusHook
}

// Option configures the client.
type Option func(*Client)

// WithTransport is a no-op in offline builds.
func WithTransport(transport Transport) Option { return func(c *Client) {} }

// WithBaseURL is a no-op in offline builds.
func WithBaseURL(baseURL string) Option { return func(c *Client) {} }

// WithRetryPolicy is a no-op in offline builds.
func WithRetryPolicy(policy RetryPolicy) Option { return func(c *Client) {} }

// WithRateLimit is a no-op in offline builds.
func WithRateLimit(requestsPerSecond float64) Option { return func(c *Client) {} }

// NewClient returns an offline no-op client.
func NewClient(token string, opts ...Option) *Client {
	return &Client{}
}

// Request always fails in offline builds.
func (c *Client) Request(ctx context.Context, method, path string, body any, out any) error {
	return errors.New("notion import disabled in offline build")
}
