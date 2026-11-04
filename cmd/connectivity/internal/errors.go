package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/sdkerrors"
)

// ResponseError converts an SDK error envelope into the SDK's error type.
func ResponseError(status int, apiError *connectivity.Error, rawResponse *http.Response) error {
	if status >= 200 && status < 300 {
		return nil
	}
	var body []byte
	if rawResponse != nil && rawResponse.Body != nil {
		var err error
		original := rawResponse.Body
		body, err = io.ReadAll(original)
		if err != nil {
			rawResponse.Body = struct {
				io.Reader
				io.Closer
			}{Reader: io.MultiReader(bytes.NewReader(body), original), Closer: original}
			return fmt.Errorf("read SDK error response: %w", err)
		}
		_ = original.Close()
		rawResponse.Body = io.NopCloser(bytes.NewReader(body))
	} else if apiError != nil {
		var err error
		body, err = json.Marshal(apiError)
		if err != nil {
			return fmt.Errorf("encode SDK error response: %w", err)
		}
	}
	return sdkerrors.NewSDKError(http.StatusText(status), status, string(body), rawResponse)
}
