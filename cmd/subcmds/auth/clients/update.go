package clients

import (
	"github.com/formancehq/auth/authclient"
	"github.com/formancehq/fctl/cmd/cmdbuilder"
	"github.com/formancehq/fctl/cmd/config"
	"github.com/formancehq/fctl/cmd/subcmds/auth/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TODO: This command is a copy/paste of the create command
// We should handle membership side the patch of the client OR
// We should get the client before updating it to get replace informations
func NewAuthClientsUpdateCommand() *cobra.Command {
	const (
		publicFlag                = "public"
		trustedFlag               = "trusted"
		descriptionFlag           = "description"
		redirectUriFlag           = "redirect-uri"
		postLogoutRedirectUriFlag = "post-logout-redirect-uri"
	)
	return cmdbuilder.NewCommand("update [CLIENT_ID]",
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
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

			authClient, err := internal.NewAuthClient(cmd, cfg)
			if err != nil {
				return err
			}

			public := viper.GetBool(publicFlag)
			trusted := viper.GetBool(trustedFlag)
			description := viper.GetString(descriptionFlag)

			client, _, err := authClient.DefaultApi.UpdateClient(cmd.Context(), args[0]).Body(authclient.ClientOptions{
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

			internal.PrintAuthClient(cmd.OutOrStdout(), *client.Data)

			return nil
		}),
	)
}
