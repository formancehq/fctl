package wallets

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func walletFlags() []pluginsdk.FlagSpec {
	return []pluginsdk.FlagSpec{commandapi.StringFlag("id", "Wallet ID"), commandapi.StringFlag("name", "Exact wallet name")}
}

func targetWallet(op commandapi.Operation, required bool) commandapi.Operation {
	op.Spec.Flags = append(op.Spec.Flags, walletFlags()...)
	if required {
		op.Spec.Inputs = append(op.Spec.Inputs, pluginsdk.InputSpec{Title: "Wallet", Kind: "select", Flag: "id", AlternativeFlag: "name", Required: true,
			Source: &pluginsdk.ChoiceSource{CommandPath: []string{"wallets", "list"}, ValueField: "id", LabelFields: []string{"name", "id"}, EmptyMessage: "No wallets available"}})
	}
	op.ValidateRoute = validateWalletRoute
	op.Run = walletOperation
	return op
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
	return id, commandapi.Identifier(id)
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

// Validate the dynamic wallet ID locally before body validation or HTTP lookup.
func validateWalletRoute(r pluginsdk.ExecuteRequest, op commandapi.Operation) error {
	if r.Flags["id"] == "" && r.Flags["name"] != "" {
		r.Flags = maps.Clone(r.Flags)
		r.Flags["id"] = "resolved-wallet"
	}
	_, err := commandapi.Route(op.Segments, r)
	return err
}
