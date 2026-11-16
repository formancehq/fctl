package demo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	fctl "github.com/formancehq/fctl/v3/pkg"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

func TestFailedCommandRendersFinalSummaryAfterLateAcknowledgement(t *testing.T) {
	for _, format := range []string{"plain", "json"} {
		t.Run(format, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			secondStarted := make(chan struct{})
			failure := errors.New("connection lost after commit")
			client := &concurrentLedger{send: func(ctx context.Context, _ operations.V2CreateTransactionRequest, ordinal int) error {
				if ordinal == 0 {
					select {
					case <-secondStarted:
						return failure
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				close(secondStarted)
				<-ctx.Done()
				return nil // An acknowledged success still counts after a sibling failed.
			}}
			c := &controller{client: func(*cobra.Command, ledger.V2SchemaDataInput) (ledgerClient, error) { return client, nil }}
			cmd := newCommand(c)
			cmd.Flags().String(fctl.OutputFlag, format, "")
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs([]string{"--type", "omnibus", "--confirm", "--rate", "100", "--concurrency", "2"})
			err := cmd.ExecuteContext(ctx)
			require.ErrorIs(t, err, failure, "rendering must retain the original write error")
			require.EqualValues(t, 1, c.store.Transactions)
			require.EqualValues(t, 1, c.store.Errors)
			require.Zero(t, c.store.Interrupted)
			if format == "json" {
				decoder := json.NewDecoder(&stdout)
				var result struct {
					Data runSummary `json:"data"`
				}
				require.NoError(t, decoder.Decode(&result))
				require.EqualValues(t, 1, result.Data.Transactions)
				require.EqualValues(t, 1, result.Data.Errors)
				require.Zero(t, result.Data.Interrupted)
				require.NotEmpty(t, result.Data.Ledger)
				var extra any
				require.ErrorIs(t, decoder.Decode(&extra), io.EOF, "only one JSON summary may be written")
			} else {
				require.Equal(t, 1, strings.Count(stdout.String(), "Ledger "))
				require.Contains(t, stdout.String(), "1 transactions, 1 errors, 0 interrupted")
			}
			require.NotContains(t, stdout.String(), "Usage:")
			require.Contains(t, stderr.String(), "Stopped:")
			client.mu.Lock()
			require.Zero(t, client.active, "summary must follow draining")
			client.mu.Unlock()
		})
	}
}
