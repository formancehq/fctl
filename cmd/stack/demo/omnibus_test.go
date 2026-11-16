package demo

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
)

func TestOmnibusSchema(t *testing.T) {
	t.Parallel()
	schema, err := omnibusSchema()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"banks", "clients", "platform"}, slices.Collect(maps.Keys(schema.V2ChartOfAccounts)))
	require.ElementsMatch(t, []string{"CLIENT_DEPOSIT", "UNIDENTIFIED_DEPOSIT", "SUSPENSE_RESOLUTION", "PAYOUT_RESERVE", "PAYOUT_SETTLEMENT"}, slices.Collect(maps.Keys(schema.V2TransactionTemplates)))
	require.ElementsMatch(t, []string{"CLIENT_BALANCE", "PENDING_SUSPENSE", "INFLIGHT_PAYOUTS"}, slices.Collect(maps.Keys(schema.V2QueryTemplates)))
	for name, template := range schema.V2TransactionTemplates {
		require.NotEmpty(t, template.Script, name)
		require.NotContains(t, template.Script, "@world", name)
		require.NotNil(t, template.Runtime, name)
		require.Equal(t, ledger.RuntimeExperimentalInterpreter, *template.Runtime, name)
	}
}

func TestOmnibusConservesFundsAcrossLifecycles(t *testing.T) {
	t.Parallel()
	scenario := &omnibus{random: rand.New(rand.NewPCG(42, 99))}
	balances := make(map[string]int64)
	suspense := make(map[string]int64)
	payouts := make(map[string]int64)
	seen := make(map[string]bool)
	counts := make(map[string]int)
	const bank = "banks:demo_bank:main"
	const suspenseAccount = "platform:demo:suspense:payin"
	for i := range 2000 {
		beforeBalances := scenario.balances
		beforeSuspense := slices.Clone(scenario.suspense)
		beforePayouts := slices.Clone(scenario.payouts)
		request, commit := scenario.next("demo-ledger", "run-1")
		require.Equal(t, beforeBalances, scenario.balances, "funds must not change before acknowledgement")
		require.Equal(t, beforeSuspense, scenario.suspense)
		require.Equal(t, beforePayouts, scenario.payouts)
		require.Equal(t, uint64(i), scenario.sequence)
		require.Equal(t, "demo-ledger", request.Ledger)
		require.Equal(t, schemaVersion, *request.SchemaVersion)
		require.NotNil(t, request.IdempotencyKey)
		require.Equal(t, fmt.Sprintf("run-1-%d", i), *request.IdempotencyKey)
		require.Equal(t, *request.IdempotencyKey, *request.V2PostTransaction.Reference)
		require.False(t, seen[*request.IdempotencyKey], "transaction identities must be unique")
		seen[*request.IdempotencyKey] = true
		require.Nil(t, request.Force)
		require.Nil(t, request.V2PostTransaction.Force)
		require.Empty(t, request.V2PostTransaction.Postings)
		script := request.V2PostTransaction.Script
		template := *script.Template
		counts[template]++
		vars := script.Vars
		require.Equal(t, "EUR/2", vars["asset"])
		amount, err := strconv.ParseInt(vars["amount"], 10, 64)
		require.NoError(t, err)
		require.Positive(t, amount)
		require.LessOrEqual(t, amount, int64(10000))
		require.Equal(t, "run-1", request.V2PostTransaction.V2Metadata["demo.run"])
		require.Equal(t, "omnibus", request.V2PostTransaction.V2Metadata["demo.type"])
		require.Equal(t, template, request.V2PostTransaction.V2Metadata["demo.template"])
		client := "clients:" + vars["client_id"] + ":main"
		var source, destination string
		switch template {
		case "CLIENT_DEPOSIT":
			require.Equal(t, *request.IdempotencyKey, vars["reference"])
			source, destination = bank, client
		case "UNIDENTIFIED_DEPOSIT":
			require.Equal(t, *request.IdempotencyKey, vars["reference"])
			require.Equal(t, "demo", vars["platform_name"])
			source, destination = bank, suspenseAccount
			suspense[vars["reference"]] = amount
		case "SUSPENSE_RESOLUTION":
			require.Equal(t, "demo", vars["platform_name"])
			require.Contains(t, suspense, vars["original_reference"], "resolution must identify an unresolved deposit")
			require.Equal(t, suspense[vars["original_reference"]], amount)
			delete(suspense, vars["original_reference"])
			source, destination = suspenseAccount, client
		case "PAYOUT_RESERVE":
			require.Equal(t, *request.IdempotencyKey, vars["payout_ref"])
			require.NotContains(t, payouts, vars["payout_ref"])
			payouts[vars["payout_ref"]] = amount
			source, destination = client, "banks:demo_bank:payout:"+vars["payout_ref"]
		case "PAYOUT_SETTLEMENT":
			require.Contains(t, payouts, vars["payout_ref"], "settlement must identify a reserved payout")
			require.Equal(t, payouts[vars["payout_ref"]], amount)
			require.Equal(t, "bank-"+*request.IdempotencyKey, vars["bank_reference"])
			delete(payouts, vars["payout_ref"])
			source, destination = "banks:demo_bank:payout:"+vars["payout_ref"], bank
		default:
			t.Fatalf("unknown transaction template %q", template)
		}
		require.NotContains(t, source, "world")
		require.NotContains(t, destination, "world")
		if source != bank {
			require.GreaterOrEqual(t, balances[source], amount, "%s cannot debit unfunded account %s", template, source)
		}
		balances[source] -= amount
		balances[destination] += amount
		commit()
		require.Equal(t, uint64(i+1), scenario.sequence)
		for j, balance := range scenario.balances {
			require.Equal(t, balances[fmt.Sprintf("clients:client_%02d:main", j)], balance)
			require.GreaterOrEqual(t, balance, int64(0))
		}
		var total int64
		for address, balance := range balances {
			total += balance
			if !strings.HasPrefix(address, "banks:demo_bank:main") {
				require.GreaterOrEqual(t, balance, int64(0), address)
			}
		}
		require.Zero(t, total, "every generated movement must conserve total funds")
		var pending int64
		for _, m := range scenario.suspense {
			pending += m.amount
			require.Equal(t, suspense[m.reference], m.amount)
		}
		require.Equal(t, len(suspense), len(scenario.suspense))
		require.Equal(t, pending, balances[suspenseAccount])
		require.Equal(t, len(payouts), len(scenario.payouts))
		for _, m := range scenario.payouts {
			require.Equal(t, payouts[m.reference], m.amount)
		}
	}
	require.ElementsMatch(t, []string{"CLIENT_DEPOSIT", "UNIDENTIFIED_DEPOSIT", "SUSPENSE_RESOLUTION", "PAYOUT_RESERVE", "PAYOUT_SETTLEMENT"}, slices.Collect(maps.Keys(counts)))
}

func TestOmnibusPrioritizesFullPendingQueues(t *testing.T) {
	for _, template := range []string{"PAYOUT_SETTLEMENT", "SUSPENSE_RESOLUTION"} {
		t.Run(template, func(t *testing.T) {
			scenario := newOmnibus()
			pending := make([]movement, maxPendingMovements)
			for i := range pending {
				pending[i] = movement{client: 0, amount: 100, reference: fmt.Sprintf("pending-%d", i)}
			}
			if template == "PAYOUT_SETTLEMENT" {
				scenario.payouts = pending
			} else {
				scenario.suspense = pending
			}
			request, commit := scenario.next("demo-ledger", "test")
			require.Equal(t, template, *request.V2PostTransaction.Script.Template)
			commit()
			if template == "PAYOUT_SETTLEMENT" {
				require.Len(t, scenario.payouts, maxPendingMovements-1)
			} else {
				require.Len(t, scenario.suspense, maxPendingMovements-1)
			}
		})
	}
}
