// Package orchestration supplies the historical orchestration commands through the plugin SDK.
package orchestration

import (
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
	"github.com/formancehq/fctl/misc/fctl-plugin/internal/metadata"
)

// New builds an independent plugin using the HTTP client supplied by the host.
func New(client *http.Client) pluginsdk.Plugin {
	operations := instanceOperations()
	operations = append(operations, workflowOperations()...)
	operations = append(operations, triggerOperations()...)
	return command.New("orchestration", metadata.Version, client, operations, aliases)
}
