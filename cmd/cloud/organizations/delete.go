package organizations

import (
	"github.com/formancehq/fctl/cmd/internal"
	"github.com/spf13/cobra"
)

func NewDeleteCommand() *cobra.Command {
	return internal.NewCommand("delete",
		internal.WithAliases("del", "d"),
		internal.WithShortDescription("Delete organization"),
		internal.WithArgs(cobra.ExactArgs(1)),
		internal.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal.Get(cmd)
			if err != nil {
				return err
			}

			apiClient, err := internal.NewMembershipClient(cmd, cfg)
			if err != nil {
				return err
			}

			_, err = apiClient.DefaultApi.
				DeleteOrganization(cmd.Context(), args[0]).
				Execute()
			if err != nil {
				return err
			}

			internal.Success(cmd.OutOrStdout(), "Organization '%s' deleted", args[0])

			return nil
		}),
	)
}
