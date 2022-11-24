package stack

import (
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/cmdutils"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewDeleteCommand() *cobra.Command {
	const (
		stackNameFlag = "name"
	)

	return cmdbuilder.NewMembershipCommand("delete [STACK_ID] | --name=[NAME]",
		cmdbuilder.WithShortDescription("Delete a sandbox"),
		cmdbuilder.WithAliases("del", "d"),
		cmdbuilder.WithArgs(cobra.MaximumNArgs(1)),
		cmdbuilder.WithStringFlag(stackNameFlag, "", "Sandbox to remove"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd)
			if err != nil {
				return err
			}
			organization, err := cmdbuilder.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return errors.Wrap(err, "searching default organization")
			}

			apiClient, err := config.NewClient(cmd, cfg)
			if err != nil {
				return err
			}

			var stackID string
			if len(args) == 1 {
				if cmdutils.GetString(cmd, stackNameFlag) != "" {
					return errors.New("need either an id of a name spefified using --name flag")
				}
				stackID = args[0]
			} else {
				if cmdutils.GetString(cmd, stackNameFlag) == "" {
					return errors.New("need either an id of a name specified using --name flag")
				}
				stacks, _, err := apiClient.DefaultApi.ListStacks(cmd.Context(), organization).Execute()
				if err != nil {
					return errors.Wrap(err, "listing stacks")
				}
				for _, s := range stacks.Data {
					if s.Name == cmdutils.GetString(cmd, stackNameFlag) {
						stackID = s.Id
						break
					}
				}
				if stackID == "" {
					return errors.New("stack not found")
				}
			}

			if _, err := apiClient.DefaultApi.DeleteStack(cmd.Context(), organization, stackID).Execute(); err != nil {
				return errors.Wrap(err, "deleting stack")
			}

			cmdbuilder.Success(cmd.OutOrStdout(), "Stack deleted.")

			return nil
		}),
	)
}
