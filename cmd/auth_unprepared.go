package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func unpreparedAuthCommand(preparationErr error) *cobra.Command {
	command := &cobra.Command{
		Use: "auth", Short: "Manage Auth through an installed plugin",
		Long: "Auth commands are discovered from the plugin catalogue and cached locally.\nPrepare your target with fctl plugins sync --service auth, or install a local\nexecutable with fctl plugins install --service auth --binary PATH.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return fmt.Errorf("auth plugin is not installed for this target; run fctl plugins sync --service auth or fctl plugins install --service auth --binary PATH")
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
