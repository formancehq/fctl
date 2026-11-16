package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
)

// Ledger v2 wraps _info in data; v3 returns the version at the top level.
// Detect the contract before any mutation, rather than retrying a write against
// another API after an ambiguous response.
func detectLedgerMajor(ctx context.Context, client *http.Client, uri string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(uri, "/")+"/api/ledger/_info", nil)
	if err != nil {
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("read Ledger version: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("read Ledger version: HTTP %d", response.StatusCode)
	}
	var info struct {
		Version string `json:"version"`
		Data    struct {
			Version string `json:"version"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&info); err != nil {
		return "", fmt.Errorf("read Ledger version: %w", err)
	}
	version := info.Version
	if version == "" {
		version = info.Data.Version
	}
	major := semver.Major("v" + strings.TrimPrefix(version, "v"))
	if major != "v2" && major != "v3" {
		return "", fmt.Errorf("unsupported Ledger version %q: demo requires Ledger v2 or v3", version)
	}
	return major, nil
}

func demoLedgerClient(ctx context.Context, clients *fctl.StackClients, schema ledger.V2SchemaDataInput) (ledgerClient, error) {
	major, err := detectLedgerMajor(ctx, clients.HTTPClient, clients.URI)
	if err != nil {
		return nil, err
	}
	if major == "v3" {
		return &v3LedgerClient{client: clients.SDK.Ledger.V3, schema: schema}, nil
	}
	return clients.SDK.Ledger.V2, nil
}
