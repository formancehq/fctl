package connections

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pterm/pterm"
	"github.com/stretchr/testify/require"

	connectivityinternal "github.com/formancehq/fctl/v3/cmd/connectivity/internal"
	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/connectivity"
)

func requireLinesFit(t *testing.T, output string, width int) {
	t.Helper()
	for _, line := range strings.Split(pterm.RemoveColorFromString(output), "\n") {
		require.LessOrEqual(t, utf8.RuneCountInString(strings.TrimRight(line, " ")), width, "line %q", line)
	}
}

func catalogConnection() connectivity.Connection {
	instance := instanceFixture("stripeconnect-fpss")
	instance.ConnectionConfiguration.Connector = stringPtr("ee-stripeconnect")
	instance.ConnectionConfiguration.Ledger = stringPtr("stripeconnect-fpss")
	instance.ConnectionConfiguration.Version = nil
	instance.ConnectionConfiguration.Channel = fctl.Ptr(connectivity.Channel("beta"))
	instance.ConnectionStatus.AppliedVersion = stringPtr("v0.5.0-beta.14")
	return instance
}

func TestStatusSummaryShowsIngestionOnlyWhenUnexpected(t *testing.T) {
	tests := map[string]struct {
		lifecycle connectivity.Lifecycle
		state     string
		want      string
	}{
		"active running":     {lifecycle: "active", state: "running", want: "active"},
		"active error":       {lifecycle: "active", state: "error", want: "active / error"},
		"suspended paused":   {lifecycle: "suspended", state: "paused", want: "suspended"},
		"completed finished": {lifecycle: "completed", state: "finished", want: "completed"},
		"failed error":       {lifecycle: "failed", state: "error", want: "failed / error"},
		"pending no ingest":  {lifecycle: "pending", want: "pending"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			status := &connectivity.ConnectionStatus{Lifecycle: test.lifecycle}
			if test.state != "" {
				status.IngestionStatus = &connectivity.IngestionStatus{State: fctl.Ptr(connectivity.State(test.state))}
			}
			require.Equal(t, test.want, statusSummary(status))
		})
	}
	require.Empty(t, statusSummary(nil))
}

func TestListConnectionsGivenEightyColumnsThenNameStaysAndConnectorAndLedgerAreCut(t *testing.T) {
	t.Cleanup(fctl.SetTerminalWidth(80))
	client := listInstanceClientMock{list: func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error) {
		return &testConnectionList{Items: []connectivity.Connection{catalogConnection()}, PageSize: 15}, nil
	}}

	output, err := executeCommand(testNewListCommand(factoryReturning(client)))

	require.NoError(t, err)
	plain := pterm.RemoveColorFromString(output)
	require.Contains(t, plain, "stripeconnect-fpss | ee-stripe")
	require.Contains(t, plain, "| beta   | v0.5.0-beta.14 | active")
	require.Contains(t, plain, "…")
	requireLinesFit(t, output, 80)
}

func TestListConnectionsGivenNarrowTerminalThenRowsAreStacked(t *testing.T) {
	t.Cleanup(fctl.SetTerminalWidth(50))
	client := listInstanceClientMock{list: func(context.Context, connectivityinternal.ListOptions) (*testConnectionList, error) {
		return &testConnectionList{Items: []connectivity.Connection{catalogConnection()}, PageSize: 15}, nil
	}}

	output, err := executeCommand(testNewListCommand(factoryReturning(client)))

	require.NoError(t, err)
	plain := pterm.RemoveColorFromString(output)
	require.Contains(t, plain, "stripeconnect-fpss\n  Connector  ee-stripeconnect\n")
	require.Contains(t, plain, "  Status     active")
	requireLinesFit(t, output, 50)
}

func TestShowConnectionSummarizesFiles(t *testing.T) {
	instance := catalogConnection()
	instance.ConnectionConfiguration.Files = map[string]string{"/etc/modulr/client.pem": "-----BEGIN CERTIFICATE-----\nMIIB\nMIIC\n-----END CERTIFICATE-----\n"}

	output, err := executeCommand(testNewShowCommand(factoryWithInstance(instance)), instance.Name)

	require.NoError(t, err)
	plain := pterm.RemoveColorFromString(output)
	require.Contains(t, plain, "channel beta")
	require.Contains(t, plain, "-----BEGIN CERTIFICATE----- … (4 lines)")
	require.NotContains(t, plain, "MIIB")
	require.Contains(t, plain, "# Configuration (1 value)\n")
}

func TestShowConnectionGivenEightyColumnsThenLongErrorsWrap(t *testing.T) {
	t.Cleanup(fctl.SetTerminalWidth(80))
	instance := catalogConnection()
	instance.ConnectionStatus.LastError = stringPtr("plugin modulr: fetch accounts page 3: GET https://api-sandbox.modulrfinance.com/accounts?page=3&size=500: 401 Unauthorized: HMAC signature does not match the request")

	output, err := executeCommand(testNewShowCommand(factoryWithInstance(instance)), instance.Name)

	require.NoError(t, err)
	require.Contains(t, pterm.RemoveColorFromString(output), "Last Error      | (may be historical) plugin modulr:")
	requireLinesFit(t, output, 80)
}

func TestSuspendJSONIncludesTheReturnedConnection(t *testing.T) {
	client := suspensionClientMock{patch: func(_ context.Context, name string, _ map[string]any) (*connectivity.Connection, error) {
		instance := instanceFixture(name)
		instance.ConnectionConfiguration.Suspend = fctl.Ptr(true)
		return &instance, nil
	}}
	command := testNewSuspendCommand(factoryReturning(client))
	command.Flags().String(fctl.OutputFlag, "plain", "")

	output, err := executeCommand(command, "stripe-eu", "--confirm", "--output", "json")

	require.NoError(t, err)
	var envelope struct {
		Data SuspensionStore `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, "stripe-eu", envelope.Data.Name)
	require.True(t, envelope.Data.Suspend)
	require.NotNil(t, envelope.Data.Connection)
	require.True(t, *envelope.Data.Connection.ConnectionConfiguration.Suspend)
}
