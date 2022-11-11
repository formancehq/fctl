package clients

import (
	"github.com/formancehq/auth/authclient"
	internal2 "github.com/formancehq/fctl/cmd/auth/internal"
	"github.com/formancehq/fctl/cmd/internal/cmdbuilder"
	"github.com/formancehq/fctl/cmd/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		cmdbuilder.WithArgs(cobra.ExactArgs(1)),
		cmdbuilder.WithBoolFlag(publicFlag, false, "Is client public"),
		cmdbuilder.WithBoolFlag(trustedFlag, false, "Is the client trusted"),
		cmdbuilder.WithStringFlag(descriptionFlag, "", "Client description"),
		cmdbuilder.WithStringSliceFlag(redirectUriFlag, []string{}, "Redirect URIS"),
		cmdbuilder.WithStringSliceFlag(postLogoutRedirectUriFlag, []string{}, "Post logout redirect uris"),
		cmdbuilder.WithShortDescription("Create client"),
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

			internal2.PrintAuthClient(cmd.OutOrStdout(), *client.Data)

			return nil
		}),
	)
}
