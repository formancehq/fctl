// Package api adapts the public HTTP client to the core's connection layer.
package api

import (
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

type Client = httpclient.Client
type Error = httpclient.Error

func New(base string, client *http.Client) (*Client, error) { return httpclient.New(base, client) }
func Path(segments ...string) string                        { return httpclient.Path(segments...) }
func ValidateURL(value string) error                        { return httpclient.ValidateURL(value) }
func ValidateSecureURL(value string) error                  { return httpclient.ValidateSecureURL(value) }
