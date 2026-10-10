package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

func walletFlags() []pluginsdk.FlagSpec {
	return []pluginsdk.FlagSpec{str("id", "Wallet ID"), str("name", "Exact wallet name")}
}
func metadataFlag() pluginsdk.FlagSpec {
	return str("metadata", "Metadata as JSON object or CSV key=value")
}
func ikFlag() pluginsdk.FlagSpec { return str("ik", "Idempotency key") }

func targetWallet(op operation, required bool) operation {
	op.spec.Flags = append(op.spec.Flags, walletFlags()...)
	if required {
		op.spec.Inputs = append(op.spec.Inputs, pluginsdk.InputSpec{Title: "Wallet", Kind: "select", Flag: "id", AlternativeFlag: "name", Required: true,
			Source: &pluginsdk.ChoiceSource{CommandPath: []string{"wallets", "list"}, ValueField: "id", LabelFields: []string{"name", "id"}, EmptyMessage: "No wallets available"}})
	}
	op.run = walletOperation
	return op
}

func newWallets(client *http.Client) pluginsdk.Plugin {
	create := confirmed(leaf("create", "create <name>", http.MethodPost, "wallets", 1, 1, metadataFlag(), ikFlag()))
	create.body = walletCreateBody
	update := confirmed(leaf("update", "update <wallet-id>", http.MethodPatch, "wallets/$0", 1, 1, metadataFlag(), ikFlag()))
	update.body = walletUpdateBody
	listWallets := leaf("list", "list", http.MethodGet, "wallets", 0, 0, append(pagination(), metadataFlag(), str("name", "Exact name filter"))...)
	listWallets.query = map[string]string{"page-size": "pageSize", "cursor": "cursor", "name": "name"}
	listWallets.body = validateMetadata
	listWallets.run = walletOperation
	show := targetWallet(leaf("show", "show", http.MethodGet, "wallets/@id", 0, 0), true)
	balanceCreate := confirmed(targetWallet(leaf("balances create", "create <balance-name>", http.MethodPost, "wallets/@id/balances", 1, 1, str("expires-at", "RFC3339 expiry"), str("priority", "Integer priority")), true))
	balanceCreate.body = balanceCreateBody
	balanceList := targetWallet(leaf("balances list", "list", http.MethodGet, "wallets/@id/balances", 0, 0, pagination()...), true)
	balanceList.query = map[string]string{"page-size": "pageSize", "cursor": "cursor"}
	balanceShow := targetWallet(leaf("balances show", "show <balance-name>", http.MethodGet, "wallets/@id/balances/$0", 1, 1), true)
	credit := confirmed(targetWallet(leaf("credit", "credit <amount> <asset>", http.MethodPost, "wallets/@id/credit", 2, 2,
		metadataFlag(), ikFlag(), str("balance", "Balance to credit"), str("source", "Sources as CSV or JSON string array: account=ADDRESS or wallet=id:ID/BALANCE or wallet=name:NAME/BALANCE")), true))
	credit.body = walletMovement
	debit := confirmed(targetWallet(leaf("debit", "debit <amount> <asset>", http.MethodPost, "wallets/@id/debit", 2, 2,
		metadataFlag(), ikFlag(), str("balance", "Balances as CSV or JSON string array"), boolean("pending", "Create a pending hold"), str("description", "Debit description"),
		str("destination", "account=ADDRESS or wallet=id:ID/BALANCE or wallet=name:NAME/BALANCE")), true))
	debit.body = walletMovement
	holds := targetWallet(leaf("holds list", "list", http.MethodGet, "holds", 0, 0, append(pagination(), metadataFlag())...), false)
	holds.query = map[string]string{"page-size": "pageSize", "cursor": "cursor", "id": "walletID"}
	holds.body = validateMetadata
	transactions := targetWallet(leaf("transactions list", "list", http.MethodGet, "transactions", 0, 0, pagination()...), false)
	transactions.query = map[string]string{"page-size": "pageSize", "cursor": "cursor", "id": "walletID"}
	confirm := confirmed(leaf("holds confirm", "confirm <hold-id>", http.MethodPost, "holds/$0/confirm", 1, 1, boolean("final", "Close hold after confirming"), str("amount", "Integer amount (default 0)"), ikFlag()))
	confirm.body = confirmHoldBody
	return newPlugin("wallets", client, []operation{create, update, listWallets, show, balanceCreate, balanceList, balanceShow, credit, debit, holds, transactions, confirm,
		leaf("holds show", "show <hold-id>", http.MethodGet, "holds/$0", 1, 1),
		confirmed(leaf("holds void", "void <hold-id>", http.MethodPost, "holds/$0/void", 1, 1, ikFlag())),
	})
}

