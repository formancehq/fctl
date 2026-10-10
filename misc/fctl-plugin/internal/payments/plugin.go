// Package payments implements the historical payments commands.
package payments

import (
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
	"github.com/formancehq/fctl/misc/fctl-plugin/internal/metadata"
)

// New builds the Payments SDK plugin using the injected HTTP client.
func New(client *http.Client) pluginsdk.Plugin {
	operations := accountOperations()
	operations = append(operations, paymentOperations()...)
	operations = append(operations, bankAccountOperations()...)
	operations = append(operations, poolOperations()...)
	operations = append(operations, transferOperations()...)
	operations = append(operations, taskOperations()...)
	operations = append(operations, connectorOperations()...)
	operations = append(operations, orderOperations()...)
	operations = append(operations, conversionOperations()...)
	operations = append(operations, paymentListOperations()...)
	operations = append(operations, paymentExchangeOperations()...)
	operations = append(operations, paymentMetadataOperations()...)
	operations = append(operations,
		forwardBankAccountOperation(),
		addPoolAccountOperation(),
		poolBalancesOperation(),
		transferStatusOperation(),
	)
	for i := range operations {
		operations[i].Run = paymentsOperation
		if operations[i].Command == "connectors get-config" || operations[i].Command == "connectors uninstall" || operations[i].Command == "connectors update-config" {
			alternative := "provider"
			if operations[i].Command == "connectors update-config" {
				alternative = ""
			}
			operations[i].Spec.Inputs = append(operations[i].Spec.Inputs, pluginsdk.InputSpec{Title: "Connector", Kind: "select", Flag: "connector-id", AlternativeFlag: alternative, Required: true,
				Source: &pluginsdk.ChoiceSource{CommandPath: []string{"payments", "connectors", "list"}, ValueField: "id", LabelFields: []string{"name", "provider", "id"}, EmptyMessage: "No connectors available"}})
		}
	}
	return commandapi.New("payments", metadata.Version, client, operations, aliases)
}
