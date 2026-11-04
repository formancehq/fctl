package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"

	oidcclient "github.com/formancehq/go-libs/v4/oidc/client"

	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type ClientFactory func(*cobra.Command) (API, error)

type nonInteractiveContextKey struct{}

func WithNonInteractive(ctx context.Context) context.Context {
	return context.WithValue(ctx, nonInteractiveContextKey{}, true)
}

func IsNonInteractive(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	nonInteractive, _ := ctx.Value(nonInteractiveContextKey{}).(bool)
	return nonInteractive
}

type clientFactoryDependencies struct {
	loadAndAuthenticateCurrentProfile func(*cobra.Command) (*fctl.Config, *fctl.Profile, string, oidcclient.RelyingParty, error)
	resolveStackID                    func(*cobra.Command, fctl.Profile) (string, string, error)
	readStackToken                    func(*cobra.Command, string, string, string) (*fctl.AccessToken, error)
	newStackClientsFromFlags          func(*cobra.Command, oidcclient.RelyingParty, fctl.Dialog, string, fctl.Profile) (*fctl.StackClients, error)
}

func NewClientFactory() ClientFactory {
	return newClientFactory(clientFactoryDependencies{
		loadAndAuthenticateCurrentProfile: fctl.LoadAndAuthenticateCurrentProfile,
		resolveStackID:                    fctl.ResolveStackID,
		readStackToken:                    fctl.ReadStackToken,
		newStackClientsFromFlags:          fctl.NewStackClientsFromFlags,
	})
}

func newClientFactory(dependencies clientFactoryDependencies) ClientFactory {
	return func(cmd *cobra.Command) (API, error) {
		_, profile, profileName, relyingParty, err := dependencies.loadAndAuthenticateCurrentProfile(cmd)
		if err != nil {
			return nil, err
		}

		dialog := fctl.NewPTermDialog()
		if IsNonInteractive(cmd.Context()) {
			organizationID, stackID, err := dependencies.resolveStackID(cmd, *profile)
			if err != nil {
				return nil, err
			}
			stackToken, err := dependencies.readStackToken(cmd, profileName, organizationID, stackID)
			if err != nil {
				return nil, err
			}
			if stackToken == nil || stackToken.Expired() {
				return nil, fmt.Errorf("connectivity completion requires an existing unexpired stack token")
			}
			dialog = silentDialog{}
		}

		clients, err := dependencies.newStackClientsFromFlags(
			cmd,
			relyingParty,
			dialog,
			profileName,
			*profile,
		)
		if err != nil {
			return nil, err
		}

		if clients == nil || clients.HTTPClient == nil {
			return nil, fmt.Errorf("connectivity authenticated HTTP client is required")
		}
		return NewAPI(clients.URI, clients.HTTPClient), nil
	}
}

type silentDialog struct{}

func (silentDialog) Info(string, ...any) {}

// API is the SDK Connectivity operation surface used by commands.
type API interface {
	ConnectivityInfo(ctx context.Context, opts ...operations.Option) (*operations.ConnectivityInfoResponse, error)
	CreateConnection(ctx context.Context, request connectivity.ConnectionCreate, security *operations.CreateConnectionSecurity, opts ...operations.Option) (*operations.CreateConnectionResponse, error)
	DeleteConnection(ctx context.Context, request operations.DeleteConnectionRequest, security *operations.DeleteConnectionSecurity, opts ...operations.Option) (*operations.DeleteConnectionResponse, error)
	GetConnection(ctx context.Context, request operations.GetConnectionRequest, security *operations.GetConnectionSecurity, opts ...operations.Option) (*operations.GetConnectionResponse, error)
	GetConnector(ctx context.Context, request operations.GetConnectorRequest, security *operations.GetConnectorSecurity, opts ...operations.Option) (*operations.GetConnectorResponse, error)
	GetConnectorFacets(ctx context.Context, request operations.GetConnectorFacetsRequest, security *operations.GetConnectorFacetsSecurity, opts ...operations.Option) (*operations.GetConnectorFacetsResponse, error)
	GetConnectorVersion(ctx context.Context, request operations.GetConnectorVersionRequest, security *operations.GetConnectorVersionSecurity, opts ...operations.Option) (*operations.GetConnectorVersionResponse, error)
	GetQueryCapabilities(ctx context.Context, security *operations.GetQueryCapabilitiesSecurity, opts ...operations.Option) (*operations.GetQueryCapabilitiesResponse, error)
	ListConnections(ctx context.Context, request operations.ListConnectionsRequest, security *operations.ListConnectionsSecurity, opts ...operations.Option) (*operations.ListConnectionsResponse, error)
	ListConnectorVersions(ctx context.Context, request operations.ListConnectorVersionsRequest, security *operations.ListConnectorVersionsSecurity, opts ...operations.Option) (*operations.ListConnectorVersionsResponse, error)
	ListConnectors(ctx context.Context, request operations.ListConnectorsRequest, security *operations.ListConnectorsSecurity, opts ...operations.Option) (*operations.ListConnectorsResponse, error)
	PatchConnection(ctx context.Context, request operations.PatchConnectionRequest, security *operations.PatchConnectionSecurity, opts ...operations.Option) (*operations.PatchConnectionResponse, error)
	ReplaceConnection(ctx context.Context, request operations.ReplaceConnectionRequest, security *operations.ReplaceConnectionSecurity, opts ...operations.Option) (*operations.ReplaceConnectionResponse, error)
}

var _ API = (*sdk.FormanceV1)(nil)

const (
	ResourceConnectors        = "connectors"
	ResourceConnectorVersions = "connectorversions"
	ResourceConnections       = "connections"
	VersionAliasLatest        = "latest"
)

var VersionAliases = []string{VersionAliasLatest, "stable", "rc", "beta", "alpha"}

type ListOptions struct {
	PageSize int32
	Cursor   string
	Query    string
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]any
}

func (e *APIError) Error() string {
	return fmt.Sprintf("connectivity API error: status %d, code %s: %s", e.StatusCode, e.Code, e.Message)
}
func ResponseError(status int, apiError *connectivity.Error) error {
	if status >= 200 && status < 300 {
		return nil
	}
	e := &APIError{StatusCode: status, Message: http.StatusText(status)}
	if apiError != nil {
		e.Code = apiError.Code
		e.Message = apiError.Message
		e.Details = apiError.Details
	}
	return e
}

type patchBodyKey struct{}

// WithPatchBody preserves merge-patch nulls and empty maps omitted by the generated model.
func WithPatchBody(ctx context.Context, body []byte) context.Context {
	return context.WithValue(ctx, patchBodyKey{}, append([]byte(nil), body...))
}

type patchHTTPClient struct{ client *http.Client }

func (c *patchHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if body, ok := req.Context().Value(patchBodyKey{}).([]byte); ok && req.Method == http.MethodPatch {
		req = req.Clone(req.Context())
		if req.Body != nil {
			_ = req.Body.Close()
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	}
	return c.client.Do(req)
}

func OptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// NewAPI uses the SDK operations with an invocation-local merge-patch body override.
func NewAPI(serverURL string, client *http.Client) API {
	return sdk.New(sdk.WithServerURL(serverURL), sdk.WithClient(&patchHTTPClient{client: client})).Connectivity.V1
}
