package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

type VersionsClient interface {
	GetVersions(ctx context.Context) ([]capabilities.ComponentVersion, error)
}

type HTTPVersionsClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func (c HTTPVersionsClient) GetVersions(ctx context.Context) ([]capabilities.ComponentVersion, error) {
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	endpoint, err := url.JoinPath(c.BaseURL, "versions")
	if err != nil {
		return nil, fmt.Errorf("build versions url: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	rsp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode < 200 || rsp.StatusCode >= 300 {
		return nil, fmt.Errorf("get versions failed: status %d", rsp.StatusCode)
	}

	var response versionsResponse
	if err := json.NewDecoder(rsp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode versions response: %w", err)
	}

	versions := make([]capabilities.ComponentVersion, 0, len(response.Versions))
	for _, version := range response.Versions {
		versions = append(versions, capabilities.ComponentVersion{
			Product: capabilities.Product(version.Name),
			Version: version.Version,
			Health:  version.Health,
		})
	}
	return versions, nil
}

type versionsResponse struct {
	Versions []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Health  bool   `json:"health"`
	} `json:"versions"`
}

func componentVersionFor(versions []capabilities.ComponentVersion, product capabilities.Product) (capabilities.ComponentVersion, bool) {
	for _, version := range versions {
		if version.Product == product {
			return version, true
		}
	}
	return capabilities.ComponentVersion{}, false
}
