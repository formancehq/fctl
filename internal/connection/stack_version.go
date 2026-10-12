package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/api"
)

// StackVersion reads the Cloud compatibility line from Membership. Direct
// service connections have no stack line and use their service API version.
func (s *Settings) StackVersion(ctx context.Context, cmd *cobra.Command) (string, error) {
	options, _, _, _, err := s.Resolve(cmd)
	if err != nil {
		return "", err
	}
	if options.AuthMode != "cloud" {
		return "", nil
	}
	if options.Organization == "" || options.Stack == "" {
		return "", fmt.Errorf("choose an organization and stack before selecting a service provider")
	}
	client, err := s.Client(ctx, cmd, "cloud")
	if err != nil {
		return "", err
	}
	data, err := client.Do(ctx, http.MethodGet, api.Path("organizations", options.Organization, "stacks", options.Stack), nil, nil, nil)
	if err != nil {
		return "", fmt.Errorf("discover stack compatibility line: %w", err)
	}
	var result struct {
		Data struct {
			Version      string `json:"version"`
			ID           string `json:"id"`
			Organization string `json:"organizationId"`
		} `json:"data"`
	}
	if err = json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	if result.Data.ID != options.Stack || result.Data.Organization != options.Organization || result.Data.Version == "" {
		return "", fmt.Errorf("membership returned an incomplete or mismatched stack identity")
	}
	return result.Data.Version, nil
}
