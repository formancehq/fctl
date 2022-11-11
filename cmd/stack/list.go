package stack

import (
	"fmt"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/formancehq/fctl/cmd/internal/membership"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newSandboxListCommand() *cobra.Command {
	return cmdbuilder.NewMembershipCommand("list",
		cmdbuilder.WithShortDescription("list sandboxes"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			organization, err := cmdbuilder.ResolveOrganizationID(cmd, cfg)
			if err != nil {
				return errors.Wrap(err, "searching default organization")
			}

			apiClient, err := membership.NewClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			rsp, _, err := apiClient.DefaultApi.ListStacks(cmd.Context(), organization).Execute()
			if err != nil {
				return errors.Wrap(err, "listing sandboxs")
			}

			if len(rsp.Data) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No sandboxs found.")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Stacks: ")
			for _, s := range rsp.Data {
				fmt.Fprintf(cmd.OutOrStdout(), "\t- %s: %s\r\n", s.Id, s.Name)
			}
			return nil
		}),
	)
}
