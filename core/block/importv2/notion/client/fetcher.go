package client

import (
	"context"
	"errors"
	"io"
	"time"
)

type FetcherOption func(*noopFileFetcher)

// WithFetcherTransport is a no-op.
func WithFetcherTransport(transport Transport) FetcherOption { return func(f *noopFileFetcher) {} }

// WithFetcherRetryPolicy is a no-op.
func WithFetcherRetryPolicy(policy RetryPolicy) FetcherOption { return func(f *noopFileFetcher) {} }

// WithFetcherStallTimeout is a no-op.
func WithFetcherStallTimeout(d time.Duration) FetcherOption { return func(f *noopFileFetcher) {} }

type noopFileFetcher struct{}

// NewFileFetcher returns an offline file fetcher that always fails.
func NewFileFetcher(opts ...FetcherOption) FileFetcher {
	return &noopFileFetcher{}
}

// Fetch always fails in offline builds.
func (f *noopFileFetcher) Fetch(ctx context.Context, url string, dst io.Writer) error {
	return errors.New("notion file fetch disabled in offline build")
}
