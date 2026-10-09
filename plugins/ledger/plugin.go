// Package ledger embeds the product-owned Ledger fctl plugin.
// Service commands, payloads, forms and contract tests live in the independent
// github.com/formancehq/ledger/fctl-plugin module.
package ledger

import (
	"net/http"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	ledgerplugin "github.com/formancehq/ledger/fctl-plugin"
)

// New adapts the product factory to the host's embedded registration boundary.
func New(client *http.Client) pluginsdk.Plugin { return ledgerplugin.New(client) }
