package connectivityclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	sdk "github.com/formancehq/connectivity/pkg/client/http"
	"github.com/formancehq/connectivity/pkg/client/http/models/components"
)

const connectivityPath = "/api/connectivity"

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

// This SDK revision substitutes path parameters verbatim. Escape each input
// segment before handing it to an operation; the SDK still owns route selection.
type sdkOperation func(context.Context, *sdk.SDK) error

func listPage[T any](ctx context.Context, c *client, options ListOptions, operation sdkOperation, validate func(*T) error) (*page[T], error) {
	envelope := &page[T]{}
	if err := c.invoke(ctx, http.StatusOK, envelope, options.PageSize == 0, operation); err != nil {
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
	query, size, cursor := sdkListOptions(options)
	result, err := listPage(ctx, c, options, func(ctx context.Context, s *sdk.SDK) error {
		_, err := s.Connectors.ListConnectors(ctx, query, size, cursor)
		return err
	}, validateConnector)
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
	if err := c.invoke(ctx, http.StatusOK, result, false, func(ctx context.Context, s *sdk.SDK) error {
		_, err := s.Connectors.GetConnectorFacets(ctx, optionalString(query))
		return err
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) GetQueryCapabilities(ctx context.Context) (*QueryCapabilities, error) {
	result := &QueryCapabilities{}
	if err := c.invoke(ctx, http.StatusOK, result, false, func(ctx context.Context, s *sdk.SDK) error {
		_, err := s.Query.GetQueryCapabilities(ctx)
		return err
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) GetConnector(ctx context.Context, name string) (*Connector, error) {
	result := &Connector{}
	if err := c.invoke(ctx, http.StatusOK, result, false, func(ctx context.Context, s *sdk.SDK) error {
		_, err := s.Connectors.GetConnector(ctx, url.PathEscape(name))
		return err
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) ListConnectorVersions(ctx context.Context, connector string, options ListOptions) (*ConnectorVersionList, error) {
	query, size, cursor := sdkListOptions(options)
	result, err := listPage(ctx, c, options, func(ctx context.Context, s *sdk.SDK) error {
		_, err := s.Connectors.ListConnectorVersions(ctx, url.PathEscape(connector), query, size, cursor)
		return err
	}, validateConnectorVersionSummary)
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
	if err := c.invoke(ctx, http.StatusOK, result, false, func(ctx context.Context, s *sdk.SDK) error {
		_, err := s.Connectors.GetConnectorVersion(ctx, url.PathEscape(connector), url.PathEscape(version))
		return err
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) ListConnections(ctx context.Context, options ListOptions) (*ConnectionList, error) {
	query, size, cursor := sdkListOptions(options)
	result, err := listPage(ctx, c, options, func(ctx context.Context, s *sdk.SDK) error {
		_, err := s.Connection.ListConnections(ctx, query, size, cursor)
		return err
	}, validateConnection)
	if err != nil {
		return nil, err
	}
	return &ConnectionList{
		Items:    result.Cursor.Data,
		PageSize: result.Cursor.PageSize,
		HasMore:  result.Cursor.HasMore,
		Next:     result.Cursor.Next,
	}, nil
}

func (c *client) CreateConnection(ctx context.Context, instance ConnectionCreate) (*Connection, error) {
	if strings.TrimSpace(instance.Name) == "" {
		return nil, fmt.Errorf("connection name is required")
	}
	if err := validateConfigurationWrite(instance.Configuration); err != nil {
		return nil, err
	}
	var request components.ConnectionCreate
	if err := sdkRequest(instance, &request); err != nil {
		return nil, err
	}
	result := &Connection{}
	if err := c.invoke(ctx, http.StatusCreated, result, false, func(ctx context.Context, s *sdk.SDK) error {
		_, err := s.Connection.CreateConnection(ctx, request)
		return err
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) GetConnection(ctx context.Context, name string) (*Connection, error) {
	result := &Connection{}
	if err := c.invoke(ctx, http.StatusOK, result, false, func(ctx context.Context, s *sdk.SDK) error {
		_, err := s.Connection.GetConnection(ctx, url.PathEscape(name))
		return err
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) ReplaceConnection(ctx context.Context, name string, update ConnectionUpdate) (*Connection, error) {
	if err := validateConfigurationWrite(update.Configuration); err != nil {
		return nil, err
	}
	var request components.ConnectionUpdate
	if err := sdkRequest(update, &request); err != nil {
		return nil, err
	}
	result := &Connection{}
	if err := c.invoke(ctx, http.StatusOK, result, false, func(ctx context.Context, s *sdk.SDK) error {
		_, err := s.Connection.ReplaceConnection(ctx, url.PathEscape(name), request)
		return err
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) PatchConnection(ctx context.Context, name string, patch ConnectionPatch) (*Connection, error) {
	if err := validatePatch(patch); err != nil {
		return nil, err
	}
	var request components.ConnectionPatch
	if err := sdkRequest(patch, &request); err != nil {
		return nil, err
	}
	result := &Connection{}
	if err := c.invoke(ctx, http.StatusOK, result, false, func(ctx context.Context, s *sdk.SDK) error {
		_, err := s.Connection.PatchConnection(ctx, url.PathEscape(name), request)
		return err
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) DeleteConnection(ctx context.Context, name string) error {
	return c.invoke(ctx, http.StatusNoContent, nil, false, func(ctx context.Context, s *sdk.SDK) error {
		_, err := s.Connection.DeleteConnection(ctx, url.PathEscape(name))
		return err
	})
}

// Convert only validated input. The generated patch preserves absent/null/value
// states, including null entries that a redacted GET never returned.
func sdkRequest(input, destination any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("marshal connectivity request: %w", err)
	}
	if err := json.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("convert connectivity SDK request: %w", err)
	}
	return nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func sdkListOptions(options ListOptions) (*string, *int, *string) {
	var size *int
	if options.PageSize != 0 {
		value := int(options.PageSize)
		size = &value
	}
	return optionalString(options.Query), size, optionalString(options.Cursor)
}

// Each SDK instance and response capture belongs to one invocation. Generated
// response models erase required-field presence, so validate the original bytes
// rather than marshaling their zero values back into the facade's models.
func (c *client) invoke(ctx context.Context, expectedStatus int, destination any, omitPageSize bool, operation sdkOperation) error {
	endpoint, err := serviceURL(c.stackURI)
	if err != nil {
		return err
	}
	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	transport := &sdkHTTPClient{client: httpClient, omitPageSize: omitPageSize, expectedStatus: expectedStatus}
	s := sdk.New(sdk.WithServerURL(endpoint.String()), sdk.WithClient(transport))
	sdkErr := operation(ctx, s)
	if transport.response == nil {
		return fmt.Errorf("send connectivity request: %w", sdkErr)
	}
	if transport.readErr != nil {
		return fmt.Errorf("read connectivity response: %w", transport.readErr)
	}
	if transport.response.StatusCode != expectedStatus {
		response := *transport.response
		response.Body = io.NopCloser(bytes.NewReader(transport.body))
		return decodeAPIError(&response)
	}
	if destination == nil {
		return nil
	}
	// SDK decoding or Content-Type errors do not replace the facade's existing
	// JSON validation: the complete original body remains authoritative here.
	return decodeResponse(bytes.NewReader(transport.body), destination, true)
}

type sdkHTTPClient struct {
	client         *http.Client
	omitPageSize   bool
	expectedStatus int
	response       *http.Response
	body           []byte
	readErr        error
}

func (c *sdkHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if c.omitPageSize {
		req = req.Clone(req.Context())
		endpoint := *req.URL
		req.URL = &endpoint
		query := endpoint.Query()
		query.Del("pageSize")
		req.URL.RawQuery = query.Encode()
	}
	response, err := c.client.Do(req)
	if err != nil {
		return response, err
	}
	c.response = response
	// Close the real network body exactly once, including read failures. Give the
	// SDK an independent replay; its decoder must not consume our validation input.
	if response.StatusCode != http.StatusNoContent || c.expectedStatus != http.StatusNoContent {
		c.body, c.readErr = io.ReadAll(response.Body)
	}
	_ = response.Body.Close()
	response.Body = io.NopCloser(bytes.NewReader(c.body))
	return response, nil
}

func serviceURL(stackURI string) (*url.URL, error) {
	endpoint, err := url.Parse(stackURI)
	if err != nil {
		return nil, fmt.Errorf("parse stack URI: %w", err)
	}
	if endpoint.Scheme == "" || endpoint.Host == "" {
		return nil, fmt.Errorf("parse stack URI: expected absolute URI")
	}
	rawPath := strings.TrimSuffix(endpoint.EscapedPath(), "/") + connectivityPath
	endpoint.Path = strings.TrimSuffix(endpoint.Path, "/") + connectivityPath
	endpoint.RawPath = rawPath
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
		return validateConnectorVersion(value.Version)
	case *Connection:
		return validateConnection(value)
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
	if connector == nil || strings.TrimSpace(connector.Name) == "" {
		return fmt.Errorf("connector name is required")
	}
	return nil
}

func validateConnectorVersion(version string) error {
	if strings.TrimSpace(version) == "" {
		return fmt.Errorf("connector version version is required")
	}
	return nil
}

func validateConnectorVersionSummary(summary *ConnectorVersionSummary) error {
	return validateConnectorVersion(summary.Version)
}

func validateConnection(instance *Connection) error {
	if instance == nil || strings.TrimSpace(instance.Name) == "" {
		return fmt.Errorf("connection name is required")
	}
	if strings.TrimSpace(instance.Configuration.Ledger) == "" {
		return fmt.Errorf("connection configuration.ledger is required")
	}
	if status := instance.Status; status != nil {
		if !oneOf(status.Lifecycle, "pending", "active", "suspended", "completed", "failed", "unknown") {
			return fmt.Errorf("connection status.lifecycle is invalid")
		}
		if status.SuspendedBy == nil {
			return fmt.Errorf("connection status.suspendedBy must be a non-null array")
		}
		for _, source := range status.SuspendedBy {
			if !oneOf(source.Source, "request", "policy") {
				return fmt.Errorf("connection status.suspendedBy entry is invalid")
			}
		}
		if ingestion := status.Ingestion; ingestion != nil && ingestion.State != nil && !oneOf(*ingestion.State, "running", "error", "failed", "stopped", "paused", "finished", "unknown") {
			return fmt.Errorf("connection status.ingestion.state is invalid")
		}
		if failure := status.Error; failure != nil {
			if !oneOf(failure.Code, "configuration_invalid", "execution_failed", "service_error", "failure_unknown") {
				return fmt.Errorf("connection status.error is invalid")
			}
		}
	}
	return nil
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func validateConfigurationWrite(configuration ConnectionConfiguration) error {
	if strings.TrimSpace(configuration.Connector) == "" {
		return fmt.Errorf("configuration.connector is required on writes")
	}
	if strings.TrimSpace(configuration.Ledger) == "" {
		return fmt.Errorf("configuration.ledger is required on writes")
	}
	if (configuration.Version == nil) == (configuration.Channel == nil) {
		return fmt.Errorf("configuration requires exactly one of version and channel")
	}
	if configuration.Version != nil && strings.TrimSpace(*configuration.Version) == "" {
		return fmt.Errorf("configuration.version must not be empty")
	}
	if configuration.Channel != nil && !oneOf(*configuration.Channel, "stable", "rc", "beta", "alpha") {
		return fmt.Errorf("configuration.channel is invalid")
	}
	return nil
}

// A patch may be partial, so selector and immutable-field validation belongs to
// the service after merging. Validate the public shape without losing nulls.
func validatePatch(patch ConnectionPatch) error {
	data, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshal connectivity request: %w", err)
	}
	if patch == nil {
		return fmt.Errorf("configuration patch must be an object")
	}
	var fields struct {
		Configuration json.RawMessage `json:"configuration"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fields); err != nil {
		return fmt.Errorf("invalid connectivity patch: %w", err)
	}
	if len(fields.Configuration) == 0 {
		return nil
	}
	if bytes.Equal(bytes.TrimSpace(fields.Configuration), []byte("null")) {
		return fmt.Errorf("configuration patch must be an object")
	}
	var configuration struct {
		Connector     *string            `json:"connector"`
		Version       *string            `json:"version"`
		Channel       *string            `json:"channel"`
		Ledger        *string            `json:"ledger"`
		Suspend       *bool              `json:"suspend"`
		InitialCursor *[]byte            `json:"initialCursor"`
		PollInterval  *string            `json:"pollInterval"`
		Env           map[string]*string `json:"env"`
		Files         map[string]*string `json:"files"`
	}
	decoder = json.NewDecoder(bytes.NewReader(fields.Configuration))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&configuration); err != nil {
		return fmt.Errorf("invalid connectivity patch configuration: %w", err)
	}
	if configuration.Channel != nil && !oneOf(*configuration.Channel, "stable", "rc", "beta", "alpha") {
		return fmt.Errorf("configuration.channel is invalid")
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
