// Package auth embeds the product-owned Auth fctl plugin.
// Service commands, payloads, forms and contract tests live in the independent
// github.com/formancehq/auth/misc/fctl-plugin module.
package auth

import (
	"net/http"

	authplugin "github.com/formancehq/auth/misc/fctl-plugin"
	"github.com/formancehq/fctl/pkg/pluginsdk"
)

// New adapts the product factory to the host's embedded registration boundary.
func New(client *http.Client) pluginsdk.Plugin { return authplugin.New(client) }
