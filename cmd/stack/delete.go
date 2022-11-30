package stack

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewDeleteCommand() *cobra.Command {
	const (
		stackNameFlag = "name"
	)

	return internal.NewMembershipCommand("delete [STACK_ID] | --name=[NAME]",
		internal.WithShortDescription("Delete a sandbox"),
		internal.WithAliases("del", "d"),
		internal.WithArgs(cobra.MaximumNArgs(1)),
		internal.WithStringFlag(stackNameFlag, "", "Sandbox to remove"),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}
			organization, err := internal.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return errors.Wrap(err, "searching default organization")
			}

			apiClient, err := internal.NewMembershipClient(cmd, cfg)
			if err != nil {
				return err
			}

			var stackID string
			if len(args) == 1 {
				if internal.GetString(cmd, stackNameFlag) != "" {
					return errors.New("need either an id of a name spefified using --name flag")
				}
				stackID = args[0]
			} else {
				if internal.GetString(cmd, stackNameFlag) == "" {
					return errors.New("need either an id of a name specified using --name flag")
				}
				stacks, _, err := apiClient.DefaultApi.ListStacks(cmd.Context(), organization).Execute()
				if err != nil {
					return errors.Wrap(err, "listing stacks")
				}
				for _, s := range stacks.Data {
					if s.Name == internal.GetString(cmd, stackNameFlag) {
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

			internal.Success(cmd.OutOrStdout(), "Stack deleted.")

			return nil
		}),
	)
}
