// Package api implements the shared HTTP boundary for service modules.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// Client targets one service endpoint, including any gateway prefix.
type Client struct {
	base     string
	http     *http.Client
	metadata map[string]string
}

// Error keeps a structured service failure, including partial bulk results.
type Error struct {
	StatusCode int
	Code       string
	Message    string
	Body       json.RawMessage
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("HTTP %d (%s): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, http.StatusText(e.StatusCode))
}

func (c *Client) Endpoint() string         { return c.base }
func (c *Client) HTTPClient() *http.Client { return c.http }

// WithContext returns a client copy with non-secret host metadata. Values must
// never contain authentication material. Metadata is not sent as HTTP headers.
func (c *Client) WithContext(values map[string]string) *Client {
	copyClient := *c
	copyClient.metadata = maps.Clone(values)
	return &copyClient
}

// Context returns a defensive copy of the host metadata, or nil when unset.
func (c *Client) Context() map[string]string { return maps.Clone(c.metadata) }

func New(base string, client *http.Client) (*Client, error) {
	if err := ValidateURL(base); err != nil {
		return nil, err
	}
	return &Client{base: strings.TrimRight(base, "/"), http: client}, nil
}

// ValidateURL rejects ambiguous endpoints and credentials embedded in URLs.
func ValidateURL(value string) error {
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("endpoint must be an absolute HTTP(S) URL without credentials, query or fragment")
	}
	return nil
}

// ValidateSecureURL permits plaintext authentication only on loopback.
func ValidateSecureURL(value string) error {
	if err := ValidateURL(value); err != nil {
		return err
	}
	u, err := url.Parse(value)
	if err != nil {
		return err
	}
	if u.Scheme == "https" {
		return nil
	}
	ip := net.ParseIP(u.Hostname())
	if u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback()) {
		return nil
	}
	return fmt.Errorf("authenticated endpoints require HTTPS except on loopback")
}

// Path encodes each user-provided identifier as exactly one path segment.
func Path(segments ...string) string {
	encoded := make([]string, len(segments))
	for i, segment := range segments {
		encoded[i] = url.PathEscape(segment)
	}
	return "/" + strings.Join(encoded, "/")
}

// Do preserves JSON numbers and performs each operation exactly once.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body json.RawMessage, headers http.Header) (result json.RawMessage, err error) {
	u := c.base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build service request: %w", err)
	}
	req.Header = headers.Clone()
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("service request failed: %w", err)
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20+1))
	if err != nil {
		return nil, fmt.Errorf("read service response: %w", err)
	}
	if len(data) > 32<<20 {
		return nil, fmt.Errorf("service response exceeds 32 MiB; reduce page size")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var details struct {
			Code    string `json:"errorCode"`
			Message string `json:"errorMessage"`
		}
		failure := &Error{StatusCode: resp.StatusCode}
		if json.Unmarshal(data, &details) == nil {
			failure.Code, failure.Message, failure.Body = details.Code, details.Message, data
		}
		return nil, failure
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return json.RawMessage("null"), nil
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("service returned invalid JSON")
	}
	return data, nil
}
