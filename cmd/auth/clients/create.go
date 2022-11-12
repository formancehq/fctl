package clients

import (
	"strings"

	"github.com/formancehq/auth/authclient"
	internal2 "github.com/formancehq/fctl/cmd/auth/internal"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/cmdutils"
	"github.com/formancehq/fctl/cmd/internal/config"
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
	return cmdbuilder.NewCommand("create",
		cmdbuilder.WithAliases("c"),
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithBoolFlag(publicFlag, false, "Is client public"),
		cmdbuilder.WithBoolFlag(trustedFlag, false, "Is the client trusted"),
		cmdbuilder.WithStringFlag(descriptionFlag, "", "Client description"),
		cmdbuilder.WithStringSliceFlag(redirectUriFlag, []string{}, "Redirect URIS"),
		cmdbuilder.WithStringSliceFlag(postLogoutRedirectUriFlag, []string{}, "Post logout redirect uris"),
		cmdbuilder.WithShortDescription("Create client"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get(cmd.Context())
			if err != nil {
				return err
			}

			authClient, err := internal2.NewAuthClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			public := cmdutils.Viper(cmd.Context()).GetBool(publicFlag)
			trusted := cmdutils.Viper(cmd.Context()).GetBool(trustedFlag)
			description := cmdutils.Viper(cmd.Context()).GetString(descriptionFlag)

			response, _, err := authClient.DefaultApi.CreateClient(cmd.Context()).Body(authclient.ClientOptions{
				Public:                 &public,
				RedirectUris:           cmdutils.Viper(cmd.Context()).GetStringSlice(redirectUriFlag),
				Description:            &description,
				Name:                   args[0],
				Trusted:                &trusted,
				PostLogoutRedirectUris: cmdutils.Viper(cmd.Context()).GetStringSlice(postLogoutRedirectUriFlag),
			}).Execute()
			if err != nil {
				return err
			}

			tableData := pterm.TableData{}
			tableData = append(tableData, []string{pterm.LightCyan("ID"), response.Data.Id})
			tableData = append(tableData, []string{pterm.LightCyan("Name"), response.Data.Name})
			tableData = append(tableData, []string{pterm.LightCyan("Description"), cmdbuilder.StringPointerToString(response.Data.Description)})
			tableData = append(tableData, []string{pterm.LightCyan("Public"), cmdbuilder.BoolPointerToString(response.Data.Public)})
			tableData = append(tableData, []string{pterm.LightCyan("Redirect URIs"), strings.Join(response.Data.RedirectUris, ",")})
			tableData = append(tableData, []string{pterm.LightCyan("Post logout redirect URIs"), strings.Join(response.Data.PostLogoutRedirectUris, ",")})
			return pterm.DefaultTable.
				WithWriter(cmd.OutOrStdout()).
				WithData(tableData).
				Render()
		}),
	)
}
