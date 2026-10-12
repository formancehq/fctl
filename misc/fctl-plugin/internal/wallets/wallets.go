package wallets

import (
	"encoding/json"
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func walletOperations() []commandapi.Operation {
	create := commandapi.Confirmed(commandapi.Leaf("create", "create <name>", http.MethodPost, "wallets", 1, 1, metadataFlag(), ikFlag()))
	create.Body = walletCreateBody
	update := commandapi.Confirmed(commandapi.Leaf("update", "update <wallet-id>", http.MethodPatch, "wallets/$0", 1, 1, metadataFlag(), ikFlag()))
	update.Body = walletUpdateBody
	listWallets := commandapi.Leaf("list", "list", http.MethodGet, "wallets", 0, 0, append(commandapi.PaginationFlags(), metadataFlag(), commandapi.StringFlag("name", "Exact name filter"))...)
	listWallets.Query = map[string]string{"page-size": "pageSize", "cursor": "cursor", "name": "name"}
	listWallets.Body = validateMetadata
	listWallets.Run = walletOperation
	show := targetWallet(commandapi.Leaf("show", "show", http.MethodGet, "wallets/@id", 0, 0), true)
	return []commandapi.Operation{create, update, listWallets, show}
}

func walletCreateBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	if err := commandapi.Identifier(r.Args[0]); err != nil {
		return nil, err
	}
	metadata, err := commandapi.Pairs(r.Flags["metadata"])
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"name": r.Args[0], "metadata": metadata})
}

func walletUpdateBody(r pluginsdk.ExecuteRequest) (json.RawMessage, error) {
	metadata, err := commandapi.Pairs(r.Flags["metadata"])
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"metadata": metadata})
}
