package clarity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	fctl "github.com/formancehq/fctl/v3/pkg"
)

const apiPrefix = "/api/reconciliation"

type Client struct {
	HTTPClient *http.Client
	BaseURL    string
}

func NewClient(cmd *cobra.Command) (*Client, error) {
	_, profile, profileName, relyingParty, err := fctl.LoadAndAuthenticateCurrentProfile(cmd)
	if err != nil {
		return nil, err
	}
	clients, err := fctl.NewStackClientsFromFlags(cmd, relyingParty, fctl.NewPTermDialog(), profileName, *profile)
	if err != nil {
		return nil, err
	}
	return &Client{HTTPClient: clients.HTTPClient, BaseURL: clients.URI}, nil
}

func (c *Client) Do(cmd *cobra.Command, method, requestPath string, query url.Values, body any, response any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	endpoint := strings.TrimRight(c.BaseURL, "/") + apiPrefix + requestPath
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(cmd.Context(), method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode >= http.StatusMultipleChoices {
		var apiError ErrorResponse
		if json.Unmarshal(responseBody, &apiError) == nil && apiError.ErrorMessage != "" {
			if apiError.Details != "" {
				return fmt.Errorf("reconciliation API returned %d (%s): %s: %s", resp.StatusCode, apiError.ErrorCode, apiError.ErrorMessage, apiError.Details)
			}
			return fmt.Errorf("reconciliation API returned %d (%s): %s", resp.StatusCode, apiError.ErrorCode, apiError.ErrorMessage)
		}
		return fmt.Errorf("reconciliation API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	if response == nil || len(responseBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(responseBody, response); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

func ResourcePath(resource string) string {
	return url.PathEscape(resource)
}
