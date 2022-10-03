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
	sandboxNameFlag  = "name"
)

var (
	apiClient *membershipclient.APIClient
)

var sandboxsCommand = &cobra.Command{
	Use:   "sandbox",
	Short: "manage your sandbox",
}

var createStackCommand = &cobra.Command{
	Use:   "create",
	Args:  cobra.ExactArgs(1),
	Short: "create a new sandbox",
	RunE: func(cmd *cobra.Command, args []string) error {

		organization, err := findOrganizationId(cmd.Context())
		if err != nil {
			return err
		}

		sandbox, _, err := apiClient.DefaultApi.CreateStack(cmd.Context(), organization).Body(membershipclient.StackData{
			Name: args[0],
		}).Execute()
		if err != nil {
			return errors.Wrap(err, "creating sandbox")
		}

		baseUrl, err := fctl.ServicesBaseUrl(*currentProfile, sandbox.Data.OrganizationId, sandbox.Data.Id)
		if err != nil {
			return err
		}
		baseUrlStr := baseUrl.String()

		fmt.Fprintf(cmd.OutOrStdout(), "Stack created with ID: %s\r\n", sandbox.Data.Id)
		fmt.Fprintf(cmd.OutOrStdout(), "Your dashboard will be reachable on: %s\r\n", baseUrlStr)
		fmt.Fprintln(cmd.OutOrStdout(), "You can access your sandbox apis using following urls :")
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
	Short: "delete a sandbox",
	RunE: func(cmd *cobra.Command, args []string) error {

		organization, err := findOrganizationId(cmd.Context())
		if err != nil {
			return errors.Wrap(err, "searching default organization")
		}

		var sandboxId string
		if len(args) == 1 {
			if viper.GetString(sandboxNameFlag) != "" {
				return errors.New("need either an id of a name spefified using --name flag")
			}
			sandboxId = args[0]
		} else {
			if viper.GetString(sandboxNameFlag) == "" {
				return errors.New("need either an id of a name spefified using --name flag")
			}
			sandboxs, _, err := apiClient.DefaultApi.ListStacks(context.Background(), organization).Execute()
			if err != nil {
				return errors.Wrap(err, "listing sandboxs")
			}
			for _, s := range sandboxs.Data {
				if s.Name == viper.GetString(sandboxNameFlag) {
					sandboxId = s.Id
					break
				}
			}
			if sandboxId == "" {
				return errors.New("sandbox not found")
			}
		}

		if _, err := apiClient.DefaultApi.DeleteStack(cmd.Context(), organization, sandboxId).Execute(); err != nil {
			return errors.Wrap(err, "deleting sandbox")
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Stack deleted.")

		return nil
	},
}

var listStacks = &cobra.Command{
	Use:   "list",
	Short: "list sandboxs",
	RunE: func(cmd *cobra.Command, args []string) error {

		organization, err := findOrganizationId(cmd.Context())
		if err != nil {
			return errors.Wrap(err, "searching default organization")
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
	},
}

func init() {
	deleteStackCommand.Flags().StringP(sandboxNameFlag, "n", "", "Stack name to delete")
	sandboxsCommand.PersistentFlags().String(organizationFlag, "", "Specific organization to target")
	sandboxsCommand.AddCommand(createStackCommand, deleteStackCommand, listStacks)
	rootCommand.AddCommand(sandboxsCommand)
}
