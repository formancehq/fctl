package plugin

import (
	"context"
	"fmt"
	"net/http"

	formance "github.com/formancehq/formance-sdk-go/v3"
)

// ServiceVersions maps service names to their versions (e.g. "ledger" -> "3.1.0").
type ServiceVersions map[string]string

// DetectServiceVersions calls the stack's /versions endpoint and returns a map
// of service name to version string. Works for both Cloud and direct stack profiles
// since both expose the same endpoint.
func DetectServiceVersions(ctx context.Context, stackClient *formance.Formance) (ServiceVersions, error) {
	resp, err := stackClient.GetVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to call /versions: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from /versions", resp.StatusCode)
	}

	if resp.GetVersionsResponse == nil {
		return nil, fmt.Errorf("/versions returned no data")
	}

	versions := make(ServiceVersions)
	for _, v := range resp.GetVersionsResponse.GetVersions() {
		versions[v.GetName()] = v.GetVersion()
	}

	return versions, nil
}

// DetectServiceVersion returns the version of a specific service on the stack.
func DetectServiceVersion(ctx context.Context, stackClient *formance.Formance, serviceName string) (string, error) {
	versions, err := DetectServiceVersions(ctx, stackClient)
	if err != nil {
		return "", err
	}

	version, ok := versions[serviceName]
	if !ok {
		return "", fmt.Errorf("service %q not found in /versions response", serviceName)
	}

	return version, nil
}
