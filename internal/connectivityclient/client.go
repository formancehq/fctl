package connectivityclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const connectivityPath = "/api/connectivity"

type client struct {
	stackURI   string
	httpClient *http.Client
}

func (e *APIError) Error() string {
	return fmt.Sprintf("connectivity API error: status %d, code %s: %s", e.StatusCode, e.Code, e.Message)
}

func (c *client) ListPlugins(ctx context.Context, options ListOptions) (*PluginList, error) {
	result := &PluginList{}
	if err := c.requestJSON(ctx, http.MethodGet, "plugins", "", listQuery(options), nil, "", http.StatusOK, result, true); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) GetPlugin(ctx context.Context, name string) (*Plugin, error) {
	result := &Plugin{}
	if err := c.requestJSON(ctx, http.MethodGet, "plugins", name, nil, nil, "", http.StatusOK, result, true); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) ListInstances(ctx context.Context, options ListOptions) (*InstanceList, error) {
	result := &InstanceList{}
	if err := c.requestJSON(ctx, http.MethodGet, "instances", "", listQuery(options), nil, "", http.StatusOK, result, true); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) CreateInstance(ctx context.Context, instance InstanceCreate) (*Instance, error) {
	result := &Instance{}
	if err := c.requestJSON(ctx, http.MethodPost, "instances", "", nil, instance, "application/json", http.StatusCreated, result, true); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) GetInstance(ctx context.Context, name string) (*Instance, error) {
	result := &Instance{}
	if err := c.requestJSON(ctx, http.MethodGet, "instances", name, nil, nil, "", http.StatusOK, result, true); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) PatchInstance(ctx context.Context, name string, patch InstancePatch) (*Instance, error) {
	result := &Instance{}
	if err := c.requestJSON(ctx, http.MethodPatch, "instances", name, nil, patch, "application/merge-patch+json", http.StatusOK, result, true); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) DeleteInstance(ctx context.Context, name string) error {
	return c.requestJSON(ctx, http.MethodDelete, "instances", name, nil, nil, "", http.StatusNoContent, nil, false)
}

func listQuery(options ListOptions) url.Values {
	query := url.Values{}
	if options.Limit != 0 {
		query.Set("limit", strconv.FormatInt(int64(options.Limit), 10))
	}
	if options.Continue != "" {
		query.Set("continue", options.Continue)
	}
	if options.Plugin != "" {
		query.Set("plugin", options.Plugin)
	}
	return query
}

func (c *client) requestJSON(ctx context.Context, method, resource, name string, query url.Values, body any, contentType string, expectedStatus int, destination any, requireObject bool) error {
	endpoint, err := endpointURL(c.stackURI, resource, name)
	if err != nil {
		return err
	}
	endpoint.RawQuery = query.Encode()

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal connectivity request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return fmt.Errorf("create connectivity request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	response, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send connectivity request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		return decodeAPIError(response)
	}
	if destination == nil {
		return nil
	}
	return decodeResponse(response.Body, destination, requireObject)
}

func endpointURL(stackURI, resource, name string) (*url.URL, error) {
	endpoint, err := url.Parse(stackURI)
	if err != nil {
		return nil, fmt.Errorf("parse stack URI: %w", err)
	}
	if endpoint.Scheme == "" || endpoint.Host == "" {
		return nil, fmt.Errorf("parse stack URI: expected absolute URI")
	}

	basePath := strings.TrimSuffix(endpoint.Path, "/")
	baseRawPath := strings.TrimSuffix(endpoint.EscapedPath(), "/")
	if baseRawPath == "" {
		baseRawPath = basePath
	}
	path := connectivityPath + "/" + resource
	rawPath := path
	if name != "" {
		path += "/" + name
		rawPath += "/" + url.PathEscape(name)
	}

	endpoint.Path = basePath + path
	endpoint.RawPath = baseRawPath + rawPath
	endpoint.RawQuery = ""
	endpoint.Fragment = ""
	return endpoint, nil
}

func decodeResponse(body io.Reader, destination any, requireObject bool) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("read connectivity response: %w", err)
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return fmt.Errorf("decode connectivity response: empty response body")
	}
	if requireObject && (bytes.Equal(trimmed, []byte("{}")) || bytes.Equal(trimmed, []byte("null"))) {
		return fmt.Errorf("decode connectivity response: empty object response")
	}
	if err := json.Unmarshal(trimmed, destination); err != nil {
		return fmt.Errorf("decode connectivity response: %w", err)
	}
	return nil
}

func decodeAPIError(response *http.Response) error {
	payload := struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	}{}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return &APIError{StatusCode: response.StatusCode, Message: response.Status}
	}
	return &APIError{
		StatusCode: response.StatusCode,
		Code:       payload.Code,
		Message:    payload.Message,
		Details:    payload.Details,
	}
}
