package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	fctl "github.com/formancehq/fctl/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zitadel/oidc/pkg/client/rp"
	"golang.org/x/oauth2"
)

func init() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
}

func newRootCommand() *cobra.Command {
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	const (
		profileFlag     = "profile"
		configFileFlag  = "config"
		debugFlag       = "debug"
		insecureTlsFlag = "insecure-tls"
	)

	return newCommand("fctl",
		withShortDescription("Formance Control CLI"),
		withSilenceUsage(),
		withPersistentPreRunE(func(cmd *cobra.Command, args []string) (err error) {
			if err = viper.BindPFlags(cmd.Flags()); err != nil {
				return err
			}

			var (
				httpClient         = http.DefaultClient
				config             *fctl.Config
				configManager      *fctl.ConfigManager
				currentProfileName string
			)

			if viper.GetBool(insecureTlsFlag) {
				httpTransport := http.DefaultTransport.(*http.Transport)
				httpTransport.TLSClientConfig = &tls.Config{
					InsecureSkipVerify: true,
				}
				httpClient = &http.Client{
					Transport: httpTransport,
				}
			}

			configManager = fctl.NewConfigManager(viper.GetString(configFileFlag))

			config, err = configManager.Load()
			if err != nil {
				return err
			}

			currentProfileName = config.CurrentProfile
			if currentProfileName == "" {
				currentProfileName = "default"
				config.CurrentProfile = currentProfileName
			}
			fctl.DebugLn(cmd.Context(), cmd.OutOrStdout(), "Current profile:", currentProfileName)
			if selectedProfile := viper.GetString(profileFlag); selectedProfile != "" {
				fctl.DebugLn(cmd.Context(), cmd.OutOrStdout(), "Override profile by flag:", selectedProfile)
				currentProfileName = selectedProfile
			}

			currentProfile := config.GetProfileOrDefault(currentProfileName, &fctl.Profile{
				MembershipURI:  viper.GetString(membershipUriFlag),
				BaseServiceURI: viper.GetString(baseServiceUriFlag),
			})
			fctl.DebugLn(cmd.Context(), cmd.OutOrStdout(), "Selected profile membership uri:", currentProfile.MembershipURI)
			fctl.DebugLn(cmd.Context(), cmd.OutOrStdout(), "Selected base service uri:", currentProfile.BaseServiceURI)

			fctl.IfDebug(cmd.Context(), func() {
				fctl.DebugLn(cmd.Context(), cmd.OutOrStdout(), "Configure http round tripper logger")
				httpClient.Transport = fctl.DebugRoundTripper(httpClient.Transport)
			})
			if currentProfile.Token != nil {
				if currentProfile.Token.Expiry.Before(time.Now()) {
					fctl.DebugLn(cmd.Context(), cmd.OutOrStdout(), "Detect expired auth token against membership, trying to refresh token")
					relyingParty, err := rp.NewRelyingPartyOIDC(currentProfile.MembershipURI, authClient, "",
						"", []string{"openid", "email", "offline_access"}, rp.WithHTTPClient(httpClient))
					if err != nil {
						return err
					}

					newToken, err := relyingParty.OAuthConfig().
						TokenSource(context.WithValue(context.TODO(), oauth2.HTTPClient, httpClient), currentProfile.Token).
						Token()
					if err != nil {
						return err
					}

					currentProfile.Token = newToken

					if err := configManager.UpdateConfig(config); err != nil {
						return err
					}

				} else {
					fctl.DebugLn(cmd.Context(), cmd.OutOrStdout(), "Detect active auth token against membership, reuse it")
				}
			}

			ctx := cmd.Context()
			ctx = fctl.WithHttpClient(ctx, httpClient)
			ctx = fctl.WithCurrentProfile(ctx, currentProfile)
			ctx = fctl.WithConfig(ctx, config)
			ctx = fctl.WithConfigManager(ctx, configManager)
			ctx = fctl.WithCurrentProfileName(ctx, currentProfileName)
			ctx = fctl.WithDebug(ctx, viper.GetBool(debugFlag))
			cmd.SetContext(ctx)

			return nil
		}),
		withChildCommands(
			newLedgerCommand(),
			newPaymentsCommand(),
			newProfilesCommand(),
			newSandboxCommand(),
			newUICommand(),
			newVersionCommand(),
			newLoginCommand(),
		),
		withPersistentStringPFlag(profileFlag, "p", "", "config profile to use"),
		withPersistentStringPFlag(configFileFlag, "c", fmt.Sprintf("%s/.formance/fctl.config", homedir), "Debug mode"),
		withPersistentBoolPFlag(debugFlag, "d", false, "Debug mode"),
		withPersistentBoolFlag(insecureTlsFlag, false, "Insecure TLS"),
	)
}

func Execute() {
	_ = newRootCommand().Execute()
}
