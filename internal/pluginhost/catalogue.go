package pluginhost

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/formancehq/fctl/v4/internal/connection"
	"github.com/formancehq/fctl/v4/internal/pluginmanager"
)

func servicePluginCatalogue(cmd *cobra.Command, settings *connection.Settings, manager *pluginmanager.Manager, service, catalogue string) (string, error) {
	if catalogue != "" {
		return catalogue, nil
	}
	if value := os.Getenv("FCTL_PLUGIN_CATALOGUE"); value != "" {
		return value, nil
	}
	lock, err := cachedServiceLock(settings, cmd, manager, service)
	if err == nil && lock.Catalogue != "" {
		return lock.Catalogue, nil
	}
	return pluginmanager.DefaultCatalogue, nil
}
