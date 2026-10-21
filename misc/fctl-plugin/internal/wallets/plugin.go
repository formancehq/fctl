// Package wallets implements the historical wallets commands.
package wallets

import (
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
	"github.com/formancehq/fctl/misc/fctl-plugin/internal/metadata"
)

// New builds the Wallets SDK plugin using the injected HTTP client.
func New(client *http.Client) pluginsdk.Plugin {
	operations := walletOperations()
	operations = append(operations, balanceOperations()...)
	operations = append(operations, movementOperations()...)
	operations = append(operations, holdOperations()...)
	operations = append(operations, transactionOperations()...)
	return commandapi.New("wallets", metadata.Version, client, operations, aliases)
}
