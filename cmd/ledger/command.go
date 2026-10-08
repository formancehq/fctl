// Package ledger exposes the Ledger v3 data-plane HTTP API.
// Routes follow release/v3.0 at 71b0feb549a7cbc22bddaca2859ae5d91a24d378.
package ledger

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/api"
	"github.com/formancehq/fctl/v4/internal/command"
)

// NewCommand builds an independently embeddable Ledger module. The runtime
// resolves the Ledger endpoint and authentication; this module owns v3 routes.
func NewCommand(r command.Runtime) *cobra.Command {
	m := module{runtime: r}
	root := &cobra.Command{
		Use: "ledger", Short: "Use the Ledger v3 data-plane API",
		Long:    "Use the Ledger v3 data-plane API. Nested commands select a ledger with --ledger.\nList commands return one complete JSON page, including continuation tokens.\nPass the next or previous token to --cursor with the same filters and order.",
		Example: "  fctl ledger create books\n  fctl ledger --ledger books accounts list --filter 'address ^= \"users:\"'\n  fctl ledger --ledger books transactions create --data @transaction.json --idempotency-key payment-42",
	}
	root.PersistentFlags().StringVar(&m.ledger, "ledger", "", "Ledger name for nested commands")
	root.PersistentFlags().StringVar(&m.consistency, "consistency", "", "Read consistency: linearizable or stale (server default: linearizable)")
	root.AddCommand(
		m.endpoint(operation{use: "list", short: "List ledgers", global: true, path: func([]string) []string { return []string{"v3", ""} }, page: true, reverse: true}),
		m.endpoint(operation{use: "create [name]", short: "Create a ledger", ledgerArg: true, method: http.MethodPost, body: bodyDefault, idempotency: true}),
		m.endpoint(operation{use: "show [name]", short: "Show a ledger", ledgerArg: true}),
		m.endpoint(operation{use: "delete [name]", short: "Delete a ledger", ledgerArg: true, method: http.MethodDelete, idempotency: true}),
		m.endpoint(operation{use: "stats", short: "Show ledger statistics", path: fixed("stats")}),
		m.endpoint(operation{use: "balances", short: "Aggregate ledger balances and volumes", path: fixed("volumes"), filter: true,
			boolQuery: map[string]string{"use-max-precision": "useMaxPrecision", "collapse-colors": "collapseColors"}, stringQuery: map[string]string{"group-by-prefixes": "groupByPrefixes"}}),
		m.endpoint(operation{use: "info", short: "Show Ledger server version information", global: true, path: func([]string) []string { return []string{"_info"} }}),
		m.accounts(), m.transactions(), m.metadata(nil, false), m.indexes(),
	)
	logs := &cobra.Command{Use: "logs", Short: "Read ledger logs (requires the LOG index)"}
	logs.AddCommand(m.endpoint(operation{use: "list", short: "List one page of ledger logs", path: fixed("logs"), page: true, reverse: true, filter: true, dates: true}))
	root.AddCommand(logs)
	bulk := m.endpoint(operation{use: "bulk", short: "Submit v3 bulk operations once", method: http.MethodPost, path: fixed("bulk"), body: bodyRequired, idempotency: true, bulk: true,
		boolQuery: map[string]string{"atomic": "atomic", "continue-on-failure": "continueOnFailure"}})
	bulk.Long = "Submit a JSON array of v3 bulk operations once. With --atomic, --idempotency-key identifies the whole batch. Otherwise each element's ik identifies its operation and the header is ignored. Inspect every returned element for business failures, including when --continue-on-failure is enabled. This is not a log import or backup restore."
	root.AddCommand(bulk)
	return root
}

func (m *module) accounts() *cobra.Command {
	group := &cobra.Command{Use: "accounts", Short: "Read accounts and balances, update metadata"}
	group.AddCommand(
		m.endpoint(operation{use: "list", short: "List accounts", path: fixed("accounts"), page: true, reverse: true, filter: true}),
		m.endpoint(operation{use: "show <address>", short: "Show an account including volumes and metadata", args: 1, path: resource("accounts"), boolQuery: map[string]string{"collapse-colors": "collapseColors"}}),
		m.endpoint(operation{use: "balances <address>", short: "Show account balances in the full account envelope", args: 1, path: resource("accounts"), boolQuery: map[string]string{"collapse-colors": "collapseColors"}}),
		m.metadata([]string{"accounts"}, false),
	)
	return group
}

