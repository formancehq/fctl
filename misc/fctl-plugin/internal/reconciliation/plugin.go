// Package reconciliation supplies the historical reconciliation commands through the plugin SDK.
package reconciliation

import (
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	"github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
	"github.com/formancehq/fctl/misc/fctl-plugin/internal/metadata"
)

// New builds an independent plugin using the HTTP client supplied by the host.
func New(client *http.Client) pluginsdk.Plugin {
	operations := []command.Operation{
		command.Leaf("get", "get <reconciliation-id>", http.MethodGet, "reconciliations/$0", 1, 1),
	}
	operations = append(operations, policyOperations()...)
	operations = append(operations, ruleOperations()...)
	operations = append(operations,
		command.Leaf("evaluations get", "get <evaluation-id>", http.MethodGet, "evaluations/$0", 1, 1),
		clarityList("evaluations"),
	)
	operations = append(operations, alertOperations()...)
	operations = append(operations, paginatedList("list", "list", "reconciliations", 0))
	return command.New("reconciliation", metadata.Version, client, operations, aliases)
}
