package internal

import (
	"fmt"
	"net/http"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

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
