package search

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/cmdutils"
	"github.com/formancehq/fctl/cmd/internal/collections"
	"github.com/formancehq/fctl/cmd/internal/config"
	searchclient "github.com/formancehq/search/client"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func newSearchClient(cmd *cobra.Command, cfg *config.Config) (*searchclient.APIClient, error) {
	profile := config.GetCurrentProfile(cmd, cfg)

	organizationID, err := cmdbuilder.ResolveOrganizationID(cmd, cfg)
	if err != nil {
		return nil, err
	}

	stack, err := cmdbuilder.ResolveStack(cmd, cfg, organizationID)
	if err != nil {
		return nil, err
	}

	httpClient := config.GetHttpClient(cmd)

	token, err := profile.GetStackToken(cmd.Context(), httpClient, stack)
	if err != nil {
		return nil, err
	}

	apiConfig := searchclient.NewConfiguration()
	apiConfig.Servers = searchclient.ServerConfigurations{{
		URL: profile.ApiUrl(stack, "search").String(),
	}}
	apiConfig.AddDefaultHeader("Authorization", fmt.Sprintf("Bearer %s", token))
	apiConfig.HTTPClient = httpClient

	return searchclient.NewAPIClient(apiConfig), nil
}

func NewCommand() *cobra.Command {
	const (
		sizeFlag = "size"
	)
	return cmdbuilder.NewCommand("search",
		cmdbuilder.WithAliases("se"),
		cmdbuilder.WithArgs(cobra.MinimumNArgs(1)),
		cmdbuilder.WithIntFlag(sizeFlag, 5, "Number of items to fetch"),
		cmdbuilder.WithValidArgs("ANY", "ACCOUNT", "TRANSACTION", "ASSET"),
		cmdbuilder.WithShortDescription("Search in all services"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {

			cfg, err := config.Get(cmd)
			if err != nil {
				return err
			}

			searchClient, err := newSearchClient(cmd, cfg)
			if err != nil {
				return err
			}

			target := strings.ToUpper(args[0])
			if target == "ANY" {
				target = ""
			}
			terms := make([]string, 0)
			if len(args) > 1 {
				terms = args[1:]
			}
			size := int32(cmdutils.GetInt(cmd, sizeFlag))

			response, _, err := searchClient.DefaultApi.Search(cmd.Context()).Query(searchclient.Query{
				Size:   &size,
				Terms:  terms,
				Target: &target,
			}).Execute()
			if err != nil {
				return err
			}

			if target == "" {
				tableData := make([][]string, 0)
				for kind, values := range response.Data {
					for _, value := range values.([]any) {
						dataAsJson, err := json.Marshal(value)
						if err != nil {
							return err
						}

						dataAsJsonString := string(dataAsJson)
						if len(dataAsJsonString) > 100 {
							dataAsJsonString = dataAsJsonString[:100] + "..."
						}

						tableData = append(tableData, []string{
							kind, dataAsJsonString,
						})
					}
				}
				tableData = collections.Prepend(tableData, []string{"Kind", "Object"})

				return pterm.DefaultTable.
					WithHasHeader().
					WithWriter(cmd.OutOrStdout()).
					WithData(tableData).
					Render()
			}

			switch target {
			case "TRANSACTION":
				err = displayTransactions(cmd.OutOrStdout(), response.Cursor.Data)
			case "ACCOUNT":
				err = displayAccounts(cmd.OutOrStdout(), response.Cursor.Data)
			case "ASSET":
				err = displayAssets(cmd.OutOrStdout(), response.Cursor.Data)
			}
			return err
		}),
	)
}

func displayAssets(out io.Writer, assets []map[string]interface{}) error {
	tableData := make([][]string, 0)
	for _, asset := range assets {
		tableData = append(tableData, []string{
			asset["ledger"].(string),
			asset["name"].(string),
			asset["account"].(string),
			fmt.Sprint(asset["input"].(float64)),
			fmt.Sprint(asset["output"].(float64)),
		})
	}
	tableData = collections.Prepend(tableData, []string{"Ledger", "Asset", "Account", "Input", "Output"})

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(out).
		WithData(tableData).
		Render()
}

func displayAccounts(out io.Writer, accounts []map[string]interface{}) error {
	tableData := make([][]string, 0)
	for _, account := range accounts {
		tableData = append(tableData, []string{
			// TODO: Missing property 'ledger' on api response
			//account["ledger"].(string),
			account["address"].(string),
		})
	}
	tableData = collections.Prepend(tableData, []string{ /*"Ledger",*/ "Address"})

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(out).
		WithData(tableData).
		Render()
}

func displayTransactions(out io.Writer, txs []map[string]interface{}) error {
	tableData := make([][]string, 0)
	for _, tx := range txs {
		tableData = append(tableData, []string{
			tx["ledger"].(string),
			fmt.Sprint(tx["txid"].(float64)),
			tx["reference"].(string),
			tx["timestamp"].(string),
		})
	}
	tableData = collections.Prepend(tableData, []string{"Ledger", "ID", "Reference", "Date"})

	return pterm.DefaultTable.
		WithHasHeader().
		WithWriter(out).
		WithData(tableData).
		Render()
}
