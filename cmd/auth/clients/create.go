package clients

import (
	"strings"

	internal2 "github.com/formancehq/fctl/cmd/internal"
	"github.com/formancehq/formance-sdk-go"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func NewCreateCommand() *cobra.Command {
	const (
		publicFlag                = "public"
		trustedFlag               = "trusted"
		descriptionFlag           = "description"
		redirectUriFlag           = "redirect-uri"
		postLogoutRedirectUriFlag = "post-logout-redirect-uri"
	)
	return internal2.NewCommand("create",
		internal2.WithAliases("c"),
		internal2.WithArgs(cobra.ExactArgs(1)),
		internal2.WithBoolFlag(publicFlag, false, "Is client public"),
		internal2.WithBoolFlag(trustedFlag, false, "Is the client trusted"),
		internal2.WithStringFlag(descriptionFlag, "", "Client description"),
		internal2.WithStringSliceFlag(redirectUriFlag, []string{}, "Redirect URIS"),
		internal2.WithStringSliceFlag(postLogoutRedirectUriFlag, []string{}, "Post logout redirect uris"),
		internal2.WithShortDescription("Create client"),
		internal2.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := internal2.Get(cmd)
			if err != nil {
				return err
			}

			authClient, err := internal2.NewStackClient(cmd, cfg)
			if err != nil {
				return err
			}

			public := internal2.GetBool(cmd, publicFlag)
			trusted := internal2.GetBool(cmd, trustedFlag)
			description := internal2.GetString(cmd, descriptionFlag)

			response, _, err := authClient.ClientsApi.CreateClient(cmd.Context()).Body(formance.ClientOptions{
				Public:                 &public,
				RedirectUris:           internal2.GetStringSlice(cmd, redirectUriFlag),
				Description:            &description,
				Name:                   args[0],
				Trusted:                &trusted,
				PostLogoutRedirectUris: internal2.GetStringSlice(cmd, postLogoutRedirectUriFlag),
			}).Execute()
			if err != nil {
				return err
			}

			tableData := pterm.TableData{}
			tableData = append(tableData, []string{pterm.LightCyan("ID"), response.Data.Id})
			tableData = append(tableData, []string{pterm.LightCyan("Name"), response.Data.Name})
			tableData = append(tableData, []string{pterm.LightCyan("Description"), internal2.StringPointerToString(response.Data.Description)})
			tableData = append(tableData, []string{pterm.LightCyan("Public"), internal2.BoolPointerToString(response.Data.Public)})
			tableData = append(tableData, []string{pterm.LightCyan("Redirect URIs"), strings.Join(response.Data.RedirectUris, ",")})
			tableData = append(tableData, []string{pterm.LightCyan("Post logout redirect URIs"), strings.Join(response.Data.PostLogoutRedirectUris, ",")})
			return pterm.DefaultTable.
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
