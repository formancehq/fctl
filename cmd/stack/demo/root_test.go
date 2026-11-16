package demo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

type fakeLedger struct {
	creates      int
	schemas      int
	requests     []operations.V2CreateTransactionRequest
	send         func(context.Context, operations.V2CreateTransactionRequest) error
	createStatus int
	schemaErr    error
}

func (f *fakeLedger) CreateLedger(_ context.Context, r operations.V2CreateLedgerRequest, _ ...operations.Option) (*operations.V2CreateLedgerResponse, error) {
	f.creates++
	if r.Ledger == "" {
		return nil, fmt.Errorf("empty ledger")
	}
	status := f.createStatus
	if status == 0 {
		status = 204
	}
	return &operations.V2CreateLedgerResponse{StatusCode: status}, nil
}
func (f *fakeLedger) InsertSchema(_ context.Context, r operations.V2InsertSchemaRequest, _ ...operations.Option) (*operations.V2InsertSchemaResponse, error) {
	f.schemas++
	if len(r.V2SchemaData.V2TransactionTemplates) != 5 {
		return nil, fmt.Errorf("missing templates")
	}
	return &operations.V2InsertSchemaResponse{StatusCode: 204}, f.schemaErr
}
func (f *fakeLedger) CreateTransaction(ctx context.Context, r operations.V2CreateTransactionRequest, _ ...operations.Option) (*operations.V2CreateTransactionResponse, error) {
	f.requests = append(f.requests, r)
	if f.send != nil {
		if err := f.send(ctx, r); err != nil {
			return nil, err
		}
	}
	return &operations.V2CreateTransactionResponse{StatusCode: 200}, nil
}

func TestListAndInvalidOptionsDoNotAuthenticate(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--list"}, ""},
		{nil, "--type is required"},
		{[]string{"--type", "unknown"}, "unknown demo type"},
		{[]string{"--type", "omnibus", "--rate", "0"}, "--rate"},
		{[]string{"--type", "omnibus", "--rate", "-1"}, "--rate"},
		{[]string{"--type", "omnibus", "--rate", "NaN"}, "--rate"},
		{[]string{"--type", "omnibus", "--rate", "+Inf"}, "--rate"},
		{[]string{"--type", "omnibus", "--rate", "10001"}, "--rate"},
		{[]string{"--type", "omnibus", "--rate", "1e-20"}, "--rate"},
		{[]string{"--type", "omnibus", "--duration", "-1s"}, "--duration"},
		{[]string{"--type", "omnibus", "--concurrency", "-1"}, "--concurrency"},
		{[]string{"--type", "omnibus", "--concurrency", "1025"}, "--concurrency"},
	} {
		t.Run(fmt.Sprint(tc.args), func(t *testing.T) {
			cmd := NewCommand()
			cmd.SetContext(t.Context())
			require.NoError(t, cmd.ParseFlags(tc.args))
			c := &controller{client: func(*cobra.Command, ledger.V2SchemaDataInput) (ledgerClient, error) {
				t.Fatal("unexpected authentication")
				return nil, nil
			}}
			_, err := c.Run(cmd, nil)
			if tc.want == "" {
				require.NoError(t, err)
				require.Len(t, c.store.Recipes, len(recipeNames()))
				require.Nil(t, c.store.runSummary)
				for i, recipe := range c.store.Recipes {
					require.Equal(t, recipeNames()[i], recipe.Name)
					require.NotEmpty(t, recipe.Description)
				}
			} else {
				require.ErrorContains(t, err, tc.want)
			}
		})
	}
}

func TestGenerateRateAndDuration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := &fakeLedger{}
		c := &controller{recipeName: "omnibus", store: newRunStore("demo-test", "test")}
		schema, err := omnibusSchema()
		require.NoError(t, err)
		var progress bytes.Buffer
		require.NoError(t, c.generate(t.Context(), client, schema, 2, 2250*time.Millisecond, &progress))
		require.Equal(t, 1, client.creates)
		require.Equal(t, 1, client.schemas)
		require.Len(t, client.requests, 5)
		require.EqualValues(t, 5, c.store.Transactions)
		require.Equal(t, 2.25, c.store.ElapsedSeconds)
		for i, r := range client.requests {
			require.Equal(t, fmt.Sprintf("test-%d", i), *r.IdempotencyKey)
			require.Equal(t, "demo-test", r.Ledger)
			require.Equal(t, schemaVersion, *r.SchemaVersion)
		}
		require.Contains(t, progress.String(), "data retained")
	})
}

