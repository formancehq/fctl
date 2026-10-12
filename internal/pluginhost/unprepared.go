package pluginhost

import (
	"fmt"

	"github.com/spf13/cobra"
)

// UnpreparedServiceCommand explains how to prepare a missing service provider.
func UnpreparedServiceCommand(service, title string, preparationErr error) *cobra.Command {
	command := &cobra.Command{
		Use: service, Short: "Manage " + title + " through an installed plugin",
		Long: fmt.Sprintf("%s commands are discovered from the plugin catalogue and cached locally.\nPrepare your target with fctl plugins sync --service %s, or install a local\nexecutable with fctl plugins install --service %s --binary PATH.", title, service, service),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return fmt.Errorf("%s plugin is not installed for this target; run fctl plugins sync --service %s or fctl plugins install --service %s --binary PATH", service, service, service)
		},
	}
	command.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		if preparationErr != nil {
			return preparationErr
		}
		return err
	})
	return command
}
