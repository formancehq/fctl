package internal

import (
	"bytes"
	"context"
	"io"
	"net/http"

	fctl "github.com/formancehq/fctl/v3/pkg"
	sdk "github.com/formancehq/fctl/v3/pkg/client"
)

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

// PatchAPI supplies the authenticated SDK with only the merge-patch body adapter.
func PatchAPI(clients *fctl.StackClients) *sdk.FormanceV1 {
	return sdk.New(sdk.WithServerURL(clients.URI), sdk.WithClient(&patchHTTPClient{client: clients.HTTPClient})).Connectivity.V1
}
