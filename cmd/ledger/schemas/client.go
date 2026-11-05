package schemas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"time"

	fctl "github.com/formancehq/fctl/v3/pkg"
	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

// QueryTemplate keeps params opaque because legacy SDK union members silently
// drop time bounds when params includes a redundant resource field.
type QueryTemplate struct {
	Description *string                              `json:"description,omitempty"`
	Resource    *ledger.V2QueryResource              `json:"resource,omitempty"`
	Params      json.RawMessage                      `json:"params,omitempty"`
	Vars        map[string]ledger.V2QueryTemplateVar `json:"vars,omitempty"`
	Body        map[string]any                       `json:"body,omitempty"`
}

type SchemaData struct {
	Chart        map[string]ledger.V2ChartSegment        `json:"chart"`
	Transactions map[string]ledger.V2TransactionTemplate `json:"transactions,omitempty"`
	Queries      map[string]QueryTemplate                `json:"queries,omitempty"`
}

type Schema struct {
	Version      string                                  `json:"version"`
	CreatedAt    time.Time                               `json:"createdAt"`
	Chart        map[string]ledger.V2ChartSegment        `json:"chart"`
	Transactions map[string]ledger.V2TransactionTemplate `json:"transactions,omitempty"`
	Queries      map[string]QueryTemplate                `json:"queries,omitempty"`
}

type schemaCursor struct {
	Data     []Schema `json:"data"`
	HasMore  bool     `json:"hasMore"`
	Previous *string  `json:"previous,omitempty"`
	Next     *string  `json:"next,omitempty"`
	PageSize int64    `json:"pageSize"`
}

func getSchema(ctx context.Context, clients *fctl.StackClients, request operations.V2GetSchemaRequest) (*Schema, error) {
	var response struct {
		Data *Schema `json:"data"`
	}
	err := invokeSchema(clients, http.StatusOK, nil, &response, func(sdk *formance.Formance) error {
		_, err := sdk.Ledger.V2.GetSchema(ctx, request)
		return err
	})
	if err != nil {
		return nil, err
	}
	if response.Data == nil {
		return nil, fmt.Errorf("decode schema response: data must be an object")
	}
	return response.Data, nil
}

func listSchemas(ctx context.Context, clients *fctl.StackClients, request operations.V2ListSchemasRequest) (*schemaCursor, error) {
	var response struct {
		Cursor *schemaCursor `json:"cursor"`
	}
	err := invokeSchema(clients, http.StatusOK, nil, &response, func(sdk *formance.Formance) error {
		_, err := sdk.Ledger.V2.ListSchemas(ctx, request)
		return err
	})
	if err != nil {
		return nil, err
	}
	if response.Cursor == nil || response.Cursor.Data == nil {
		return nil, fmt.Errorf("decode schema response: cursor.data must be an array")
	}
	return response.Cursor, nil
}

func insertSchema(ctx context.Context, clients *fctl.StackClients, request operations.V2InsertSchemaRequest, data *SchemaData) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encode schema: %w", err)
	}
	return invokeSchema(clients, http.StatusNoContent, body, nil, func(sdk *formance.Formance) error {
		_, err := sdk.Ledger.V2.InsertSchema(ctx, request)
		return err
	})
}

// Generated operations still own routes, HTTP methods and error normalization.
// The invocation-local transport preserves the original schema JSON at the
// boundary where the generated params union cannot represent the server shape.
func invokeSchema(clients *fctl.StackClients, expectedStatus int, requestBody []byte, destination any, operation func(*formance.Formance) error) error {
	transport := &schemaTransport{client: clients.HTTPClient, requestBody: requestBody}
	sdk := formance.New(formance.WithServerURL(clients.URI), formance.WithClient(transport))
	sdkErr := operation(sdk)
	if transport.readErr != nil {
		return fmt.Errorf("read schema response: %w", transport.readErr)
	}
	if transport.response == nil || transport.response.StatusCode != expectedStatus {
		if sdkErr != nil {
			return sdkErr
		}
		return fmt.Errorf("unexpected response while accessing ledger schema")
	}
	if destination == nil {
		return sdkErr
	}
	contentType, _, _ := mime.ParseMediaType(transport.response.Header.Get("Content-Type"))
	if contentType != "application/json" {
		if sdkErr != nil {
			return sdkErr
		}
		return fmt.Errorf("unexpected schema response content type %q", contentType)
	}
	if err := json.Unmarshal(transport.body, destination); err != nil {
		return fmt.Errorf("decode schema response: %w", err)
	}
	return nil
}

type schemaTransport struct {
	client      *http.Client
	requestBody []byte
	response    *http.Response
	body        []byte
	readErr     error
}

func (t *schemaTransport) Do(request *http.Request) (*http.Response, error) {
	if t.requestBody != nil {
		request = request.Clone(request.Context())
		if request.Body != nil {
			_ = request.Body.Close()
		}
		request.Body = io.NopCloser(bytes.NewReader(t.requestBody))
		request.ContentLength = int64(len(t.requestBody))
		request.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(t.requestBody)), nil
		}
	}
	response, err := t.client.Do(request)
	if err != nil {
		return response, err
	}
	t.response = response
	t.body, t.readErr = io.ReadAll(response.Body)
	_ = response.Body.Close()
	response.Body = io.NopCloser(bytes.NewReader(t.body))
	return response, nil
}

func (q *QueryTemplate) UnmarshalJSON(data []byte) error {
	type plain QueryTemplate
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(value.Params) != 0 {
		var params map[string]json.RawMessage
		if err := json.Unmarshal(value.Params, &params); err != nil {
			return fmt.Errorf("query params must be an object: %w", err)
		}
	}
	*q = QueryTemplate(value)
	return nil
}

func (s *Schema) UnmarshalJSON(data []byte) error {
	type plain Schema
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, name := range []string{"version", "createdAt", "chart"} {
		if raw := fields[name]; len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fmt.Errorf("schema %s is required", name)
		}
	}
	*s = Schema(value)
	return nil
}
