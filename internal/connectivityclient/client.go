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

const (
	connectorsResource         = "connectors"
	connectorVersionsResource  = "versions"
	connectorInstancesResource = "connectorinstances"
	facetsSegment              = "_facets"
	querySegment               = "_query"
	capabilitiesSegment        = "capabilities"
)

type client struct {
	stackURI   string
	httpClient *http.Client
}

func (e *APIError) Error() string {
	return fmt.Sprintf("connectivity API error: status %d, code %s: %s", e.StatusCode, e.Code, e.Message)
}

type page[T any] struct {
	Cursor *struct {
		PageSize int32  `json:"pageSize"`
		HasMore  bool   `json:"hasMore"`
		Next     string `json:"next"`
		Data     []T    `json:"data"`
	} `json:"cursor"`
}

func listPage[T any](ctx context.Context, c *client, path []string, options ListOptions, validate func(*T) error) (*page[T], error) {
	envelope := &page[T]{}
	if err := c.requestJSON(ctx, http.MethodGet, path, listQuery(options), nil, "", http.StatusOK, envelope, true); err != nil {
		return nil, err
	}
	if envelope.Cursor == nil {
		return nil, fmt.Errorf("decode connectivity response: cursor must be a non-null object")
	}
	if envelope.Cursor.Data == nil {
		return nil, fmt.Errorf("decode connectivity response: cursor.data must be a non-null array")
	}
	for i := range envelope.Cursor.Data {
		if err := validate(&envelope.Cursor.Data[i]); err != nil {
			return nil, fmt.Errorf("decode connectivity response: cursor.data[%d]: %w", i, err)
		}
	}
	return envelope, nil
}

func (c *client) ListConnectors(ctx context.Context, options ListOptions) (*ConnectorList, error) {
	result, err := listPage(ctx, c, []string{connectorsResource}, options, validateConnector)
	if err != nil {
		return nil, err
	}
	return &ConnectorList{
		Items:    result.Cursor.Data,
		PageSize: result.Cursor.PageSize,
		HasMore:  result.Cursor.HasMore,
		Next:     result.Cursor.Next,
	}, nil
}

func (c *client) GetConnectorFacets(ctx context.Context, query string) (*FacetDistribution, error) {
	result := &FacetDistribution{}
	values := url.Values{}
	if query != "" {
		values.Set("query", query)
	}
	path := []string{connectorsResource, facetsSegment}
	if err := c.requestJSON(ctx, http.MethodGet, path, values, nil, "", http.StatusOK, result, true); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) GetQueryCapabilities(ctx context.Context) (*QueryCapabilities, error) {
	result := &QueryCapabilities{}
	path := []string{querySegment, capabilitiesSegment}
	if err := c.requestJSON(ctx, http.MethodGet, path, nil, nil, "", http.StatusOK, result, true); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) GetConnector(ctx context.Context, name string) (*Connector, error) {
	result := &Connector{}
	if err := c.requestJSON(ctx, http.MethodGet, []string{connectorsResource, name}, nil, nil, "", http.StatusOK, result, true); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) ListConnectorVersions(ctx context.Context, connector string, options ListOptions) (*ConnectorVersionList, error) {
	path := []string{connectorsResource, connector, connectorVersionsResource}
	result, err := listPage(ctx, c, path, options, validateConnectorVersionSummary)
	if err != nil {
		return nil, err
	}
	return &ConnectorVersionList{
		Items:    result.Cursor.Data,
		PageSize: result.Cursor.PageSize,
		HasMore:  result.Cursor.HasMore,
		Next:     result.Cursor.Next,
	}, nil
}

func (c *client) GetConnectorVersion(ctx context.Context, connector, version string) (*ConnectorVersion, error) {
	result := &ConnectorVersion{}
	path := []string{connectorsResource, connector, connectorVersionsResource, version}
	if err := c.requestJSON(ctx, http.MethodGet, path, nil, nil, "", http.StatusOK, result, true); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) ListConnectorInstances(ctx context.Context, options ListOptions) (*ConnectorInstanceList, error) {
	result, err := listPage(ctx, c, []string{connectorInstancesResource}, options, validateConnectorInstance)
	if err != nil {
		return nil, err
	}
	return &ConnectorInstanceList{
		Items:    result.Cursor.Data,
		PageSize: result.Cursor.PageSize,
		HasMore:  result.Cursor.HasMore,
		Next:     result.Cursor.Next,
	}, nil
}

