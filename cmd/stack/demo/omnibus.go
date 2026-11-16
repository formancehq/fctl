package demo

import (
	"fmt"
	"math/rand/v2"
	"strconv"

	"github.com/formancehq/go-libs/v4/pointer"

	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

const schemaVersion = "v1"

const maxPendingMovements = 100

type movement struct {
	client    int
	amount    int64
	reference string
}

// State only advances after the server acknowledges a transaction. Each pending
// movement retains its identity and amount until it is resolved or settled.
type omnibus struct {
	namespace string
	balances  [20]int64
	suspense  []movement
	payouts   []movement
	sequence  uint64
	random    *rand.Rand
}

func newOmnibus() *omnibus {
	return &omnibus{random: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))}
}

func (o *omnibus) next(ledgerName, runID string) (operations.V2CreateTransactionRequest, func()) {
	reference := transactionReference(runID, o.namespace, o.sequence)
	client := o.random.IntN(len(o.balances))
	amount := int64(100 + o.random.IntN(9901))
	vars := map[string]string{"asset": "EUR/2", "amount": strconv.FormatInt(amount, 10)}
	template := "CLIENT_DEPOSIT"
	commit := func() { o.balances[client] += amount }
	choice := o.random.IntN(100)
	switch {
	case len(o.payouts) >= maxPendingMovements || len(o.payouts) > 0 && choice < 25:
		pending := o.payouts[0]
		template = "PAYOUT_SETTLEMENT"
		vars["amount"] = strconv.FormatInt(pending.amount, 10)
		vars["bank_id"] = o.accountSegment("demo_bank")
		vars["payout_ref"] = pending.reference
		vars["bank_reference"] = "bank-" + reference
		commit = func() { o.payouts = o.payouts[1:] }
	case len(o.suspense) >= maxPendingMovements || len(o.suspense) > 0 && choice < 50:
		pending := o.suspense[0]
		template = "SUSPENSE_RESOLUTION"
		vars["amount"] = strconv.FormatInt(pending.amount, 10)
		vars["platform_name"] = o.accountSegment("demo")
		vars["client_id"] = o.accountSegment(fmt.Sprintf("client_%02d", pending.client))
		vars["original_reference"] = pending.reference
		commit = func() { o.balances[pending.client] += pending.amount; o.suspense = o.suspense[1:] }
	case o.balances[client] > 0 && (choice < 75 || o.balances[client] > 1000000):
		template = "PAYOUT_RESERVE"
		amount = min(amount, o.balances[client])
		vars["amount"] = strconv.FormatInt(amount, 10)
		vars["client_id"] = o.accountSegment(fmt.Sprintf("client_%02d", client))
		vars["bank_id"] = o.accountSegment("demo_bank")
		vars["payout_ref"] = reference
		commit = func() {
			o.balances[client] -= amount
			o.payouts = append(o.payouts, movement{client, amount, reference})
		}
	case choice < 85:
		template = "UNIDENTIFIED_DEPOSIT"
		vars["bank_id"] = o.accountSegment("demo_bank")
		vars["platform_name"] = o.accountSegment("demo")
		vars["reference"] = reference
		commit = func() { o.suspense = append(o.suspense, movement{client, amount, reference}) }
	default:
		vars["bank_id"] = o.accountSegment("demo_bank")
		vars["client_id"] = o.accountSegment(fmt.Sprintf("client_%02d", client))
		vars["reference"] = reference
	}
	return operations.V2CreateTransactionRequest{
		Ledger:         ledgerName,
		SchemaVersion:  pointer.For(schemaVersion),
		IdempotencyKey: pointer.For(reference),
		V2PostTransaction: ledger.V2PostTransaction{
			Reference:  pointer.For(reference),
			V2Metadata: map[string]string{"demo.run": runID, "demo.type": "omnibus", "demo.template": template},
			Script:     &ledger.V2PostTransactionScript{Template: pointer.For(template), Vars: vars},
		},
	}, func() { commit(); o.sequence++ }
}

// Independent streams never share spendable accounts. Within a stream only one
// request is in flight, so dependencies always follow acknowledged writes.
func (o *omnibus) accountSegment(name string) string { return accountSegment(o.namespace, name) }