func TestGenerateDoesNotCatchUpAfterSlowRequest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var starts []time.Time
		client := &fakeLedger{send: func(ctx context.Context, _ operations.V2CreateTransactionRequest) error {
			starts = append(starts, time.Now())
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(1500 * time.Millisecond):
				return nil
			}
		}}
		c := &controller{recipeName: "omnibus", store: newRunStore("demo-test", "test")}
		schema, err := omnibusSchema()
		require.NoError(t, err)
		require.NoError(t, c.generate(t.Context(), client, schema, 1, 4*time.Second, &bytes.Buffer{}))
		require.Len(t, starts, 3)
		require.Equal(t, 1500*time.Millisecond, starts[1].Sub(starts[0]))
		require.EqualValues(t, 2, c.store.Transactions)
	})
}

func TestGenerateStopsOnWriteFailure(t *testing.T) {
	failure := errors.New("connection lost after commit")
	client := &fakeLedger{send: func(context.Context, operations.V2CreateTransactionRequest) error { return failure }}
	c := &controller{recipeName: "omnibus", store: newRunStore("demo-test", "test")}
	schema, err := omnibusSchema()
	require.NoError(t, err)
	var progress bytes.Buffer
	err = c.generate(t.Context(), client, schema, 1, 0, &progress)
	require.ErrorIs(t, err, failure)
	require.ErrorContains(t, err, "test-0")
	require.Len(t, client.requests, 1)
	require.Zero(t, c.store.Transactions)
	require.EqualValues(t, 1, c.store.Errors)
	require.Contains(t, progress.String(), "Stopped:")
}

func TestGenerateCancelsInflightRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	client := &fakeLedger{send: func(ctx context.Context, _ operations.V2CreateTransactionRequest) error {
		cancel()
		<-ctx.Done()
		return ctx.Err()
	}}
	c := &controller{recipeName: "omnibus", store: newRunStore("demo-test", "test")}
	schema, err := omnibusSchema()
	require.NoError(t, err)
	var progress bytes.Buffer
	require.NoError(t, c.generate(ctx, client, schema, 1, 0, &progress))
	require.Len(t, client.requests, 1)
	require.Zero(t, c.store.Transactions)
	require.Contains(t, progress.String(), "commit status is unknown")
	require.EqualValues(t, 1, c.store.Interrupted)
}

func TestSetupFailurePreventsTransactions(t *testing.T) {
	for _, tc := range []struct {
		name    string
		client  *fakeLedger
		schemas int
	}{
		{"create", &fakeLedger{createStatus: 500}, 0},
		{"schema", &fakeLedger{schemaErr: errors.New("unsupported schema")}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &controller{recipeName: "omnibus", store: newRunStore("demo-test", "test")}
			schema, err := omnibusSchema()
			require.NoError(t, err)
			require.Error(t, c.generate(t.Context(), tc.client, schema, 1, 0, &bytes.Buffer{}))
			require.Equal(t, tc.schemas, tc.client.schemas)
			require.Empty(t, tc.client.requests)
		})
	}
}

func TestDecliningApprovalDoesNotWrite(t *testing.T) {
	cmd := NewCommand()
	cmd.SetContext(fctl.WithApprovalPrompt(t.Context(), func(string) (string, error) { return "n", nil }))
	require.NoError(t, cmd.ParseFlags([]string{"--type", "omnibus"}))
	client := &fakeLedger{}
	c := &controller{client: func(*cobra.Command, ledger.V2SchemaDataInput) (ledgerClient, error) { return client, nil }}
	_, err := c.Run(cmd, nil)
	require.ErrorIs(t, err, fctl.ErrMissingApproval)
	require.Zero(t, client.creates)
	require.Zero(t, client.schemas)
	require.Empty(t, client.requests)
}

func TestDemoAcceptsRateUpToTenThousand(t *testing.T) {
	for _, rate := range []string{"101", "10000"} {
		t.Run(rate, func(t *testing.T) {
			cmd := NewCommand()
			cmd.SetContext(t.Context())
			require.NoError(t, cmd.ParseFlags([]string{"--type", "omnibus", "--rate", rate}))
			authenticated := errors.New("validation passed")
			c := &controller{client: func(*cobra.Command, ledger.V2SchemaDataInput) (ledgerClient, error) { return nil, authenticated }}
			_, err := c.Run(cmd, nil)
			require.ErrorIs(t, err, authenticated)
		})
	}
}