func (c *client) CreateConnectorInstance(ctx context.Context, instance ConnectorInstanceCreate) (*ConnectorInstance, error) {
	result := &ConnectorInstance{}
	if err := c.requestJSON(ctx, http.MethodPost, []string{connectorInstancesResource}, nil, instance, "application/json", http.StatusCreated, result, true); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) GetConnectorInstance(ctx context.Context, name string) (*ConnectorInstance, error) {
	result := &ConnectorInstance{}
	if err := c.requestJSON(ctx, http.MethodGet, []string{connectorInstancesResource, name}, nil, nil, "", http.StatusOK, result, true); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) PatchConnectorInstance(ctx context.Context, name string, patch ConnectorInstancePatch) (*ConnectorInstance, error) {
	result := &ConnectorInstance{}
	path := []string{connectorInstancesResource, name}
	if err := c.requestJSON(ctx, http.MethodPatch, path, nil, patch, "application/merge-patch+json", http.StatusOK, result, true); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) DeleteConnectorInstance(ctx context.Context, name string) error {
	return c.requestJSON(ctx, http.MethodDelete, []string{connectorInstancesResource, name}, nil, nil, "", http.StatusNoContent, nil, false)
}

func listQuery(options ListOptions) url.Values {
	query := url.Values{}
	if options.PageSize != 0 {
		query.Set("pageSize", strconv.FormatInt(int64(options.PageSize), 10))
	}
	if options.Cursor != "" {
		query.Set("cursor", options.Cursor)
	}
	if options.Query != "" {
		query.Set("query", options.Query)
	}
	return query
}

func (c *client) requestJSON(ctx context.Context, method string, path []string, query url.Values, body any, contentType string, expectedStatus int, destination any, requireObject bool) error {
	endpoint, err := endpointURL(c.stackURI, path)
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

func endpointURL(stackURI string, segments []string) (*url.URL, error) {
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
	path := connectivityPath
	rawPath := connectivityPath
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		path += "/" + segment
		rawPath += "/" + url.PathEscape(segment)
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
	if requireObject {
		var root any
		if err := json.Unmarshal(trimmed, &root); err != nil {
			return fmt.Errorf("decode connectivity response: %w", err)
		}
		if _, ok := root.(map[string]any); !ok {
			return fmt.Errorf("decode connectivity response: root must be an object")
		}
	}
	if err := json.Unmarshal(trimmed, destination); err != nil {
		return fmt.Errorf("decode connectivity response: %w", err)
	}
	if err := validateResponse(destination); err != nil {
		return fmt.Errorf("decode connectivity response: %w", err)
	}
	return nil
}

func validateResponse(destination any) error {
	switch value := destination.(type) {
	case *Connector:
		return validateConnector(value)
	case *ConnectorVersion:
		return validateConnectorVersion(value.Version, value.Image)
	case *ConnectorInstance:
		return validateConnectorInstance(value)
	case *FacetDistribution:
		if value.Facets == nil {
			return fmt.Errorf("facets must be a non-null object")
		}
	case *QueryCapabilities:
		if value.Resources == nil {
			return fmt.Errorf("resources must be a non-null object")
		}
	}
	return nil
}

func validateConnector(connector *Connector) error {
	if connector == nil || connector.Metadata.Name == nil || strings.TrimSpace(*connector.Metadata.Name) == "" {
		return fmt.Errorf("connector metadata.name is required")
	}
	return nil
}

func validateConnectorVersion(version, image string) error {
	if strings.TrimSpace(version) == "" {
		return fmt.Errorf("connector version version is required")
	}
	if strings.TrimSpace(image) == "" {
		return fmt.Errorf("connector version image is required")
	}
	return nil
}

func validateConnectorVersionSummary(summary *ConnectorVersionSummary) error {
	return validateConnectorVersion(summary.Version, summary.Image)
}

func validateConnectorInstance(instance *ConnectorInstance) error {
	if instance == nil || instance.Metadata.Name == nil || strings.TrimSpace(*instance.Metadata.Name) == "" {
		return fmt.Errorf("connector instance metadata.name is required")
	}
	if strings.TrimSpace(instance.Spec.Connector) == "" {
		return fmt.Errorf("connector instance spec.connector is required")
	}
	if strings.TrimSpace(instance.Spec.Ledger) == "" {
		return fmt.Errorf("connector instance spec.ledger is required")
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
