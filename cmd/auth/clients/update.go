package clients

import (
	"strings"

	"github.com/formancehq/auth/authclient"
	internal2 "github.com/formancehq/fctl/cmd/auth/internal"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TODO: This command is a copy/paste of the create command
// We should handle membership side the patch of the client OR
// We should get the client before updating it to get replace informations
func NewUpdateCommand() *cobra.Command {
	const (
		publicFlag                = "public"
		trustedFlag               = "trusted"
		descriptionFlag           = "description"
		redirectUriFlag           = "redirect-uri"
		postLogoutRedirectUriFlag = "post-logout-redirect-uri"
	)
	return cmdbuilder.NewCommand("update [CLIENT_ID]",
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithShortDescription("Update client"),
		cmdbuilder.WithAliases("u", "upd"),
		cmdbuilder.WithBoolFlag(publicFlag, false, "Is client public"),
		cmdbuilder.WithBoolFlag(trustedFlag, false, "Is the client trusted"),
		cmdbuilder.WithStringFlag(descriptionFlag, "", "Client description"),
		cmdbuilder.WithStringSliceFlag(redirectUriFlag, []string{}, "Redirect URIS"),
		cmdbuilder.WithStringSliceFlag(postLogoutRedirectUriFlag, []string{}, "Post logout redirect uris"),
		cmdbuilder.WithRunE(func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}

			authClient, err := internal2.NewAuthClient(cmd.Context(), cfg)
			if err != nil {
				return err
			}

			public := viper.GetBool(publicFlag)
			trusted := viper.GetBool(trustedFlag)
			description := viper.GetString(descriptionFlag)

			response, _, err := authClient.DefaultApi.UpdateClient(cmd.Context(), args[0]).Body(authclient.ClientOptions{
				Public:                 &public,
				RedirectUris:           viper.GetStringSlice(redirectUriFlag),
				Description:            &description,
				Name:                   args[0],
				Trusted:                &trusted,
				PostLogoutRedirectUris: viper.GetStringSlice(postLogoutRedirectUriFlag),
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