func integer(value string, nonnegative bool) (json.Number, error) {
	n, ok := new(big.Int).SetString(value, 10)
	if !ok || (nonnegative && n.Sign() < 0) {
		return "", fmt.Errorf("expected %sinteger, got %q", map[bool]string{true: "nonnegative ", false: ""}[nonnegative], value)
	}
	return json.Number(n.String()), nil
}

func validateMetadata(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	_, err := pairs(r.Flags["metadata"])
	return nil, err
}

func walletMovement(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	n, err := integer(r.Args[0], true)
	if err != nil {
		return nil, err
	}
	if err := identifier(r.Args[1]); err != nil {
		return nil, fmt.Errorf("asset: %w", err)
	}
	metadata, err := pairs(r.Flags["metadata"])
	if err != nil {
		return nil, err
	}
	body := map[string]any{"amount": map[string]any{"amount": n, "asset": r.Args[1]}, "metadata": metadata}
	if r.CommandPath[1] == "credit" {
		body["balance"] = r.Flags["balance"]
		sources, err := list(r.Flags["source"])
		if err != nil {
			return nil, err
		}
		if err := validateSubjects(sources); err != nil {
			return nil, err
		}
	} else {
		body["pending"] = r.Flags["pending"] == "true"
		body["description"] = r.Flags["description"]
		balances, err := list(r.Flags["balance"])
		if err != nil {
			return nil, err
		}
		body["balances"] = balances
		if value := r.Flags["destination"]; value != "" {
			if _, err := parseSubject(value); err != nil {
				return nil, err
			}
		}
	}
	return marshal(body)
}

type subject struct{ kind, id, name, balance string }

func parseSubject(value string) (subject, error) {
	if id, found := strings.CutPrefix(value, "account="); found {
		if err := identifier(id); err != nil {
			return subject{}, err
		}
		return subject{kind: "ACCOUNT", id: id}, nil
	}
	if !strings.HasPrefix(value, "wallet=") {
		return subject{}, fmt.Errorf("invalid subject %q", value)
	}
	definition, balance, hasBalance := strings.Cut(strings.TrimPrefix(value, "wallet="), "/")
	if !hasBalance {
		balance = "main"
	}
	if err := identifier(balance); err != nil {
		return subject{}, err
	}
	kind, id, found := strings.Cut(definition, ":")
	if !found || (kind != "id" && kind != "name") || identifier(id) != nil {
		return subject{}, fmt.Errorf("wallet subject requires id:ID or name:NAME")
	}
	out := subject{kind: "WALLET", balance: balance}
	if kind == "name" {
		out.name = id
	} else {
		out.id = id
	}
	return out, nil
}

func walletByName(ctx context.Context, client *httpclient.Client, name string) (string, error) {
	data, err := client.Do(ctx, http.MethodGet, httpclient.Path("wallets"), url.Values{"name": {name}, "pageSize": {"2"}}, nil, nil)
	if err != nil {
		return "", err
	}
	var envelope struct {
		Cursor struct {
			HasMore bool `json:"hasMore"`
			Data    []struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"cursor"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", err
	}
	if len(envelope.Cursor.Data) == 0 {
		return "", fmt.Errorf("wallet %q not found", name)
	}
	if len(envelope.Cursor.Data) != 1 || envelope.Cursor.HasMore {
		return "", fmt.Errorf("wallet name %q is ambiguous", name)
	}
	id := envelope.Cursor.Data[0].ID
	return id, identifier(id)
}

func resolveSubject(ctx context.Context, client *httpclient.Client, value string) (map[string]string, error) {
	s, err := parseSubject(value)
	if err != nil {
		return nil, err
	}
	if s.name != "" {
		s.id, err = walletByName(ctx, client, s.name)
		if err != nil {
			return nil, err
		}
	}
	out := map[string]string{"type": s.kind, "identifier": s.id}
	if s.balance != "" {
		out["balance"] = s.balance
	}
	return out, nil
}

func walletOperation(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op operation, body json.RawMessage) (json.RawMessage, error) {
	var err error
	if op.command != "list" {
		r, err = resolveWalletTarget(ctx, client, r)
		if err != nil {
			return nil, err
		}
	}
	if op.command == "credit" || op.command == "debit" {
		body, err = walletSubjectBody(ctx, client, r, body)
		if err != nil {
			return nil, err
		}
	}
	query, err := walletQuery(op, r)
	if err != nil {
		return nil, err
	}
	path, err := route(op.segments, r)
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	if r.Flags["ik"] != "" {
		headers.Set("Idempotency-Key", r.Flags["ik"])
	}
	return client.Do(ctx, op.method, path, query, body, headers)
}

func walletCreateBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if err := identifier(r.Args[0]); err != nil {
		return nil, err
	}
	metadata, err := pairs(r.Flags["metadata"])
	if err != nil {
		return nil, err
	}
	return marshal(map[string]any{"name": r.Args[0], "metadata": metadata})
}

func walletUpdateBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	metadata, err := pairs(r.Flags["metadata"])
	if err != nil {
		return nil, err
	}
	return marshal(map[string]any{"metadata": metadata})
}

func balanceCreateBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if err := identifier(r.Args[0]); err != nil {
		return nil, err
	}
	body := map[string]any{"name": r.Args[0]}
	if v := r.Flags["expires-at"]; v != "" {
		if err := timestamp(v); err != nil {
			return nil, err
		}
		body["expiresAt"] = v
	}
	if v := r.Flags["priority"]; v != "" {
		n, err := integer(v, false)
		if err != nil {
			return nil, err
		}
		body["priority"] = n
	}
	return marshal(body)
}

func confirmHoldBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	amount := r.Flags["amount"]
	if amount == "" {
		amount = "0"
	}
	n, err := integer(amount, true)
	if err != nil {
		return nil, err
	}
	return marshal(map[string]any{"amount": n, "final": r.Flags["final"] == "true"})
}

func validateSubjects(values []string) error {
	for _, value := range values {
		if _, err := parseSubject(value); err != nil {
			return err
		}
	}
	return nil
}

func resolveWalletTarget(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest) (pluginsdk.ExecuteRequest, error) {
	if r.Flags["id"] != "" && r.Flags["name"] != "" {
		return r, fmt.Errorf("provide --id or --name, not both")
	}
	if r.Flags["name"] != "" {
		id, err := walletByName(ctx, client, r.Flags["name"])
		if err != nil {
			return r, err
		}
		r.Flags["id"] = id
	}
	return r, nil
}

func walletSubjectBody(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, body json.RawMessage) (json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		return nil, err
	}
	if r.CommandPath[1] == "credit" {
		sources, err := walletSources(ctx, client, r.Flags["source"])
		if err != nil {
			return nil, err
		}
		object["sources"] = sources
	} else if r.Flags["destination"] != "" {
		destination, err := resolveSubject(ctx, client, r.Flags["destination"])
		if err != nil {
			return nil, err
		}
		encoded, err := json.Marshal(destination)
		if err != nil {
			return nil, err
		}
		object["destination"] = encoded
	}
	return marshal(object)
}

func walletSources(ctx context.Context, client *httpclient.Client, value string) (json.RawMessage, error) {
	values, err := list(value)
	if err != nil {
		return nil, err
	}
	sources := []map[string]string{}
	for _, value := range values {
		s, err := resolveSubject(ctx, client, value)
		if err != nil {
			return nil, err
		}
		sources = append(sources, s)
	}
	return marshal(sources)
}

func walletQuery(op operation, r pluginsdk.ExecuteRequest) (url.Values, error) {
	query, err := queryValues(op.query, r)
	if err != nil {
		return nil, err
	}
	if cursor := query.Get("cursor"); cursor != "" {
		return url.Values{"cursor": {cursor}}, nil
	}
	if op.method == http.MethodGet && r.Flags["metadata"] != "" {
		metadata, err := pairs(r.Flags["metadata"])
		if err != nil {
			return nil, err
		}
		for key, value := range metadata {
			query.Set("metadata["+key+"]", value)
		}
	}
	return query, nil
}
