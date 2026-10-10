// Package webhooks supplies the historical webhooks commands through the plugin SDK.
package webhooks

import (
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
	"github.com/formancehq/fctl/misc/fctl-plugin/internal/metadata"
)

// New builds an independent plugin using the HTTP client supplied by the host.
func New(client *http.Client) pluginsdk.Plugin {
	operations := configOperations()
	operations = append(operations, deliveryOperations()...)
	return command.New("webhooks", metadata.Version, client, operations, aliases)
}