func (m *module) transactions() *cobra.Command {
	group := &cobra.Command{Use: "transactions", Short: "Read, create and revert transactions"}
	group.AddCommand(
		m.endpoint(operation{use: "list", short: "List transactions (newest first by default)", path: fixed("transactions"), page: true, reverse: true, filter: true, dates: true}),
		m.endpoint(operation{use: "show <id>", short: "Show a transaction", args: 1, transactionID: true, path: resource("transactions")}),
		m.endpoint(operation{use: "create", short: "Create a transaction from v3 JSON (postings or Numscript)", method: http.MethodPost, path: fixed("transactions"), body: bodyRequired, idempotency: true}),
		m.endpoint(operation{use: "revert <id>", short: "Revert a transaction", args: 1, transactionID: true, method: http.MethodPost, path: resource("transactions", "revert"), body: bodyOptional, idempotency: true}),
		m.metadata([]string{"transactions"}, true),
	)
	return group
}

// Metadata reads use the owning resource's GET route: v3 has no GET /metadata.
func (m *module) metadata(parent []string, transactionID bool) *cobra.Command {
	group := &cobra.Command{Use: "metadata", Short: "Read metadata in the resource envelope, set or delete keys"}
	args := 0
	argLabel := ""
	if len(parent) > 0 {
		args = 1
		argLabel = " <address>"
		if transactionID {
			argLabel = " <id>"
		}
	}
	base := func(a []string) []string {
		segments := append([]string{}, parent...)
		if args > 0 {
			segments = append(segments, a[0])
		}
		return segments
	}
	group.AddCommand(
		m.endpoint(operation{use: "show" + argLabel, short: "Show the resource including metadata", args: args, transactionID: transactionID, path: base}),
		m.endpoint(operation{use: "set" + argLabel, short: "Merge a JSON metadata object", args: args, transactionID: transactionID, method: http.MethodPost, body: bodyRequired, idempotency: true, path: func(a []string) []string { return append(base(a), "metadata") }}),
		m.endpoint(operation{use: "delete" + argLabel + " <key>", short: "Delete one raw metadata key", args: args + 1, transactionID: transactionID, method: http.MethodDelete, idempotency: true, path: func(a []string) []string { return append(base(a), "metadata", a[args]) }}),
	)
	return group
}

func (m *module) indexes() *cobra.Command {
	group := &cobra.Command{Use: "indexes", Short: "Manage and inspect ledger indexes"}
	group.AddCommand(
		m.endpoint(operation{use: "list", short: "List ledger indexes", path: fixed("indexes")}),
		m.endpoint(operation{use: "show <canonical-id>", short: "Show an index", args: 1, path: resource("indexes")}),
		m.endpoint(operation{use: "status <canonical-id>", short: "Show index build status", args: 1, path: resource("indexes", "status")}),
		m.endpoint(operation{use: "create", short: "Create an index from JSON, e.g. {\"id\":\"log_builtin:LOG_BUILTIN_INDEX_DATE\"}", method: http.MethodPost, path: fixed("indexes"), body: bodyRequired, idempotency: true}),
		m.endpoint(operation{use: "delete <canonical-id>", short: "Drop an index", args: 1, method: http.MethodDelete, path: resource("indexes"), idempotency: true}),
	)
	inspect := m.endpoint(operation{use: "inspect <canonical-id>", short: "Inspect a metadata index", args: 1, path: resource("indexes", "inspect"), inspect: true})
	group.AddCommand(inspect)
	return group
}

func fixed(segments ...string) func([]string) []string {
	return func([]string) []string { return segments }
}

func resource(collection string, suffix ...string) func([]string) []string {
	return func(args []string) []string { return append([]string{collection, args[0]}, suffix...) }
}

func (m *module) path(op operation, args []string) (string, error) {
	if op.global {
		return api.Path(op.path(args)...), nil
	}
	name := m.ledger
	if op.ledgerArg && len(args) > 0 {
		if name != "" && name != args[0] {
			return "", fmt.Errorf("positional ledger name conflicts with --ledger")
		}
		name = args[0]
	}
	if name == "" {
		return "", fmt.Errorf("select a ledger with --ledger (create/show/delete also accept a positional name)")
	}
	segments := []string{"v3", name}
	if op.path != nil {
		segments = append(segments, op.path(args)...)
	}
	return api.Path(segments...), nil
}
