package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dividebyzero/univelop-api-mcp/internal/types"
)

// DefaultUserAgent is the User-Agent header sent with every request.
const DefaultUserAgent = "univelop-mcp/1.0"

// Max response body: 50 MB
const maxBodySize = 50 * 1024 * 1024

// idempotentMethods tracks which HTTP methods are safe to retry.
var idempotentMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodPut:     true,
	http.MethodDelete:  true,
	http.MethodHead:    true,
	http.MethodOptions: true,
}

// retryableStatuses are HTTP status codes that may succeed on retry.
var retryableStatuses = map[int]bool{
	http.StatusTooManyRequests:      true,
	http.StatusBadGateway:           true,
	http.StatusServiceUnavailable:   true,
	http.StatusGatewayTimeout:       true,
}

// Client wraps an HTTP client for talking to the Univelop API.
type Client struct {
	BaseURL    string // normalized, no trailing slash
	APIKey     string
	Label      string
	httpClient *http.Client
	maxRetries int
	limiter    *RateLimiter
}

// Options configures a Client.
type Options struct {
	Timeout    time.Duration
	MaxRetries int
	RateLimit  float64 // requests per second, 0 = unlimited
	Label      string
}

// New creates a new API client for a workspace.
func New(baseURL, apiKey string, opts Options) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("client: baseURL is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("client: apiKey is required")
	}

	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("client: invalid baseURL %q: %w", baseURL, err)
	}
	u.Path = strings.TrimRight(u.Path, "/")
	if u.Scheme == "" {
		u.Scheme = "https"
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	maxRetries := opts.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	return &Client{
		BaseURL:    strings.TrimRight(u.String(), "/"),
		APIKey:     apiKey,
		Label:      opts.Label,
		httpClient: &http.Client{Timeout: timeout},
		maxRetries: maxRetries,
		limiter:    NewRateLimiter(opts.RateLimit),
	}, nil
}

// Query is a URL query parameter builder.
type Query struct {
	url.Values
}

// NewQuery creates a new Query builder.
func NewQuery() Query {
	return Query{url.Values{}}
}

// SetString adds a string param if non-empty.
func (q Query) SetString(key, val string) {
	if val != "" {
		q.Set(key, val)
	}
}

// SetInt adds an integer param if non-zero.
func (q Query) SetInt(key string, val int) {
	if val != 0 {
		q.Set(key, strconv.Itoa(val))
	}
}

// SetBool adds a bool param as "true" if true.
func (q Query) SetBool(key string, val bool) {
	if val {
		q.Set(key, "true")
	}
}

// Encode returns the encoded query string (with leading ? if non-empty).
func (q Query) Encode() string {
	if len(q.Values) == 0 {
		return ""
	}
	return "?" + q.Values.Encode()
}

// request is the shared request builder and executor.
func (c *Client) request(ctx context.Context, method, path string, query url.Values, body []byte) (*types.Response, error) {
	// Wait for rate limiter
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit: %w", err)
		}
	}

	// Build URL
	fullURL := c.BaseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	isIdempotent := idempotentMethods[method]

	// Retry loop
	var lastErr error
	attempts := 1 + c.maxRetries
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			// Backoff: 200ms * 2^(attempt-1) + jitter
			backoff := time.Duration(200*(1<<(attempt-1))) * time.Millisecond
			backoff += time.Duration(rand.Intn(50)) * time.Millisecond
			if backoff > 3*time.Second {
				backoff = 3 * time.Second
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		// Build request in-loop so the body reader is fresh on retry
		var attemptBody io.Reader
		if body != nil {
			attemptBody = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, fullURL, attemptBody)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("x-api-key", c.APIKey)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", DefaultUserAgent)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if isIdempotent && attempt < attempts-1 {
				continue
			}
			return nil, lastErr
		}

		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBodySize+1))
		resp.Body.Close()

		if readErr != nil {
			lastErr = fmt.Errorf("reading response body: %w", readErr)
			if isIdempotent && attempt < attempts-1 {
				continue
			}
			return nil, lastErr
		}

		if len(bodyBytes) > maxBodySize {
			lastErr = fmt.Errorf("response body exceeds %d bytes", maxBodySize)
			return nil, lastErr
		}

		// Success
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return &types.Response{
				StatusCode: resp.StatusCode,
				Header:     resp.Header.Clone(),
				Body:       bodyBytes,
			}, nil
		}

		// Retryable
		if isIdempotent && retryableStatuses[resp.StatusCode] && attempt < attempts-1 {
			lastErr = fmt.Errorf("%s %s: HTTP %d", method, fullURL, resp.StatusCode)
			continue
		}

		// Non-retryable error
		msg := types.ParseErrorBody(bodyBytes, resp.StatusCode)
		return nil, &types.APIError{
			StatusCode: resp.StatusCode,
			Method:     method,
			URL:        fullURL,
			Msg:        msg,
		}
	}

	return nil, lastErr
}

// Get issues a GET request.
func (c *Client) Get(ctx context.Context, path string, query url.Values) (*types.Response, error) {
	return c.request(ctx, http.MethodGet, path, query, nil)
}

// Post issues a POST request with a JSON body.
func (c *Client) Post(ctx context.Context, path string, query url.Values, body []byte) (*types.Response, error) {
	return c.request(ctx, http.MethodPost, path, query, body)
}

// Put issues a PUT request with a JSON body.
func (c *Client) Put(ctx context.Context, path string, query url.Values, body []byte) (*types.Response, error) {
	return c.request(ctx, http.MethodPut, path, query, body)
}

// Patch issues a PATCH request with a JSON body.
func (c *Client) Patch(ctx context.Context, path string, query url.Values, body []byte) (*types.Response, error) {
	return c.request(ctx, http.MethodPatch, path, query, body)
}

// Delete issues a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) (*types.Response, error) {
	return c.request(ctx, http.MethodDelete, path, nil, nil)
}

// EscapePath encodes a path segment for safe URL construction.
func EscapePath(seg string) string {
	return url.PathEscape(seg)
}