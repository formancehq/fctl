package stack

import (
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewDeleteCommand() *cobra.Command {
	const (
		stackNameFlag = "name"
	)

	return fctl.NewMembershipCommand("delete [STACK_ID] | --name=[NAME]",
		fctl.WithShortDescription("Delete a stack"),
		fctl.WithAliases("del", "d"),
		fctl.WithArgs(cobra.MaximumNArgs(1)),
		fctl.WithStringFlag(stackNameFlag, "", "Stack to remove"),
		fctl.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := fctl.GetConfig(cmd)
			if err != nil {
				return err
			}
			organization, err := fctl.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return errors.Wrap(err, "searching default organization")
			}

			apiClient, err := fctl.NewMembershipClient(cmd, cfg)
			if err != nil {
				return err
			}

			var stackID string
			if len(args) == 1 {
				if fctl.GetString(cmd, stackNameFlag) != "" {
					return errors.New("need either an id of a name spefified using --name flag")
				}
				stackID = args[0]
			} else {
				if fctl.GetString(cmd, stackNameFlag) == "" {
					return errors.New("need either an id of a name specified using --name flag")
				}
				stacks, _, err := apiClient.DefaultApi.ListStacks(cmd.Context(), organization).Execute()
				if err != nil {
					return errors.Wrap(err, "listing stacks")
				}
				for _, s := range stacks.Data {
					if s.Name == fctl.GetString(cmd, stackNameFlag) {
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

			fctl.Success(cmd.OutOrStdout(), "Stack deleted.")

			return nil
		}),
	)
}
