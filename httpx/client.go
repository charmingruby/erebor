package httpx

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrCreateRequest indicates that an HTTP request could not be constructed.
	ErrCreateRequest = errors.New("create request")
	// ErrEncodeRequestBody indicates that a request body could not be encoded.
	ErrEncodeRequestBody = errors.New("encode request body")
	// ErrDoRequest indicates that an HTTP request failed.
	ErrDoRequest = errors.New("do request")
	// ErrUnexpectedStatusCode indicates that a response has an unexpected status.
	ErrUnexpectedStatusCode = errors.New("unexpected status code")
	// ErrDecodeResponse indicates that a response body could not be read or decoded.
	ErrDecodeResponse = errors.New("decode response")
	// ErrMissingBaseURL indicates that a relative request URL has no configured base URL.
	ErrMissingBaseURL = errors.New("missing base URL")
)

// Client sends HTTP requests with an optional base URL.
type Client struct {
	client  *http.Client
	baseURL string
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// NewClient creates a client with a five-second default timeout.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		client: &http.Client{Timeout: 5 * time.Second},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithHTTPClient sets the underlying HTTP client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.client = hc
	}
}

// WithBaseURL sets the base URL used for relative request URLs.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// Request describes an HTTP request and its expected response status.
type Request struct {
	Body           any
	Headers        http.Header
	Method         string
	URL            string
	ExpectedStatus int
}

// Do sends a request and decodes its JSON response into T.
func Do[T any](
	ctx context.Context,
	c *Client,
	r Request,
) (*T, error) {
	endpoint, err := c.resolveURL(r.URL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateRequest, err)
	}

	var bodyReader io.Reader
	if r.Body != nil {
		buf := &bytes.Buffer{}
		if err := json.MarshalWrite(buf, r.Body); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrEncodeRequestBody, err)
		}
		bodyReader = buf
	}

	req, err := http.NewRequestWithContext(ctx, r.Method, endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateRequest, err)
	}

	req.Header = r.Headers.Clone()
	if req.Header == nil {
		req.Header = make(http.Header)
	}

	if r.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDoRequest, err)
	}

	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != r.ExpectedStatus {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf(
			"%w: got %d, want %d, body: %s",
			ErrUnexpectedStatusCode,
			res.StatusCode,
			r.ExpectedStatus,
			body,
		)
	}

	if res.StatusCode == http.StatusNoContent {
		var result T
		return &result, nil
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecodeResponse, err)
	}

	var result T
	if len(bytes.TrimSpace(body)) == 0 {
		return &result, nil
	}

	if err := json.UnmarshalRead(bytes.NewReader(body), &result); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecodeResponse, err)
	}

	return &result, nil
}

func (c *Client) resolveURL(rawURL string) (string, error) {
	if u, err := url.Parse(rawURL); err != nil {
		return "", err
	} else if u.IsAbs() {
		return rawURL, nil
	}

	if c.baseURL == "" {
		return "", ErrMissingBaseURL
	}

	return strings.TrimSuffix(c.baseURL, "/") + "/" + strings.TrimPrefix(rawURL, "/"), nil
}
