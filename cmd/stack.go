package cmd

import (
	"context"
	"fmt"

	fctl "github.com/formancehq/fctl/pkg"
	membershipclient "github.com/numary/membership-api/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	organizationFlag = "organization"
	stackNameFlag    = "name"
)

var (
	apiClient *membershipclient.APIClient
)

var stacksCommand = &cobra.Command{
	Use:   "stacks",
	Short: "manage your organization stacks",
}

var createStackCommand = &cobra.Command{
	Use:   "create",
	Args:  cobra.ExactArgs(1),
	Short: "create a new stack",
	RunE: func(cmd *cobra.Command, args []string) error {

		organization, err := findOrganizationId(cmd.Context())
		if err != nil {
			return err
		}

		stack, _, err := apiClient.DefaultApi.CreateStack(cmd.Context(), organization).Body(membershipclient.StackData{
			Name: args[0],
		}).Execute()
		if err != nil {
			return errors.Wrap(err, "creating stack")
		}

		baseUrl, err := fctl.ServicesBaseUrl(*currentProfile, stack.Data.OrganizationId, stack.Data.Id)
		if err != nil {
			return err
		}
		baseUrlStr := baseUrl.String()

		fmt.Fprintf(cmd.OutOrStdout(), "Stack created with ID: %s\r\n", stack.Data.Id)
		fmt.Fprintf(cmd.OutOrStdout(), "Your dashboard will be reachable on: %s\r\n", baseUrlStr)
		fmt.Fprintln(cmd.OutOrStdout(), "You can access your stack apis using following urls :")
		fmt.Fprintf(cmd.OutOrStdout(), "Ledger: %s/api/ledger\r\n", baseUrlStr)
		fmt.Fprintf(cmd.OutOrStdout(), "Payments: %s/api/payments\n", baseUrlStr)
		fmt.Fprintf(cmd.OutOrStdout(), "Search: %s/api/search\n", baseUrlStr)
		fmt.Fprintf(cmd.OutOrStdout(), "Auth: %s/api/auth\n", baseUrlStr)

		return nil
	},
}

var deleteStackCommand = &cobra.Command{
	Use:   "delete [STACK_ID] | --name=[NAME]",
	Args:  cobra.MaximumNArgs(1),
	Short: "delete a stack",
	RunE: func(cmd *cobra.Command, args []string) error {

		organization, err := findOrganizationId(cmd.Context())
		if err != nil {
			return errors.Wrap(err, "searching default organization")
		}

		var stackId string
		if len(args) == 1 {
			if viper.GetString(stackNameFlag) != "" {
				return errors.New("need either an id of a name spefified using --name flag")
			}
			stackId = args[0]
		} else {
			if viper.GetString(stackNameFlag) == "" {
				return errors.New("need either an id of a name spefified using --name flag")
			}
			stacks, _, err := apiClient.DefaultApi.ListStacks(context.Background(), organization).Execute()
			if err != nil {
				return errors.Wrap(err, "listing stacks")
			}
			for _, s := range stacks.Data {
				if s.Name == viper.GetString(stackNameFlag) {
					stackId = s.Id
					break
				}
			}
			if stackId == "" {
				return errors.New("stack not found")
			}
		}

		if _, err := apiClient.DefaultApi.DeleteStack(cmd.Context(), organization, stackId).Execute(); err != nil {
			return errors.Wrap(err, "deleting stack")
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Stack deleted.")

		return nil
	},
}

var listStacks = &cobra.Command{
	Use:   "list",
	Short: "list stacks",
	RunE: func(cmd *cobra.Command, args []string) error {

		organization, err := findOrganizationId(cmd.Context())
		if err != nil {
			return errors.Wrap(err, "searching default organization")
		}

		rsp, _, err := apiClient.DefaultApi.ListStacks(cmd.Context(), organization).Execute()
		if err != nil {
			return errors.Wrap(err, "listing stacks")
		}

		if len(rsp.Data) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No stacks found.")
			return nil
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Stacks: ")
		for _, s := range rsp.Data {
			fmt.Fprintf(cmd.OutOrStdout(), "\t- %s: %s\r\n", s.Id, s.Name)
		}
		return nil
	},
}

func init() {
	deleteStackCommand.Flags().StringP(stackNameFlag, "n", "", "Stack name to delete")
	stacksCommand.PersistentFlags().String(organizationFlag, "", "Specific organization to target")
	stacksCommand.AddCommand(createStackCommand, deleteStackCommand, listStacks)
	rootCommand.AddCommand(stacksCommand)
}
