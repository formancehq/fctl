package cmd

import (
	"github.com/formancehq/auth/authclient"
	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAuthClientsCreateCommand() *cobra.Command {
	const (
		publicFlag                = "public"
		trustedFlag               = "trusted"
		descriptionFlag           = "description"
		redirectUriFlag           = "redirect-uri"
		postLogoutRedirectUriFlag = "post-logout-redirect-uri"
	)
	return newCommand("create",
		withArgs(cobra.ExactArgs(1)),
		withBoolFlag(publicFlag, false, "Is client public"),
		withBoolFlag(trustedFlag, false, "Is the client trusted"),
		withStringFlag(descriptionFlag, "", "Client description"),
		withStringSliceFlag(redirectUriFlag, []string{}, "Redirect URIS"),
		withStringSliceFlag(postLogoutRedirectUriFlag, []string{}, "Post logout redirect uris"),
		withRunE(func(cmd *cobra.Command, args []string) error {
			authClient, err := fctl.NewAuthClientFromContext(cmd.Context())
			if err != nil {
				return err
			}

			public := viper.GetBool(publicFlag)
			trusted := viper.GetBool(trustedFlag)
			description := viper.GetString(descriptionFlag)

			client, _, err := authClient.DefaultApi.CreateClient(cmd.Context()).Body(authclient.ClientOptions{
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

			fctl.PrintAuthClient(cmd.OutOrStdout(), *client.Data)

			return nil
		}),
	)
}
